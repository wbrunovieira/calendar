package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Connect establishes a connection to the PostgreSQL database
func Connect(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return db, nil
}

// RunMigrations creates the finance schema and initial tables
func RunMigrations(db *sql.DB) error {
	log.Println("Running database migrations...")

	migrations := []string{
		// Create finance schema
		`CREATE SCHEMA IF NOT EXISTS finance`,

		// Create profiles table
		`CREATE TABLE IF NOT EXISTS finance.profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			calendar_id VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL CHECK (type IN ('PERSONAL', 'BUSINESS')),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create accounts table
		`CREATE TABLE IF NOT EXISTS finance.accounts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL CHECK (type IN ('personal', 'business')),
			bank_name VARCHAR(255),
			account_number VARCHAR(100),
			balance DECIMAL(15, 2) NOT NULL DEFAULT 0,
			currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create transactions table
		`CREATE TABLE IF NOT EXISTS finance.transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id UUID NOT NULL REFERENCES finance.accounts(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL CHECK (type IN ('income', 'expense', 'transfer')),
			category VARCHAR(255),
			amount DECIMAL(15, 2) NOT NULL,
			description TEXT,
			transaction_date DATE NOT NULL,
			is_recurring BOOLEAN NOT NULL DEFAULT false,
			recurrence_rule TEXT,
			tags TEXT[],
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create investments table
		`CREATE TABLE IF NOT EXISTS finance.investments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id UUID NOT NULL REFERENCES finance.accounts(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL CHECK (type IN ('stock', 'fii', 'treasury', 'cdb', 'lci', 'lca')),
			ticker VARCHAR(20) NOT NULL,
			quantity DECIMAL(15, 6) NOT NULL,
			average_price DECIMAL(15, 2) NOT NULL,
			current_price DECIMAL(15, 2),
			purchase_date DATE NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_profiles_calendar_id ON finance.profiles(calendar_id)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON finance.accounts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON finance.transactions(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_date ON finance.transactions(transaction_date)`,
		`CREATE INDEX IF NOT EXISTS idx_investments_account_id ON finance.investments(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_investments_ticker ON finance.investments(ticker)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("✓ Migrations completed successfully")
	return nil
}
