# Calendar

**Personal life-ops platform** — calendar, finances, health and AI agents in one multi-container app. Integrates Google Calendar, Linear, Mercado Pago, Binance and B3 market data, with LLM-powered agents (LangGraph + Langfuse) doing transaction entry and CRM research.

[![CI](https://github.com/wbrunovieira/calendar/actions/workflows/ci.yml/badge.svg)](https://github.com/wbrunovieira/calendar/actions/workflows/ci.yml)
[![CD](https://github.com/wbrunovieira/calendar/actions/workflows/deploy.yml/badge.svg)](https://github.com/wbrunovieira/calendar/actions/workflows/deploy.yml)

![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?logo=typescript&logoColor=white)
![NestJS](https://img.shields.io/badge/NestJS-E0234E?logo=nestjs&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-000000?logo=next.js&logoColor=white)
![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3776AB?logo=python&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)

## Services

| Service | Stack | Port | Purpose |
|---------|-------|------|---------|
| calendar-core | NestJS 11 + Prisma | 3334 | Events, habits, TODOs, reminders, recurrence (DDD) |
| calendar-finances | Go 1.23 + Gorilla Mux | 3335 | Transactions, invoices, investments, B3/Binance sync |
| calendar-health | Go 1.23 + Gorilla Mux | 3336 | Workouts, exercises, activity tracking |
| agents | Python 3.12 + FastAPI + LangGraph | 3337 | AI agents (transaction entry, CRM research) |
| calendar-frontend | Next.js 15 + React 19 | 3000 | Calendar UI |
| finances-frontend | Next.js 15 | 3003 | Finance dashboard |
| health-frontend | Next.js 15 | 3004 | Health dashboard |
| postgres | PostgreSQL 15 | 5433 | Shared DB (`public`, `finance`, `health` schemas) |
| Langfuse v3 | 6 containers | 3100 | LLM observability |

## Development

```bash
cp .env.example .env       # configure secrets (see CLAUDE.md)
docker-compose up -d       # backends run in Docker
cd services/calendar-frontend && npm run dev   # frontends run locally
```

Tests:

```bash
docker-compose exec calendar-core npm test
cd services/calendar-finances && go test ./...
cd services/calendar-health && go test ./...
docker-compose exec agents python -m pytest tests/
```

## CI/CD

- **CI** (`.github/workflows/ci.yml`): builds and tests every service on push/PR to `main` — NestJS unit + E2E (real Postgres), Go test suites, pytest, and the three Next.js builds.
- **CD** (`.github/workflows/deploy.yml`): after CI passes on `main`, waits for manual approval (environment `production`), then deploys via SSH to the VPS as a dedicated non-root user — only rebuilding the services whose code changed, running Prisma migrations when needed, and health-checking every endpoint.

Architecture details and conventions: see [CLAUDE.md](./CLAUDE.md).
