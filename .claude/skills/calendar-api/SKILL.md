---
name: calendar-api
description: Como usar a API do calendar-core (agenda/eventos, hábitos, todos, lembretes, calendários) via HTTP — base URLs, token de auth, e o Swagger. Use ao ler/criar/editar/apagar eventos-hábitos-todos por API, ou ao mandar pra outro Claude/agente como consumir a API.
---

# calendar-api — usar a API do calendar-core

Agenda (EVENT), hábito (HABIT), todo (TODO) e lembrete (REMINDER) são **o mesmo recurso `/events`**, distinguidos pelo campo `eventType`. O `calendarId` no corpo define em qual calendário o evento entra.

## Base URLs
- Produção: `https://calendar-api.wbdigitalsolutions.com`
- Local (dev): `http://localhost:3334`

## Autenticação
A API exige um token quando `API_TOKEN` está setado (em produção, está). Header: `Authorization: Bearer <token>` (ou `X-API-Key: <token>`).
- Token salvo em `~/.calendar-api-token` (chmod 600). NUNCA ecoar; ler via `$(cat ...)`.
- É o mesmo valor do `API_TOKEN` no `.env` do servidor. Token errado/ausente → **401** (não parseie o corpo, confie no status).
- Os `/docs` e `/docs-json` são **públicos** (não exigem token).

## Documentação viva (Swagger)
- UI: `https://calendar-api.wbdigitalsolutions.com/docs` — Authorize 🔓 → cola o token → Try it out.
- Spec OpenAPI (pra agentes lerem): `https://calendar-api.wbdigitalsolutions.com/docs-json` — tem cada rota, corpo (DTO) e resposta. **Leia o spec** em vez de assumir campos.

## Rotas principais (base sem prefixo global)
```bash
TOK=$(cat ~/.calendar-api-token)
BASE=https://calendar-api.wbdigitalsolutions.com

# Listar (filtra por tipo): eventType=EVENT|HABIT|TODO|REMINDER
curl -s -H "Authorization: Bearer $TOK" "$BASE/events?calendarId=<id>&eventType=TODO"

# Calendários (pra pegar o calendarId)
curl -s -H "Authorization: Bearer $TOK" "$BASE/calendars"

# Criar (obrigatórios: calendarId, title, startTime "HH:mm"; eventType default EVENT)
curl -s -X POST "$BASE/events" -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"calendarId":"<id>","title":"Academia","startTime":"07:00","eventType":"HABIT","recurrenceRule":"FREQ=WEEKLY;BYDAY=MO,WE,FR"}'

# Editar / apagar
curl -s -X PUT    "$BASE/events/<id>" -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" -d '{...}'
curl -s -X DELETE "$BASE/events/<id>" -H "Authorization: Bearer $TOK"

# Concluir hábito/todo numa data
curl -s -X POST "$BASE/events/executions/toggle" -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"eventId":"<id>","executionDate":"2026-07-20","completed":true}'
```
Outras: `GET /events/habits/stats`, `GET /events/habits/weekly-progress`, `POST /events/reorder`, `DELETE /events/<id>/recurring?scope=this|future|all`. Detalhes completos no `/docs-json`.

## Gotchas
- Sem prefixo global (`/events`, não `/api/events`).
- Não há validação de dono: `calendarId` no corpo é o único escopo — mande o certo.
- Ativar/rotacionar o token: setar `API_TOKEN` no `.env` do servidor E rebuildar os frontends (o token é build arg `NEXT_PUBLIC_*`, embutido no bundle). Ver [[wb-project-manager-tracking]] pra rastrear o trabalho.
