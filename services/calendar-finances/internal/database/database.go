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

		// Migration: Add linked_account_id to bank_accounts (for linking credit cards to parent accounts)
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'linked_account_id'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN linked_account_id UUID REFERENCES finance.bank_accounts(id) ON DELETE SET NULL;
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_bank_accounts_linked ON finance.bank_accounts(linked_account_id)`,

		// Create credit_card_invoices table
		`CREATE TABLE IF NOT EXISTS finance.credit_card_invoices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			bank_account_id UUID NOT NULL REFERENCES finance.bank_accounts(id) ON DELETE CASCADE,
			reference_date DATE NOT NULL,
			opening_date DATE NOT NULL,
			closing_date DATE NOT NULL,
			due_date DATE NOT NULL,
			amount NUMERIC(15, 2) NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED', 'PAID')),
			paid_at TIMESTAMP,
			paid_amount NUMERIC(15, 2),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_invoice_account_reference UNIQUE (bank_account_id, reference_date)
		)`,

		// Indexes for credit_card_invoices
		`CREATE INDEX IF NOT EXISTS idx_invoices_bank_account ON finance.credit_card_invoices(bank_account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_status ON finance.credit_card_invoices(status)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_closing_date ON finance.credit_card_invoices(closing_date)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_due_date ON finance.credit_card_invoices(due_date)`,

		// Migration: Add invoice_id to transactions (for linking credit card transactions to invoices)
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'transactions'
				AND column_name = 'invoice_id'
			) THEN
				ALTER TABLE finance.transactions
				ADD COLUMN invoice_id UUID REFERENCES finance.credit_card_invoices(id) ON DELETE SET NULL;
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_invoice ON finance.transactions(invoice_id)`,

		// Migration: Add investment-specific columns to bank_accounts
		`DO $$
		BEGIN
			-- Add investment_type column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'investment_type'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN investment_type VARCHAR(50) CHECK (investment_type IN ('SAVINGS_BOX', 'CDB', 'LCI', 'LCA', 'STOCKS', 'FUNDS', 'CRYPTO', 'TREASURY', 'OTHER'));
			END IF;

			-- Add yield_type column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'yield_type'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN yield_type VARCHAR(50) CHECK (yield_type IN ('FIXED', 'CDI_PERCENTAGE', 'IPCA_PLUS', 'VARIABLE'));
			END IF;

			-- Add yield_rate column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'yield_rate'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN yield_rate NUMERIC(8, 4);
			END IF;

			-- Add maturity_date column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'maturity_date'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN maturity_date DATE;
			END IF;

			-- Add broker column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'broker'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN broker VARCHAR(100);
			END IF;
		END $$`,

		// Index for investment accounts
		`CREATE INDEX IF NOT EXISTS idx_bank_accounts_investment_type ON finance.bank_accounts(investment_type) WHERE type = 'INVESTMENT'`,

		// Migration: Add FII to investment_type constraint and add quota columns
		`DO $$
		BEGIN
			-- Update investment_type constraint to include FII
			IF EXISTS (
				SELECT 1 FROM information_schema.check_constraints
				WHERE constraint_schema = 'finance'
				AND constraint_name LIKE '%investment_type%'
			) THEN
				ALTER TABLE finance.bank_accounts DROP CONSTRAINT IF EXISTS bank_accounts_investment_type_check;
				ALTER TABLE finance.bank_accounts
				ADD CONSTRAINT bank_accounts_investment_type_check
				CHECK (investment_type IN ('SAVINGS_BOX', 'CDB', 'LCI', 'LCA', 'STOCKS', 'FUNDS', 'FII', 'CRYPTO', 'TREASURY', 'OTHER'));
			END IF;

			-- Add number_of_quotas column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'number_of_quotas'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN number_of_quotas NUMERIC(15, 6);
			END IF;

			-- Add quota_price column
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'quota_price'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN quota_price NUMERIC(15, 6);
			END IF;
		END $$`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("✓ Migrations completed successfully")
	return nil
}
