//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"encoding/json"
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

// These tests drive the real routes against a real database, with the same wiring
// main.go uses. Unit tests here run against fakes that answer whatever the test wants,
// which is how a route that returns 500 on every call shipped unnoticed: the fake
// repository ignored the filter the real one rejects.
//
// Run with: go test -tags=integration ./...
// Needs TEST_DATABASE_URL, defaulting to the local compose database.

func testDB(t *testing.T) *sql.DB {
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
		t.Skipf("test database unavailable (%v) — start it with: docker compose up -d postgres", err)
	}
	// The suite owns its schema, so a blank database is enough to run against.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("preparing the schema: %v", err)
	}

	// Closed by the process, not by the test: t.Cleanup callbacks that tidy the seed
	// data run after any defer in the test body would have closed it.
	return db
}

// router wires the routes under test exactly as main.go does.
func router(t *testing.T, db *sql.DB) *mux.Router {
	t.Helper()

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	invoiceRepo := persistence.NewInvoiceRepository(db)

	handler := httpHandlers.NewBankAccountHandlers(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetCreditUsageUseCase(usecases.NewGetCreditUsageUseCase(accountRepo, invoiceRepo, txRepo))

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/bank-accounts/{id}/credit-usage", handler.CreditUsage).Methods("GET")
	return r
}

func get(t *testing.T, r *mux.Router, path string) (int, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("seeding: %v\nquery: %s", err, query)
	}
}
