//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/database"
	httpHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// The deletion bug this branch fixes was invisible to unit tests for years: they
// passed nil as the balance recalculator, short-circuiting the very call that
// destroyed the correction. These drive the real route against a real database.
//
// Run with: go test -tags=integration ./...

func deleteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgresql://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("connecting to the test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("test database unavailable (%v) — start it with: docker compose up -d postgres", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("preparing the schema: %v", err)
	}
	return db
}

func deleteRouter(t *testing.T, db *sql.DB) *mux.Router {
	t.Helper()

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	invoiceRepo := persistence.NewInvoiceRepository(db)
	checkpointRepo := persistence.NewCheckpointRepository(db)

	recalculateUC := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, checkpointRepo)
	deleteUC := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalculateUC)
	deleteUC.SetInvoiceRecalculator(usecases.NewRecalculateInvoiceAmountUseCase(invoiceRepo, txRepo))

	handler := httpHandlers.NewTransactionHandlers(nil, nil, nil, nil, nil, deleteUC)

	r := mux.NewRouter()
	r.PathPrefix("/api/v1").Subrouter().
		HandleFunc("/transactions/{id}", handler.Delete).Methods("DELETE")
	return r
}

func deleteRequest(t *testing.T, r *mux.Router, id string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/transactions/"+id, nil))
	return rec.Code
}

func seedForDelete(t *testing.T, db *sql.DB) (profileID string) {
	t.Helper()
	if err := db.QueryRow(`
		INSERT INTO finance.profiles (calendar_id, name, type)
		VALUES ('e2e-del-' || gen_random_uuid(), 'E2E Delete', 'PERSONAL') RETURNING id
	`).Scan(&profileID); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}
	t.Cleanup(func() {
		mustExec(t, db, `DELETE FROM finance.transactions WHERE profile_id = $1`, profileID)
		mustExec(t, db, `DELETE FROM finance.bank_accounts WHERE profile_id = $1`, profileID)
		mustExec(t, db, `DELETE FROM finance.profiles WHERE id = $1`, profileID)
	})
	return profileID
}

func seedAccountFor(t *testing.T, db *sql.DB, profileID, name string, initial float64) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`
		INSERT INTO finance.bank_accounts (profile_id, name, type, initial_balance, current_balance, currency)
		VALUES ($1, $2, 'CHECKING', $3, $3, 'BRL') RETURNING id
	`, profileID, name, initial).Scan(&id); err != nil {
		t.Fatalf("seeding account: %v", err)
	}
	return id
}

func balanceOf(t *testing.T, db *sql.DB, accountID string) float64 {
	t.Helper()
	var balance float64
	if err := db.QueryRow(`SELECT current_balance FROM finance.bank_accounts WHERE id = $1`, accountID).Scan(&balance); err != nil {
		t.Fatalf("reading balance: %v", err)
	}
	return balance
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("seeding: %v\nquery: %s", err, query)
	}
}

// Deleting a confirmed expense has to give the money back. It did not: the balance
// was recomputed while the row was still in the ledger, so the old number went
// straight back and the row then vanished.
func TestDeleteTransactionRoute_RestoresTheBalance(t *testing.T) {
	db := deleteTestDB(t)
	profileID := seedForDelete(t, db)
	accountID := seedAccountFor(t, db, profileID, "Conta", 1000)

	var txID string
	if err := db.QueryRow(`
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, 'EXPENSE', 'CONFIRMED', 300, 'BRL', 'Mercado', now()) RETURNING id
	`, profileID, accountID).Scan(&txID); err != nil {
		t.Fatalf("seeding transaction: %v", err)
	}
	mustExec(t, db, `UPDATE finance.bank_accounts SET current_balance = 700 WHERE id = $1`, accountID)

	if code := deleteRequest(t, deleteRouter(t, db), txID); code != http.StatusNoContent && code != http.StatusOK {
		t.Fatalf("expected the delete to succeed, got %d", code)
	}

	if got := balanceOf(t, db, accountID); got != 1000 {
		t.Fatalf("expected 1000 back on the account, got %.2f", got)
	}
}

// Both legs of a transfer, and both must end where they started.
func TestDeleteTransactionRoute_ReversesBothLegsOfATransfer(t *testing.T) {
	db := deleteTestDB(t)
	profileID := seedForDelete(t, db)
	source := seedAccountFor(t, db, profileID, "Origem", 1000)
	destination := seedAccountFor(t, db, profileID, "Destino", 0)

	var txID string
	if err := db.QueryRow(`
		INSERT INTO finance.transactions
			(profile_id, bank_account_id, destination_account_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, $3, 'TRANSFER', 'CONFIRMED', 300, 'BRL', 'Aporte', now()) RETURNING id
	`, profileID, source, destination).Scan(&txID); err != nil {
		t.Fatalf("seeding transfer: %v", err)
	}
	mustExec(t, db, `UPDATE finance.bank_accounts SET current_balance = 700 WHERE id = $1`, source)
	mustExec(t, db, `UPDATE finance.bank_accounts SET current_balance = 300 WHERE id = $1`, destination)

	if code := deleteRequest(t, deleteRouter(t, db), txID); code != http.StatusNoContent && code != http.StatusOK {
		t.Fatalf("expected the delete to succeed, got %d", code)
	}

	if got := balanceOf(t, db, source); got != 1000 {
		t.Errorf("expected the source restored to 1000, got %.2f", got)
	}
	if got := balanceOf(t, db, destination); got != 0 {
		t.Errorf("expected the destination credit removed, got %.2f — phantom balance", got)
	}
}

func TestDeleteTransactionRoute_UnknownTransaction(t *testing.T) {
	db := deleteTestDB(t)

	if code := deleteRequest(t, deleteRouter(t, db), "00000000-0000-0000-0000-000000000000"); code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown transaction, got %d", code)
	}
}
