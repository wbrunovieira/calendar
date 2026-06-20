# Plano: mover a regra de "obrigação recorrente quitada?" para o backend

**Status:** proposto
**Autor:** Claude + Bruno
**Data:** 2026-06-20
**Serviço:** `calendar-finances` (Go) + `finances-frontend` (Next.js)

## Contexto

Despesas recorrentes (ex.: "Tim Celular") variam de valor todo mês: a recorrência é
cadastrada em R$ 99,00, mas a fatura real vem 102,14 / 109,99 / 112,30… O usuário precisa
**editar o valor do mês e confirmar**, sem duplicar e sem a pendência "grudar".

### Bug encontrado (já corrigido — fix paliativo "B")
A regra **"esta recorrência já foi quitada no mês?"** vivia no **frontend**
(`finances-frontend/src/components/finances/TodayAlerts.tsx`), fazendo *fuzzy match*
entre recorrência e transações por **tipo + valor + descrição + mês**. O match por
**valor** quebrava com valores variáveis: ao confirmar o Tim em 109,99 (≠ 99,00), o match
falhava e a recorrência continuava aparecendo como "a pagar".

**Fix B (entregue):** extraída a regra para `finances-frontend/src/utils/pendingAlerts.ts`
(função pura `findPendingRecurrings`, com Vitest — red→green), removendo a comparação de
valor e passando a casar por **tipo + descrição + conta + mês (±7 dias)**.

O fix B resolve o sintoma, mas **a regra continua na camada de view**. Este plano (A) move a
regra para o backend, que é onde ela pertence (o projeto é DDD).

## Por que pertence ao backend

1. **É domínio financeiro**, não apresentação. "Uma obrigação recorrente do período está
   satisfeita?" é regra de negócio.
2. **Fonte única da verdade.** Hoje qualquer cliente (web, agents, n8n) teria que
   reimplementar o mesmo heurístico e divergir.
3. **Testabilidade.** O backend tem suíte Go; o frontend não tinha teste algum até o fix B.
4. **A causa raiz é uma lacuna do backend** (ver abaixo), não da view.

## Lacunas atuais no backend (causa raiz)

1. **Sem vínculo transação → recorrência.** A tabela `finance.transactions` não tem
   `recurring_transaction_id`. Quando `ProcessRecurringTransactionsUseCase` gera o
   lançamento (`internal/application/usecases/process_recurring_transactions.go`), ele
   **não grava de qual recorrência veio**. Sem o elo, o cliente precisa adivinhar a
   correspondência — e adivinhava por valor.
2. **`Process` não é idempotente.** `FindDueRecurring` filtra só por
   `status='ACTIVE' AND next_occurrence <= now`. Chamar `POST /recurring-transactions/process`
   duas vezes antes do `next_occurrence` avançar gera **transações duplicadas** (não há
   checagem "ocorrência X já gerada?" nem unique constraint).
3. **`Process` não é agendado.** É só endpoint HTTP (`cmd/api/main.go`), sem cron. Os jobs
   existentes (auto-close de fatura, Binance, B3, close-month) seguem o padrão de goroutine
   em `main.go` — o Process deveria entrar igual.

## Arquitetura proposta

Vincular cada lançamento gerado à sua recorrência + competência, tornar o Process
idempotente e agendado, e expor a regra de "satisfeita?" no domínio. O cliente só renderiza.

| Camada | Mudança |
|---|---|
| **Schema** | `transactions.recurring_transaction_id UUID NULL` (FK → `recurring_transactions`, `ON DELETE SET NULL`) + `transactions.reference_month DATE NULL` (primeiro dia da competência). Índice em `(recurring_transaction_id, reference_month)`. |
| **Domínio** | `Transaction` ganha `RecurringTransactionID *string` e `ReferenceMonth *time.Time`. Regra: *uma ocorrência (recorrência, competência) está satisfeita ⟺ existe transação vinculada não-CANCELLED para aquela competência*. **Sem comparar valor.** |
| **Process** | Grava o vínculo + `reference_month` no lançamento. **Idempotente**: antes de criar, checar se já existe transação vinculada para `(recurring_id, competência)`; se sim, pular. Avançar `next_occurrence` só após sucesso. |
| **Cron** | Goroutine diária em `main.go` chamando o Process (mesmo padrão dos outros jobs). |
| **API** | A listagem de recorrências (ou um novo `GET /recurring-transactions/pending?on=YYYY-MM-DD`) retorna, por recorrência, se a competência corrente está satisfeita / a transação vinculada. Frontend lê isso. |
| **Confirm/Pay** | Ao confirmar/pagar uma recorrência pelo app ("Já paguei"), reusar/atualizar o lançamento PLANNED vinculado (editar valor + status→CONFIRMED) em vez de criar novo. Endpoint de update já aceita amount+status juntos (`PUT /transactions/{id}`). |
| **Frontend** | Remover o fuzzy-match de `pendingAlerts.ts`; passar a confiar no que o backend reporta (mostrar o PLANNED vinculado, editável). |

Com o vínculo por `(recurring_id, competência)`, **variar o valor "simplesmente funciona"**:
o Process cria o PLANNED da competência, o usuário edita pro valor real e confirma — a
pendência some porque o match é por id+competência, não por valor.

## Fases (cada uma em TDD)

1. **Migração + domínio.** Adicionar colunas/índice (SQL idempotente em
   `internal/database/database.go`), campos no domínio `Transaction` e persistência.
   Backfill opcional dos vínculos históricos por heurístico (descrição+mês) — *one-off*.
2. **Process idempotente + vínculo.** Gravar `recurring_transaction_id` + `reference_month`;
   skip se a competência já tem transação vinculada. Testes: "não duplica ao rodar 2×",
   "grava vínculo", "avança next_occurrence só no sucesso".
3. **Regra de domínio "satisfeita?" + endpoint.** `IsOccurrenceSatisfied(recurringID, month)`
   no domínio/usecase; expor na API. Testes de unidade.
4. **Cron diário.** Goroutine em `main.go`. Teste do use-case (o agendador em si é fino).
5. **Confirm reusa o PLANNED vinculado.** Garantir que "Já paguei"/edição não cria duplicata.
6. **Frontend.** Trocar o fuzzy-match por consumo do backend; ajustar/retirar testes de
   `pendingAlerts.ts` conforme a regra sair da view.

## Riscos / considerações

- **Migração de dados existentes:** transações já criadas não têm o vínculo. O backfill por
  heurístico é "best effort"; aceitável porque o histórico já está reconciliado.
- **Backward-compat do frontend:** manter o fix B funcionando até a fase 6 (o backend novo é
  aditivo). Sem big-bang.
- **Idempotência vs. reprocessamento legítimo:** a checagem é por `(recurring_id, competência,
  não-CANCELLED)` — se o usuário cancelar a transação, o Process pode regerar.
- **Regra de "satisfeita" e conta:** competência + recorrência bastam; a conta vem da própria
  recorrência (não precisa casar conta como no paliativo).

## Referências de código

- `internal/application/usecases/process_recurring_transactions.go` — Process atual
- `internal/domain/recurringtransaction/recurring_transaction.go` — domínio recorrência
- `internal/domain/transaction/transaction.go` — domínio transação (onde entra o vínculo)
- `internal/database/database.go` — schema `transactions` (migração)
- `cmd/api/main.go` — onde entra a goroutine do cron
- `finances-frontend/src/utils/pendingAlerts.ts` — regra paliativa (fix B), a ser simplificada
- `finances-frontend/src/components/finances/TodayAlerts.tsx` — consumidor
