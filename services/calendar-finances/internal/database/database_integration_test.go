//go:build integration

package database

import (
	"os"
	"testing"
)

// TestMigrations_ExternalIDUnique garante o guard-rail no banco contra
// transações duplicadas por external_id (regressão do bug do FindByExternalID
// que gerou ~223 mil trades Binance duplicados em produção).
func TestMigrations_ExternalIDUnique(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable"
	}

	db, err := Connect(dbURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	profileID := "9f2b7c1e-0000-4000-8000-000000000001"
	cleanup := func() {
		db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", profileID)
		db.Exec("DELETE FROM finance.bank_accounts WHERE profile_id = $1", profileID)
		db.Exec("DELETE FROM finance.profiles WHERE id = $1", profileID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := db.Exec(
		"INSERT INTO finance.profiles (id, calendar_id, name, type) VALUES ($1, 'test-cal-uq', 'test-uq', 'PERSONAL')",
		profileID,
	); err != nil {
		t.Fatalf("failed to insert profile: %v", err)
	}

	var accountID string
	if err := db.QueryRow(
		"INSERT INTO finance.bank_accounts (profile_id, name, type) VALUES ($1, 'test-uq-acc', 'CHECKING') RETURNING id",
		profileID,
	).Scan(&accountID); err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}

	insert := func() error {
		_, err := db.Exec(`
			INSERT INTO finance.transactions
				(profile_id, bank_account_id, type, status, amount, currency, occurred_on, external_id)
			VALUES ($1, $2, 'INCOME', 'CONFIRMED', 1.00, 'BRL', '2026-07-09', 'test-uq-binance-1')`,
			profileID, accountID,
		)
		return err
	}

	if err := insert(); err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}
	if err := insert(); err == nil {
		t.Fatal("second insert with same external_id should fail (unique index missing?)")
	}

	// external_id NULL continua livre (índice é parcial)
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`
			INSERT INTO finance.transactions
				(profile_id, bank_account_id, type, status, amount, currency, occurred_on)
			VALUES ($1, $2, 'EXPENSE', 'CONFIRMED', 2.00, 'BRL', '2026-07-09')`,
			profileID, accountID,
		); err != nil {
			t.Fatalf("insert with NULL external_id should always succeed: %v", err)
		}
	}
}
