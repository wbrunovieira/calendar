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
		// Skipping locally is a convenience. Skipping in CI would turn this
		// whole file into a green tick that ran nothing.
		if os.Getenv("CI") != "" {
			t.Fatalf("integration DB unavailable in CI: %v", err)
		}
		t.Skipf("integration DB unavailable (%v); skipping", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() {
		invariantsCleanup(t, db)
		_ = db.Close()
	})
	invariantsCleanup(t, db)
	return db
}

// invariantsCleanup reports what it could not remove. Swallowing the error
// leaves the next test failing on a duplicate key, which hides the real cause.
func invariantsCleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"DELETE FROM finance.transactions WHERE profile_id = $1",
		"DELETE FROM finance.bank_accounts WHERE profile_id = $1",
		"DELETE FROM finance.profiles WHERE id = $1",
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement, invariantsProfileID); err != nil {
			t.Fatalf("cleanup failed (%s): %v", statement, err)
		}
	}
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

	// The cron reads the status code and nothing else, so it must never
	// disagree with the report it accompanies. The shared database may hold
	// other drifting accounts, which is why this is asserted as a relationship
	// rather than a fixed 200.
	wantCode := http.StatusConflict
	if body.Data.OK {
		wantCode = http.StatusOK
	}
	if rec.Code != wantCode {
		t.Errorf("status = %d but ok = %v; the code must follow the report", rec.Code, body.Data.OK)
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

// The reason this must not fail the check, proven against the real schema:
// credit_card_invoices.amount defaults to 0 and no write path maintains it. A
// card used once already reports a drift equal to the whole bill, and for a PAID
// invoice there is no route that can bring it to zero — POST /invoices/{id}/
// recalculate refuses PAID. An alarm nobody can clear stops being read.
func TestInvariantsRoute_ReportsAStaleInvoiceTotalWithoutFailingTheCheck(t *testing.T) {
	db := invariantsDB(t)
	invariantsSeed(t, db, 0, 0, 0)

	const cardID = "3a1f0000-0000-4000-8000-000000000003"
	const invoiceID = "3a1f0000-0000-4000-8000-000000000004"

	if _, err := db.Exec(`
		INSERT INTO finance.bank_accounts (id, profile_id, name, type, closing_day, due_day, current_balance)
		VALUES ($1, $2, 'Cartão E2E', 'CREDIT_CARD', 5, 15, 0)`,
		cardID, invariantsProfileID); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO finance.credit_card_invoices
			(id, bank_account_id, reference_date, opening_date, closing_date, due_date, amount, status)
		VALUES ($1, $2, '2026-03-01', '2026-02-06', '2026-03-05', '2026-03-15', 0, 'PAID')`,
		invoiceID, cardID); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO finance.transactions
			(profile_id, bank_account_id, invoice_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, $3, 'EXPENSE', 'CONFIRMED', 923.04, 'Compra', '2026-02-20')`,
		invariantsProfileID, cardID, invoiceID); err != nil {
		t.Fatalf("seed charge: %v", err)
	}

	_, report := invariantsRequest(t, db)

	var found *usecases.InvoiceInvariant
	for i := range report.InvoiceDrifts {
		if report.InvoiceDrifts[i].InvoiceID == invoiceID {
			found = &report.InvoiceDrifts[i]
		}
	}
	if found == nil {
		t.Fatal("the stale invoice total is missing from the report")
	}
	if found.StoredAmount != 0 || found.ComputedAmount != 923.04 {
		t.Errorf("stored/computed = %v/%v, want 0/923.04", found.StoredAmount, found.ComputedAmount)
	}
	if found.Note == "" {
		t.Error("a stale invoice total must explain that the column has no writer")
	}

	// The test database is shared, so another suite's rows can legitimately
	// make the overall report red. What must hold is narrower and stronger: an
	// invoice may never be the reason. Every invoice difference carries a note,
	// so invoices can never gate the check.
	for _, drift := range report.InvoiceDrifts {
		if drift.Note == "" {
			t.Errorf("an invoice difference without a note would fail a check nobody can clear: %+v", drift)
		}
	}

	// The card itself was never written either, so it must not fail the check.
	for _, drift := range report.AccountDrifts {
		if drift.AccountID == cardID && drift.Note == "" {
			t.Errorf("a card at zero balance must carry a note: %+v", drift)
		}
	}
}
