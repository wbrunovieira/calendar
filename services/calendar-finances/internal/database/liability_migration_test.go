//go:build integration
// +build integration

package database

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// The domain accepting LIABILITY means nothing if the column refuses it. A CHECK
// constraint rejects the insert at the last moment, which is the worst place to
// find out: every layer above reports success right up to the write.
func TestMigrations_LiabilityIsAcceptedByTheColumn(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgresql://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("test database unavailable (%v)", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	var profileID string
	if err := db.QueryRow(`
		INSERT INTO finance.profiles (calendar_id, name, type)
		VALUES ('liab-' || gen_random_uuid(), 'Liability Test', 'PERSONAL') RETURNING id
	`).Scan(&profileID); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.bank_accounts WHERE profile_id = $1`, profileID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, profileID)
	})

	if _, err := db.Exec(`
		INSERT INTO finance.bank_accounts (profile_id, name, type, currency, initial_balance, current_balance)
		VALUES ($1, 'Divida com as irmas', 'LIABILITY', 'BRL', 0, 0)`, profileID); err != nil {
		t.Fatalf("the column refused LIABILITY: %v", err)
	}

	// And the constraint still does its job for a type that does not exist.
	if _, err := db.Exec(`
		INSERT INTO finance.bank_accounts (profile_id, name, type, currency, initial_balance, current_balance)
		VALUES ($1, 'invalida', 'NAO_EXISTE', 'BRL', 0, 0)`, profileID); err == nil {
		t.Fatal("the constraint was dropped and never put back: any string is now a valid account type")
	}
}
