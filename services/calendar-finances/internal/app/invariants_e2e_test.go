//go:build integration
// +build integration

package app_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/app"
	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/database"
	_ "github.com/lib/pq"
)

const (
	invariantsProfileID = "3a1f0000-0000-4000-8000-000000000001"
	invariantsAccountID = "3a1f0000-0000-4000-8000-000000000002"
)

func invariantsDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("integration DB unavailable (%v); skipping", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() {
		invariantsCleanup(db)
		_ = db.Close()
	})
	invariantsCleanup(db)
	return db
}

func invariantsCleanup(db *sql.DB) {
	db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", invariantsProfileID)
	db.Exec("DELETE FROM finance.bank_accounts WHERE profile_id = $1", invariantsProfileID)
	db.Exec("DELETE FROM finance.profiles WHERE id = $1", invariantsProfileID)
}

// invariantsSeed creates one checking account seeded with `initial` and one
// CONFIRMED income of `income`, so the ledger says initial+income.
func invariantsSeed(t *testing.T, db *sql.DB, initial, income, storedBalance float64) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'Invariants E2E', 'PERSONAL')`, invariantsProfileID, "invariants-e2e"); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO finance.bank_accounts (id, profile_id, name, type, initial_balance, current_balance)
		VALUES ($1, $2, 'Conta E2E', 'CHECKING', $3, $4)`,
		invariantsAccountID, invariantsProfileID, initial, storedBalance); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, 'INCOME', 'CONFIRMED', $3, 'Salário', CURRENT_DATE)`,
		invariantsProfileID, invariantsAccountID, income); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
}

func invariantsRequest(t *testing.T, db *sql.DB) (int, usecases.CheckInvariantsResult) {
	t.Helper()
	application, err := app.New(db)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	rec := httptest.NewRecorder()
	application.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health/invariants", nil))

	var body struct {
		Data usecases.CheckInvariantsResult `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the report: %v", err)
	}
	return rec.Code, body.Data
}

// findAccountDrift returns the report entry for the seeded account. The database
// is shared with other suites, so the report legitimately carries other rows.
func findAccountDrift(report usecases.CheckInvariantsResult) *usecases.AccountInvariant {
	for i := range report.AccountDrifts {
		if report.AccountDrifts[i].AccountID == invariantsAccountID {
			return &report.AccountDrifts[i]
		}
	}
	return nil
}

func TestInvariantsRoute_ReportsNoDriftForAConsistentAccount(t *testing.T) {
	db := invariantsDB(t)
	// 100 seed + 250 confirmed income = 350, and the column says 350.
	invariantsSeed(t, db, 100, 250, 350)

	_, report := invariantsRequest(t, db)

	if drift := findAccountDrift(report); drift != nil {
		t.Errorf("a consistent account must not be reported: %+v", *drift)
	}
	if report.CheckedAccounts == 0 {
		t.Error("the report checked no accounts at all")
	}
}

// This is the failure the endpoint exists to catch: a stored balance that no
// transaction justifies — exactly what a dual-write leaves behind.
func TestInvariantsRoute_CatchesAStoredBalanceNoTransactionJustifies(t *testing.T) {
	db := invariantsDB(t)
	invariantsSeed(t, db, 100, 250, 350)

	// Something wrote 87.65 into the column without a transaction behind it.
	if _, err := db.Exec(
		"UPDATE finance.bank_accounts SET current_balance = 437.65 WHERE id = $1",
		invariantsAccountID,
	); err != nil {
		t.Fatalf("corrupting the stored balance: %v", err)
	}

	status, report := invariantsRequest(t, db)

	if status != http.StatusConflict {
		t.Errorf("status = %d, want 409", status)
	}
	if report.OK {
		t.Error("ok = true while an account drifts")
	}

	drift := findAccountDrift(report)
	if drift == nil {
		t.Fatal("the drifting account is missing from the report")
	}
	if drift.StoredBalance != 437.65 {
		t.Errorf("storedBalance = %v, want 437.65", drift.StoredBalance)
	}
	if drift.ComputedBalance != 350 {
		t.Errorf("computedBalance = %v, want 350", drift.ComputedBalance)
	}
	if drift.Drift != 87.65 {
		t.Errorf("drift = %v, want 87.65", drift.Drift)
	}
	if drift.Note != "" {
		t.Errorf("a checking account must carry no excuse, got %q", drift.Note)
	}
}

// A CANCELLED transaction is not part of the ledger. If the endpoint counted it,
// every cancellation would show up as a phantom drift.
func TestInvariantsRoute_IgnoresCancelledTransactions(t *testing.T) {
	db := invariantsDB(t)
	invariantsSeed(t, db, 100, 250, 350)

	if _, err := db.Exec(`
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, 'EXPENSE', 'CANCELLED', 999.99, 'Compra cancelada', CURRENT_DATE)`,
		invariantsProfileID, invariantsAccountID); err != nil {
		t.Fatalf("seed cancelled transaction: %v", err)
	}

	_, report := invariantsRequest(t, db)

	if drift := findAccountDrift(report); drift != nil {
		t.Errorf("a cancelled transaction must not move the ledger: %+v", *drift)
	}
}

// PLANNED transactions are forecasts, not money. They must not count either.
func TestInvariantsRoute_IgnoresPlannedTransactions(t *testing.T) {
	db := invariantsDB(t)
	invariantsSeed(t, db, 100, 250, 350)

	if _, err := db.Exec(`
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, 'EXPENSE', 'PLANNED', 500.00, 'Aluguel previsto', CURRENT_DATE)`,
		invariantsProfileID, invariantsAccountID); err != nil {
		t.Fatalf("seed planned transaction: %v", err)
	}

	_, report := invariantsRequest(t, db)

	if drift := findAccountDrift(report); drift != nil {
		t.Errorf("a planned transaction must not move the ledger: %+v", *drift)
	}
}
