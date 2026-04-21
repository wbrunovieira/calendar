# Plano: Integração Mercado Pago — Sync Automático de Lançamentos

---

## ⚡ Atualização: Mercado Pago está no Open Finance

> **Descoberta posterior à criação deste documento:**
> O Mercado Pago **participa do Open Finance regulado pelo BACEN** como transmissor de dados,
> confirmado via Pluggy (cobertura verificada em abr/2026).
>
> Isso muda completamente a abordagem — em vez de CSV via API proprietária do MP,
> é possível usar **o mesmo agregador Open Finance do Nubank** (Belvo ou Pluggy)
> para cobrir MP e Nubank com uma única integração, mesmo código, mesmos webhooks.
>
> **A abordagem CSV descrita neste documento passa a ser fallback/legado.**
> O plano unificado está em [`open-finance-unified.md`](./open-finance-unified.md).

### Cobertura MP via Open Finance (Pluggy confirmado)

| Produto | Open Finance | Observação |
|---|---|---|
| Conta corrente (saldo + extrato) | ✅ | Accounts + Transactions |
| Cartão Mercado Pago (lançamentos da fatura) | ✅ | Credit Cards — **antes sem API, agora disponível** |
| Cofrinhos (posição) | ✅ | Investments |
| Rendimentos | ✅ | Dentro de Transactions |
| Movimentos de investimento | ❌ | Investment Transactions — não disponível ainda |
| Conta PJ / MP Empresas | ✅ | Contexto Business confirmado |

**Isso resolve o maior gap anterior**: os lançamentos individuais do cartão Mercado Pago
(que antes exigiam upload manual de CSV de fatura) agora são acessíveis via Open Finance.

---

## Cobertura por produto MP

> Análise baseada nos CSVs reais do extrato (4 meses, 94 tipos de transação distintos)
> e na pesquisa da API pública do Mercado Pago.
> **Seção mantida como referência para o modo fallback (CSV).**

### Conta principal (Mercado Pago Checking)

| O que | Automatizável | Como |
|---|---|---|
| Rendimentos diários | ✅ | CSV automático via API |
| Pix enviados/recebidos | ✅ | CSV automático via API |
| Pagamentos (QR, app, conta) | ✅ | CSV automático via API |
| Transferências entre contas | ✅ | CSV automático via API |

**Status: 100% automatizável** — é o escopo principal deste plano.

---

### Cofrinhos (Reservas / Caixinhas MP)

Os cofrinhos aparecem no CSV da conta principal como dois tipos de entrada:

| Tipo no CSV | Significado | O que fazer |
|---|---|---|
| `Reserva por gastos [nome]` | Depósito automático no cofrinho | Registrar como TRANSFER: Conta → Cofrinho |
| `Reserva programada [nome]` | Depósito agendado no cofrinho | Registrar como TRANSFER: Conta → Cofrinho |
| `Dinheiro reservado [nome]` | Depósito manual maior no cofrinho | Registrar como TRANSFER: Conta → Cofrinho |
| `Dinheiro retirado [nome]` | Saque do cofrinho para a conta | Registrar como TRANSFER: Cofrinho → Conta |

**O saldo do cofrinho em si não tem API** — não é possível consultar o saldo programaticamente.
Mas como todos os depósitos e saques passam pelo CSV da conta principal,
o saldo pode ser **derivado** acumulando as transferências desde a criação.

| Cobertura | Detalhe |
|---|---|
| Movimentos (in/out) | ✅ Estão no CSV principal |
| Saldo atual em tempo real | ❌ Sem API — somente app |
| Rendimentos do cofrinho | ✅ Entram como `Rendimentos` no CSV principal (mesma linha) |

**Status: movimentos 100% automatizáveis; saldo derivado (sem confirmação em tempo real).**

**Mudança no plano existente**: hoje ignoramos `Reserva por gastos` e `Dinheiro reservado`.
Para automatizar cofrinhos, passaríamos a processá-los como TRANSFER entre as contas,
em vez de ignorar. Requer que o usuário tenha o cofrinho cadastrado como conta no sistema.

---

### Cartão de Crédito Mercado Pago

O cartão tem dois fluxos distintos que precisam de tratamento separado:

#### Fluxo 1 — Pagamento da fatura (já capturado)

Quando você paga a fatura, aparece no CSV da conta principal como:
```
Pagamento Cartão de crédito  -2408,97
```
Este já é capturado como EXPENSE no sync automático da conta principal. ✅

#### Fluxo 2 — Lançamentos individuais do cartão (fatura)

Os gastos realizados *com* o cartão (o que aparece na fatura) estão em um
**extrato separado** dentro do app do MP. **Não existe API pública para isso.**

| Cobertura | Detalhe |
|---|---|
| Pagamento da fatura | ✅ No CSV da conta principal |
| Lançamentos individuais | ❌ Sem API — extrato separado, somente app/download manual |
| Webhook de nova compra | ❌ Não disponível para conta pessoal |

**Status: parcialmente automatizável.**
- Pagamento da fatura → automático via CSV da conta
- Lançamentos individuais → **continua manual** (download do extrato do cartão no app)
  — mesma mecânica do upload manual, porém com CSV de formato diferente do extrato da conta

O sistema já possui a entidade `Invoice` e o fluxo de faturas.
O que falta é um parser para o formato CSV específico do cartão MP
e um endpoint de upload de fatura: `POST /invoices/{id}/import-statement`.

---

### Investimentos (CDB, conta remunerada)

| Produto | Cobertura | Como aparece |
|---|---|---|
| Rendimentos da conta (100% CDI) | ✅ | `Rendimentos` no CSV principal — já capturado |
| Rendimentos dos cofrinhos | ✅ | `Rendimentos` no CSV principal — mesma linha |
| CDB Mercado Pago | ⚠️ | Provavelmente aparece como `Rendimentos` — sem dados suficientes |
| Posição/saldo dos investimentos | ❌ | Sem API — somente app |
| Resgate de CDB | ⚠️ | Provavelmente como `Dinheiro retirado` — sem dados suficientes |

**Status: rendimentos automatizáveis; posição sem API.**

O MP não expõe saldo de investimentos via API. Para CDB especificamente,
como é um produto mais recente, seria necessário analisar um extrato real para
mapear os tipos de transação corretos antes de incluir no parser.

---

### Resumo geral por produto

| Produto | Automatização | Esforço adicional |
|---|---|---|
| Conta principal | ✅ 100% | Escopo principal deste plano |
| Cofrinhos (movimentos) | ✅ Via CSV principal | Pequeno: tratar Reserva/Dinheiro retirado como TRANSFER |
| Cofrinhos (saldo real-time) | ❌ Sem API | — |
| Cartão — pagamento fatura | ✅ Via CSV principal | Zero (já capturado como EXPENSE) |
| Cartão — lançamentos fatura | 🔶 Manual assistido | Médio: parser para CSV do cartão + endpoint de upload |
| Rendimentos (conta+cofrinhos) | ✅ Via CSV principal | Zero (já capturado) |
| CDB posição/saldo | ❌ Sem API | — |

---

## Contexto e objetivo

O Mercado Pago não oferece uma API de Open Banking completa para consumidores.
A API pública é voltada para *vendedores* (receber pagamentos via checkout/QR).
Porém, existe a **Account Money Report API** — a mesma fonte dos CSVs baixados manualmente —
que pode ser gerada e baixada programaticamente.

**Objetivo**: eliminar o lançamento manual de transações do MP, mantendo o fluxo manual
como fallback. Ambos os caminhos devem compartilhar o mesmo processador de lançamentos.

---

## O que a API do MP permite

| Recurso | Disponível | Observação |
|---|---|---|
| Gerar relatório CSV via API | ✅ | `POST /v1/account/settlement_report` |
| Baixar relatório gerado | ✅ | `GET /v1/account/settlement_report/list` |
| Webhook quando relatório fica pronto | ✅ | Notificação via POST no endpoint registrado |
| Listar pagamentos recebidos (checkout) | ✅ | `GET /v1/payments/search` — só vendedor |
| Feed de lançamentos em tempo real | ❌ | Não existe |
| Leitura de Pix avulsos como consumidor | ❌ | Não existe na API pública |
| OAuth para conta pessoal | ✅ | Fluxo padrão para obter access_token do usuário |

**Conclusão**: o ciclo é `solicitar relatório → webhook avisa → baixar CSV → processar`.
Cadência mínima: 1x por dia (relatório do dia anterior).

---

## Arquitetura

### Visão geral

```
┌─────────────────────────────────────────────────────────────────┐
│                         Scheduler (diário)                       │
│   00:05 UTC → RequestReportUseCase (D-1)                        │
└───────────────────────────┬─────────────────────────────────────┘
                            │ POST /v1/account/settlement_report
                            ▼
              ┌─────────────────────────┐
              │   Mercado Pago API      │
              └──────────┬──────────────┘
                         │ webhook (report ready)
                         ▼
        ┌────────────────────────────────────┐
        │  POST /webhooks/mercadopago/report  │  ← HTTP handler
        └──────────────┬─────────────────────┘
                       │
                       ▼
          ┌────────────────────────────┐
          │  ProcessReportUseCase      │  ← use case compartilhado
          │  (parse CSV + dedup + create) │
          └────────────────────────────┘
                       ▲
                       │ mesmo use case
        ┌──────────────┘
        │
┌───────────────────┐
│  Manual CSV Upload │  ← POST /bank-accounts/{id}/import-statement
│  (fallback)        │
└───────────────────┘
```

### Princípio central

> `ProcessReportUseCase` é a única fonte de verdade para importar lançamentos.
> A origem (webhook automático vs upload manual) é irrelevante para o processamento.

---

## Domain: `mpsync`

Novo domínio em `internal/domain/mpsync/`.

```
internal/domain/mpsync/
├── sync_log.go          # entidade SyncLog
└── repository.go        # interface do repositório
```

### `SyncLog` — rastreia cada execução de sync

```go
type SyncLog struct {
    ID              string
    BankAccountID   string
    ReportID        string        // ID do relatório no MP
    ReportDate      time.Time     // data do período do relatório
    Source          SyncSource    // AUTOMATIC | MANUAL
    Status          SyncStatus    // PENDING | DOWNLOADING | PROCESSING | DONE | FAILED
    Imported        int           // lançamentos novos criados
    Skipped         int           // duplicatas ignoradas (já existiam)
    Errors          int
    ErrorDetail     *string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type SyncSource string
const (
    SyncSourceAutomatic SyncSource = "AUTOMATIC"
    SyncSourceManual    SyncSource = "MANUAL"
)

type SyncStatus string
const (
    SyncStatusPending     SyncStatus = "PENDING"
    SyncStatusDownloading SyncStatus = "DOWNLOADING"
    SyncStatusProcessing  SyncStatus = "PROCESSING"
    SyncStatusDone        SyncStatus = "DONE"
    SyncStatusFailed      SyncStatus = "FAILED"
)
```

### `Repository` interface

```go
type Repository interface {
    Save(log *SyncLog) error
    FindByID(id string) (*SyncLog, error)
    FindByReportID(reportID string) (*SyncLog, error)
    ListByBankAccount(bankAccountID string, limit int) ([]*SyncLog, error)
}
```

---

## Infrastructure: `internal/infrastructure/mercadopago/`

```
internal/infrastructure/mercadopago/
├── client.go            # interface MPClient + implementação HTTP
├── report_parser.go     # parse do CSV para []MPEntry
└── client_test.go       # testes do parser (unitários, sem HTTP)
```

### Interface `MPClient`

```go
type MPClient interface {
    RequestReport(ctx context.Context, req ReportRequest) (*ReportResponse, error)
    DownloadReport(ctx context.Context, reportID string) ([]byte, error)
    ListPendingReports(ctx context.Context) ([]*ReportInfo, error)
}

type ReportRequest struct {
    BeginDate time.Time
    EndDate   time.Time
    FileNamePrefix string
}

type ReportResponse struct {
    ID        string
    Status    string  // "generating" | "created"
    CreatedAt time.Time
}

type MPEntry struct {
    Date            time.Time
    TransactionType string  // "Rendimentos", "Pix enviado", "Pagamento com QR Pix", etc.
    ReferenceID     string  // ID único do MP (usado como ExternalID para dedup)
    Amount          float64 // positivo = crédito, negativo = débito
}
```

### `report_parser.go`

Parser puro (sem IO). Recebe `[]byte` e retorna `[]MPEntry`.
Facilita testes unitários sem mock de HTTP.

```go
func ParseCSV(data []byte) ([]MPEntry, error)
```

### Mapeamento de tipos do MP para o domínio

| TransactionType (CSV) | Tipo no sistema | Observação |
|---|---|---|
| `Rendimentos` | `INCOME` | Categoria: rendimento da conta |
| `Pix recebido` | `INCOME` | Categoria: a mapear |
| `Dinheiro retirado *` | `INCOME` | Resgate de caixinha → fica como INCOME (caixinha é conta separada) |
| `Transferência enviada` | `INCOME` | Estorno/devolução |
| `Pix enviado` | `EXPENSE` | |
| `Pagamento com QR Pix` | `EXPENSE` | |
| `Pagamento Cartão de crédito` | `EXPENSE` | |
| `Pagamento *` | `EXPENSE` | |
| `Compra *` | `EXPENSE` | |
| `Dinheiro reservado *` | ignorar | Reserva interna (não afeta saldo real da conta principal) |
| `Reserva por gastos *` | ignorar | Reserva interna de caixinha |
| `Transferência cancelada *` | ignorar | Par anulado — já cancela a transação original |

**Regra de dedup**: `ExternalID = "mp-{ReferenceID}"`. Se já existe no banco, pula.

---

## Use Cases

### 1. `RequestMPReportUseCase`

Solicita geração de relatório para um período. Cria `SyncLog` com status `PENDING`.

```go
type RequestMPReportInput struct {
    BankAccountID string
    Date          time.Time // dia a ser reportado (normalmente D-1)
}

type RequestMPReportResult struct {
    SyncLogID string
    ReportID  string
}
```

### 2. `ProcessMPReportUseCase`

Recebe `[]byte` (CSV bruto) + contexto. Faz parse, dedup por `ExternalID`, cria transações.
**Usado tanto pelo webhook quanto pelo upload manual.**

```go
type ProcessMPReportInput struct {
    BankAccountID string
    ProfileID     string
    CSVData       []byte
    Source        mpsync.SyncSource
    SyncLogID     *string // nil = cria um novo log
}

type ProcessMPReportResult struct {
    Imported int
    Skipped  int
    Errors   int
    Details  []EntryResult
}
```

Lógica interna:
1. Parse CSV → `[]MPEntry`
2. Filtrar entradas ignoradas (Reserva, cancelada)
3. Para cada entrada:
   - Verificar `FindByExternalID("mp-{ReferenceID}")`
   - Se existe → `Skipped++`
   - Se não existe → mapear tipo, criar `transaction.Transaction` com `ExternalID`
4. Atualizar `SyncLog` → status `DONE`
5. Disparar `RecalculateBalance` para a conta

### 3. `HandleMPWebhookUseCase`

Recebe payload do webhook do MP (relatório pronto).
Busca `SyncLog` pelo `ReportID`, baixa CSV via `MPClient.DownloadReport`, delega para `ProcessMPReportUseCase`.

```go
type HandleMPWebhookInput struct {
    ReportID  string
    Signature string // header x-signature para validação
}
```

### 4. Fluxo manual (existente, adaptado)

O endpoint `POST /bank-accounts/{id}/import-statement` passa a delegar para `ProcessMPReportUseCase`
com `Source: SyncSourceManual`. Mesmo processador, mesma dedup, mesmo log.

---

## HTTP Handlers

```
POST /api/v1/bank-accounts/{id}/sync-report       → dispara RequestMPReportUseCase
GET  /api/v1/bank-accounts/{id}/sync-logs         → lista SyncLogs da conta
POST /api/v1/bank-accounts/{id}/import-statement  → upload manual (já existe, adaptar)
POST /api/v1/webhooks/mercadopago/report          → HandleMPWebhookUseCase (sem auth JWT, valida x-signature)
```

---

## Banco de dados

```sql
CREATE TABLE IF NOT EXISTS finance.mp_sync_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_account_id UUID NOT NULL REFERENCES finance.bank_accounts(id) ON DELETE CASCADE,
    report_id       TEXT,
    report_date     DATE NOT NULL,
    source          TEXT NOT NULL CHECK (source IN ('AUTOMATIC', 'MANUAL')),
    status          TEXT NOT NULL DEFAULT 'PENDING',
    imported        INT NOT NULL DEFAULT 0,
    skipped         INT NOT NULL DEFAULT 0,
    errors          INT NOT NULL DEFAULT 0,
    error_detail    TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT now(),
    updated_at      TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_mp_sync_logs_account ON finance.mp_sync_logs (bank_account_id, report_date DESC);
CREATE INDEX idx_mp_sync_logs_report_id ON finance.mp_sync_logs (report_id) WHERE report_id IS NOT NULL;
```

A coluna `external_id` em `finance.transactions` já existe. A dedup usa `external_id = 'mp-{ReferenceID}'`.

---

## Configuração

Variáveis de ambiente necessárias:

```env
MP_ACCESS_TOKEN=APP_USR-...        # token OAuth da conta pessoal
MP_WEBHOOK_SECRET=...              # segredo para validar x-signature
MP_BANK_ACCOUNT_ID=6882e83a-...   # ID da conta MP no sistema
MP_REPORT_ENABLED=true             # feature flag para desabilitar sem redeploy
```

---

## Testes

### Unitários — `internal/application/usecases/`

**`mp_sync_test.go`**

| Teste | O que valida |
|---|---|
| `TestProcessReport_CreatesNewTransactions` | Entradas novas viram transações |
| `TestProcessReport_SkipsDuplicateByExternalID` | Mesma `ReferenceID` não duplica |
| `TestProcessReport_IgnoresReservaEntries` | "Reserva por gastos" ignorada |
| `TestProcessReport_IgnoresCanceledTransfers` | "Transferência cancelada" ignorada |
| `TestProcessReport_MapsRendimentosAsIncome` | Rendimentos → INCOME |
| `TestProcessReport_MapsPixEnviadoAsExpense` | Pix enviado → EXPENSE |
| `TestProcessReport_UpdatesSyncLogOnSuccess` | SyncLog fica DONE |
| `TestProcessReport_UpdatesSyncLogOnError` | SyncLog fica FAILED se parse falha |
| `TestRequestReport_CreatesPendingSyncLog` | SyncLog criado com status PENDING |
| `TestHandleWebhook_RejectsInvalidSignature` | Signature errada → erro |
| `TestHandleWebhook_DownloadsAndProcesses` | Fluxo completo com fakes |

**`internal/infrastructure/mercadopago/report_parser_test.go`**

| Teste | O que valida |
|---|---|
| `TestParseCSV_ValidFile` | Parse correto de CSV real |
| `TestParseCSV_EmptyFile` | Retorna slice vazio, sem erro |
| `TestParseCSV_MalformedRow` | Linha malformada ignorada sem panic |
| `TestParseCSV_NegativeAmount` | Débito com sinal negativo parsed corretamente |
| `TestParseCSV_BrazilianNumberFormat` | "1.234,56" → 1234.56 |
| `TestParseCSV_DateFormat` | "21-04-2026" → time.Time correto |

### E2E / Integração — `internal/test/`

**`mp_sync_e2e_test.go`** (tag `// +build integration`)

| Teste | O que valida |
|---|---|
| `TestE2E_ManualCSVUpload_ImportsTransactions` | Upload de CSV real → transações criadas no DB |
| `TestE2E_ManualCSVUpload_IdempotentOnReupload` | Segundo upload não duplica |
| `TestE2E_WebhookFlow_EndToEnd` | POST /webhooks → processa → DB correto |
| `TestE2E_SyncLog_TracksImportHistory` | Histórico de syncs listável por conta |
| `TestE2E_BalanceRecalculatedAfterSync` | Saldo da conta atualiza após import |

Infra de teste: Docker Compose com Postgres real (já existe em `internal/test/`).
MPClient mockado via interface — sem chamadas reais à API do MP nos testes.

---

## Fakes para testes unitários

```go
// fakeMPClient implementa MPClient para testes
type fakeMPClient struct {
    reportResponse *mercadopago.ReportResponse
    csvData        []byte
    downloadErr    error
}

// fakeSyncLogRepo implementa mpsync.Repository para testes
type fakeSyncLogRepo struct {
    logs []*mpsync.SyncLog
}
```

---

## Plano de implementação

| Fase | O que fazer | Dependências |
|---|---|---|
| **1 — Domain** | Criar `internal/domain/mpsync/` (SyncLog + Repository) | — |
| **2 — Parser** | Criar `report_parser.go` + testes unitários | — |
| **3 — ProcessReport UC** | Use case + fakes + testes | Fase 1 + 2 |
| **4 — Persistence** | Migration + `mp_sync_log_repository.go` | Fase 1 |
| **5 — MPClient** | Interface + implementação HTTP real + testes com CSV fixture | Fase 2 |
| **6 — RequestReport UC** | Use case + scheduler diário | Fase 5 |
| **7 — Webhook handler** | HTTP handler + validação x-signature | Fase 3 + 5 |
| **8 — Manual import** | Adaptar endpoint existente para usar ProcessReport UC | Fase 3 |
| **9 — E2E tests** | Testes de integração completos | Fases 1–8 |
| **10 — Feature flag + deploy** | `MP_REPORT_ENABLED`, config OAuth, teste em produção | Fase 9 |
| **11 — Cofrinhos como TRANSFER** | Tratar `Reserva por gastos` e `Dinheiro retirado` como TRANSFER no parser | Fase 3 |
| **12 — Upload fatura cartão MP** | Parser CSV do cartão + `POST /invoices/{id}/import-statement` | Independente |

---

## Decisões de design

**Por que `ProcessMPReportUseCase` centraliza tudo?**
Manual e automático diferem apenas na *origem* dos bytes do CSV. O processamento
(parse, dedup, mapeamento de tipos, criação de transação, recalculate) é idêntico.
Duplicar essa lógica seria um bug em espera.

**Por que ignorar "Reserva por gastos" e "Dinheiro reservado"?**
Essas entradas são movimentos internos dentro do MP (da conta principal para caixinhas).
No nosso modelo, as caixinhas são contas separadas (`SAVINGS_BOX`). A transferência
MP→Caixinha já é registrada como TRANSFER entre as duas contas. Importar a "Reserva"
geraria duplicata — exatamente o problema que tivemos com os Resgates.

**Por que `ExternalID = "mp-{ReferenceID}"`?**
O campo `reference_id` do CSV é o ID único que o MP atribui a cada operação.
Prefixar com `"mp-"` evita colisão com `ExternalID`s de outras integrações (brapi dividendos, binance, etc.).

**Por que não usar `GET /v1/payments/search`?**
Retorna apenas pagamentos processados como *vendedor*. Pix avulsos recebidos,
rendimentos e despesas pessoais não aparecem. O relatório CSV é a única fonte
completa para conta pessoal.
