---
name: track-work
description: Rastreia todo trabalho do WB Calendar (melhorias, correções, features, débito técnico) como issues no WB Project Manager via a API, e mantém o status atualizado. Use SEMPRE ao planejar/iniciar trabalho não-trivial, ao descobrir um bug/melhoria, ao começar (→ In Progress) e ao concluir (→ Done). Também ao pedir "cria as issues", "atualiza o board".
---

# track-work — rastrear trabalho do WB Calendar como issues

Toda melhoria/correção vira issue no projeto **WB Calendar** do WB Project
Manager, com status em dia. Não rastreie trivialidades (typo, 1 linha).

## Constantes
- Base URL: `https://projects.wbdigitalsolutions.com`
- `projectId: cmor7uoo4000jpa01zd6gywrq`  |  `workspaceId: cmge96f200001wa7ouziczg0w`
- API key: `~/.wb-project-manager-api-key` — NUNCA ecoar; ler via `$(cat ...)`.

| Status | statusId | type |
|---|---|---|
| Backlog | cmge9i3pt0005walququqw1rx | BACKLOG |
| Todo | cmge9i3pv0007walqv7is970v | TODO |
| In Progress | cmge9i3pv0009walqbwhmule6 | IN_PROGRESS |
| Done | cmge9i3pw000bwalqn1glwrn4 | DONE |
| Canceled | cmge9i3pw000dwalqi5qgpguo | CANCELED |

## Comandos
```bash
KEY=$(cat ~/.wb-project-manager-api-key)
BASE=https://projects.wbdigitalsolutions.com

# Listar (evite duplicar) — no GET o filtro é status=<TYPE>
curl -s -H "Authorization: Bearer $KEY" "$BASE/api/issues?projectId=cmor7uoo4000jpa01zd6gywrq"

# Criar issue — no POST/PATCH use statusId=<cuid>
curl -s -X POST "$BASE/api/issues" -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"title":"...","description":"...","workspaceId":"cmge96f200001wa7ouziczg0w","projectId":"cmor7uoo4000jpa01zd6gywrq","statusId":"cmge9i3pv0007walqv7is970v","type":"IMPROVEMENT","priority":"MEDIUM"}'

# Mudar status (In Progress / Done) — dispara SLA automático
curl -s -X PATCH "$BASE/api/issues/ISSUE_ID" -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"statusId":"cmge9i3pv0009walqbwhmule6"}'
```
Milestones: `POST /api/milestones` com `{name, projectId, targetDate?}`. Lote: `POST /api/issues/bulk` `{workspaceId, issues:[...]}` (até 100). Doc: `/api/docs`.

## Gotchas
- GET filtra por `status=<TYPE>`; POST/PATCH usam `statusId=<cuid>`.
- Datas ISO 8601; `milestoneId`/`assigneeId` aceitam `null`. Bulk máx 100.
- Key errada → 401; não parseie o corpo, confie no status.
- A UI não auto-atualiza após criar via API — dê refresh para ver.
