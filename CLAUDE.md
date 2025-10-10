# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Multi-container calendar application integrating Google Calendar accounts (professional and personal), Linear task management, and financial tracking with AI-powered agents.

## Architecture

### Container Structure
- **calendar-core** (NestJS): Main API, authentication, Google Calendar sync, Linear API integration - Runs in Docker on port 3334
- **calendar-frontend** (Next.js 15.5): Web interface with separate calendar and financial dashboard pages - Runs locally on port 3000
- **calendar-ai** (Python - planned): AI services using Llama, PyTorch, Langchain, CrewAI
- **calendar-worker** (Go/Rust - planned): Heavy processing, batch jobs, ML data preparation
- **postgres**: Primary database (PostgreSQL 15) - Port 5433

### Architecture Decision: No Redis
This is a personal project for a single user, so Redis was removed to reduce costs and complexity:
- **Cache**: Using NestJS in-memory cache or PostgreSQL native caching
- **Job Queues**: Using PostgreSQL-based queues (pg-boss or BullMQ with PostgreSQL adapter)
- **Sessions**: Using stateless JWT tokens or PostgreSQL session storage

### Domain-Driven Design Structure
The calendar-core service follows DDD architecture:
- `src/domains/[domain]/domain/entities/`: Domain entities (e.g., CalendarEvent)
- `src/domains/[domain]/application/`: Use cases and application logic (planned)
- `src/domains/[domain]/infrastructure/`: External integrations and repositories (planned)

Example domain: `src/domains/google-calendar/domain/entities/calendar-event.entity.ts`

### Key Integrations
- Google Calendar API (OAuth2 for bruno@wbdigitalsolutions.com and wbrunovieira77@gmail.com)
- Linear API for project task tracking and auto-management
- Mercado Pago API for financial transactions
- Nubank data import (CSV/OFX or email parsing)

### Frontend Development (Local)

The frontend runs locally (not in Docker) for better performance:

```bash
# Navigate to frontend directory
cd services/calendar-frontend

# Install dependencies
npm install

# Run development server (http://localhost:3000)
npm run dev

# Build for production
npm run build

# Run linter
npm run lint
```

**Note**: Frontend requires backend running on port 3334. Start backend with `docker-compose up -d` first.

## Development Commands

### Docker Operations (Backend Only)
```bash
# Start all services
docker-compose up -d

# Start with rebuild
docker-compose up -d --build

# View logs for specific service
docker-compose logs -f calendar-core

# Stop all services
docker-compose down

# Access container shell
docker-compose exec calendar-core sh
```

### calendar-core (NestJS) Commands
```bash
# From project root, run commands in container
docker-compose exec calendar-core npm run [command]

# Or navigate to service directory
cd services/calendar-core

# Build
npm run build

# Run in development mode (with hot-reload)
npm run start:dev

# Run in debug mode
npm run start:debug

# Linting
npm run lint

# Format code
npm run format

# Run unit tests
npm run test

# Run tests in watch mode
npm run test:watch

# Run specific test file
npm run test -- calendar-event.entity.spec.ts

# Run tests with coverage
npm run test:cov

# Run e2e tests
npm run test:e2e
```

### Database Access
```bash
# Connect to PostgreSQL (from host)
psql -h localhost -p 5433 -U calendar -d calendar_db

# Connect to PostgreSQL (from inside container)
docker-compose exec postgres psql -U calendar -d calendar_db
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

## Financial Module Architecture

Financial dashboard is a separate page from calendar:
- Only recurring bills appear on calendar view
- Dashboard handles: transaction analysis, categorization, spending insights via AI
- Integrations: Mercado Pago API, Nubank CSV/OFX imports

## Worker Service Responsibilities

The calendar-worker (planned) will handle:
- Mass synchronization (Google Calendar history, CSV/ICS imports)
- Pattern analysis and reporting
- ML data preparation (embeddings, clustering)
- Recurring operations (backups, nightly sync, cleanup)
- Heavy integrations (bulk Linear exports, OCR, attachments)
- Batch notifications (weekly summaries, daily digests, alerts)