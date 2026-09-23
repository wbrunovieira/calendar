# CRM → finances contract sync

`POST /api/v1/contracts/sync` mirrors a deal decided in the CRM into the ledger.

It is a **server-to-server** route and not part of the dashboard API.

**On reachability, stated accurately — measured, not assumed.** An earlier version
of this document claimed the route was "never published through Nginx". That is
false: `calendar-finances.wbdigitalsolutions.com` proxies `location /` straight to
the container.

What actually guards it today is Authelia. The live vhost includes
`authelia-authrequest.conf` on `location /`, and an unauthenticated
`POST /api/v1/contracts/sync` against the public name answers **302** to the login
gate — measured on the server, not inferred from the playbook, which had drifted
and does not contain that include.

On top of that the configs now `deny all` on `/api/v1/contracts/`, so the route is
closed at the edge even if the Authelia include is ever lost. The CRM is unaffected
either way: it reaches the container over the Docker bridge and never passes through
Nginx at all.

## What it is, and what it is deliberately not

The endpoint stores a **mirror** of the deal — value, currency, status, client — and
nothing else. It does **not** create receivables, and that is a decision rather than
an omission:

- The CRM does not transmit an instalment plan, on purpose. A payment schedule has a
  due date and a paid/unpaid state, and the CRM holds neither. The schedule is born
  here, where the contract form lives; duplicating it there would create two sources
  of truth.
- Generating forecast entries from the OPEN half of the funnel would be confidently
  wrong today. The CRM drops any deal with no organization linked, and as of
  23/09/2026 only 1 of 15 open deals has one — R$ 2.850 of a R$ 9.090 pipeline would
  arrive, with nothing marking the rest as missing. A forecast that looks complete
  and is not is worse than no forecast.

What arrives complete is the CLOSED half: all 14 won deals have an organization,
because winning converts the lead into a client. That is what the mirror is for.

## Request

```
POST /api/v1/contracts/sync
Content-Type: application/json
x-webhook-secret: <CRM_SYNC_WEBHOOK_SECRET>
```

```json
{
  "source": "wb-crm",
  "deal": {
    "id": "<uuid in the CRM>",
    "title": "Site institucional",
    "totalValue": 350.00,
    "currency": "BRL",
    "status": "open" | "won" | "lost",
    "closedAt": "2026-09-23T12:00:00.000Z",
    "updatedAt": "2026-09-23T12:00:00.000Z"
  },
  "organization": { "id": "<uuid in the CRM>", "name": "Refrigeracao Garrido" }
}
```

State is **complete on every delivery, never a delta**. A `totalValue` of `null`
means the deal has no value — it is stored as NULL, not as zero, because a zero
reads back as free work.

## Responses

| Code | Meaning |
|---|---|
| `200` | Stored. `data.outcome` says what happened. |
| `400` | The payload will never work. Do not retry it. |
| `401` | Wrong or missing `x-webhook-secret`. |
| `500` | Our failure. Retry. |
| `503` | `CRM_SYNC_WEBHOOK_SECRET` is not configured on the server. |

```json
{"data": {"contractId": "...", "costCenterId": "...",
          "outcome": "created|updated|unchanged|stale",
          "changes": [{"field": "status", "from": "OPEN", "to": "WON"}]}}
```

`stale` is a success: the delivery was recognised as one the contract has already
passed and was dropped without writing.

## The two rules that make it safe to retry

**`deal.id` is the idempotency key.** Upsert is on `(source, external_id)`, and the
uniqueness lives in a database index, not only in the use case — two deliveries for
the same deal can race, and only the database can settle that. The loser of the
insert re-reads the winner's row and applies on top instead of failing.

**`deal.updatedAt` is the ordering stamp.** A delivery whose stamp is not strictly
newer than the stored one is dropped. Equal counts as already seen: the sender's
`updatedAt` changes on every write, so the same stamp is the same write.

Without the second rule, a redelivery that arrives out of order overwrites new state
with old — and a sale that fell through would quietly come back to life in the
forecast.

## The client

The `organization` resolves to a cost center via `(profile_id, external_source,
external_id)`, created the first time that organization is seen. An existing cost
center is **never renamed** by a sync: its name is a local label that appears across
the ledger and may have been adjusted on purpose, while the external reference —
the part that actually links the two systems — never changes.

## Configuration

| Variable | Meaning |
|---|---|
| `CRM_SYNC_WEBHOOK_SECRET` | The shared secret. **Unset means the route answers 503 and stores nothing** — it fails closed. Distinct from `CRM_WEBHOOK_SECRET`, which is what `agents` uses to call *into* the CRM: opposite direction, different trust relationship. |
| `FINANCE_BUSINESS_PROFILE_ID` | The profile deals are filed under. Defaults to WB Digital Solutions and is logged at startup. Deliberately **not** taken from the payload: the sender has no idea which ledger profile it is feeding, and letting it name one would let a webhook write into any profile. |

## Timestamps

`remote_updated_at` and `closed_at` are `TIMESTAMPTZ`, and that is load-bearing
rather than stylistic. A plain `TIMESTAMP` keeps the wall clock and **discards the
offset**: a stamp of `15:00-03:00` stores as `15:00` and reads back as `15:00Z`,
three hours before the instant that was sent. Two things break at once — a
genuinely older delivery beats a newer one (the exact failure the ordering rule
exists to prevent), and the stored value is permanently behind the incoming one, so
nothing is ever recognised as stale and every delivery rewrites the row.

Send whatever ISO form is convenient; ordering is by instant.

## Still to build

Generating the receivable when a deal turns `won`, and reporting a divergence when
remote state contradicts a **CONFIRMED** entry — a confirmed entry is never rewritten
by a sync; the conflict is reported instead.
