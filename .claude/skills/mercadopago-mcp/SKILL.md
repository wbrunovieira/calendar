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

## ⚠️ Account migration in progress (2026-09-05)

Bruno moved to a **new mcp.ai account** so the PJ plan can hold the CNPJ connection:

| | Old (abandoned) | New (current) |
|---|---|---|
| Login | wbrunovieira77@gmail.com | **bruno@wbdigitalsolutions.com** |
| User ID | `usr_kv7dh20mdvh5` | **`usr_c2sc1drm4hsl`** (active since 2026-09-05) |
| Toolkit | `tk_5ey5wn9nsfx8` | **`tk_6gkli6i5iokw`** |
| Plan | free trial | **PJ Ilimitado, R$ 99,90/mês — subscribed** |
| Connections | Mercado Pago only | 3 connected on 2026-09-05 — see table below |

## Connections on the current account (2026-09-05)

Pass `item` (item_id, connector_id or bank name) on every call — there is more than one now.

| Bank | connector | item_id | Accounts |
|---|---|---|---|
| Mercado Pago | 606 | `41421d5e-bc52-4318-91db-494d75368009` | BANK `74e75926-60f5-4755-a3f6-6d2451b2af8a` (conta) · CREDIT `2c9d2ca0-6490-410a-9523-aba1a8fc45a5` (final 1529) |
| Clear Corretora | 623 | `978ed108-6a15-48c1-8353-07782162ed74` | BANK `6278c236-58d7-4d5c-93f8-05d8224b9fc0` (Banco XP 348) · BANK `f864ee76-128f-4bf1-879f-1d5d84c5ddbd` (XP CCTVM 102, onde caem os dividendos) |
| **Nubank PF** | 612 | `4fba1e15-0c91-4420-8389-dcfc89be60c3` | BANK `b298efbd-6eb8-4b8d-b170-4cb4710f1e21` (41722077-9) · CREDIT `832402c5-2d76-4dad-b661-5f0b8d1d7271` (final 5435, Mastercard) |
| **Nubank Empresas** | 664 | `3272dd35-fb8b-4579-aefd-dd64a0732171` | BANK `d1772561-029e-49e2-ad3d-b78c70824a08` (28156336-9) · CREDIT `4c275204-1c7e-4cb9-b8ee-036709e141ff` (final 0471) |

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

## Bruno's connection on the OLD account (found 2026-08-11 — historical reference)

| What | ID | Detail |
|---|---|---|
| Connection (`item_id`) | `0cd91348-06c1-4675-a570-8af6ba414ad1` | connector_id `606`, "Mercado Pago" |
| Checking account | `b1b50f0c-3b05-4f40-86b7-039a4fa949c2` | "Mercado Pago (Conta Pré-paga)", número 7750044166-2 |
| Credit card | `cd8caca1-6f86-46fa-8681-136534e994fe` | Visa, final 1529 |

If IDs stop working (reconnect expired, or a second bank gets added later), rediscover
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
  -d '{"account_id":"b1b50f0c-3b05-4f40-86b7-039a4fa949c2","from":"2026-08-01","to":"2026-08-31"}'
```

**Fatura (credit card bills)** — returns CLOSED bills only, with a derived
`payment_status` (`PAID` / `OPEN` / `PAST_DUE_UNCONFIRMED` / `PAST_DUE_UNPAID` — read
the `payment_status_legend` in the response, it explains the cross-bill heuristic):
```bash
curl -s -X POST "https://api.mcp.ai/api/openfinance/credit-card-bills/list" \
  -H "Authorization: Bearer $MCP_AI_API_KEY" -H "Content-Type: application/json" \
  -d '{"account_id":"cd8caca1-6f86-46fa-8681-136534e994fe"}'
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

Real Mercado Pago **credit card** via this API: limit R$ 4.400,00, used R$ 2.818,69,
available R$ 1.581,31. The **internal system** record for "Cartão Mercado Pago"
(`73567965-1f0b-4d2e-868a-b1cfdc05fbbd`) has `creditLimit: 2600` and
`currentBalance: -739.79` — doesn't match. Don't patch either number; this needs the
same investigation approach as [[finances-reconciliation-open-2026-08]] (find the
missing/wrong transactions, or confirm the card's real limit changed and update the
static field — never force the balance).
