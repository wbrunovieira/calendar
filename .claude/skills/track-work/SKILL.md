---
name: track-work
description: Tracks all WB Calendar work (improvements, fixes, features, tech debt) as issues in the WB Project Manager via its API, and keeps their status current. Use ALWAYS when planning/starting non-trivial work, when spotting a bug/improvement, when starting (→ In Progress) and when finishing (→ Done). Also on "create the issues", "update the board".
---

# track-work — track WB Calendar work as issues

Every improvement/fix becomes an issue in the **WB Calendar** project of the WB
Project Manager, with its status kept current. Don't track trivia (typos, one-liners).

## Constants
- Base URL: `https://projects.wbdigitalsolutions.com`
- `projectId: cmor7uoo4000jpa01zd6gywrq`  |  `workspaceId: cmge96f200001wa7ouziczg0w`
- API key: `~/.wb-project-manager-api-key` — NEVER echo it; read it via `$(cat ...)`.

| Status | statusId | type |
|---|---|---|
| Backlog | cmge9i3pt0005walququqw1rx | BACKLOG |
| Todo | cmge9i3pv0007walqv7is970v | TODO |
| In Progress | cmge9i3pv0009walqbwhmule6 | IN_PROGRESS |
| Done | cmge9i3pw000bwalqn1glwrn4 | DONE |
| Canceled | cmge9i3pw000dwalqi5qgpguo | CANCELED |

## Commands
```bash
KEY=$(cat ~/.wb-project-manager-api-key)
BASE=https://projects.wbdigitalsolutions.com

# List (avoid duplicates) — on GET the filter is status=<TYPE>
curl -s -H "Authorization: Bearer $KEY" "$BASE/api/issues?projectId=cmor7uoo4000jpa01zd6gywrq"

# Create an issue — on POST/PATCH use statusId=<cuid>
curl -s -X POST "$BASE/api/issues" -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"title":"...","description":"...","workspaceId":"cmge96f200001wa7ouziczg0w","projectId":"cmor7uoo4000jpa01zd6gywrq","statusId":"cmge9i3pv0007walqv7is970v","type":"IMPROVEMENT","priority":"MEDIUM"}'

# Change status (In Progress / Done) — triggers SLA tracking automatically
curl -s -X PATCH "$BASE/api/issues/ISSUE_ID" -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"statusId":"cmge9i3pv0009walqbwhmule6"}'
```
Milestones: `POST /api/milestones` with `{name, projectId, targetDate?}`. Bulk: `POST /api/issues/bulk` `{workspaceId, issues:[...]}` (up to 100). Docs: `/api/docs`.

## Gotchas
- GET filters by `status=<TYPE>`; POST/PATCH use `statusId=<cuid>`.
- ISO 8601 dates; `milestoneId`/`assigneeId` accept `null`. Bulk max 100.
- Wrong key → 401; don't parse the body, trust the status.
- The UI does not auto-refresh after an API create — hit refresh to see it.

> Issue titles and descriptions stay in Portuguese: the board is private and its
> existing issues are in Portuguese. Everything public on GitHub is English.
