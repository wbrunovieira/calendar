---
name: calendar-api
description: How to use the calendar-core API (agenda/events, habits, todos, reminders, calendars) over HTTP — base URLs, auth token, and Swagger. Use when reading/creating/editing/deleting events-habits-todos via the API, or when handing another Claude/agent instructions on how to consume it.
---

# calendar-api — using the calendar-core API

Agenda (EVENT), habit (HABIT), todo (TODO) and reminder (REMINDER) are **the same `/events` resource**, told apart by the `eventType` field. The `calendarId` in the body decides which calendar the event lands in.

## Base URLs
- Production: `https://calendar-api.wbdigitalsolutions.com`
- Local (dev): `http://localhost:3334`

## Authentication
The API requires a token whenever `API_TOKEN` is set (it is, in production). Header: `Authorization: Bearer <token>` (or `X-API-Key: <token>`).
- Token stored in `~/.calendar-api-token` (chmod 600). NEVER echo it; read it via `$(cat ...)`.
- Same value as `API_TOKEN` in the server's `.env`. Wrong/missing token → **401** (don't parse the body, trust the status).
- `/docs` and `/docs-json` are **public** (no token required).

## Living documentation (Swagger)
- UI: `https://calendar-api.wbdigitalsolutions.com/docs` — Authorize 🔓 → paste the token → Try it out.
- OpenAPI spec (for agents to read): `https://calendar-api.wbdigitalsolutions.com/docs-json` — every route, body (DTO) and response. **Read the spec** instead of assuming fields.

## Main routes (no global prefix)
```bash
TOK=$(cat ~/.calendar-api-token)
BASE=https://calendar-api.wbdigitalsolutions.com

# List (filter by type): eventType=EVENT|HABIT|TODO|REMINDER
curl -s -H "Authorization: Bearer $TOK" "$BASE/events?calendarId=<id>&eventType=TODO"

# Calendars (to get the calendarId)
curl -s -H "Authorization: Bearer $TOK" "$BASE/calendars"

# Create (required: calendarId, title, startTime "HH:mm"; eventType defaults to EVENT)
curl -s -X POST "$BASE/events" -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"calendarId":"<id>","title":"Academia","startTime":"07:00","eventType":"HABIT","recurrenceRule":"FREQ=WEEKLY;BYDAY=MO,WE,FR"}'

# Edit / delete
curl -s -X PUT    "$BASE/events/<id>" -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" -d '{...}'
curl -s -X DELETE "$BASE/events/<id>" -H "Authorization: Bearer $TOK"

# Complete a habit/todo on a given date
curl -s -X POST "$BASE/events/executions/toggle" -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"eventId":"<id>","executionDate":"2026-07-20","completed":true}'
```
Others: `GET /events/habits/stats`, `GET /events/habits/weekly-progress`, `POST /events/reorder`, `DELETE /events/<id>/recurring?scope=this|future|all`. Full details in `/docs-json`.

## Gotchas
- No global prefix (`/events`, not `/api/events`).
- There is no ownership validation: `calendarId` in the body is the only scoping — send the right one.
- Enabling/rotating the token: set `API_TOKEN` in the server's `.env` AND rebuild the frontends (the token is a `NEXT_PUBLIC_*` build arg, inlined into the bundle). See [[wb-project-manager-tracking]] to track the work.
