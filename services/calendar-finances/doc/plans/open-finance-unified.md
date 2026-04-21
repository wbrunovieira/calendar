# Plano: Integração Open Finance Unificada (Nubank + Mercado Pago)

## Descoberta central

Ao investigar a viabilidade de automatizar o Nubank, confirmou-se que
**Mercado Pago também participa do Open Finance regulado pelo BACEN**.

Ambas as instituições expõem **exatamente os mesmos escopos** via agregadores:

| Produto | Nubank | Mercado Pago | Via |
|---|---|---|---|
| Conta corrente PF (saldo + extrato) | ✅ | ✅ | Open Finance |
| Conta PJ / empresas | ✅ | ✅ | Open Finance |
| Cartão de crédito (lançamentos + fatura) | ✅ | ✅ | Open Finance |
| Investimentos / Caixinha (posição) | ✅ | ✅ | Open Finance |
| Movimentos de investimento | ❌ | ❌ | Não disponível ainda |
| Webhook de novas transações | ✅ | ✅ | Agregador |

**Consequência arquitetural**: uma única integração com um agregador Open Finance
serve para Nubank PF, Nubank PJ, MP pessoal e MP empresas — mesmo código,
mesmo fluxo de consentimento, mesmos webhooks.

---

## Impacto: o que muda em relação ao plano CSV

| Aspecto | Plano CSV (MP) | Open Finance (ambos) |
|---|---|---|
| Cartão MP lançamentos | ❌ Sem API, manual | ✅ Automático |
| Cartão Nubank lançamentos | ✅ Via OF | ✅ Via OF (igual) |
| Latência | D-1 (relatório CSV) | Minutos (webhook) |
| Cofrinhos / Caixinha | Apenas movimentos no CSV | ✅ Posição real |
| Contas PJ | ✅ Parcial via CSV | ✅ Completo via OF |
| Código necessário | Um client por banco | **Um client para todos** |

O plano CSV do MP passa a ser **fallback de contingência** caso o consentimento
expire ou o usuário não queira renovar.

---

## Escolha de agregador

### Pluggy vs Belvo — decisão final

| Critério | Pluggy | Belvo |
|---|---|---|
| MP no Open Finance | ✅ Confirmado | ✅ Parceiro direto do MP |
| Nubank PF + PJ | ✅ | ✅ |
| Cartão crédito ambos | ✅ | ✅ (inclui `Bill` detalhado) |
| Investimentos | ✅ posição | ✅ posição + movimentos |
| Webhook real-time | ✅ | ✅ `new_transactions_available` |
| Free tier | Trial 14 dias / 20 contas | **25 conexões reais grátis** |
| Custo pago | R$2.500/mês | US$1.000/mês (Launch) |

**Escolha: Belvo** para este projeto.

Motivos:
- 25 conexões reais gratuitas cobre indefinidamente: Nubank PF + Nubank PJ + MP PF + MP PJ = 4 conexões
- Belvo tem **parceria direta com o Mercado Pago** (MP usa Belvo para crédito)
- Suporte a `investment_transactions` (Pluggy ainda não)
- Relação com MP aumenta probabilidade de suporte prioritário

### Custo: análise do plano gratuito

| Plano | Preço | Links | Uso |
|---|---|---|---|
| Test | Grátis | 25 reais | Desenvolvimento / uso pessoal |
| Launch | US$1.000/mês | Ilimitado | Produção comercial |
| Growth | Negociação | Ilimitado | Escala |

**Nossa necessidade: 4 links** — bem dentro do limite de 25 do plano gratuito.

#### Riscos do plano gratuito

1. **Posicionamento "Test"**: o plano gratuito se chama "Test" e a Belvo posiciona o "Launch" como plano de produção. Para um app pessoal com 1 usuário, US$1.000/mês é inviável.

2. **Open Finance pode exigir ativação comercial**: conectores Open Finance (que cobrem Nubank e MP) podem estar disponíveis apenas a partir do plano pago ou via contato com o time comercial — isso não está explícito na documentação pública.

#### Ação necessária antes de implementar

Enviar email para o suporte da Belvo com a seguinte pergunta:

> "I have a personal finance app, single user, needing 4 permanent links via Open Finance (Nubank PF, Nubank PJ, Mercado Pago PF, Mercado Pago PJ). Can the Test plan be used for production with this minimal volume, or is there a cost?"

Dado que o MP tem parceria direta com a Belvo e o volume é mínimo (4 links, 1 usuário), há boa chance de liberação informal ou desconto comercial.

#### Plano B se Belvo exigir plano pago

| Conta | Alternativa |
|---|---|
| Mercado Pago (conta + cartão) | CSV automático via Account Money Report API (gratuito) |
| Nubank PF + PJ | Pluggy (trial 14 dias, plano pago mais acessível para baixo volume) |
| Investimentos / Caixinha | Atualização manual por enquanto |

A arquitetura documentada aqui permanece válida — troca-se apenas o agregador sem alterar domain ou use cases.

---

## Arquitetura unificada

```
┌─────────────────────────────────────────────────────────────┐
│              Belvo Webhooks (todas as instituições)          │
│  historical_update / new_transactions_available / error     │
└──────────────────────────┬──────────────────────────────────┘
                           │ POST /webhooks/belvo
                           ▼
           ┌───────────────────────────────┐
           │  HandleBelvoWebhookUseCase    │
           └──────────────┬────────────────┘
                          │ despacha por tipo de evento + instituição
          ┌───────────────┼───────────────────┐
          ▼               ▼                   ▼
  SyncAccountUC    SyncCardUC        SyncInvestmentsUC
  (conta corrente)  (fatura+lançamentos) (posição)
          │               │                   │
          └───────────────┼───────────────────┘
                          ▼
              NormalizeTransactionsUseCase   ← normaliza para domínio interno
                          │
                          ▼
              ProcessTransactionUseCase      ← dedup ExternalID, cria tx, recalcula
```

### Contas suportadas

| Conta no sistema | Instituição Belvo | Contexto |
|---|---|---|
| Nubank (PF) | `nubank_br` | Personal |
| Nubank PJ (empresa) | `nubank_br` | Business |
| Mercado Pago (PF) | `mercadopago_br` | Personal |
| Mercado Pago Empresas | `mercadopago_br` | Business |
| Cartão Nubank | Mesmo link do Nubank PF | `credit_cards` scope |
| Cartão Mercado Pago | Mesmo link do MP PF | `credit_cards` scope |

**Um link Belvo por conta bancária** cobre a conta corrente + cartão vinculado.

---

## Domain: `openfinance`

```
internal/domain/openfinance/
├── link.go        # Link: conexão autorizada com uma instituição
├── sync_log.go    # SyncLog: rastreio de cada execução
└── repository.go  # interfaces
```

### `Link`

```go
type Link struct {
    ID               string
    BankAccountID    string       // conta corrente no nosso sistema
    Provider         Provider     // BELVO
    ExternalLinkID   string       // link_id do Belvo
    Institution      Institution  // NUBANK_PF | NUBANK_PJ | MP_PF | MP_PJ
    Status           LinkStatus   // ACTIVE | EXPIRED | REVOKED | ERROR
    LastSyncAt       *time.Time
    ConsentExpiresAt *time.Time   // Open Finance consents ~12 meses
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type Institution string
const (
    InstitutionNubankPF  Institution = "nubank_pf"
    InstitutionNubankPJ  Institution = "nubank_pj"
    InstitutionMPPF      Institution = "mercadopago_pf"
    InstitutionMPPJ      Institution = "mercadopago_pj"
)
```

### `SyncLog`

```go
type SyncLog struct {
    ID            string
    LinkID        string
    Event         SyncEvent   // HISTORICAL | INCREMENTAL | INVESTMENTS | CARD
    Status        SyncStatus  // PENDING | RUNNING | DONE | FAILED
    Imported      int
    Skipped       int
    Errors        int
    ErrorDetail   *string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

---

## Infrastructure: `internal/infrastructure/belvo/`

```
internal/infrastructure/belvo/
├── client.go                  # interface BelvoClient + HTTP
├── transaction_mapper.go      # BelvoTransaction → []domain.NormalizedEntry
├── card_mapper.go             # BelvoBill + transactions → Invoice + []NormalizedEntry
├── investment_mapper.go       # BelvoInvestment → balance update
└── *_mapper_test.go           # testes unitários dos mappers (sem HTTP)
```

### Interface `BelvoClient`

```go
type BelvoClient interface {
    ListAccounts(ctx context.Context, linkID string) ([]*BelvoAccount, error)
    ListTransactions(ctx context.Context, linkID string, from, to time.Time) ([]*BelvoTransaction, error)
    ListBills(ctx context.Context, linkID string) ([]*BelvoBill, error)
    ListBillTransactions(ctx context.Context, billID string) ([]*BelvoTransaction, error)
    ListInvestments(ctx context.Context, linkID string) ([]*BelvoInvestment, error)
    ListInvestmentTransactions(ctx context.Context, linkID string) ([]*BelvoInvestmentTransaction, error)
}
```

### `NormalizedEntry` — formato interno agnóstico de origem

```go
// NormalizedEntry é o contrato entre os mappers e ProcessTransactionUseCase.
// Mesmo formato para Belvo, CSV do MP, ou qualquer fonte futura.
type NormalizedEntry struct {
    ExternalID  string    // "belvo-{id}" | "mp-{reference_id}"
    Date        time.Time
    Amount      float64   // sempre positivo
    Direction   Direction // CREDIT | DEBIT
    Description string
    CategoryHint string   // sugestão de categoria (pode ser ignorada)
}
```

### Mapeamento Belvo → NormalizedEntry

| Campo Belvo | Mapeamento |
|---|---|
| `transaction.id` | `ExternalID = "belvo-{id}"` |
| `type = INFLOW` | `Direction = CREDIT` → INCOME |
| `type = OUTFLOW` | `Direction = DEBIT` → EXPENSE |
| `type = TRANSFER_IN` | `Direction = CREDIT` → TRANSFER destino |
| `type = TRANSFER_OUT` | `Direction = DEBIT` → TRANSFER origem |
| `amount` | `Amount` (sempre positivo no Belvo) |
| `description` | `Description` |
| `category` | `CategoryHint` (melhor esforço) |

---

## Use Cases

### 1. `ConnectOpenFinanceUseCase`
Cria sessão de consentimento no Belvo, retorna URL do widget, salva Link PENDING.

### 2. `HandleBelvoWebhookUseCase`
Recebe evento, valida HMAC, despacha:
- `historical_update` → `SyncAccountUseCase(mode=FULL, from=12meses atrás)`
- `new_transactions_available` → `SyncAccountUseCase(mode=INCREMENTAL, from=last_sync)`
- `accounts_completed` → `SyncInvestmentsUseCase`
- `link_error` → marca Link como ERROR

### 3. `SyncAccountUseCase`
Busca transações via Belvo, mapeia para NormalizedEntry, delega para ProcessTransactionUseCase.
Funciona para qualquer instituição (Nubank ou MP) — a diferença está no Link.

### 4. `SyncCardUseCase`
Busca faturas do cartão, mapeia para Invoice + NormalizedEntries.
Integra com o sistema de Invoice já existente.

### 5. `SyncInvestmentsUseCase`
Atualiza `BankAccount.CurrentBalance` para caixinhas/investimentos.

### 6. `RevokeOpenFinanceLinkUseCase`
Revoga no Belvo, marca Link como REVOKED.

---

## HTTP Handlers

```
POST   /api/v1/bank-accounts/{id}/connect-open-finance  → ConnectOpenFinanceUseCase
GET    /api/v1/bank-accounts/{id}/open-finance-link     → status do Link + last_sync
DELETE /api/v1/bank-accounts/{id}/open-finance-link     → RevokeOpenFinanceLinkUseCase
POST   /api/v1/webhooks/belvo                           → HandleBelvoWebhookUseCase (sem JWT, valida HMAC)
```

---

## Banco de dados

```sql
CREATE TABLE IF NOT EXISTS finance.open_finance_links (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_account_id   UUID NOT NULL REFERENCES finance.bank_accounts(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL DEFAULT 'BELVO',
    external_link_id  TEXT NOT NULL,
    institution       TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'ACTIVE',
    last_sync_at      TIMESTAMP,
    consent_expires_at TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT now(),
    updated_at        TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (bank_account_id, provider)
);

CREATE TABLE IF NOT EXISTS finance.open_finance_sync_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id        UUID NOT NULL REFERENCES finance.open_finance_links(id),
    event          TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'PENDING',
    imported       INT NOT NULL DEFAULT 0,
    skipped        INT NOT NULL DEFAULT 0,
    errors         INT NOT NULL DEFAULT 0,
    error_detail   TEXT,
    created_at     TIMESTAMP NOT NULL DEFAULT now(),
    updated_at     TIMESTAMP NOT NULL DEFAULT now()
);
```

---

## Configuração

```env
BELVO_SECRET_ID=...
BELVO_SECRET_PASSWORD=...
BELVO_WEBHOOK_SECRET=...
BELVO_ENVIRONMENT=sandbox   # sandbox | production
OF_ENABLED=true             # feature flag
```

---

## Testes

### Unitários

| Arquivo | Testes |
|---|---|
| `transaction_mapper_test.go` | INFLOW/OUTFLOW/TRANSFER mapeados, ExternalID prefixado, amount sempre positivo |
| `card_mapper_test.go` | fatura mapeada para Invoice, lançamentos para NormalizedEntry |
| `investment_mapper_test.go` | saldo da caixinha extraído corretamente |
| `sync_account_test.go` | dedup funciona, novos criados, antigos skipped |
| `sync_card_test.go` | Invoice criada/atualizada, lançamentos importados |
| `webhook_handler_test.go` | HMAC inválido rejeitado, eventos despachados corretamente |
| `connect_test.go` | Link PENDING criado, URL de widget retornada |

### E2E (tag `integration`)

| Teste | Valida |
|---|---|
| `TestE2E_HistoricalSync_NubankPF` | 12 meses importados, dedup no re-run |
| `TestE2E_HistoricalSync_MPPF` | mesma lógica para conta MP |
| `TestE2E_IncrementalSync_OnlyNew` | webhook new_tx → só novidades |
| `TestE2E_CardSync_InvoiceAndItems` | fatura + lançamentos no DB |
| `TestE2E_InvestmentSync_BalanceUpdated` | caixinha saldo correto |
| `TestE2E_ConsentExpiry_LinkMarkedExpired` | expiração detectada e notificada |
| `TestE2E_Revoke_LinkRevoked` | revogação refletida |

BelvoClient mockado via interface em todos os testes.

---

## Plano de implementação

| Fase | O que fazer | Prioridade |
|---|---|---|
| **1** | Domain `openfinance` (Link + SyncLog + repos) | Alta |
| **2** | `BelvoClient` interface + mappers | Alta |
| **3** | Testes unitários dos mappers | Alta |
| **4** | `ConnectOpenFinanceUseCase` + handler | Alta |
| **5** | `SyncAccountUseCase` (conta corrente) | Alta |
| **6** | `HandleBelvoWebhookUseCase` | Alta |
| **7** | Migration + repositórios | Alta |
| **8** | `SyncCardUseCase` (fatura + lançamentos) | Média |
| **9** | `SyncInvestmentsUseCase` (caixinha) | Média |
| **10** | `RevokeOpenFinanceLinkUseCase` | Média |
| **11** | E2E tests completos | Alta |
| **12** | Teste com sandbox Belvo (ambas instituições) | Alta |
| **13** | Consentimento real, monitorar primeiros syncs | — |
| **14** | Deprecar sync CSV do MP (manter como fallback) | Baixa |

---

## Consentimento: renovação anual

O Open Finance Brasil exige renovação de consentimento pelo usuário ~a cada 12 meses.

O sistema deve:
1. Monitorar `consent_expires_at` em todos os Links
2. Notificar o usuário 30 dias antes da expiração
3. Exibir no frontend o status do Link (ACTIVE / EXPIRANDO / EXPIRED)
4. Ao expirar, continuar com o fallback CSV do MP (se disponível)
   ou simplesmente parar o sync automático até renovação

---

## Decisões de design

**Por que um `NormalizedEntry` em vez de criar `transaction.Transaction` diretamente nos mappers?**
Os mappers Belvo não devem conhecer lógica de negócio (recalculate, categorias, etc.).
`NormalizedEntry` é um DTO simples de transporte. `ProcessTransactionUseCase`
é quem decide o que fazer com ele — e é reutilizável para CSV, Belvo, ou qualquer fonte.

**Por que não criar um domain `belvo` em vez de `openfinance`?**
`openfinance` é o conceito de negócio. `belvo` é um detalhe de implementação.
Se amanhã Pluggy se tornar mais barato ou Belvo sair do ar, só a infra muda —
o domain e os use cases continuam iguais.

**Por que o fallback CSV do MP continua existindo?**
1. Usuários que não quiserem fazer o fluxo de consentimento
2. Expiração de consentimento sem renovação imediata
3. Se o Belvo tiver indisponibilidade, o CSV garante continuidade
