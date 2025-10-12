# Calendar Finances Service

Financial management service built with Go for the Calendar application.

## Overview

This service handles all financial operations including:
- Account management (personal and business)
- Transaction tracking (income, expenses, transfers)
- Investment portfolio management
- Financial reports and analytics
- Bank integrations (Nubank, Mercado Pago)

## Tech Stack

- **Language**: Go 1.23
- **HTTP Router**: Gorilla Mux
- **Database**: PostgreSQL 15 (shared with calendar-core)
- **Container**: Docker
- **Port**: 3335

## Project Structure

```
calendar-finances/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── database/
│   │   └── database.go          # DB connection and migrations
│   ├── domain/                  # Domain entities (future)
│   ├── handlers/
│   │   └── handlers.go          # HTTP handlers
│   └── repository/              # Data access layer (future)
├── pkg/                         # Shared packages (future)
├── Dockerfile
├── go.mod
└── go.sum
```

## Database Schema

The service uses the `finance` schema in PostgreSQL:

### Tables

**finance.accounts**
- Personal and business bank accounts
- Tracks balance and account details

**finance.transactions**
- All financial transactions (income/expense/transfer)
- Supports recurring transactions
- Category and tag support

**finance.investments**
- Investment portfolio tracking
- Supports: stocks, FIIs, treasury bonds, CDB, LCI, LCA
- Tracks quantity, prices, and performance

## API Endpoints

### Health & Info
- `GET /health` - Service health check
- `GET /` - API information

### Accounts (planned)
- `GET /api/v1/accounts` - List all accounts
- `POST /api/v1/accounts` - Create new account
- `GET /api/v1/accounts/:id` - Get account details
- `PUT /api/v1/accounts/:id` - Update account
- `DELETE /api/v1/accounts/:id` - Delete account

### Transactions (planned)
- `GET /api/v1/transactions` - List transactions
- `POST /api/v1/transactions` - Create transaction
- `GET /api/v1/transactions/:id` - Get transaction
- `PUT /api/v1/transactions/:id` - Update transaction
- `DELETE /api/v1/transactions/:id` - Delete transaction

### Investments (planned)
- `GET /api/v1/investments` - List investments
- `POST /api/v1/investments` - Add investment
- `GET /api/v1/investments/:id` - Get investment details

## Development

### Run with Docker Compose
```bash
# From project root
docker-compose up -d calendar-finances

# View logs
docker-compose logs -f calendar-finances

# Rebuild
docker-compose up -d --build calendar-finances
```

### Run locally (for development)
```bash
cd services/calendar-finances

# Download dependencies
go mod download

# Run the application
go run cmd/api/main.go
```

### Environment Variables

Create `.env` file (see `.env.example`):
```
PORT=3335
DATABASE_URL=postgres://calendar:calendar123@postgres:5432/calendar_db?sslmode=disable
```

## Testing

```bash
# Health check
curl http://localhost:3335/health

# API info
curl http://localhost:3335/
```

## Future Features

- [ ] Complete CRUD for accounts
- [ ] Complete CRUD for transactions
- [ ] Investment tracking
- [ ] Bank integrations (Nubank, Mercado Pago)
- [ ] Financial reports and dashboards
- [ ] Cash flow predictions
- [ ] Budget management
- [ ] Recurring bill management
- [ ] Export/Import (CSV, OFX)
- [ ] Multi-currency support

## Integration with Calendar

- Recurring bills from finances appear in the calendar view
- Frontend displays financial dashboard at `/finances`
- Uses same PostgreSQL database but separate schema
- Independent scaling and deployment
