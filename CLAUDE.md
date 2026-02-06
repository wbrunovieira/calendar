# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Multi-container calendar application integrating Google Calendar accounts (professional and personal), Linear task management, and financial tracking with AI-powered agents. Single-user personal project.

## Quick Start

```bash
# 1. Start backend services
docker-compose up -d

# 2. Start calendar frontend (new terminal)
cd services/calendar-frontend && npm install && npm run dev

# 3. Start finances frontend (new terminal, optional)
cd services/finances-frontend && npm install && npm run dev
```

## Architecture

### Container Structure
- **calendar-core** (NestJS 11): Main API on port 3334 - Runs in Docker with hot-reload
- **calendar-frontend** (Next.js 15.5 + React 19): Calendar web interface on port 3000 - Runs locally
- **calendar-finances** (Go 1.23 + Gorilla Mux): Financial service on port 3335 - Runs in Docker
- **finances-frontend** (Next.js 15.5): Financial dashboard on port 3003 - Runs locally
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
    ├── repositories/        # Repository interfaces
    └── persistence/         # Prisma implementations
```

**Event Types:** Events have `eventType` field: `EVENT` (appointment), `HABIT` (habit), `TODO` (task), `REMINDER`. All stored in the same `events` table with type-specific fields (priority for TODOs, reminderDaysBefore for REMINDERs, weeklyTargetCount for flexible habits).

**Recurrence Pattern:** Master events with RRule format. Derived instances link via `recurrenceMasterId`. Overrides and exceptions tracked in separate tables.

**Path Aliases (vitest/tsconfig):**
- `@/` → `src/`
- `@domains/` → `src/domains/`
- `@common/` → `src/common/`

### Financial Module (calendar-finances)

Located in `services/calendar-finances/internal/`. Manual dependency wiring in `cmd/api/main.go` (repo → usecase → handler). Routes registered on Gorilla Mux with `/api/v1` prefix. Migrations are manual SQL strings in `internal/database/database.go`.

### AI Agents Module (agents)

FastAPI app with LangGraph. Graph compiled once at startup (lifespan), invoked per-request. Services communicate internally via Docker network (`http://calendar-finances:3335`, `http://langfuse-web:3000`).

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

### Frontends - Run locally
```bash
cd services/calendar-frontend && npm run dev     # Port 3000
cd services/finances-frontend && npm run dev     # Port 3003
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

## Database Schema

### Calendar Schema (Prisma)
Schema: `services/calendar-core/prisma/schema.prisma`

**Key Tables:**
- `events` - Main events with RRule recurrence, eventType (EVENT/HABIT/TODO/REMINDER)
- `event_completions` - Execution tracking (separate from modifications)
- `recurrence_exceptions` - Removed dates from recurring events
- `recurrence_overrides` - Modified instances of recurring events
- `calendars` - Professional/personal separation
- `category_types` - Modern categorization (health, work, leisure, etc.)
- `categories` - Legacy structure with M2M to category_types
- `labels` - Event labels

### Finance Schema (PostgreSQL)
Schema prefix: `finance.` — Migrations in Go code (`internal/database/database.go`), not Prisma.

**Key Tables:** `finance.profiles`, `finance.transactions`, `finance.bank_accounts`, `finance.recurring_transactions`, `finance.budget_targets`, `finance.categories`, `finance.invoices`

## API Endpoints

### calendar-core
- `/events` - CRUD + `/events/executions/toggle`, `/events/:id/executions`, `/events/stats`
- `/calendars`, `/categories`, `/category-types`, `/labels` - Standard CRUD

### calendar-finances
All prefixed with `/api/v1`:
- `/profiles`, `/bank-accounts`, `/transactions`, `/recurring-transactions`, `/budgets`, `/categories`

### agents
- `GET /health` — liveness (reports LangGraph availability)
- `GET /health?deep=1` — readiness (executes ping-pong graph)

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
