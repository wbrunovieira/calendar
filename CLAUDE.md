# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Multi-container calendar application integrating Google Calendar accounts (professional and personal), Linear task management, and financial tracking with AI-powered agents.

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
- **calendar-agents** (Python 3.12 + FastAPI + LangGraph): AI agent service on port 3337 - Runs in Docker
- **postgres** (PostgreSQL 15): Database on port 5433 (host) / 5432 (container)

#### Langfuse v3 (Observability Stack)
- **langfuse-web** (Langfuse 3): Dashboard UI on port 3100
- **langfuse-worker** (Langfuse Worker 3): Background job processor on port 3030 (localhost only)
- **langfuse-postgres** (PostgreSQL 17): Langfuse database on port 5434 (localhost only)
- **langfuse-clickhouse** (ClickHouse): Analytics engine on port 8123/9001 (localhost only)
- **langfuse-minio** (MinIO): S3-compatible object storage on port 9090 (API) / 9091 (console, localhost only)
- **langfuse-redis** (Redis 7): Queue/cache on port 6379 (localhost only)

### Key Design Decisions
- **No Redis for calendar**: Single-user personal project uses in-memory cache and PostgreSQL-based queues (Redis is only used by Langfuse)
- **Separate finance schema**: `finance.*` tables isolated from `public.*` calendar tables
- **Separate Langfuse PostgreSQL**: Dedicated database to avoid migration conflicts with calendar-postgres
- **Frontends run locally**: Better DX with faster hot-reload outside Docker
- **Secrets via .env**: Langfuse sensitive keys (`LANGFUSE_ENCRYPTION_KEY`, `LANGFUSE_NEXTAUTH_SECRET`, `LANGFUSE_SALT`) have no defaults in docker-compose.yml — must be set in `.env`

### Domain-Driven Design (calendar-core)

Located in `services/calendar-core/src/domains/`:

| Domain | Description |
|--------|-------------|
| `events/` | Main event management with recurrence (most complex) |
| `calendars/` | Professional/personal calendar separation |
| `categories/` | Event categorization (legacy) |
| `category-types/` | Modern category system (health, work, leisure, etc.) |
| `google-calendar/` | Google Calendar API integration (partial) |

**Domain Structure:**
```
src/domains/[domain]/
├── domain/entities/          # Domain entities
├── application/
│   ├── use-cases/           # Business logic
│   └── dto/                 # Data transfer objects
└── infrastructure/
    ├── controllers/         # HTTP endpoints
    ├── repositories/        # Repository interfaces
    └── persistence/         # Prisma implementations
```

### Financial Module (calendar-finances)

Located in `services/calendar-finances/internal/`:

| Directory | Description |
|-----------|-------------|
| `domain/` | Entities: transaction, profile, bankaccount, category, budgettarget, recurringtransaction, invoice |
| `application/usecases/` | Business logic |
| `infrastructure/http/` | HTTP handlers and routes |
| `infrastructure/persistence/` | Repository implementations |
| `database/` | PostgreSQL connection and migrations |

**Design Decision:** Only recurring bills appear on calendar; all financial analysis happens in dedicated dashboard.

### AI Agents Module (calendar-agents)

Located in `services/calendar-agents/`:

```
services/calendar-agents/
├── Dockerfile              # Multi-stage (builder + runtime), non-root user
├── requirements.txt        # fastapi, uvicorn, langgraph, langchain-core, langfuse, httpx
├── .env.example
└── app/
    ├── __init__.py
    ├── main.py             # FastAPI + lifespan (imports/compiles graph at startup, no execution)
    ├── health.py           # GET /health (liveness) + GET /health?deep=1 (readiness with graph execution)
    └── graph.py            # StateGraph ping-pong (build_graph + verify_graph)
```

**Health Endpoints:**
- `GET /health` — liveness check, reports if LangGraph compiled successfully
- `GET /health?deep=1` — readiness check, executes the ping-pong graph and returns result

### Langfuse v3 (Observability)

LLM observability platform for tracing and monitoring AI agent executions. Accessible at `http://localhost:3100`.

**Infrastructure:** 6 containers (web, worker, postgres, clickhouse, minio, redis) all on `calendar-network`. ClickHouse uses a custom Keeper config (`config/clickhouse/clickhouse-config.xml`) for `ReplicatedMergeTree` support.

**Internal communication:** Containers use internal Docker hostnames (e.g., `http://langfuse-web:3000`, `http://langfuse-minio:9000`), not localhost.

### n8n (Workflow Automation)

External n8n server available for workflow automation. **Not part of docker-compose** — runs independently.

**Role:** Thin orchestration layer — handles scheduling, webhook routing and notification delivery only. Never contains business logic, LLM prompts or data analysis.

| Use case | How |
|----------|-----|
| **Cron scheduling** | n8n triggers calendar-agents endpoints on schedule (e.g., daily briefing at 7h, weekly review on Mondays) |
| **Messaging webhooks** | n8n receives Telegram/WhatsApp webhooks via native nodes, extracts text + chat_id, forwards to `POST /agents/parse` on calendar-agents |
| **Notification delivery** | calendar-agents produces notifications via API, n8n consumes and delivers to the right channel (Telegram, email, WhatsApp, SMS) |

**Design principle:** `n8n = when to run + where to deliver` / `calendar-agents = what to do + how to decide`

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
docker-compose exec calendar-core npm run lint           # Lint
docker-compose exec calendar-core npm run format         # Format
docker-compose exec calendar-core npm run test           # Unit tests (Vitest)
docker-compose exec calendar-core npm run test:watch     # Watch mode
docker-compose exec calendar-core npm run test:cov       # Coverage
docker-compose exec calendar-core npm run test:e2e       # E2E tests
```

### calendar-finances (Go) - Run inside Docker
```bash
docker-compose exec calendar-finances go test ./...                    # All tests
docker-compose exec calendar-finances go test -v ./...                 # Verbose
docker-compose exec calendar-finances go test ./internal/domain/...    # Specific package
docker-compose exec calendar-finances go test -cover ./...             # Coverage
docker-compose exec calendar-finances go build -o bin/api cmd/api/main.go  # Build
```

### calendar-agents (Python) - Runs in Docker
```bash
docker-compose logs -f calendar-agents                         # View logs
curl http://localhost:3337/health                              # Liveness check
curl http://localhost:3337/health?deep=1                       # Deep check (executes graph)
docker-compose exec calendar-agents pip install <package>      # Install package
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
docker-compose exec calendar-core npx prisma migrate dev
docker-compose exec calendar-core npx prisma generate
docker-compose exec calendar-core npx prisma studio
```

### Test Database Setup
```bash
bash scripts/setup-test-db.sh    # Create test database
bash scripts/reset-test-db.sh    # Reset test database
bash scripts/seed-test-db.sh     # Seed test database
```

## Database Schema

### Calendar Schema (Prisma)
Schema: `services/calendar-core/prisma/schema.prisma`

**Key Tables:**
- `events` - Main events with RRule recurrence support
- `event_completions` - Execution tracking (separate from modifications)
- `recurrence_exceptions` - Removed dates from recurring events
- `recurrence_overrides` - Modified instances of recurring events
- `calendars` - Professional/personal separation
- `category_types` - Modern categorization (health, work, leisure, etc.)
- `categories` - Legacy structure with M2M to category_types

**Recurrence Pattern:** Master events with derived instances using RRule format.

### Finance Schema (PostgreSQL)
Schema prefix: `finance.`

**Key Tables:**
- `finance.profiles` - Financial profiles linked to calendar users
- `finance.transactions` - Financial transactions with status
- `finance.recurring_transactions` - Recurring bills/income
- `finance.budget_targets` - Monthly budget limits per category

## API Endpoints

### calendar-core
- `/events` - CRUD + `/events/executions/toggle`, `/events/:id/executions`, `/events/stats`
- `/calendars`, `/categories`, `/category-types` - Standard CRUD

### calendar-finances
All prefixed with `/api/v1`:
- `/profiles`, `/bank-accounts`, `/transactions`, `/recurring-transactions`, `/budgets`, `/categories`

### calendar-agents
- `GET /health` — liveness (reports LangGraph availability)
- `GET /health?deep=1` — readiness (executes ping-pong graph)

## Testing

### Frameworks
- **calendar-core**: Vitest + vitest-mock-extended + supertest (E2E)
- **calendar-finances**: Go native testing with fake repositories

### Test Files
- calendar-core: `*.spec.ts` next to source, E2E in `test/`
- calendar-finances: `*_test.go` next to source

### Test Helpers
- calendar-core: `src/test/helpers/` - fixtures, mock-builders, test-utils
- calendar-finances: `internal/test/helpers/` - fixtures

### Important Notes
- Use `fixedTime()` / `useFakeTimers()` for date-dependent tests
- Tests use `America/Sao_Paulo` timezone by default
- Mock external APIs (Google Calendar, Linear, Mercado Pago)

## Environment

Copy `.env.example` to `.env` and configure:
- Database credentials (defaults work with docker-compose)
- Google OAuth credentials
- Linear API key
- Mercado Pago access token
- JWT secret
- Langfuse secrets (**required**, no defaults): `LANGFUSE_ENCRYPTION_KEY`, `LANGFUSE_NEXTAUTH_SECRET`, `LANGFUSE_SALT` — generate with `openssl rand -hex 32`
- Langfuse API keys (for calendar-agents): `LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY` — create in Langfuse UI at http://localhost:3100
- LLM provider key: `OPENAI_API_KEY`