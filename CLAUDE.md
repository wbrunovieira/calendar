# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a multi-container calendar application that integrates Google Calendar accounts (professional and personal), Linear for task management, and financial tracking with AI-powered agents.

## Architecture

### Container Structure
The project uses Docker containers with the following services:
- **calendar-core**: Main API (NestJS/Go/Rust - TBD), handles authentication, Google Calendar sync, Linear API integration
- **calendar-frontend**: Next.js web interface with separate pages for calendar and financial dashboard
- **calendar-ai**: Python container for AI services using Llama, PyTorch, Langchain, and CrewAI
- **calendar-worker**: Go/Rust service for heavy processing, batch jobs, and ML data preparation
- **postgres**: Primary database
- **redis**: Cache and message queue

### Key Integrations
- Google Calendar API (OAuth2 for two accounts: bruno@wbdigitalsolutions.com and wbrunovieira77@gmail.com)
- Linear API for project task tracking and auto-management
- Mercado Pago API for financial transactions
- Nubank data import (CSV/OFX or email parsing)

## Development Commands

Since the project is in initial setup phase, the following commands will be available once containers are configured:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f [service-name]

# Rebuild specific service
docker-compose build [service-name]

# Run database migrations (once implemented)
docker-compose exec calendar-core npm run migration:run

# Access container shell
docker-compose exec [service-name] sh
```

## Development Phases

Currently in planning phase. Next steps:
1. Docker Compose setup with all containers
2. NestJS backend with Google Calendar OAuth2
3. Next.js frontend with calendar views
4. PostgreSQL schema and Redis configuration
5. Worker service implementation
6. AI service integration
7. Financial module

## Linear Integration

The application self-manages its development by creating Linear issues automatically. When implementing features:
- Issues are created via Linear API in the "Calendar App" project
- Status updates happen automatically (pending → in_progress → done)
- Implementation details are added as comments

## Financial Module

The financial dashboard is a separate page from the calendar. Only recurring bills appear on the calendar view. The dashboard handles:
- Transaction analysis and categorization
- Spending insights via AI
- Integration with Mercado Pago API
- Nubank data imports

## Worker Jobs

The calendar-worker handles:
- Mass synchronization (Google Calendar history, CSV/ICS imports)
- Pattern analysis and reporting
- ML data preparation (embeddings, clustering)
- Recurring operations (backups, nightly sync)
- Heavy integrations (bulk Linear exports, OCR)
- Batch notifications (weekly summaries, daily digests)