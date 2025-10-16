# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Multi-container calendar application integrating Google Calendar accounts (professional and personal), Linear task management, and financial tracking with AI-powered agents.

## Architecture

### Container Structure
- **calendar-core** (NestJS): Main API, authentication, Google Calendar sync, Linear API integration - Runs in Docker on port 3334
- **calendar-frontend** (Next.js 15.5): Calendar web interface - Runs locally on port 3000
- **calendar-finances** (Go 1.23): Financial service with Gorilla Mux - Runs in Docker on port 3335
- **finances-frontend** (Next.js 15.5): Financial dashboard - Runs locally on port 3003
- **postgres**: Primary database (PostgreSQL 15) - Port 5433 (host), 5432 (container)

### Architecture Decision: No Redis
This is a personal project for a single user, so Redis was removed to reduce costs and complexity:
- **Cache**: Using NestJS in-memory cache or PostgreSQL native caching
- **Job Queues**: Using PostgreSQL-based queues (pg-boss or BullMQ with PostgreSQL adapter)
- **Sessions**: Using stateless JWT tokens or PostgreSQL session storage

### Domain-Driven Design Structure
The calendar-core service follows DDD architecture with 5 implemented domains:

**Domains:**
- `calendars/` - Calendar management (professional/personal separation)
- `categories/` - Event categorization (legacy structure)
- `category-types/` - Modern category type system (health, work, leisure, etc.)
- `events/` - Main event management with recurrence support (most complex)
- `google-calendar/` - Google Calendar API integration (partial)

**Domain Structure:**
- `src/domains/[domain]/domain/entities/` - Domain entities
- `src/domains/[domain]/application/use-cases/` - Business logic (create, update, delete, list)
- `src/domains/[domain]/application/dto/` - Data transfer objects
- `src/domains/[domain]/infrastructure/controllers/` - HTTP endpoints
- `src/domains/[domain]/infrastructure/repositories/` - Data persistence
- `src/domains/[domain]/infrastructure/persistence/` - Prisma implementations

**Events Domain Use Cases:**
- create-event, update-event, delete-event, list-events
- toggle-event-execution (mark as complete/incomplete)
- get-event-executions (completion history)
- get-events-stats (analytics by day/week/month/category)

### Key Integrations
- Google Calendar API (OAuth2 for bruno@wbdigitalsolutions.com and wbrunovieira77@gmail.com)
- Linear API for project task tracking and auto-management
- Mercado Pago API for financial transactions
- Nubank data import (CSV/OFX or email parsing)

### Frontend Development (Local)

Both frontends run locally (not in Docker) for better performance:

**calendar-frontend (port 3000):**
```bash
cd services/calendar-frontend
npm install
npm run dev        # Start dev server
npm run build      # Build for production
npm run lint       # Run linter
```

**finances-frontend (port 3003):**
```bash
cd services/finances-frontend
npm install
npm run dev        # Start dev server
npm run build      # Build for production
```

**Note**: Frontends require backend services running. Start with `docker-compose up -d` first.

## Development Commands

**IMPORTANT**: calendar-core ALWAYS runs inside Docker. NEVER try to run `npm run start:dev` or any npm commands outside of Docker for the backend. The backend is already running with hot-reload inside the container.

### Docker Operations (Backend Only)
```bash
# Start all services (backend is already running with hot-reload)
docker-compose up -d

# Start with rebuild
docker-compose up -d --build

# View logs for specific service
docker-compose logs -f calendar-core

# Stop all services
docker-compose down

# Access container shell
docker-compose exec calendar-core sh

# Restart a specific service
docker-compose restart calendar-core
```

### calendar-core (NestJS) Commands
**IMPORTANT**: All backend commands MUST be run inside the Docker container:

```bash
# From project root, run commands in container
docker-compose exec calendar-core npm run [command]

# Examples:
# Linting
docker-compose exec calendar-core npm run lint

# Format code
docker-compose exec calendar-core npm run format

# Run unit tests
docker-compose exec calendar-core npm run test

# Run tests in watch mode
docker-compose exec calendar-core npm run test:watch

# Run specific test file
docker-compose exec calendar-core npm run test -- calendar-event.entity.spec.ts

# Run tests with coverage
docker-compose exec calendar-core npm run test:cov

# Run e2e tests
docker-compose exec calendar-core npm run test:e2e
```

**Note**: DO NOT run `npm run start:dev` manually - the container is already running with hot-reload via docker-compose.

### calendar-finances (Go) Commands
**IMPORTANT**: All Go backend commands MUST be run inside the Docker container:

```bash
# From project root, run commands in container
docker-compose exec calendar-finances [command]

# Examples:
# Run tests
docker-compose exec calendar-finances go test ./...

# Build the application
docker-compose exec calendar-finances go build -o bin/server cmd/server/main.go

# View logs
docker-compose logs -f calendar-finances
```

### Database Access
```bash
# Connect to PostgreSQL (from host)
psql -h localhost -p 5433 -U calendar -d calendar_db

# Connect to PostgreSQL (from inside container)
docker-compose exec postgres psql -U calendar -d calendar_db

# Run Prisma commands (calendar-core)
docker-compose exec calendar-core npx prisma migrate dev
docker-compose exec calendar-core npx prisma generate
docker-compose exec calendar-core npx prisma studio
docker-compose exec calendar-core npm run prisma:seed
```

## Environment Configuration

Copy `.env.example` to `.env` and configure:
- Database credentials (defaults work with docker-compose)
- Google OAuth credentials (GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET)
- Linear API key
- Mercado Pago access token
- JWT secret

## Linear Integration

Application self-manages development via Linear API:
- Creates issues in "Calendar App" project
- Automatically updates status: pending → in_progress → done
- Adds implementation details as comments

## Database Schema (Prisma)

Located at: `services/calendar-core/prisma/schema.prisma`

**Core Tables:**
- `users` - User accounts with timezone support (default: America/Sao_Paulo)
- `calendars` - Professional/personal calendar separation
- `events` - Main events with recurrence support (RRule format)
- `category_types` - Modern categorization (health, work, leisure, finance, family, personal, education, social, other)
- `categories` - Legacy category structure
- `category_to_types` - M2M relationship between categories and types
- `event_completions` - Event execution tracking (separate from modifications)
- `recurrence_exceptions` - Removed dates from recurring events
- `recurrence_overrides` - Modified instances of recurring events
- `event_reminders` - Notification settings
- `monthly_goals` - Target execution counts per category/month
- `reports` - Monthly statistics and AI insights

**Key Patterns:**
- Recurrence hierarchy: Master events with derived instances using RRule
- Separate completion tracking from event modifications
- M2M support for flexible categorization
- Timezone-aware date handling

## API Endpoints (calendar-core)

**Events API** (`/events`):
```
GET    /events                      # List events (filters: calendarId, categoryId, search, startDate, endDate)
POST   /events                      # Create event
PUT    /events/:id                  # Update event
DELETE /events/:id                  # Delete single event
DELETE /events/:id/recurring        # Delete recurring (scope: this/future/all)
POST   /events/executions/toggle    # Toggle completion status
GET    /events/:id/executions       # Get execution history
GET    /events/stats                # Get stats (groupBy: day/week/month/category/categoryType/total)
```

**Other Domains:** Calendars, Categories, CategoryTypes have similar CRUD endpoints

## Financial Module Architecture

**Services:**
- **calendar-finances** (Go): REST API with clean architecture
  - `internal/application/usecases/` - Business logic
  - `internal/domain/` - Domain entities (profile, bankaccount)
  - `internal/infrastructure/http/` - HTTP handlers and routes
  - `internal/infrastructure/persistence/` - Repository implementations
  - `internal/database/` - PostgreSQL connection

- **finances-frontend** (Next.js): Financial dashboard on port 3003
  - Separate from calendar interface
  - Transaction analysis and categorization
  - Spending insights via AI

**Integrations:** Mercado Pago API, Nubank CSV/OFX imports

**Design Decision:** Only recurring bills appear on calendar view; all financial analysis happens in dedicated dashboard