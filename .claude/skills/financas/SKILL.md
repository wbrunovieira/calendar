---
name: financas
description: Financial system (calendar-finances). Use ALWAYS when the user asks to POST/register/execute a transaction, confirm PENDING or OVERDUE entries, pay/confirm an invoice, reconcile an account against the bank, check a balance, or touch recurrences. Carries profile/account/category IDs, auth, and the golden rule (balance = strict sum of transactions; never patch a balance). Writes ALWAYS via the API, never SQL.
---

# Calendar Finances — API (production)

How to create/confirm/query transactions safely. **For writes, ALWAYS the API** (it recalculates the balance, processes recurrence, updates the invoice). Direct SQL is for reads or category labels only — never for posting entries.

Account and category names below are kept in Portuguese on purpose: they are the
literal labels stored in the database, and translating them would break lookups.

## ⚠️ Where you run — the finances agent lives in a git worktree
Two Claude agents work this repo in parallel, in separate terminal tabs. A git
checkout has **one HEAD per working directory**, so if both share a directory,
`git checkout` by one silently moves the other onto a foreign branch. The terminal
emulator is irrelevant — separate Warp tabs, iTerm, or tmux all hit the same
`.git/HEAD`. The fix is a **separate working directory**, via `git worktree`.

| Agent | Working directory | Owns |
|---|---|---|
| calendar | `~/projects/calendar` (main checkout) | `calendar-core`, `calendar-frontend`, the local Docker stack |
| **finances (you)** | **`~/projects/calendar-finances`** (worktree) | `calendar-finances`, `finances-frontend`, production transactions |

Both directories share one `.git` (4.8 MB — the bulk of the 3.2 GB is
`node_modules`), so branches, remotes and fetches are common; only HEAD is
independent.

**Rules while working here:**
- **Run every git command from `~/projects/calendar-finances`.** Never `git checkout`
  in the main checkout — that is the other agent's HEAD.
- A branch checked out in one worktree **cannot** be checked out in the other. That
  is the guard rail doing its job, not a bug.
- **Do not run `docker-compose up` from the worktree.** `container_name` is pinned in
  the compose file, so it collides with the shared stack; the local stack belongs to
  the main checkout. Go tests don't need it (`go test ./...` runs locally against fake
  repositories), and production work goes over SSH anyway.
- `.env`, `.env.test` and `services/calendar-finances/.env` are **symlinks** to the
  main checkout — one file, edited in either place. They are gitignored, so a fresh
  worktree has to relink them.
- The agent memory directory is keyed by the working directory path, so
  `~/.claude/projects/-Users-brunovieira-projects-calendar-finances/memory` is a
  symlink to the main project's `memory/`. Both agents therefore share memory —
  see [[agent-split-calendar-vs-finances]] for who owns which file.

**Recreating the worktree from scratch** (e.g. after a machine wipe):
```bash
git -C ~/projects/calendar worktree add ~/projects/calendar-finances <branch>
ln -sfn ~/projects/calendar/.env                            ~/projects/calendar-finances/.env
ln -sfn ~/projects/calendar/.env.test                       ~/projects/calendar-finances/.env.test
ln -sfn ~/projects/calendar/services/calendar-finances/.env ~/projects/calendar-finances/services/calendar-finances/.env
ln -sfn ~/.claude/projects/-Users-brunovieira-projects-calendar/memory \
        ~/.claude/projects/-Users-brunovieira-projects-calendar-finances/memory
cd ~/projects/calendar-finances/services/finances-frontend && npm ci
```

## ⚠️ GOLDEN RULE — the balance is a result, never a target
This is a financial system: **the balance must represent STRICTLY the sum/subtraction of the transactions.**
- **NEVER "patch" the balance** directly (not via the recalc API, not via SQL, not by editing `currentBalance`).
- If the system balance **does not match** the bank → **do not adjust the balance**. The difference is always a **missing, duplicated or wrong transaction**.
- **Hunt the fix down in the transactions, TOGETHER with the user** — work out which entry is missing/wrong, show the difference, and only post/correct the transaction after confirming with them.
- E.g.: system 931.78 vs bank 933.74 → difference +1.96 = unposted interest → post the interest (INCOME), don't touch the balance.
- The balance recalc (`/bank-accounts/{id}/recalculate-balance`) only **recomputes from the transactions** — use it to fix a derived balance, never to force a value.

## Access / auth
- **Internal API, no token**, on the VPS. Reach it over SSH:
  ```bash
  ssh -i ~/.ssh/id_rsa root@45.90.123.190 "curl -s http://127.0.0.1:3335/api/v1/<route>"
  ```
- Base URL: `http://127.0.0.1:3335/api/v1`
- Filter SSH noise out of pipes: `| grep -vE "WARNING|post-quantum|decrypt later|openssh"`
- Responses: `{"data": ...}`. POST/PUT send JSON with `-X POST -H 'Content-Type: application/json' -d '{...}'`.

## Profiles
| Profile | ID | Type |
|---|---|---|
| Bruno Pessoal | `259866ac-6f74-4f7f-98b3-9a4896fb6758` | PERSONAL |
| WB Digital Solutions | `268b1f32-fe80-4293-9d03-87f1c63317b9` | BUSINESS |

## Accounts — Bruno Pessoal
Default account (checking) = **Mercado Pago** `6882e83a-b4fb-4cd5-b364-e0cfa61a90c3`.
| Account | ID | Type |
|---|---|---|
| Mercado Pago | `6882e83a-b4fb-4cd5-b364-e0cfa61a90c3` | CHECKING |
| Nubank | `2e9875d5-49be-4b0f-b1ef-c9adb421f42a` | CHECKING |
| Wise (BRL) | `62b5726d-1efb-44df-8210-cba0a6a4d4b2` | CHECKING |
| Wise (EUR) | `cfa39f00-540b-4951-a511-76f4276b1761` | CHECKING (EUR) |
| Wise (USD) | `d8199fe6-efd5-4978-990b-f89a00f7fa8c` | CHECKING (USD) |
| Cartão Pessoal Nubank | `00e3c475-d21f-43f5-89ba-399d312c5095` | CREDIT_CARD |
| Cartão Mercado Pago | `73567965-1f0b-4d2e-868a-b1cfdc05fbbd` | CREDIT_CARD |
| Caixinha Mercado Pago | `b7c6aba6-df07-4a70-be56-2676541ada58` | INVESTMENT |
| Caixinha Nubank Pessoal | `c1f6172e-f3d9-4756-9f02-cb212f934fcd` | INVESTMENT |
| Binance (spot) | `cada06d0-d4cb-4a0e-b600-77bc589729a1` | EXCHANGE |
| (FIIs HGLG11/KNCR11/XPML11/KNRI11/SNAG11/CPTS11, CDBs, Clear, Binance crypto) | — | INVESTMENT |

## Accounts — WB Digital Solutions
| Account | ID | Type |
|---|---|---|
| Nubank Juridica | `22296070-8ef1-44e5-90b2-d7c92a87dba3` | CHECKING |
| Nubank Juridica Cartão | `174c6a9c-1b68-4a76-88d3-7cae3f1cf069` | CREDIT_CARD |
| Caixinha Nubank Juridica | `056a334a-bc51-4749-8668-34a14ca2adfc` | INVESTMENT |

## Most-used categories — Bruno Pessoal
Streaming/IPTV `5441f3b1-7da0-4221-b260-b650efd6dba3` · Supermercado `e2f502a3-75a2-43f4-ba17-91105430783d` · Restaurante `fc5b2f6c-14b9-4a94-97b8-3b951dbd9244` · iFood `70a21cf2-1ea8-4bb9-93be-86a506adaae4` · Aluguel `2ed8988c-323c-437b-8ba3-90e0b4777866` · Contas `95b71fc6-e4c2-494d-8444-bfd020aed939` · Celular `cf992387-1dd2-474e-a69f-fe3681c7d802` · Uber `e6d7b8da-6c89-49e4-9b4c-f5b85c4fb236` · Combustível `97fb6741-1892-48dd-8a7b-934629fba215` · Saúde `614c3479-7122-44e3-b618-044cb33f971b` · Academia `6e7248c0-3558-4298-9f7c-1e7866ef217f` · Lazer `b9f169f5-cd51-4ff8-8352-18d8f37ff32d` · Rendimentos (INCOME) `ee4c3034-b0dc-4e18-872f-411c9dbbd65c` · Salário (INCOME) `5848835b-352e-4ffe-b7cc-d21b90af3f51` · Transferências (TRANSFER) `9e89e8d6-cba1-48bf-aee3-70f14473acb8`.

## Most-used categories — WB
Receitas (INCOME) `8985a57a-e567-4016-b849-7fde3afe8f82` (Projetos/Servicos `520eb116-…`, Manutencao Mensal `c59f94be-…`, Aporte Socio `f8ac5b09-…`, Rendimentos Caixinha `da34a431-…`) · Despesas com Pessoal `6f08588f-…` (Pró-labore `63d95c6d-…`) · Infraestrutura `e6b897fe-…` (Software/SaaS `2ef99342-…`, Servidores `91dce52f-…`, Telefonia `216551dc-…`) · Marketing `377c50a5-…` · Taxas/Impostos `e37eae51-…` · Cursos/Capacitacao `51420c96-…` · Recrutamento/Freelancers `a4af3432-…` · Transporte `2996e140-…` (Uber `645d3a43-…`).

> **Full IDs can change** — for the current list:
> `curl -s ".../categories?profileId=<id>"` and `.../bank-accounts?profileId=<id>`.

## Querying
```bash
# pending/overdue (PLANNED)
curl -s ".../transactions?profileId=<PID>&status=PLANNED&pageSize=100"
# transactions for a period — CAREFUL: /transactions uses occurredFrom/occurredTo (NOT from/to)
curl -s ".../transactions?profileId=<PID>&bankAccountId=<ACC>&occurredFrom=2026-06-01&occurredTo=2026-06-30&pageSize=200"
# (from/to only work on /transactions/expense-analysis and /financial-summary)
# account balances
curl -s ".../bank-accounts?profileId=<PID>"   # currentBalance field
```
Possible statuses: **PLANNED · CONFIRMED · CANCELLED**. Types: **INCOME · EXPENSE · TRANSFER**.

## Confirming a pending/overdue entry (the "Confirmar" button)
The right way to "execute" an overdue recurrence that already exists as PLANNED:
```bash
ssh -i ~/.ssh/id_rsa root@45.90.123.190 "curl -s -X PUT \
  http://127.0.0.1:3335/api/v1/transactions/<TX_ID>/status \
  -H 'Content-Type: application/json' \
  -d '{\"status\":\"CONFIRMED\",\"occurredOn\":\"2026-06-23\"}'"
```
- `occurredOn` = the real payment date (not the planned one). It recalculates the account balance.

## Creating a new transaction
```bash
ssh -i ~/.ssh/id_rsa root@45.90.123.190 "curl -s -X POST \
  http://127.0.0.1:3335/api/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{
    \"profileId\":\"259866ac-6f74-4f7f-98b3-9a4896fb6758\",
    \"bankAccountId\":\"6882e83a-b4fb-4cd5-b364-e0cfa61a90c3\",
    \"categoryId\":\"5441f3b1-7da0-4221-b260-b650efd6dba3\",
    \"type\":\"EXPENSE\",
    \"amount\":25.00,
    \"currency\":\"BRL\",
    \"description\":\"IPTV\",
    \"occurredOn\":\"2026-06-23\",
    \"status\":\"CONFIRMED\"
  }'"
```
Required fields: `profileId, bankAccountId, categoryId, type, amount, currency(BRL), description, occurredOn`; `status` (defaults to PLANNED → send CONFIRMED to settle it right away); optional `notes, dueOn, reminderOn, destinationAccountId` (transfer), `installmentNumber/Total` (installments).

## Transfers (between accounts)
**There is no dedicated route.** A transfer is `POST /transactions` with `type=TRANSFER` + `destinationAccountId`. The behaviour **changes with the profiles involved** (balances only move when `status=CONFIRMED`):

**Same profile** → creates **1 `TRANSFER` transaction**. Debits the source, credits the destination. If `categoryId` is sent it must be a TRANSFER-type category (e.g. Transferências pessoal `9e89e8d6-...`); it is optional.
```bash
# withdrawal from the savings pot to checking, same profile
-d '{"profileId":"<PID>","bankAccountId":"<SOURCE>","destinationAccountId":"<DESTINATION>",
  "type":"TRANSFER","status":"CONFIRMED","amount":1170,"currency":"BRL",
  "description":"Dinheiro retirado","occurredOn":"2026-07-05"}'
```
⚠️ If the **source is a credit card**, the balance does NOT move (credit-card guard) — only the destination is credited.

**Cross-profile** (e.g. Bruno Pessoal's MP → WB's Nubank Jurídica) → creates **2 linked transactions** (`linkedTransactionId`):
- the source becomes an **EXPENSE** on the source profile (`categoryId` optional; if sent it must be EXPENSE);
- the destination becomes an **INCOME** on the destination profile, using **`destinationCategoryId` (REQUIRED**, an INCOME category of the destination profile).
```bash
# personal -> WB contribution: EXPENSE Empréstimos (personal) + INCOME Aporte Sócio (WB)
-d '{"profileId":"259866ac-...","bankAccountId":"6882e83a-...(MP)",
  "destinationAccountId":"22296070-...(Nubank Jur)",
  "categoryId":"d359a66d-...(Empréstimos, EXPENSE personal)",
  "destinationCategoryId":"f8ac5b09-8647-4004-adf2-ba63dd5683e8 (Aporte Socio, INCOME WB)",
  "type":"TRANSFER","status":"CONFIRMED","amount":770,"currency":"BRL",
  "description":"Aporte WB Digital (via MP)","occurredOn":"2026-07-05"}'
```
The POST returns only the source transaction (with `linkedTransactionId` pointing at the destination's INCOME).

## Editing a transaction (PUT /transactions/{id})
`PUT /transactions/{id}` is a **FULL REPLACE** — omitted fields are **zeroed/wiped** (including `categoryId` and `notes`). To change only the amount: GET the current object and **resend every field that is set** (`bankAccountId, categoryId, type, status, amount, currency, description, occurredOn`), changing only `amount`. Keep `bankAccountId` and `occurredOn` unchanged so the invoice link is not recomputed. The invoice total is derived on read (recalculated from the sum of its transactions), so it reflects the change on the next GET by itself.

## Recurrences
Model: a recurrence uses an **RRULE** + dates, not `dayOfMonth`. Actual fields: `recurrenceRule` (e.g. `FREQ=MONTHLY;BYMONTHDAY=13`), `startOn`, `nextOccurrence`, `amount`, `status` (ACTIVE/PAUSED/CANCELLED).
```bash
curl -s ".../recurring-transactions?profileId=<PID>"               # list (full object)
curl -s -X POST ".../recurring-transactions/process" -d '{...}'    # generate the period's PLANNED entries
curl -s -X PATCH ".../recurring-transactions/<id>/status" -d '{"status":"PAUSED"}'
```
**Confirming the month's recurrence** = confirming the PLANNED entry it generated (`PUT /transactions/{id}/status`), NOT touching the recurrence.
**Changing the day (e.g. 13th → 23rd)**: `PUT /recurring-transactions/{id}` is a **full replace** (`CreateRecurringTransactionInput`) — send ALL fields, changing only what you need. GET the current object and resend it:
```bash
curl -s -X PUT ".../recurring-transactions/<id>" -H 'Content-Type: application/json' -d '{
  "profileId":"...","bankAccountId":"...","categoryId":"...","type":"EXPENSE","amount":25,
  "currency":"BRL","description":"IPTV","recurrenceRule":"FREQ=MONTHLY;BYMONTHDAY=23",
  "startOn":"2026-02-13","nextOccurrence":"2026-07-23","status":"ACTIVE","notes":"..."}'
```
Input dates are `YYYY-MM-DD`. `nextOccurrence` = when the next one falls (e.g. the 23rd of next month).

## Gotchas (learned the hard way)
- The `/transactions` list is capped by `pageSize` (send it high, e.g. 200) and **ignores `from/to`** — use `occurredFrom`/`occurredTo`.
- **Bulk creation** (e.g. several entries): the `ssh 'bash -s' <<EOF` heredoc failed; use a **local loop** with one `ssh` per item. **CRITICAL: `ssh -n`** inside `while read` (otherwise ssh eats the heredoc's stdin and the loop stops at the first item). Working pattern: `while IFS='|' read -r D A C DESC; do ssh -n ... "curl ... -d '{...\"amount\":$A,\"description\":\"$DESC\"...}'"; done <<ENTRIES` with lines `date|amount|catId|description` (the `|` delimiter tolerates spaces and accents in the description). JSON: escape `\"`, keep `-d '...'` single-quoted, local vars still expand.
- **Credit card**: posting an expense = a normal POST with `bankAccountId`=the card; the system links it to the open invoice by itself. The "Pagamento fatura ..." entry that shows up on the card is the payment credit (not an expense).
- **IOF / international FX**: the amount stored can land one cent below what the bank actually charged (rounding). When reconciling a card against its invoice, compare **line by line** and fix each IOF to the invoice's exact amount (via `PUT /transactions/{id}`). Nubank's "Total a pagar" already applies its own rounding (the raw sum of the lines can come out 1-2 cents above the official total) — you cannot match it to the cent by summing; pay the amount actually charged.
- **Parsing with Python**: run it LOCALLY (after the `| python3 -c '...'`), with plain double quotes (do NOT escape); only the curl goes inside the `ssh "..."`.
- Noisy SSH: `-o LogLevel=ERROR` plus `2>/dev/null` on the curl leaves just the JSON.
- Mercado Pago interest (checking account) is posted as daily INCOME entries "rendimento da conta", category Rendimentos `ee4c3034-...`. Weekends sometimes have none.

## Rules (project memory)
- **Writes only via the API** (it recalculates balance/invoice). Direct SQL: reads or category labels only.
- **Confirm the category with the user** before posting anything new.
- **Never adjust an entry just to "match" a balance** — hunt down the missing entry.
- Credit-card invoice payment: **`POST /invoices/{id}/pay`** (POST, not PUT) already creates the debit (don't post it manually). Body: `{"paidAmount":769.51,"paidAt":"2026-07-05"}` (`paidAt` = `YYYY-MM-DD`). It debits the **account linked to the card** (`linkedAccountId`), so no account needs to be passed. Check it first: `GET /bank-accounts/{cardId}` → `linkedAccountId`. Find the invoice with `GET /invoices?bankAccountId=<cardId>` (the one to pay is the CLOSED one with the month's `dueDate`). `paidAmount` = the amount the bank actually charged.
- Reconciliation: compare against the bank statement; when reconciling, Bruno's BRL accounts live at Mercado Pago/Nubank.
