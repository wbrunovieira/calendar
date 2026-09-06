---
name: mercadopago-mcp
description: Query Bruno's real, live bank data — balances, statements, credit-card bills and broker positions — for **Mercado Pago (PF), Nubank (PF), Nubank Empresas (PJ) and Clear/XP**, via the Banco MCP REST API (Open Finance Brasil, powered by Pluggy, read-only). Use when asked to check a real/current balance, extrato, fatura or posição, or to cross-check/reconcile any of those accounts against the internal calendar-finances system. Read-only: cannot move money, make Pix, or pay anything.
---

# Bruno's banks via Banco MCP (Open Finance)

Banco MCP (mcp.ai) aggregates bank data through **Open Finance Brasil** (regulated by
BACEN, powered by Pluggy under the hood). It is **read-only by architecture** — none
of its endpoints can move money, make a Pix, or pay a bill. Four institutions are
connected (2026-09-05): Mercado Pago PF, Nubank PF, Nubank Empresas and Clear/XP.

This is a **cross-check / verification tool against the real bank**, not a
replacement for [[financas]]. `calendar-finances` stays the system of record for
transactions you log — use this skill to confirm real numbers, then follow the
[[finances-reconciliation-rule]] golden rule (never patch a balance; find the
missing/extra transaction) if something doesn't match.

## Auth

```
Authorization: Bearer <MCP_AI_API_KEY>
Content-Type: application/json
```
Key stored in `services/calendar-finances/.env` as `MCP_AI_API_KEY` (gitignored,
treat as a secret — same as `BRAPI_TOKEN`/`KEY_BINANCE` in that file). A **separate**
value, `BANCO_MCP_URL`, is also in that `.env` — that one is only for registering
Banco MCP as an MCP connector in Claude Desktop/Code (`claude mcp add`), not needed
for the plain REST calls below.

Base URL: `https://api.mcp.ai/api/openfinance`. Every call is a `POST` to base URL +
path. Success: `{"ok": true, "tool": "<id>", "result": <payload>}`. Errors come back
as `{"error": {"code", "message", ...}}` — read the body, don't use `curl --fail`.

## Account (mcp.ai)

| | |
|---|---|
| Login | in `MCP_AI_ACCOUNT_LOGIN` (`.env`) |
| User ID | in `MCP_AI_USER_ID` (`.env`) |
| Toolkit | in `MCP_AI_TOOLKIT_ID` (`.env`) |
| Plan | **PJ Ilimitado, R$ 99,90/month** — the PF tiers cannot connect a CNPJ |

> This repository is **public**. Account and bank identifiers are deliberately kept
> out of it and live in `services/calendar-finances/.env` (gitignored) and in the
> agent's private memory. Do not paste real ones back into this file — none of them
> is a credential on its own, but an account number plus a card's last four digits
> plus the account e-mail is exactly the material used to impersonate someone on a
> bank's support line.

Credentials live in `services/calendar-finances/.env` (gitignored):
`MCP_AI_API_KEY` for the REST API, `BANCO_MCP_URL` for registering the MCP server.
Both point at this account.

**The MCP server resolves its account when the session starts.** If `toolkit_info`
reports a different `toolkit_id` or `acting_as.user_id` than the ones in `.env`, the running
session is bound to stale credentials and no amount of retrying will fix it — start a
new Claude Code session. A session bound to an expired account answers every call with
`free_tier_limit_reached`, which reads like a billing problem and is not one.

## Connections on the current account

Four institutions are connected: **Mercado Pago (PF)**, **Clear Corretora**,
**Nubank PF** and **Nubank Empresas (PJ)**. Each contributes a BANK account, and
Mercado Pago, Nubank PF and Nubank Empresas each also contribute a CREDIT account.

**Item and account IDs are not written down here** — discover them at runtime, which
is more reliable anyway since they change whenever a connection is re-authorised:

```bash
curl -s -X POST "https://api.mcp.ai/api/openfinance/connections/list" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" -d '{}'
curl -s -X POST "https://api.mcp.ai/api/openfinance/accounts/list" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" -d '{}'
```

`accounts/list` returns each account's `account_id`, `type` (BANK / CREDIT), bank name
and masked number, which is enough to pick the one you need. The connector numbers are
stable and safe to note: Mercado Pago 606, Nubank PF 612, Clear 623, Nubank Empresas 664.

All 4 slots of the PJ Ilimitado plan are used — a 5th bank would cost R$ 19,90/month
extra, charged automatically as a metered Stripe item.

**Known data limits** (hit on 2026-09-05, worth remembering before promising a
reconciliation): Clear's CCTVM statement over Open Finance stops at **25/05/2026** even
after a forced sync, and `investments/transactions/list` needs an `investment_id` that
no longer exists once a position is fully sold — so **sales are invisible** there. For
Clear, ask Bruno for the XLSX export (Extrato → período personalizado); it goes back to
2024 and carries the running balance per line.

**Clear today holds no positions** — everything was sold on 30/07 (SNAG11) and 11/08
(the other five), so `investments/list` returns `total: 0` and only the two cash
accounts answer. That is the real state, not a sync failure. Dividends land in the
**CCTVM (102)** account; the **Banco XP (348)** account only sees the TEDs in and out.

**Credentials are current.** `MCP_AI_API_KEY` and `BANCO_MCP_URL` in
`services/calendar-finances/.env`, plus the `banco-mcp` entry in `~/.claude.json`, were
all repointed at the new account on 2026-09-05 and verified working over REST. Note the
two live independently: the REST key works in any session, but the **MCP tools only pick
up a new connector URL in a new Claude Code session**.

Consequently every ID in the table below is stale — rediscover with
`openfinance_list_connections` / `openfinance_list_accounts` after reconnecting. Also
revoke the old MP consent (11/08) in the Mercado Pago app, since it still feeds the
abandoned workspace.

If IDs stop working (a reconnect expired, or a bank is added later), rediscover them
via `openfinance_list_connections` and `openfinance_list_accounts` (examples below).

## Common calls

**Real-time balance / list accounts** (also returns credit card limit & due data):
```bash
curl -s -X POST "https://api.mcp.ai/api/openfinance/accounts/list" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" -d '{}'
```

**Extrato (statement) for the checking account**, date range `YYYY-MM-DD`:
```bash
curl -s -X POST "https://api.mcp.ai/api/openfinance/transactions/list" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" \
  -d '{"account_id":"<BANK account_id from accounts/list>","from":"2026-08-01","to":"2026-08-31"}'
```

**Fatura (credit card bills)** — returns CLOSED bills only, with a derived
`payment_status` (`PAID` / `OPEN` / `PAST_DUE_UNCONFIRMED` / `PAST_DUE_UNPAID` — read
the `payment_status_legend` in the response, it explains the cross-bill heuristic):
```bash
curl -s -X POST "https://api.mcp.ai/api/openfinance/credit-card-bills/list" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" \
  -d '{"account_id":"<CREDIT account_id from accounts/list>"}'
```
The **currently open** cycle (not yet closed) doesn't show up here — its running
total is the `balance` field from `accounts/list` on the CREDIT account instead.

**Force a fresh sync before reading** (waits up to ~60s):
```bash
curl -s -X POST "https://api.mcp.ai/api/openfinance/connections/sync" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" \
  -d '{"items":["606"]}'
```

Full endpoint list (18 tools: connections, accounts, transactions, credit-card-bills,
investments, loans, categories) is self-documented, always current, at:
`GET https://api.mcp.ai/api/openfinance/_endpoints` (JSON) or `/_skill.md` (markdown).
Report a broken/empty response with `POST .../report -d '{"message":"..."}'`.

## Errors

- `401` — bad/missing key. `403 key_scope_mismatch` — key not allowed on this MCP.
- `409 not_connected` — need to (re)connect at the `connect_url` the tool returns.
- `402` — free-tier calls exhausted (top up or upgrade plan).
- `429` — rate limited, honor `Retry-After`.

## Scope / plan limits

Bruno is on the **free trial** (R$0, 1 bank connected, 10 questions in a 24h window
from the first one) — only Mercado Pago PF is connected right now. A "question" is a
**tool call**, not a chat turn: the vendor FAQ says *"O que acontecer antes — 10
chamadas de ferramenta ou 24h — encerra a janela de teste."* One conversational
question can therefore burn several. Paid plans make this unlimited (`429` still
applies as a rate limit). Adding Nubank PF
+ Nubank Jurídica would need the **PJ Unlimited plan (R$ 99,90/mês)**, since Nubank
Jurídica is a CNPJ account and the cheaper PF-only tiers can't connect one. Full
research/pricing/architecture notes (including a **Clear Corretora** connector that
was confirmed to exist, still untested) are in
`services/calendar-finances/doc/plans/open-finance-unified.md`.

## Known discrepancy to watch (found while testing, 2026-08-11)

The real Mercado Pago **credit card** limit reported over Open Finance does not match
the `creditLimit` recorded for "Cartão Mercado Pago"
(`73567965-1f0b-4d2e-868a-b1cfdc05fbbd`) in `calendar-finances`, and the balances
disagree too. Figures are in the agent's private memory, not here.

Don't patch either number. This needs the same approach as
[[finances-reconciliation-open-2026-08]]: find the missing/wrong transactions, or
confirm the card's real limit changed and update that static field — never force a
balance.

⚠️ Before treating a card difference as real, check WHICH quantity each side reports.
The bank's "used" figure consumes the limit and includes future installments, while the
internal `currentBalance` tracks the open bill — comparing the two invents a
discrepancy. For the open cycle prefer `credit-card-bills/list` with
`include_open_bill: true` over the `balance` field of `accounts/list`, whose meaning is
connector-dependent.
