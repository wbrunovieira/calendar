# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ Always track work as issues (skill `track-work`)
Every improvement, fix, feature or piece of tech debt MUST become an issue in the
"WB Calendar" project (cmor7uoo4000jpa01zd6gywrq) of the WB Project Manager, with
its status kept current. Invoke the `track-work` skill
(.claude/skills/track-work/SKILL.md). API docs:
https://projects.wbdigitalsolutions.com/api/docs

## ⚠️ Language
Everything public on GitHub is written in **English** — code, comments, commit
messages, PR titles and bodies, and these docs. Conversation with Bruno happens in
Portuguese, and so do the WB Project Manager issues (that board is private), but
that must not leak into the repository. Domain labels that exist as data (account
and category names such as `Caixinha Mercado Pago` or `Aluguel`) stay in Portuguese
because they are the literal values stored in the database.

## Project Overview

Multi-container calendar application integrating Google Calendar accounts (professional and personal), Linear task management, financial tracking, and health/fitness tracking with AI-powered agents. Single-user personal project.

## Architecture

### Container Structure
- **calendar-core** (NestJS 11): Main API on port 3334 - Runs in Docker with hot-reload
- **calendar-frontend** (Next.js 15.5 + React 19): Calendar web interface on port 3000 - Runs locally
- **calendar-finances** (Go 1.23 + Gorilla Mux): Financial service on port 3335 - Runs in Docker
- **finances-frontend** (Next.js 15.5): Financial dashboard on port 3003 - Runs locally
- **calendar-health** (Go 1.23 + Gorilla Mux): Health & fitness tracking service on port 3336 - Runs in Docker
- **health-frontend** (Next.js 15.5): Health dashboard on port 3004 - Runs locally
- **agents** (Python 3.12 + FastAPI + LangGraph): AI agent service on port 3337 - Runs in Docker
- **postgres** (PostgreSQL 15): Database on port 5433 (host) / 5432 (container)
- **Langfuse v3**: 6 containers for LLM observability (web on port 3100, plus worker, postgres, clickhouse, minio, redis)

### Key Design Decisions
- **No Redis for calendar**: Single-user project uses in-memory cache and PostgreSQL-based queues (Redis is only used by Langfuse)
- **Separate finance schema**: `finance.*` tables isolated from `public.*` calendar tables
- **Separate Langfuse PostgreSQL**: Dedicated database to avoid migration conflicts
- **Frontends run locally**: Better DX with faster hot-reload outside Docker
- **Secrets via .env**: Langfuse keys (`LANGFUSE_ENCRYPTION_KEY`, `LANGFUSE_NEXTAUTH_SECRET`, `LANGFUSE_SALT`) have no defaults — must be set in `.env`

### Domain-Driven Design (calendar-core)

Located in `services/calendar-core/src/domains/`:

| Domain | Description |
|--------|-------------|
| `events/` | Main event management with recurrence (most complex) |
| `calendars/` | Professional/personal calendar separation |
| `categories/` | Event categorization (legacy) |
| `category-types/` | Modern category system (health, work, leisure, etc.) |
| `labels/` | Event labeling |
| `google-calendar/` | Google Calendar API integration (partial) |

**Domain Structure:**
```
src/domains/[domain]/
├── domain/entities/          # Domain entities with static create() factory
├── application/
│   └── use-cases/           # Business logic
└── infrastructure/
    ├── controllers/         # HTTP endpoints
    ├── dtos/                # class-validator DTOs (NOT in application/dto/)
    ├── repositories/        # Repository interfaces (TypeScript interfaces)
    └── persistence/         # Concrete Prisma implementations of those interfaces
```

Persistence classes call `Entity.create()` to hydrate domain objects from Prisma records. Repository implementations instantiate `PrismaClient` directly (no injection).

**Event Types:** Events have `eventType` field: `EVENT` (appointment), `HABIT` (habit), `TODO` (task), `REMINDER`. All stored in the same `events` table with type-specific optional fields:
- TODOs: `priority`, `dueDate`
- REMINDERs: `reminderDaysBefore` (array)
- HABITs: `recurrenceType` (`FIXED`/`FLEXIBLE`), `weeklyTargetCount`, `weeklyPreferredDays`

**Recurrence Pattern:** Master events with RRule format. Derived instances link via `recurrenceMasterId`. Overrides and exceptions tracked in separate tables.

**Path Aliases (vitest/tsconfig):**
- `@/` → `src/`
- `@domains/` → `src/domains/`
- `@common/` → `src/common/`

### Financial Module (calendar-finances)

Located in `services/calendar-finances/internal/`. Manual dependency wiring in `cmd/api/main.go` (repo → usecase → handler). Routes registered on Gorilla Mux with `/api/v1` prefix. Migrations are inline SQL strings in `internal/database/database.go` — idempotent, run automatically on startup. No ORM; uses `sql.DB` with connection pooling (25 max open, 5 idle).

Some handlers require post-construction injection: after creating a handler, additional use-cases are set via setter methods (e.g., `transactionHandler.SetDailyBalancesUseCase(dailyBalancesUC)`). Check `main.go` when adding new cross-handler dependencies.

**Background goroutines** started at launch (in `main.go`):
- Auto-close invoices: 24h loop
- Sync Binance trades: 30min loop
- B3 stock prices + dividends: 2h / 24h loops
- Close month checkpoints: monthly at 00:05 UTC

**Transaction model** supports splits (`CreateTransactionSplitInput` array) and installment tracking (`InstallmentNumber`, `InstallmentTotal`) for credit card flows.

### Health Module (calendar-health)

Same Go/Gorilla Mux pattern as calendar-finances. Uses `health.*` schema (separate from `public.*` and `finance.*`). Schema versioning via `health.schema_migrations` table.

### AI Agents Module (agents)

FastAPI app with LangGraph. Graphs compiled once at startup (lifespan event), never at request time. Services communicate internally via Docker network (`http://calendar-finances:3335`, `http://langfuse-web:3000`).

**Two named graphs:**
- **Transaction graph** — 5-node pipeline: `load_context → parse_message → resolve_entities → create_transaction → format_reply`. Conditional routing on errors.
- **CRM Lead Research graph** — 7-node graph with retry loops (up to 3 rounds) and supervisor review checkpoint.

**Langfuse prompt versioning:** Agents fetch prompts by name + label via `langfuse.get_prompt()` with in-memory caching. This decouples prompt changes from code deployments (production vs staging via labels).

### n8n (Workflow Automation)

External n8n server (not in docker-compose). Thin orchestration: scheduling, webhook routing, notification delivery. **Never contains business logic or LLM prompts.** Design: `n8n = when to run + where to deliver` / `agents = what to do + how to decide`.

### Key Integrations
- Google Calendar API (OAuth2)
- Linear API for project task tracking
- Mercado Pago API for financial transactions
- Nubank data import (CSV/OFX)
- Langfuse (LLM tracing and observability)
- n8n (cron scheduling, webhook routing, notification delivery)

## Development Commands

**CRITICAL**: Backend services ALWAYS run inside Docker. Never run `npm run start:dev` outside Docker.

### Docker (Backend)
```bash
docker-compose up -d                              # Start all services
docker-compose up -d --build                      # Rebuild and start
docker-compose logs -f calendar-core              # View logs
docker-compose exec calendar-core sh              # Shell access
docker-compose restart calendar-core              # Restart service
```

### calendar-core (NestJS) - Run inside Docker
```bash
docker-compose exec calendar-core npm run test                       # All unit tests (Vitest)
docker-compose exec calendar-core npm run test -- src/domains/events/application/use-cases/create-event.use-case.spec.ts  # Single file
docker-compose exec calendar-core npm run test:watch                 # Watch mode
docker-compose exec calendar-core npm run test:cov                   # Coverage (thresholds: 80% lines/functions/statements, 75% branches)
docker-compose exec calendar-core npm run test:e2e                   # E2E tests
docker-compose exec calendar-core npm run test:e2e:watch             # E2E watch mode
docker-compose exec calendar-core npm run lint                       # Lint (ESLint)
docker-compose exec calendar-core npm run format                     # Format (Prettier)
```

### calendar-finances (Go) - Run inside Docker
```bash
docker-compose exec calendar-finances go test ./...                    # All tests
docker-compose exec calendar-finances go test -v ./internal/domain/transaction/...  # Specific package
docker-compose exec calendar-finances go test -run TestCreateTransaction ./internal/application/usecases/...  # Single test by name
docker-compose exec calendar-finances go test -cover ./...             # Coverage
docker-compose exec calendar-finances go build -o bin/api cmd/api/main.go  # Build
```

### agents (Python) - Runs in Docker
```bash
docker-compose logs -f agents                         # View logs
curl http://localhost:3337/health                              # Liveness check
curl "http://localhost:3337/health?deep=1"                     # Deep check (executes graph)
docker-compose exec agents pip install <package>      # Install package
```

### calendar-health (Go) - Run inside Docker
```bash
docker-compose exec calendar-health go test ./...                      # All tests
docker-compose exec calendar-health go test -v ./internal/domain/...   # Specific package
docker-compose exec calendar-health go test -cover ./...               # Coverage
```

### Frontends - Run locally
```bash
cd services/calendar-frontend && npm run dev     # Port 3000
cd services/finances-frontend && npm run dev     # Port 3003
cd services/health-frontend && npm run dev       # Port 3004
```

### Database
```bash
psql -h localhost -p 5433 -U calendar -d calendar_db           # Connect from host
docker-compose exec postgres psql -U calendar -d calendar_db   # Connect from container

# Prisma (calendar-core)
docker-compose exec calendar-core npx prisma migrate dev --name "description"
docker-compose exec calendar-core npx prisma generate
docker-compose exec calendar-core npx prisma studio            # GUI at localhost:5555
docker-compose exec calendar-core npm run prisma:seed           # Seed data
```

### Test Database Setup
```bash
bash scripts/setup-test-db.sh    # Create test database (calendar_test_db)
bash scripts/reset-test-db.sh    # Reset test database
bash scripts/seed-test-db.sh     # Seed test database
```

## Testing

### Frameworks
- **calendar-core**: Vitest + vitest-mock-extended + supertest (E2E)
- **calendar-finances**: Go native testing with fake repositories
- **calendar-frontend**: Vitest + Testing Library

### Test Patterns

**calendar-core test helpers** (`src/test/helpers/`):
- `fixtures.ts` — `createUserFixture()`, `createCalendarFixture()`, etc.
- `mock-builders.ts` — `createMockRepository()`, `createMockPrisma()`
- `test-utils.ts` — `useFakeTimers()`, `spyOnConsole()`, `expectToThrow()`

**calendar-finances test helpers** (`internal/test/helpers/fixtures.go`):
- `FixedTime()`, `CreateTestProfile()`, `CreateTestBankAccount()`, `CreateTestTransaction()`

### Important Notes
- Use `useFakeTimers()` for date-dependent tests (defaults to `2024-11-16T10:00:00-03:00`)
- Tests use `America/Sao_Paulo` timezone (hardcoded in Docker `TZ` env)
- Mock external APIs (Google Calendar, Linear, Mercado Pago)
- E2E tests have 30s timeout (`vitest.config.e2e.ts`)
- E2E setup in `test/setup-e2e.ts` includes `cleanDatabase()` helper

## Environment

Copy `.env.example` to `.env` and configure:
- Database credentials (defaults work with docker-compose)
- Google OAuth credentials
- Linear API key
- Mercado Pago access token
- JWT secret
- Langfuse secrets (**required**, no defaults): `LANGFUSE_ENCRYPTION_KEY`, `LANGFUSE_NEXTAUTH_SECRET`, `LANGFUSE_SALT` — generate with `openssl rand -hex 32`
- Langfuse API keys (for agents): `LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY` — create in Langfuse UI at http://localhost:3100
- LLM provider key: `OPENAI_API_KEY` or `DEEPSEEK_API_KEY`

## Deployment

Ansible playbooks in `deploy/ansible/playbooks/`:
- `deploy.yml` — Full deployment
- `quick-deploy.yml` — Fast redeploy
- `rollback.yml` — Rollback to previous
- `setup-nginx-ssl.yml` — Nginx + Let's Encrypt

Vault secrets in `deploy/ansible/inventory/group_vars/all/vault.yml` (gitignored).
