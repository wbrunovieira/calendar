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
npm run lint       # Run linter
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

# Run unit tests (Vitest)
docker-compose exec calendar-core npm run test

# Run tests in watch mode
docker-compose exec calendar-core npm run test:watch

# Run tests with UI
docker-compose exec calendar-core npm run test:ui

# Run tests with coverage
docker-compose exec calendar-core npm run test:cov

# Run e2e tests
docker-compose exec calendar-core npm run test:e2e

# Run e2e tests in watch mode
docker-compose exec calendar-core npm run test:e2e:watch
```

**Note**: DO NOT run `npm run start:dev` manually - the container is already running with hot-reload via docker-compose.

### calendar-finances (Go) Commands
**IMPORTANT**: All Go backend commands MUST be run inside the Docker container:

```bash
# From project root, run commands in container
docker-compose exec calendar-finances [command]

# Examples:
# Run all tests
docker-compose exec calendar-finances go test ./...

# Run tests with verbose output
docker-compose exec calendar-finances go test -v ./...

# Run tests for specific package
docker-compose exec calendar-finances go test ./internal/application/usecases/...

# Run specific test file
docker-compose exec calendar-finances go test ./internal/application/usecases/transaction_usecases_test.go

# Run tests with coverage
docker-compose exec calendar-finances go test -cover ./...

# Build the application (note: cmd/api not cmd/server)
docker-compose exec calendar-finances go build -o bin/api cmd/api/main.go

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

## Database Schema

### Calendar Schema (Prisma)

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

### Finance Schema (Go/PostgreSQL)

Located in: `services/calendar-finances/internal/database/database.go`

**Schema:** `finance` (separate from public schema)

**Core Tables:**
- `finance.profiles` - Financial profiles linked to calendar users
- `finance.bank_accounts` - Bank account management
- `finance.categories` - Transaction categorization
- `finance.transactions` - Financial transactions with status tracking
- `finance.recurring_transactions` - Recurring bills/income (active/inactive)
- `finance.budget_targets` - Monthly budget limits per category

**Key Features:**
- Automatic migrations run on startup
- PostgreSQL `pgcrypto` extension enabled
- Connection pooling configured (25 max open, 5 idle)

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

## API Endpoints (calendar-finances)

All endpoints are prefixed with `/api/v1`:

**Profile Routes:**
```
GET    /api/v1/profiles           # List all profiles
POST   /api/v1/profiles           # Create profile
GET    /api/v1/profiles/:id       # Get specific profile
PUT    /api/v1/profiles/:id       # Update profile
DELETE /api/v1/profiles/:id       # Delete profile
```

**Bank Account Routes:**
```
GET    /api/v1/bank-accounts      # List bank accounts
POST   /api/v1/bank-accounts      # Create bank account
GET    /api/v1/bank-accounts/:id  # Get specific account
PUT    /api/v1/bank-accounts/:id  # Update account
DELETE /api/v1/bank-accounts/:id  # Delete account
```

**Transaction Routes:**
```
GET    /api/v1/transactions            # List transactions
POST   /api/v1/transactions            # Create transaction
GET    /api/v1/transactions/:id        # Get specific transaction
PUT    /api/v1/transactions/:id/status # Update transaction status
DELETE /api/v1/transactions/:id        # Delete transaction
```

**Recurring Transaction Routes:**
```
GET    /api/v1/recurring-transactions         # List recurring transactions
POST   /api/v1/recurring-transactions         # Create recurring transaction
PUT    /api/v1/recurring-transactions/:id     # Update recurring transaction
PATCH  /api/v1/recurring-transactions/:id/status  # Update status (active/inactive)
DELETE /api/v1/recurring-transactions/:id     # Delete recurring transaction
```

**Budget Routes:**
```
GET    /api/v1/budgets/summary    # Get budget summary with spending
GET    /api/v1/budgets            # List all budget targets
POST   /api/v1/budgets            # Create budget target
PUT    /api/v1/budgets/:id        # Update budget target
DELETE /api/v1/budgets/:id        # Delete budget target
```

**Category Routes:**
```
GET    /api/v1/categories         # List categories
POST   /api/v1/categories         # Create category
PUT    /api/v1/categories/:id     # Update category
DELETE /api/v1/categories/:id     # Delete category
```

## Financial Module Architecture

**Services:**
- **calendar-finances** (Go): REST API with clean architecture
  - `internal/application/usecases/` - Business logic (transaction, budget, recurring)
  - `internal/domain/` - Domain entities (transaction, profile, bankaccount, category, budgettarget, recurringtransaction)
  - `internal/infrastructure/http/` - HTTP handlers and routes
  - `internal/infrastructure/persistence/` - Repository implementations
  - `internal/database/` - PostgreSQL connection

- **finances-frontend** (Next.js): Financial dashboard on port 3003
  - Separate from calendar interface
  - Transaction analysis and categorization
  - Spending insights via AI

**Integrations:** Mercado Pago API, Nubank CSV/OFX imports

**Design Decision:** Only recurring bills appear on calendar view; all financial analysis happens in dedicated dashboard

## Testing

### Test Framework and Infrastructure

**calendar-core (NestJS):**
- **Framework:** Vitest (migrated from Jest for better performance)
- **Coverage Tool:** @vitest/coverage-v8
- **Mocking:** vitest-mock-extended
- **E2E:** Vitest with supertest
- **Test Database:** calendar_test_db (PostgreSQL)

**calendar-finances (Go):**
- **Framework:** Go native testing
- **Mocking:** Custom SQLMock (internal/test/sqlmock/)
- **Pattern:** Fake repositories for use case testing

### Test Database Setup

**Create test database:**
```bash
# From project root
bash scripts/setup-test-db.sh
```

**Reset test database:**
```bash
bash scripts/reset-test-db.sh
```

**Seed test database:**
```bash
bash scripts/seed-test-db.sh
```

**Test database connection:**
- URL: `postgresql://calendar:calendar123@localhost:5433/calendar_test_db`
- Environment: Set in `.env.test`
- Migrations: Auto-applied via Prisma

### Running Tests

**calendar-core (inside Docker):**
```bash
# Unit tests
docker-compose exec calendar-core npm run test

# Watch mode (interactive)
docker-compose exec calendar-core npm run test:watch

# Coverage report
docker-compose exec calendar-core npm run test:cov

# E2E tests
docker-compose exec calendar-core npm run test:e2e

# UI mode (browser interface)
docker-compose exec calendar-core npm run test:ui
```

**calendar-finances (inside Docker):**
```bash
# All tests
docker-compose exec calendar-finances go test ./...

# Verbose output
docker-compose exec calendar-finances go test -v ./...

# Specific package
docker-compose exec calendar-finances go test ./internal/domain/transaction/...

# With coverage
docker-compose exec calendar-finances go test -cover ./...

# Race detection
docker-compose exec calendar-finances go test -race ./...
```

### Test File Organization

**calendar-core:**
```
src/
├── domains/
│   └── [domain]/
│       ├── domain/
│       │   └── entities/
│       │       └── *.entity.spec.ts       # Entity unit tests
│       ├── application/
│       │   └── use-cases/
│       │       └── *.use-case.spec.ts     # Use case tests
│       └── infrastructure/
│           ├── controllers/
│           │   └── *.controller.spec.ts   # Controller tests
│           └── repositories/
│               └── *.repository.spec.ts   # Repository tests
└── test/
    ├── setup.ts                           # Global test setup
    └── helpers/
        ├── fixtures.ts                    # Test data fixtures
        ├── mock-builders.ts               # Mock utilities
        └── test-utils.ts                  # Test helpers

test/                                      # E2E tests directory
├── setup-e2e.ts                          # E2E setup
└── *.e2e-spec.ts                         # E2E test files
```

**calendar-finances:**
```
internal/
├── domain/
│   └── [entity]/
│       └── *_test.go                     # Entity unit tests
├── application/
│   └── usecases/
│       └── *_test.go                     # Use case tests
├── infrastructure/
│   └── persistence/
│       └── *_test.go                     # Repository tests
└── test/
    ├── sqlmock/                          # Custom SQL mock
    └── helpers/
        └── fixtures.go                   # Test fixtures
```

### Test Helpers and Fixtures

**calendar-core:**
```typescript
import {
  createEventFixture,
  createUserFixture,
  RRULE_DAILY,
  fixedTime
} from '@/test/helpers/fixtures';
import { createMockRepository } from '@/test/helpers/mock-builders';
import { useFakeTimers } from '@/test/helpers/test-utils';

// Use in tests
const event = createEventFixture({ title: 'My Event' });
const mockRepo = createMockRepository<EventRepository>();
useFakeTimers(new Date('2024-11-16'));
```

**calendar-finances:**
```go
import "github.com/brunovieira/calendar-finances/internal/test/helpers"

// Use in tests
profile := helpers.CreateTestProfile("John Doe")
account := helpers.CreateTestBankAccount(profile.ID)
tx := helpers.CreateExpenseTransaction(profile.ID, account.ID, category.ID, 100.00)
fixedTime := helpers.FixedTime()
```

### Test Patterns

**Unit Test Pattern (calendar-core):**
```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mock } from 'vitest-mock-extended';

describe('CreateEventUseCase', () => {
  let useCase: CreateEventUseCase;
  let mockRepository: MockProxy<EventRepository>;

  beforeEach(() => {
    mockRepository = mock<EventRepository>();
    useCase = new CreateEventUseCase(mockRepository);
  });

  it('should create event successfully', async () => {
    // Arrange
    const input = { title: 'Test', calendarId: 'uuid' };
    mockRepository.create.mockResolvedValue(eventEntity);

    // Act
    const result = await useCase.execute(input);

    // Assert
    expect(result).toBeDefined();
    expect(mockRepository.create).toHaveBeenCalledOnce();
  });
});
```

**Unit Test Pattern (calendar-finances):**
```go
func TestCreateProfileUseCase(t *testing.T) {
    // Arrange
    repo := &FakeProfileRepository{profiles: make(map[string]*profile.Profile)}
    useCase := NewCreateProfileUseCase(repo)
    input := CreateProfileInput{Name: "Test", Type: "PERSONAL"}

    // Act
    output, err := useCase.Execute(context.Background(), input)

    // Assert
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    if output.Name != "Test" {
        t.Errorf("Expected name 'Test', got %v", output.Name)
    }
}
```

**E2E Test Pattern (calendar-core):**
```typescript
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import request from 'supertest';
import { Test } from '@nestjs/testing';

describe('Events API (e2e)', () => {
  let app: INestApplication;

  beforeAll(async () => {
    const moduleFixture = await Test.createTestingModule({
      imports: [AppModule],
    }).compile();

    app = moduleFixture.createNestApplication();
    await app.init();
  });

  it('/events (POST)', () => {
    return request(app.getHttpServer())
      .post('/events')
      .send({ title: 'Test Event', calendarId: 'uuid' })
      .expect(201);
  });

  afterAll(async () => {
    await app.close();
  });
});
```

### Coverage Goals

- **Unit Tests:** 80%+ coverage
- **Integration Tests:** 60%+ coverage
- **E2E Tests:** Critical flows covered
- **Domain Layer:** 90%+ coverage (business logic)

### CI/CD Testing

Tests run automatically on:
- Push to `main` or `develop` branches
- Pull requests
- GitHub Actions workflow: `.github/workflows/test.yml`

**Coverage reports** uploaded to Codecov (requires CODECOV_TOKEN secret)

### Important Testing Notes

1. **Always use test database:** Never run tests against production/development database
2. **Deterministic tests:** Use `fixedTime()` and `useFakeTimers()` for date-dependent tests
3. **Clean state:** E2E tests clean database before each run
4. **Isolation:** Each test should be independent and not rely on others
5. **Mock external APIs:** Google Calendar, Linear, Mercado Pago should be mocked in tests
6. **Timezone aware:** Tests use `America/Sao_Paulo` timezone by default

### Vitest Configuration

- `vitest.config.ts` - Unit test configuration with path aliases
- `vitest.config.e2e.ts` - E2E test configuration with setup file