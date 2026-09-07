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

		// The incoming half of a transfer is found by destination_account_id, so
		// every balance calculation scans on it. It was the only id column on
		// this table without an index, which makes that half a sequential scan
		// once per account — cheap today, and the Binance sync has already put
		// six figures of rows in here once.
		`CREATE INDEX IF NOT EXISTS idx_transactions_destination_account ON finance.transactions(destination_account_id) WHERE destination_account_id IS NOT NULL`,
		// Guard-rail contra duplicatas de syncs (binance-*, dividend-*, mp-*): mesmo que o
		// dedup na aplicação quebre, o INSERT duplicado falha no banco.
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_transactions_external_id ON finance.transactions(external_id) WHERE external_id IS NOT NULL`,
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

		// Migration: Add display_order to bank_accounts (for custom ordering)
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'bank_accounts'
				AND column_name = 'display_order'
			) THEN
				ALTER TABLE finance.bank_accounts
				ADD COLUMN display_order INTEGER DEFAULT 0;
			END IF;
		END $$`,

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

		// Migration: Add review_on to recurring_transactions for pause review reminders
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'recurring_transactions'
				AND column_name = 'review_on'
			) THEN
				ALTER TABLE finance.recurring_transactions
				ADD COLUMN review_on DATE;
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_recurring_transactions_review ON finance.recurring_transactions(review_on) WHERE review_on IS NOT NULL`,

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

		// Migration: Add is_recurring and effective_until to budget_targets for recurring budgets with versioning
		`DO $$
		BEGIN
			-- Add is_recurring column (default FALSE keeps existing budgets as one-time)
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'budget_targets'
				AND column_name = 'is_recurring'
			) THEN
				ALTER TABLE finance.budget_targets
				ADD COLUMN is_recurring BOOLEAN NOT NULL DEFAULT FALSE;
			END IF;

			-- Add effective_until column for versioning (NULL means still active)
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'budget_targets'
				AND column_name = 'effective_until'
			) THEN
				ALTER TABLE finance.budget_targets
				ADD COLUMN effective_until DATE;
			END IF;
		END $$`,

		// Migration: Add reminder_on to transactions for advance reminder alerts
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'transactions'
				AND column_name = 'reminder_on'
			) THEN
				ALTER TABLE finance.transactions
				ADD COLUMN reminder_on DATE;
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_reminder ON finance.transactions(reminder_on) WHERE reminder_on IS NOT NULL`,

		// Migration: Add linked_transaction_id to transactions (for cross-profile paired transfers)
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'finance'
				AND table_name = 'transactions'
				AND column_name = 'linked_transaction_id'
			) THEN
				ALTER TABLE finance.transactions
				ADD COLUMN linked_transaction_id UUID;
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_linked ON finance.transactions(linked_transaction_id) WHERE linked_transaction_id IS NOT NULL`,

		// Goals table
		`CREATE TABLE IF NOT EXISTS finance.goals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			category_id UUID REFERENCES finance.categories(id) ON DELETE SET NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			target_amount NUMERIC(15, 2) NOT NULL CHECK (target_amount > 0),
			current_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
			priority VARCHAR(10) NOT NULL DEFAULT 'MEDIUM' CHECK (priority IN ('HIGH', 'MEDIUM', 'LOW')),
			target_date DATE,
			status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'CANCELLED')),
			link TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goals_profile ON finance.goals(profile_id)`,
		// Migration: Add EXCHANGE and WALLET account types
		`DO $$
		BEGIN
			ALTER TABLE finance.bank_accounts DROP CONSTRAINT IF EXISTS bank_accounts_type_check;
			ALTER TABLE finance.bank_accounts ADD CONSTRAINT bank_accounts_type_check
				CHECK (type IN ('CHECKING', 'SAVINGS', 'INVESTMENT', 'CREDIT_CARD', 'CASH', 'EXCHANGE', 'WALLET', 'OTHER'));
		END $$`,
		// Without an external id on the account, the importer has no way to know which
		// of our accounts a statement line belongs to, and the only fallback is
		// matching by name — the fragile matching this whole issue exists to remove.
		`ALTER TABLE finance.bank_accounts ADD COLUMN IF NOT EXISTS provider_account_id TEXT`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_bank_accounts_provider_account
			ON finance.bank_accounts(provider_account_id) WHERE provider_account_id IS NOT NULL`,

		// Every import window, so a reconciliation report can carry a true header.
		// Without it, "left over on the bank side" has no defined boundary: a line
		// genuinely without a counterpart cannot be told apart from a period never
		// fetched — the same incomplete view presented as complete that produced a
		// phantom R$ 10.651,18 discrepancy.
		`CREATE TABLE IF NOT EXISTS finance.statement_imports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id UUID NOT NULL REFERENCES finance.bank_accounts(id),
			provider VARCHAR(20) NOT NULL,
			period_from DATE NOT NULL,
			period_to DATE NOT NULL,
			lines_seen INT NOT NULL DEFAULT 0,
			lines_inserted INT NOT NULL DEFAULT 0,
			ran_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_statement_imports_account ON finance.statement_imports(account_id, period_to DESC)`,

		// Which invoice a transaction PAYS, distinct from invoice_id, which is the
		// invoice a card purchase BELONGS TO. Without it the only way to find a bill's
		// payments is matching amount and date.
		`ALTER TABLE finance.transactions ADD COLUMN IF NOT EXISTS paid_invoice_id UUID REFERENCES finance.credit_card_invoices(id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_paid_invoice ON finance.transactions(paid_invoice_id) WHERE paid_invoice_id IS NOT NULL`,

		// Every correction of a stored balance, so a recalculation leaves a mark
		// instead of erasing one.
		`CREATE TABLE IF NOT EXISTS finance.balance_adjustments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id UUID NOT NULL REFERENCES finance.bank_accounts(id),
			balance_before NUMERIC(15,2) NOT NULL,
			balance_after NUMERIC(15,2) NOT NULL,
			delta NUMERIC(15,2) NOT NULL,
			reason TEXT NOT NULL,
			adjusted_by TEXT NOT NULL,
			adjusted_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_balance_adjustments_account ON finance.balance_adjustments(account_id, adjusted_at DESC)`,

		// One row per idempotency key, so a retried write performs its effect once.
		// Four clients write to this API and n8n retries are simultaneous by nature,
		// so low volume is no protection.
		`CREATE TABLE IF NOT EXISTS finance.idempotency_keys (
			key TEXT PRIMARY KEY,
			endpoint TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			response_status INT,
			-- TEXT, not JSONB: a replay must be told exactly what the first attempt
			-- answered, and JSONB normalises whitespace and key order on the way back.
			response_body TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_idempotency_created ON finance.idempotency_keys(created_at)`,
		// A database created before this change has response_body as JSONB, and
		// CREATE TABLE IF NOT EXISTS does not alter a column. JSONB normalises
		// whitespace and key order, so a replay would be told something subtly
		// different from what the first attempt answered.
		`ALTER TABLE finance.idempotency_keys ALTER COLUMN response_body TYPE TEXT`,

		// A natural key for invoice payments is still MISSING, deliberately.
		//
		// Idempotency-Key protects a client against its own retry; it does not stop two
		// DIFFERENT clients doing the same thing — the web form and the WhatsApp agent
		// both paying the same bill. The reviewer's suggested key is
		// (invoice_id, date, amount), but the payment leg does not carry invoice_id
		// today, and the obvious substitute — (destination_account, date, amount) —
		// would refuse two genuinely identical transfers on one day, which is a real
		// thing to do.
		//
		// Current production data would not violate that broader index, and that is
		// exactly why it would be tempting: it passes today and blocks a legitimate
		// operation later. The prerequisite is putting invoice_id on the payment leg.

		// Reconciliation matches, N:N and append-only.
		//
		// A single matched_transaction_id on the line is a 1:1 model that is already
		// known to be wrong: an invoice payment covers many purchases, a Pix settles
		// two bills, a split spreads one line across entries. Building the matcher on
		// the column would mean migrating halfway through.
		//
		// Never deleted, only undone with a reason — a reconciliation that destroys
		// its own history cannot say why a line went back to pending.
		`CREATE TABLE IF NOT EXISTS finance.reconciliation_matches (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			line_id UUID NOT NULL REFERENCES finance.bank_statement_lines(id) ON DELETE CASCADE,
			transaction_id UUID NOT NULL REFERENCES finance.transactions(id),
			amount_minor BIGINT NOT NULL CHECK (amount_minor <> 0),
			method VARCHAR(20) NOT NULL
				CHECK (method IN ('EXTERNAL_ID','END_TO_END','DETERMINISTIC','FUZZY','MANUAL')),
			score NUMERIC(5,4),
			matched_by TEXT NOT NULL,
			matched_at TIMESTAMP NOT NULL DEFAULT NOW(),
			unmatched_at TIMESTAMP,
			unmatched_reason TEXT,
			CONSTRAINT match_undo_has_reason
				CHECK (unmatched_at IS NULL OR unmatched_reason IS NOT NULL)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_matches_line ON finance.reconciliation_matches(line_id) WHERE unmatched_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_matches_transaction ON finance.reconciliation_matches(transaction_id) WHERE unmatched_at IS NULL`,
		// The same pair may be matched again after being undone, but not twice at once.
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_matches_live_pair
			ON finance.reconciliation_matches(line_id, transaction_id) WHERE unmatched_at IS NULL`,

		// Migration: bank statement lines, stored verbatim.
		//
		// Reconciliation must be persisted data, not chat work. Everything the
		// reconciliation of 06-07/09/2026 established — what had been checked, against
		// which criterion, what was left over — lived only in a transcript and went
		// away with it.
		//
		// Amounts are integers in minor units. Money in float64 was already forcing
		// half-a-centavo tolerances in this codebase, and a tolerance on value is how
		// a real difference becomes "acceptable rounding" and vanishes.
		`CREATE TABLE IF NOT EXISTS finance.bank_statement_lines (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id UUID NOT NULL REFERENCES finance.bank_accounts(id),
			provider VARCHAR(20) NOT NULL,
			external_id TEXT NOT NULL,
			booked_date DATE NOT NULL,
			value_date DATE,
			amount_minor BIGINT NOT NULL,
			currency CHAR(3) NOT NULL,
			amount_account_minor BIGINT,
			-- No fx_rate column: the provider sends only the two amounts, so a stored
			-- rate would be a derived value written down — the pattern being removed
			-- from the rest of this system. It is computed on read.
			description TEXT NOT NULL DEFAULT '',
			end_to_end_id TEXT,
			provider_status VARCHAR(10) NOT NULL DEFAULT 'POSTED'
				CHECK (provider_status IN ('PENDING','POSTED')),
			bill_id TEXT,
			raw JSONB NOT NULL,
			status VARCHAR(12) NOT NULL DEFAULT 'UNMATCHED'
				CHECK (status IN ('UNMATCHED','MATCHED','IGNORED')),
			ignored_reason TEXT,
			imported_at TIMESTAMP NOT NULL DEFAULT NOW(),
			-- last_seen_at is bumped on every import that covered this line's window.
			-- Keeping the raw payload does not reveal that the bank STOPPED reporting
			-- a row; the row simply stays and the disappearance is invisible.
			last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			-- Idempotent import: the same window may be pulled any number of times.
			-- account_id belongs in the key: Pluggy ids are globally unique, but an
			-- OFX FITID is unique only WITHIN an account by specification. Without it,
			-- two statements could legitimately collide and the upsert would rewrite
			-- one account's movement while it still claimed to belong to the other.
			CONSTRAINT uq_statement_account_provider_external UNIQUE (account_id, provider, external_id),
			-- An ignored line without a motive is indistinguishable from one nobody
			-- looked at.
			CONSTRAINT statement_ignored_has_reason
				CHECK (status <> 'IGNORED' OR ignored_reason IS NOT NULL)
		)`,
		// Every version the provider ever reported, append-only. The main row is a
		// projection of the latest; this is what makes "verbatim" true.
		//
		// Pluggy rewrites transactions, and overwriting raw on conflict destroys
		// exactly the evidence that a change came from the bank and not from us. It
		// is also what lets a match be dropped WITH a reason: without the previous
		// version, a line that "went back to diverging" has no discoverable cause.
		`CREATE TABLE IF NOT EXISTS finance.bank_statement_line_revisions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			line_id UUID NOT NULL REFERENCES finance.bank_statement_lines(id) ON DELETE CASCADE,
			seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
			booked_date DATE NOT NULL,
			amount_minor BIGINT NOT NULL,
			currency CHAR(3) NOT NULL,
			amount_account_minor BIGINT,
			description TEXT NOT NULL DEFAULT '',
			raw JSONB NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_statement_revisions_line ON finance.bank_statement_line_revisions(line_id, seen_at DESC)`,
		// Incremental migration for databases that already have the first version of
		// the table. CREATE TABLE IF NOT EXISTS is a no-op there and would leave the
		// new columns missing — which is exactly how a schema change passes locally
		// and fails on a database that has been around.
		`ALTER TABLE finance.bank_statement_lines ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP NOT NULL DEFAULT NOW()`,
		`ALTER TABLE finance.bank_statement_lines ADD COLUMN IF NOT EXISTS provider_status VARCHAR(10) NOT NULL DEFAULT 'POSTED'`,
		`ALTER TABLE finance.bank_statement_lines ADD COLUMN IF NOT EXISTS bill_id TEXT`,
		`DO $$
		BEGIN
			-- account_id belongs in the uniqueness key: an OFX FITID is unique only
			-- WITHIN an account, so two statements could legitimately collide and the
			-- upsert would rewrite one account's movement under the other's id.
			ALTER TABLE finance.bank_statement_lines DROP CONSTRAINT IF EXISTS uq_statement_provider_external;
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_statement_account_provider_external') THEN
				ALTER TABLE finance.bank_statement_lines
					ADD CONSTRAINT uq_statement_account_provider_external UNIQUE (account_id, provider, external_id);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'statement_provider_status_check') THEN
				-- The CREATE TABLE path declares this inline; without it here, a new
				-- database has the constraint and a migrated one does not. Divergent
				-- schemas are the same lesson as CREATE TABLE IF NOT EXISTS, now in
				-- the very commit that documents it.
				ALTER TABLE finance.bank_statement_lines
					ADD CONSTRAINT statement_provider_status_check
					CHECK (provider_status IN ('PENDING','POSTED'));
			END IF;
			-- matched_transaction_id and its CHECK were dropped when
			-- reconciliation_matches arrived: a transaction id on the line as well
			-- would be a second source for the same fact, and two sources diverge.
			ALTER TABLE finance.bank_statement_lines DROP CONSTRAINT IF EXISTS statement_matched_has_reference;
			ALTER TABLE finance.bank_statement_lines DROP COLUMN IF EXISTS matched_transaction_id;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_statement_account_date ON finance.bank_statement_lines(account_id, booked_date)`,
		`CREATE INDEX IF NOT EXISTS idx_statement_status ON finance.bank_statement_lines(status) WHERE status = 'UNMATCHED'`,
		// The Pix end-to-end id appears on BOTH sides of an internal transfer, which
		// makes it the only key that reconciles one without guessing.
		`CREATE INDEX IF NOT EXISTS idx_statement_e2e ON finance.bank_statement_lines(end_to_end_id) WHERE end_to_end_id IS NOT NULL`,
		// Migration: a ledger reverses instead of deleting. The row is kept so it can
		// still answer what was undone, when and why; balances derive from CONFIRMED,
		// so a reversed row stops counting without disappearing.
		`ALTER TABLE finance.transactions ADD COLUMN IF NOT EXISTS reversed_at TIMESTAMP`,
		`ALTER TABLE finance.transactions ADD COLUMN IF NOT EXISTS reversal_reason TEXT`,
		`ALTER TABLE finance.transactions ADD COLUMN IF NOT EXISTS reversal_note TEXT`,
		`ALTER TABLE finance.transactions ADD COLUMN IF NOT EXISTS reversed_by TEXT`,
		// A reversed row must carry why and by whom. The constraint is what makes the
		// control operate instead of merely existing in the schema.
		`DO $$
		BEGIN
			ALTER TABLE finance.transactions DROP CONSTRAINT IF EXISTS transactions_reversal_audited;
			ALTER TABLE finance.transactions ADD CONSTRAINT transactions_reversal_audited
				CHECK (status <> 'REVERSED' OR (reversal_reason IS NOT NULL AND reversed_by IS NOT NULL));
		END $$`,
		`DO $$
		BEGIN
			ALTER TABLE finance.transactions DROP CONSTRAINT IF EXISTS transactions_status_check;
			ALTER TABLE finance.transactions ADD CONSTRAINT transactions_status_check
				CHECK (status IN ('PLANNED', 'CONFIRMED', 'CANCELLED', 'REVERSED'));
		END $$`,
		// Migration: allow PARTIALLY_PAID on credit card invoices.
		// A bill paid in parts used to be marked PAID on the first payment, which
		// erased the outstanding debt from the ledger.
		`DO $$
		BEGIN
			ALTER TABLE finance.credit_card_invoices DROP CONSTRAINT IF EXISTS credit_card_invoices_status_check;
			ALTER TABLE finance.credit_card_invoices ADD CONSTRAINT credit_card_invoices_status_check
				CHECK (status IN ('OPEN', 'CLOSED', 'PAID', 'PARTIALLY_PAID'));
		END $$`,
		// Migration: Create crypto_purchases table for tracking structured purchase data
		`CREATE TABLE IF NOT EXISTS finance.crypto_purchases (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			transaction_id UUID NOT NULL REFERENCES finance.transactions(id) ON DELETE CASCADE,
			bank_account_id UUID NOT NULL REFERENCES finance.bank_accounts(id),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id),
			asset VARCHAR(20) NOT NULL,
			quantity DECIMAL(20, 10) NOT NULL,
			price_usd DECIMAL(20, 8) NOT NULL,
			exchange_rate DECIMAL(20, 6) NOT NULL,
			invested_brl DECIMAL(20, 2) NOT NULL,
			invested_usd DECIMAL(20, 8) NOT NULL,
			occurred_on DATE NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_crypto_purchases_account ON finance.crypto_purchases(bank_account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_crypto_purchases_profile ON finance.crypto_purchases(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_crypto_purchases_asset ON finance.crypto_purchases(asset)`,
		// Migration: Add sold_quantity and strategy columns to crypto_purchases
		`ALTER TABLE finance.crypto_purchases ADD COLUMN IF NOT EXISTS sold_quantity DECIMAL(20, 10) NOT NULL DEFAULT 0`,
		`ALTER TABLE finance.crypto_purchases ADD COLUMN IF NOT EXISTS strategy VARCHAR(50) NOT NULL DEFAULT 'manual'`,
		`CREATE INDEX IF NOT EXISTS idx_crypto_purchases_strategy ON finance.crypto_purchases(strategy)`,

		// Phase 1 — PJ Profile: add nullable PJ-specific columns to profiles
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='legal_entity_type') THEN
				ALTER TABLE finance.profiles ADD COLUMN legal_entity_type VARCHAR(10) CHECK (legal_entity_type IN ('MEI','ME','EPP','LTDA','SA'));
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='company_name') THEN
				ALTER TABLE finance.profiles ADD COLUMN company_name VARCHAR(255);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='cnpj') THEN
				ALTER TABLE finance.profiles ADD COLUMN cnpj VARCHAR(20);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='simples_nacional') THEN
				ALTER TABLE finance.profiles ADD COLUMN simples_nacional BOOLEAN;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='tax_regime') THEN
				ALTER TABLE finance.profiles ADD COLUMN tax_regime VARCHAR(20) CHECK (tax_regime IN ('SIMPLES','LUCRO_PRESUMIDO','LUCRO_REAL'));
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='das_aliquota') THEN
				ALTER TABLE finance.profiles ADD COLUMN das_aliquota NUMERIC(5,2);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='profiles' AND column_name='opening_date') THEN
				ALTER TABLE finance.profiles ADD COLUMN opening_date DATE;
			END IF;
		END $$`,

		// Phase 1 — DRE Classification: add nullable classification_dre to categories
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='categories' AND column_name='classification_dre') THEN
				ALTER TABLE finance.categories ADD COLUMN classification_dre VARCHAR(20)
					CHECK (classification_dre IN ('REVENUE','TAX','FIXED_COST','VARIABLE_COST','PROLABORE','MARKETING','FINANCIAL','ASSET','CAPITAL'));
			END IF;
		END $$`,

		// Phase 1 — Capital Contributions: track owner investments in the company
		`CREATE TABLE IF NOT EXISTS finance.capital_contributions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL CHECK (type IN ('CONTRIBUTION','WITHDRAWAL','LOAN')),
			amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
			date DATE NOT NULL,
			description VARCHAR(255) NOT NULL,
			source_account_id UUID REFERENCES finance.bank_accounts(id) ON DELETE SET NULL,
			notes TEXT,
			is_returned BOOLEAN NOT NULL DEFAULT false,
			returned_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
			returned_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_capital_contributions_profile ON finance.capital_contributions(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_capital_contributions_date ON finance.capital_contributions(date)`,

		// Phase 1 — Company Assets: track fixed assets owned by the company
		`CREATE TABLE IF NOT EXISTS finance.company_assets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			category VARCHAR(20) NOT NULL CHECK (category IN ('HARDWARE','SOFTWARE','FURNITURE','VEHICLE','REAL_ESTATE','OTHER')),
			purchase_date DATE NOT NULL,
			purchase_amount NUMERIC(15,2) NOT NULL CHECK (purchase_amount > 0),
			current_value NUMERIC(15,2) NOT NULL DEFAULT 0,
			depreciation_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
			linked_transaction_id UUID REFERENCES finance.transactions(id) ON DELETE SET NULL,
			notes TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			disposal_date DATE,
			disposal_amount NUMERIC(15,2),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_company_assets_profile ON finance.company_assets(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_company_assets_active ON finance.company_assets(profile_id, is_active)`,

		// Phase 3: Cost Centers
		`CREATE TABLE IF NOT EXISTS finance.cost_centers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(20) NOT NULL CHECK (type IN ('CLIENT','PROJECT','DEPARTMENT')),
			color VARCHAR(20),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cost_centers_profile ON finance.cost_centers(profile_id)`,

		// A cost center can mirror a record in another system — a CRM
		// organization, for instance. VARCHAR on purpose, never uuid: the CRM's
		// own table holds both uuid and cuid ids, because imported records
		// bring their own, so a uuid column would reject the older half.
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='cost_centers' AND column_name='external_id') THEN
				ALTER TABLE finance.cost_centers ADD COLUMN external_id VARCHAR(255);
				ALTER TABLE finance.cost_centers ADD COLUMN external_source VARCHAR(60);
			END IF;
		END $$`,
		// One CRM organization maps to at most one cost center per profile, so a
		// repeated sync finds the existing one instead of creating a twin.
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_cost_centers_external
			ON finance.cost_centers(profile_id, external_source, external_id)
			WHERE external_id IS NOT NULL`,

		// Phase 3: Marketing Campaigns
		`CREATE TABLE IF NOT EXISTS finance.marketing_campaigns (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			profile_id UUID NOT NULL REFERENCES finance.profiles(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			platform VARCHAR(20) NOT NULL CHECK (platform IN ('META_ADS','GOOGLE_ADS','INSTAGRAM','LINKEDIN','OTHER')),
			start_date DATE NOT NULL,
			end_date DATE,
			budget NUMERIC(15,2) NOT NULL DEFAULT 0,
			revenue_attributed NUMERIC(15,2) NOT NULL DEFAULT 0,
			leads INT NOT NULL DEFAULT 0,
			conversions INT NOT NULL DEFAULT 0,
			notes TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_marketing_campaigns_profile ON finance.marketing_campaigns(profile_id)`,

		// Phase 3: Add cost_center_id and campaign_id to transactions (non-destructive, nullable)
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='transactions' AND column_name='cost_center_id') THEN
				ALTER TABLE finance.transactions ADD COLUMN cost_center_id UUID REFERENCES finance.cost_centers(id) ON DELETE SET NULL;
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='transactions' AND column_name='campaign_id') THEN
				ALTER TABLE finance.transactions ADD COLUMN campaign_id UUID REFERENCES finance.marketing_campaigns(id) ON DELETE SET NULL;
			END IF;
		END $$`,

		// Migration: Add display_order to goals (if missing from initial migration)
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='goals' AND column_name='display_order') THEN
				ALTER TABLE finance.goals ADD COLUMN display_order INTEGER NOT NULL DEFAULT 0;
			END IF;
		END $$`,

		// Migration: Add goal_type to goals
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='goals' AND column_name='goal_type') THEN
				ALTER TABLE finance.goals ADD COLUMN goal_type VARCHAR(30) NOT NULL DEFAULT 'PERSONAL_SAVINGS';
			END IF;
		END $$`,

		// Migration: Add is_personal_reimbursement to transactions
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='finance' AND table_name='transactions' AND column_name='is_personal_reimbursement') THEN
				ALTER TABLE finance.transactions ADD COLUMN is_personal_reimbursement BOOLEAN NOT NULL DEFAULT FALSE;
			END IF;
		END $$`,

		// Migration: Update Clear broker account from CHECKING to INVESTMENT
		// and fill investment_type/broker for its sub-accounts (FIIs)
		`DO $$
		DECLARE
			clear_id UUID;
		BEGIN
			-- Find the Clear account (CHECKING type, name = 'Clear')
			SELECT id INTO clear_id FROM finance.bank_accounts
			WHERE name = 'Clear' AND type = 'CHECKING' LIMIT 1;

			IF clear_id IS NOT NULL THEN
				-- Change Clear from CHECKING to INVESTMENT (broker account)
				UPDATE finance.bank_accounts
				SET type = 'INVESTMENT', broker = 'Clear', updated_at = NOW()
				WHERE id = clear_id;

				-- Set investment_type = FII for sub-accounts with ticker ending in 11
				UPDATE finance.bank_accounts
				SET investment_type = 'FII', yield_type = 'VARIABLE', broker = 'Clear', updated_at = NOW()
				WHERE linked_account_id = clear_id
				AND investment_type IS NULL
				AND name ~ '^[A-Z]{4}11$';

				-- Set investment_type = STOCKS for sub-accounts with ticker ending in 3-8
				UPDATE finance.bank_accounts
				SET investment_type = 'STOCKS', yield_type = 'VARIABLE', broker = 'Clear', updated_at = NOW()
				WHERE linked_account_id = clear_id
				AND investment_type IS NULL
				AND name ~ '^[A-Z]{4}[3-8]$';
			END IF;
		END $$`,

		// Balance checkpoints — monthly closing balances for O(recent) recalculation
		`CREATE TABLE IF NOT EXISTS finance.balance_checkpoints (
			id             UUID PRIMARY KEY,
			account_id     UUID NOT NULL REFERENCES finance.bank_accounts(id) ON DELETE CASCADE,
			reference_month DATE NOT NULL,
			closing_balance NUMERIC(15,2) NOT NULL,
			created_at     TIMESTAMP NOT NULL DEFAULT now(),
			UNIQUE (account_id, reference_month)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_balance_checkpoints_account_month
			ON finance.balance_checkpoints (account_id, reference_month DESC)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("✓ Migrations completed successfully")
	return nil
}
