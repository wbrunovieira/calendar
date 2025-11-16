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
		// Ensure required extension
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,

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

		// Create bank_accounts table
		`CREATE TABLE IF NOT EXISTS finance.bank_accounts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL CHECK (type IN ('CHECKING', 'SAVINGS', 'INVESTMENT', 'CREDIT_CARD', 'CASH', 'OTHER')),
			initial_balance NUMERIC(15, 2) NOT NULL DEFAULT 0,
			current_balance NUMERIC(15, 2) NOT NULL DEFAULT 0,
			currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
			is_active BOOLEAN NOT NULL DEFAULT true,
			bank_name VARCHAR(255),
			bank_code VARCHAR(10),
			agency VARCHAR(20),
			account_number VARCHAR(50),
			account_digit VARCHAR(5),
			color VARCHAR(7),
			icon VARCHAR(50),
			description TEXT,
			credit_limit NUMERIC(15, 2),
			due_day INTEGER CHECK (due_day >= 1 AND due_day <= 31),
			closing_day INTEGER CHECK (closing_day >= 1 AND closing_day <= 31),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Reset finance tables introduced across recent phases (dev convenience)
		`DROP TABLE IF EXISTS finance.transaction_tags`,
		`DROP TABLE IF EXISTS finance.transaction_splits`,
		`DROP TABLE IF EXISTS finance.transactions`,
		`DROP TABLE IF EXISTS finance.recurring_transactions`,
		`DROP TABLE IF EXISTS finance.budget_targets`,
		`DROP TABLE IF EXISTS finance.categories`,

		// Create categories table
		`CREATE TABLE IF NOT EXISTS finance.categories (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			name VARCHAR(120) NOT NULL,
			type VARCHAR(20) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE', 'TRANSFER')),
			color VARCHAR(7),
			icon VARCHAR(50),
			parent_id UUID REFERENCES finance.categories(id) ON DELETE SET NULL,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_categories_profile_name UNIQUE (profile_id, name)
		)`,

		// Create transactions table
		`CREATE TABLE IF NOT EXISTS finance.transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			bank_account_id UUID NOT NULL REFERENCES finance.bank_accounts(id) ON DELETE CASCADE,
			destination_account_id UUID REFERENCES finance.bank_accounts(id) ON DELETE SET NULL,
			category_id UUID REFERENCES finance.categories(id) ON DELETE SET NULL,
			type VARCHAR(20) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE', 'TRANSFER')),
			status VARCHAR(20) NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED', 'CONFIRMED', 'CANCELLED')),
			amount NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
			currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
			description TEXT,
			notes TEXT,
			cost_center VARCHAR(120),
			occurred_on DATE NOT NULL,
			due_on DATE,
			recurrence_rule TEXT,
			installment_number INTEGER,
			installment_total INTEGER,
			external_id VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create transaction_splits table
		`CREATE TABLE IF NOT EXISTS finance.transaction_splits (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			transaction_id UUID NOT NULL REFERENCES finance.transactions(id) ON DELETE CASCADE,
			category_id UUID REFERENCES finance.categories(id) ON DELETE SET NULL,
			amount NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
			memo TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create transaction_tags table
		`CREATE TABLE IF NOT EXISTS finance.transaction_tags (
			transaction_id UUID NOT NULL REFERENCES finance.transactions(id) ON DELETE CASCADE,
			tag VARCHAR(50) NOT NULL,
			PRIMARY KEY (transaction_id, tag)
		)`,

		// Create recurring transactions table
		`CREATE TABLE IF NOT EXISTS finance.recurring_transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			bank_account_id UUID REFERENCES finance.bank_accounts(id) ON DELETE SET NULL,
			category_id UUID REFERENCES finance.categories(id) ON DELETE SET NULL,
			type VARCHAR(20) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE', 'TRANSFER')),
			amount NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
			currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
			description TEXT,
			recurrence_rule TEXT NOT NULL,
			start_on DATE NOT NULL,
			end_on DATE,
			next_occurrence DATE NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAUSED', 'CANCELLED')),
			notes TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		// Create budget targets table
		`CREATE TABLE IF NOT EXISTS finance.budget_targets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			category_id UUID NOT NULL REFERENCES finance.categories(id) ON DELETE CASCADE,
			period_start DATE NOT NULL,
			amount NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
			notes TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_budget_targets UNIQUE (profile_id, category_id, period_start)
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_profiles_calendar_id ON finance.profiles(calendar_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bank_accounts_profile_id ON finance.bank_accounts(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_profile_id ON finance.categories(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON finance.categories(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_profile_occurred ON finance.transactions(profile_id, occurred_on)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_bank_account ON finance.transactions(bank_account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transaction_splits_tx ON finance.transaction_splits(transaction_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transaction_tags_tx ON finance.transaction_tags(transaction_id)`,
		`CREATE INDEX IF NOT EXISTS idx_recurring_transactions_profile ON finance.recurring_transactions(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_recurring_transactions_next ON finance.recurring_transactions(next_occurrence)`,
		`CREATE INDEX IF NOT EXISTS idx_budget_targets_profile_period ON finance.budget_targets(profile_id, period_start)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("✓ Migrations completed successfully")
	return nil
}
