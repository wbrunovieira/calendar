# Plano: Integração Nubank — Sync Automático via Open Finance

> **Ver também**: [`open-finance-unified.md`](./open-finance-unified.md) — ambos Nubank e
> Mercado Pago participam do Open Finance e podem ser integrados com **uma única implementação**.
> Este documento detalha especificidades do Nubank; o plano de implementação unificado
> está no documento de referência.

## Contexto

Diferente do Mercado Pago (que exige parsing de CSV), o **Nubank participa do Open Finance
regulado pelo Banco Central** desde 2021. Isso significa que existe uma via oficial e
estruturada para acessar dados de conta, cartão e investimentos programaticamente —
sem engenharia reversa, sem CSV, com webhooks em tempo quase real.

---

## Cobertura por produto

| Produto | Open Finance disponível | Observação |
|---|---|---|
| Conta corrente PF (extrato + saldo) | ✅ | Escopo `accounts` + `transactions` |
| Conta corrente PJ / Nu Empresas | ✅ | Mesmo escopo, contexto business |
| Cartão de crédito PF (fatura + lançamentos) | ✅ | Escopo `credit_cards` + `bill` |
| Investimentos / Caixinha (posição) | ✅ | Escopo `investments` |
| Movimentações de investimentos (compra/venda) | ⚠️ | Escopo `investment_transactions` — disponível via Belvo, ainda não via Pluggy |
| Webhook de nova transação | ✅ | Agregadores disparam `new_transactions_available` |
| Histórico retroativo | ✅ | Últimos 12 meses no momento do consentimento |

**Status: cobertura quase total.** É o banco mais completo de todos para automação.

---

## Estratégia: agregador Open Finance

O Nubank **não oferece API pública direta** para desenvolvedores.
O acesso ocorre pela via regulada do Open Finance Brasil (BACEN), que exige:
- Registro como "Receptora de Dados" no BACEN
- Certificado ICP-Brasil
- Conformidade FAPI 1.0 Advanced
- Manutenção de endpoints de DCR (Dynamic Client Registration)

Isso é inviável para um app pessoal. A solução: usar um **agregador certificado**
que já tem a habilitação e expõe uma API simples de REST + webhook.

### Comparação de agregadores

| Critério | Pluggy | Belvo |
|---|---|---|
| Nubank PF | ✅ | ✅ |
| Nubank PJ | ✅ | ✅ |
| Cartão crédito | ✅ | ✅ (inclui `Bill`) |
| Investimentos | ✅ posição | ✅ posição + movimentos |
| Webhook novos lançamentos | ✅ | ✅ (`new_transactions_available`) |
| Plano gratuito | Trial 14 dias / 20 contas | **25 conexões reais grátis** |
| Custo produção | R$2.500/mês (Basic) | Contato sales (escala) |
| SDK Go | Não oficial | Não oficial |
| Docs qualidade | Alta | Alta |
| **Recomendação** | Produção comercial | **Personal finance / baixo volume** |

**Escolha recomendada: Belvo** para este caso de uso.

Motivos:
- 25 conexões reais gratuitas cobre o caso: Nubank PF + Nubank PJ + cartão = 3–4 conexões
- Suporta `investment_transactions` (Pluggy ainda não)
- API bem documentada com SDK e webhooks

---

## Fluxo de consentimento (uma única vez por conta)

```
Usuário clica "Conectar Nubank"
        │
        ▼
Backend → POST /api/v1/bank-accounts/{id}/connect-open-finance
        │
        ▼
Belvo API → cria sessão de consentimento
        │
        ▼
Frontend abre widget Belvo (hosted UI)
        │
        ▼
Usuário faz login no Nubank dentro do widget + autoriza compartilhamento
        │
        ▼
Belvo retorna link_id (token permanente da conexão)
        │
        ▼
Backend salva link_id → começa sync automático
        │
        ▼
Belvo webhook → historical_update (dados históricos prontos)
        │
        ▼
Backend puxa 12 meses de transações, importa, deduplica
```

**Depois do consentimento inicial**: Belvo monitora a conta e dispara
`new_transactions_available` toda vez que há novidade — sem polling manual.

---

## Arquitetura

### Visão geral

```
┌──────────────────────────────────────────────────────────┐
│                  Webhooks Belvo                           │
│  historical_update / new_transactions_available          │
└───────────────────────┬──────────────────────────────────┘
                        │ POST /webhooks/belvo
                        ▼
        ┌───────────────────────────────┐
        │  HandleBelvoWebhookUseCase    │
        └───────────────┬───────────────┘
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
   SyncAccountUC  SyncCardUC   SyncInvestmentsUC
   (conta+extrato) (fatura)    (posição+movimentos)
          │             │             │
          └─────────────┼─────────────┘
                        ▼
              ProcessTransactionUC   ← mesmo use case do MP
              (dedup por ExternalID)
```

### Princípio central

> O `ProcessTransactionUseCase` é genérico — recebe uma lista normalizada de lançamentos
> independente da origem (Belvo/Nubank, Belvo/MP futuro, upload CSV).
> A dedup sempre usa `ExternalID`.

---

## Domain: `openfinance`

Novo domínio em `internal/domain/openfinance/`.

```
internal/domain/openfinance/
├── link.go          # entidade Link (conexão com agregador)
├── sync_log.go      # entidade SyncLog (igual ao MP, compartilhado ou separado)
└── repository.go    # interfaces
```

### `Link` — representa a conexão autorizada com uma instituição

```go
type Link struct {
    ID              string
    BankAccountID   string        // conta no nosso sistema
    Provider        Provider      // BELVO | PLUGGY
    ExternalLinkID  string        // link_id do agregador
    Institution     string        // "nubank_br", "nubank_pj_br"
    Status          LinkStatus    // ACTIVE | EXPIRED | REVOKED | ERROR
    LastSyncAt      *time.Time
    ConsentExpiresAt *time.Time   // Open Finance consents expire (typically 12 months)
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type Provider string
const (
    ProviderBelvo  Provider = "BELVO"
    ProviderPluggy Provider = "PLUGGY"
)

type LinkStatus string
const (
    LinkStatusActive  LinkStatus = "ACTIVE"
    LinkStatusExpired LinkStatus = "EXPIRED"
    LinkStatusRevoked LinkStatus = "REVOKED"
    LinkStatusError   LinkStatus = "ERROR"
)
```

---

## Infrastructure: `internal/infrastructure/belvo/`

```
internal/infrastructure/belvo/
├── client.go          # interface BelvoClient + implementação HTTP
├── transaction_mapper.go  # normaliza BelvoTransaction → domain.Transaction
├── investment_mapper.go   # normaliza BelvoInvestment → domain.BankAccount/Transaction
└── client_test.go     # testes dos mappers (unitários, sem HTTP)
```

### Interface `BelvoClient`

```go
type BelvoClient interface {
    // Contas
    ListAccounts(ctx context.Context, linkID string) ([]*BelvoAccount, error)
    GetBalance(ctx context.Context, linkID string) ([]*BelvoBalance, error)

    // Transações
    ListTransactions(ctx context.Context, linkID string, from, to time.Time) ([]*BelvoTransaction, error)

    // Cartão de crédito
    ListBills(ctx context.Context, linkID string) ([]*BelvoBill, error)
    ListBillTransactions(ctx context.Context, billID string) ([]*BelvoTransaction, error)

    // Investimentos
    ListInvestments(ctx context.Context, linkID string) ([]*BelvoInvestment, error)
    ListInvestmentTransactions(ctx context.Context, linkID string) ([]*BelvoInvestmentTransaction, error)
}
```

### Mapeamento de tipos Belvo → domínio

**Transações de conta corrente:**

| `transaction.type` (Belvo) | Tipo no sistema | Observação |
|---|---|---|
| `OUTFLOW` | `EXPENSE` | Débitos |
| `INFLOW` | `INCOME` | Créditos |
| `TRANSFER_IN` | TRANSFER (dest) | Transferência recebida |
| `TRANSFER_OUT` | TRANSFER (source) | Transferência enviada |

**Investimentos:**

| `investment.type` (Belvo) | Mapeamento |
|---|---|
| `FIXED_INCOME` | CDB, LCI, LCA → `SAVINGS_BOX` ou tipo específico |
| `MUTUAL_FUND` | Fundos → `FUNDS` |
| `EQUITY` | Ações → `STOCKS` |

**ExternalID**: `"belvo-{transaction.id}"` — prefixo único por provider.

---

## Use Cases

### 1. `ConnectOpenFinanceUseCase`

Inicia o fluxo de consentimento: cria uma sessão no Belvo, retorna a URL do widget.
Salva um `Link` com status `PENDING`.

```go
type ConnectOpenFinanceInput struct {
    BankAccountID string
    Provider      openfinance.Provider
    Institution   string // "nubank_br"
    RedirectURL   string
}

type ConnectOpenFinanceResult struct {
    WidgetURL string // redirecionar frontend
    LinkID    string // ID interno para rastrear
}
```

### 2. `HandleBelvoWebhookUseCase`

Recebe eventos do Belvo e despacha para o sync correto.

Eventos suportados:
- `historical_update` → puxar 12 meses de histórico completo
- `new_transactions_available` → puxar apenas novos lançamentos
- `accounts_completed` → atualizar saldos
- `link_error` → marcar Link como ERROR, notificar usuário

### 3. `SyncNubankAccountUseCase`

Busca transações de conta corrente via `BelvoClient.ListTransactions`,
normaliza e delega para `ProcessTransactionUseCase` (genérico, já existe).

```go
type SyncNubankAccountInput struct {
    LinkID string
    From   time.Time
    To     time.Time
    Mode   SyncMode // FULL_HISTORY | INCREMENTAL
}
```

### 4. `SyncNubankCardUseCase`

Busca faturas do cartão via `BelvoClient.ListBills`, cria/atualiza `Invoice`
no sistema, importa lançamentos individuais da fatura.

Integra com o sistema de `Invoice` já existente no codebase.

### 5. `SyncNubankInvestmentsUseCase`

Atualiza posições de investimento: saldo atual da caixinha, CDB, etc.
Não cria `Transaction` para cada movimento — apenas atualiza `BankAccount.CurrentBalance`.
Para `investment_transactions`, cria transações de rendimento/aplicação.

### 6. `RevokeOpenFinanceLinkUseCase`

Revoga o consentimento no Belvo e marca o `Link` como `REVOKED`.
Necessário quando o usuário quer desconectar a conta.

---

## HTTP Handlers

```
POST /api/v1/bank-accounts/{id}/connect-open-finance   → ConnectOpenFinanceUseCase
GET  /api/v1/bank-accounts/{id}/open-finance-link      → status do Link
DELETE /api/v1/bank-accounts/{id}/open-finance-link    → RevokeOpenFinanceLinkUseCase
POST /api/v1/webhooks/belvo                            → HandleBelvoWebhookUseCase
```

O endpoint de webhook é público (sem JWT), mas valida assinatura HMAC do Belvo.

---

## Banco de dados

```sql
CREATE TABLE IF NOT EXISTS finance.open_finance_links (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_account_id     UUID NOT NULL REFERENCES finance.bank_accounts(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL CHECK (provider IN ('BELVO', 'PLUGGY')),
    external_link_id    TEXT NOT NULL,
    institution         TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'ACTIVE',
    last_sync_at        TIMESTAMP,
    consent_expires_at  TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT now(),
    updated_at          TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (bank_account_id, provider)
);

CREATE INDEX idx_open_finance_links_external ON finance.open_finance_links (external_link_id);
```

A coluna `external_id` em `finance.transactions` já existe — dedup usa `"belvo-{id}"`.

---

## Configuração

```env
BELVO_SECRET_ID=...          # credencial Belvo
BELVO_SECRET_PASSWORD=...    # credencial Belvo
BELVO_WEBHOOK_SECRET=...     # para validar assinatura dos webhooks
BELVO_ENVIRONMENT=sandbox    # sandbox | production
NUBANK_PF_LINK_ID=...        # preenchido após consentimento inicial
NUBANK_PJ_LINK_ID=...        # preenchido após consentimento inicial
```

---

## Testes

### Unitários — `internal/application/usecases/`

**`nubank_sync_test.go`**

| Teste | O que valida |
|---|---|
| `TestSyncAccount_CreatesNewTransactions` | Lançamentos novos viram transações |
| `TestSyncAccount_SkipsDuplicateByExternalID` | `belvo-{id}` não duplica |
| `TestSyncAccount_MapsInflowAsIncome` | `INFLOW` → INCOME |
| `TestSyncAccount_MapsOutflowAsExpense` | `OUTFLOW` → EXPENSE |
| `TestSyncCard_CreatesInvoiceIfNotExists` | Nova fatura cria Invoice |
| `TestSyncCard_UpdatesExistingInvoice` | Fatura já existe → atualiza |
| `TestSyncCard_ImportsIndividualPurchases` | Lançamentos do cartão viram transações |
| `TestSyncInvestments_UpdatesAccountBalance` | Saldo da caixinha atualiza |
| `TestHandleWebhook_HistoricalUpdate_TriggersFull` | `historical_update` → sync completo |
| `TestHandleWebhook_NewTransactions_TriggersIncremental` | `new_transactions` → sync parcial |
| `TestHandleWebhook_LinkError_MarksLinkAsError` | Erro → Link.Status = ERROR |
| `TestHandleWebhook_RejectsInvalidSignature` | HMAC inválido → rejeitado |
| `TestConnectOpenFinance_CreatesPendingLink` | Link criado com status PENDING |

**`internal/infrastructure/belvo/transaction_mapper_test.go`**

| Teste | O que valida |
|---|---|
| `TestMapTransaction_Inflow` | INFLOW → INCOME corretamente |
| `TestMapTransaction_Outflow` | OUTFLOW → EXPENSE corretamente |
| `TestMapTransaction_Transfer` | TRANSFER_IN/OUT mapeado |
| `TestMapTransaction_ExternalIDPrefix` | ExternalID começa com "belvo-" |
| `TestMapBill_FieldsCorretos` | due_date, total, minimum mapeados |
| `TestMapInvestment_Caixinha` | Tipo FIXED_INCOME mapeado corretamente |

### E2E / Integração — `internal/test/`

**`nubank_sync_e2e_test.go`** (tag `// +build integration`)

| Teste | O que valida |
|---|---|
| `TestE2E_ConsentFlow_CreatesLink` | POST connect → Link salvo no DB |
| `TestE2E_WebhookHistorical_ImportsFullHistory` | Webhook → 12 meses importados |
| `TestE2E_WebhookNewTx_ImportsOnlyNew` | Webhook incremental → sem duplicatas |
| `TestE2E_CardSync_CreatesInvoiceAndTransactions` | Fatura + lançamentos no DB |
| `TestE2E_InvestmentSync_UpdatesBalance` | Caixinha saldo atualizado |
| `TestE2E_RevokeLink_MarksRevoked` | Revogação refletida no DB |
| `TestE2E_BalanceRecalcAfterSync` | Saldo da conta correto após sync |

BelvoClient mockado via interface — testes não fazem chamadas reais à API.

---

## Comparação com integração Mercado Pago

| Aspecto | Mercado Pago | Nubank (Belvo) |
|---|---|---|
| Mecanismo | CSV via API própria | Open Finance regulado via agregador |
| Latência | D-1 (relatório do dia anterior) | Quase real-time (webhook em minutos) |
| Cartão crédito | Manual (sem API) | ✅ Automático |
| Investimentos | Só rendimentos no CSV | ✅ Posição + movimentos |
| Consentimento | Access token pessoal | OAuth + widget de consentimento |
| Expiração | Token sem validade definida | Consentimento expira em ~12 meses (renovável) |
| Custo | Gratuito | Gratuito até 25 conexões (Belvo) |

---

## Plano de implementação

| Fase | O que fazer | Dependências |
|---|---|---|
| **1 — Domain** | Criar `internal/domain/openfinance/` (Link + Repository) | — |
| **2 — BelvoClient** | Interface + implementação HTTP + mappers | — |
| **3 — Mappers tests** | Testes unitários dos mappers Belvo | Fase 2 |
| **4 — ConnectUC** | Fluxo de consentimento + widget URL | Fase 1 + 2 |
| **5 — SyncAccountUC** | Sync conta corrente PF e PJ | Fase 2 |
| **6 — SyncCardUC** | Sync cartão + faturas | Fase 2 + sistema Invoice |
| **7 — SyncInvestmentsUC** | Sync caixinha + CDB | Fase 2 |
| **8 — WebhookUC** | Handler de eventos Belvo + despacho | Fases 4–7 |
| **9 — Persistence** | Migration + `open_finance_link_repository.go` | Fase 1 |
| **10 — HTTP handlers** | Endpoints connect, webhook, revoke | Fase 8 + 9 |
| **11 — E2E tests** | Testes completos com Belvo mockado | Fases 1–10 |
| **12 — Sandbox Belvo** | Testar com credenciais sandbox do Belvo | Fase 11 |
| **13 — Produção** | Consentimento real, monitorar primeiros syncs | Fase 12 |

---

## Decisões de design

**Por que Belvo e não Pluggy?**
Para um app pessoal com 2-4 contas, o free tier do Belvo (25 conexões reais) cobre
indefinidamente. Pluggy requer R$2.500/mês a partir do plano pago — inviável para
uso pessoal. Se o app evoluir para multi-usuário, Pluggy tem melhor escalabilidade.

**Por que não implementar Open Finance diretamente?**
Requer registro no BACEN como instituição receptora, certificado ICP-Brasil,
conformidade FAPI 1.0 Advanced e manutenção de infraestrutura regulatória.
Custo e complexidade são ordens de magnitude maiores que usar um agregador.

**Por que o consentimento expira?**
O Open Finance Brasil exige que o consentimento seja renovado periodicamente
(tipicamente 12 meses) pelo próprio usuário, como medida de segurança regulatória.
O sistema deve monitorar `consent_expires_at` e notificar antes de expirar.

**Por que o cartão de crédito é totalmente automático aqui mas não no MP?**
O Nubank expõe dados de cartão via Open Finance (`credit_cards` scope).
O Mercado Pago não tem equivalente — o cartão MP não está no Open Finance
e não tem API de fatura. São produtos distintos da mesma finteca,
mas com graus diferentes de abertura.

**Reutilização do ProcessTransactionUseCase:**
O mesmo use case que processa CSVs do MP pode ser reutilizado aqui,
desde que a entrada seja normalizada para o formato interno `[]domain.Transaction`.
Isso mantém a lógica de dedup, criação e recalculo de saldo em um único lugar.
