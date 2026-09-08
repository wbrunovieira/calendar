//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"encoding/json"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/database"
	httpHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// These drive the two routes this branch adds against a real database, with the same
// wiring main.go uses. Unit tests here run against fakes that answer whatever the test
// wants — which is how a sibling route shipped returning 500 on every call, because
// its fake ignored a filter the real repository rejects.
//
// Run with: go test -tags=integration ./...

func cashflowTestDB(t *testing.T) *sql.DB {
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

func cashflowRouter(t *testing.T, db *sql.DB) *mux.Router {
	t.Helper()

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	categoryRepo := persistence.NewCategoryRepository(db)
	recurringRepo := persistence.NewRecurringTransactionRepository(db)

	txHandler := httpHandlers.NewTransactionHandlers(nil, nil, nil, nil, nil, nil)
	txHandler.SetCashflowSummaryUseCase(usecases.NewGetCashflowSummaryUseCase(txRepo, accountRepo, categoryRepo))

	recHandler := httpHandlers.NewRecurringTransactionHandlers(nil, nil)
	recHandler.SetPendingUseCase(usecases.NewListPendingRecurringsUseCase(recurringRepo, txRepo))

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/transactions/cashflow-summary", txHandler.CashflowSummary).Methods("GET")
	api.HandleFunc("/recurring-transactions/pending", recHandler.Pending).Methods("GET")
	return r
}

func getJSON(t *testing.T, r *mux.Router, path string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("seeding: %v\nquery: %s", err, query)
	}
}

func seedProfileAndAccount(t *testing.T, db *sql.DB) (profileID, accountID string) {
	t.Helper()
	if err := db.QueryRow(`
		INSERT INTO finance.profiles (calendar_id, name, type)
		VALUES ('e2e-' || gen_random_uuid(), 'E2E', 'PERSONAL') RETURNING id
	`).Scan(&profileID); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}
	accountID = seedAccountThroughRepository(t, db, checkingAccount(profileID, "Conta E2E", 0, 0))

	t.Cleanup(func() {
		execSQL(t, db, `DELETE FROM finance.transactions WHERE profile_id = $1`, profileID)
		execSQL(t, db, `DELETE FROM finance.recurring_transactions WHERE profile_id = $1`, profileID)
		execSQL(t, db, `DELETE FROM finance.bank_accounts WHERE profile_id = $1`, profileID)
		execSQL(t, db, `DELETE FROM finance.categories WHERE profile_id = $1`, profileID)
		execSQL(t, db, `DELETE FROM finance.profiles WHERE id = $1`, profileID)
	})
	return profileID, accountID
}

func TestCashflowSummaryRoute_Answers(t *testing.T) {
	db := cashflowTestDB(t)
	profileID, accountID := seedProfileAndAccount(t, db)

	execSQL(t, db, `
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, 'INCOME', 'CONFIRMED', 5000, 'BRL', 'Salario', '2026-09-05'),
		       ($1, $2, 'INCOME', 'CONFIRMED', 12.50, 'BRL', 'Rendimento da conta', '2026-09-05'),
		       ($1, $2, 'EXPENSE', 'CONFIRMED', 200, 'BRL', 'Supermercado', '2026-09-05')
	`, profileID, accountID)

	status, body := getJSON(t, cashflowRouter(t, db),
		"/api/v1/transactions/cashflow-summary?profileId="+profileID+"&from=2026-09-01&to=2026-09-30")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — %v", status, body)
	}

	data := body["data"].(map[string]any)
	if got := data["income"].(float64); got != 5012.50 {
		t.Errorf("expected 5012.50 of income, got %v", got)
	}
	if got := data["incomeYield"].(float64); got != 12.50 {
		t.Errorf("expected the account interest counted as yield, got %v", got)
	}
	if got := data["expense"].(float64); got != 200 {
		t.Errorf("expected 200 of expense, got %v", got)
	}
}

// The month must not count the two legs of paying a card bill as real money.
func TestCashflowSummaryRoute_DropsInvoicePayments(t *testing.T) {
	db := cashflowTestDB(t)
	profileID, accountID := seedProfileAndAccount(t, db)

	var cardID string
	cardID = seedAccountThroughRepository(t, db, cardAccount(profileID, "Cartao E2E", 1000))
	var invoiceID string
	invoiceID = seedInvoiceThroughRepository(t, db, &invoice.Invoice{
		BankAccountID: cardID, Amount: 0, Status: invoice.StatusOpen,
		ReferenceDate: time.Date(2026, time.September, 27, 0, 0, 0, 0, time.UTC), OpeningDate: time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC),
		ClosingDate: time.Date(2026, time.September, 27, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC),
	})

	execSQL(t, db, `
		INSERT INTO finance.transactions (profile_id, bank_account_id, invoice_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, $3, 'EXPENSE', 'CONFIRMED', 300, 'BRL', 'Compra no cartao', '2026-09-05')
	`, profileID, cardID, invoiceID)
	// The credit that lands on the card when the bill is paid.
	execSQL(t, db, `
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, 'INCOME', 'CONFIRMED', 300, 'BRL', 'Pagamento fatura', '2026-09-10')
	`, profileID, cardID)
	// The same bill paid by hand from the checking account: an expense with no
	// destination, which is how 17 rows in production are written.
	execSQL(t, db, `
		INSERT INTO finance.transactions (profile_id, bank_account_id, invoice_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, $3, 'EXPENSE', 'CONFIRMED', 300, 'BRL', 'Pagamento fatura cartao (venc 03/10)', '2026-09-10')
	`, profileID, accountID, invoiceID)

	status, body := getJSON(t, cashflowRouter(t, db),
		"/api/v1/transactions/cashflow-summary?profileId="+profileID+"&from=2026-09-01&to=2026-09-30")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — %v", status, body)
	}

	data := body["data"].(map[string]any)
	if got := data["expense"].(float64); got != 300 {
		t.Fatalf("expected only the purchase counted (300), got %v — the bill payment was double counted", got)
	}
	if got := data["income"].(float64); got != 0 {
		t.Fatalf("expected the card credit not to count as income, got %v", got)
	}
}

func TestPendingRecurringsRoute_ReportsAnUnpaidObligation(t *testing.T) {
	db := cashflowTestDB(t)
	profileID, accountID := seedProfileAndAccount(t, db)

	execSQL(t, db, `
		INSERT INTO finance.recurring_transactions
			(profile_id, bank_account_id, type, amount, currency, description, recurrence_rule, start_on, next_occurrence, status)
		VALUES ($1, $2, 'EXPENSE', 2000, 'BRL', 'Aluguel', 'FREQ=MONTHLY;BYMONTHDAY=5', '2026-01-05', '2026-09-05', 'ACTIVE')
	`, profileID, accountID)

	status, body := getJSON(t, cashflowRouter(t, db),
		"/api/v1/recurring-transactions/pending?profileId="+profileID+"&reference=2026-09-10")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — %v", status, body)
	}

	items, _ := body["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected the unpaid rent to be pending, got %v", body["data"])
	}
	first := items[0].(map[string]any)
	if first["description"] != "Aluguel" {
		t.Errorf("expected Aluguel, got %v", first["description"])
	}
	if got := first["daysOverdue"].(float64); got != 5 {
		t.Errorf("expected 5 days overdue, got %v", got)
	}
}

func TestPendingRecurringsRoute_DropsOffOncePaid(t *testing.T) {
	db := cashflowTestDB(t)
	profileID, accountID := seedProfileAndAccount(t, db)

	execSQL(t, db, `
		INSERT INTO finance.recurring_transactions
			(profile_id, bank_account_id, type, amount, currency, description, recurrence_rule, start_on, next_occurrence, status)
		VALUES ($1, $2, 'EXPENSE', 2000, 'BRL', 'Aluguel', 'FREQ=MONTHLY;BYMONTHDAY=5', '2026-01-05', '2026-09-05', 'ACTIVE')
	`, profileID, accountID)
	execSQL(t, db, `
		INSERT INTO finance.transactions (profile_id, bank_account_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, 'EXPENSE', 'CONFIRMED', 2000, 'BRL', 'Aluguel', $3)
	`, profileID, accountID, time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))

	status, body := getJSON(t, cashflowRouter(t, db),
		"/api/v1/recurring-transactions/pending?profileId="+profileID+"&reference=2026-09-10")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — %v", status, body)
	}
	if items, _ := body["data"].([]any); len(items) != 0 {
		t.Fatalf("expected nothing pending after payment, got %v", body["data"])
	}
}

// seedAccountThroughRepository creates the account with the same statement production
// uses, instead of an INSERT written for the test.
//
// The two used to differ, and the difference was invisible: a column duplicated in
// BankAccountRepository.Create broke every POST /bank-accounts while this suite stayed
// green, because nothing here ever called it.
//
// It returns early when the row is already there, which is what the ON CONFLICT DO
// NOTHING it replaced did. Without that, a run interrupted halfway leaves rows behind
// and every later run fails on a duplicate key — turning a self-healing helper into a
// hard stop against a dirty local database.
func seedAccountThroughRepository(t *testing.T, db *sql.DB, acc *bankaccount.BankAccount) string {
	t.Helper()
	if acc.ID == "" {
		acc.ID = uuid.NewString()
	}
	repo := persistence.NewBankAccountRepository(db)
	if existing, err := repo.FindByID(acc.ID); err == nil && existing != nil {
		return acc.ID
	}
	if acc.Currency == "" {
		acc.Currency = "BRL"
	}
	acc.IsActive = true
	acc.CreatedAt, acc.UpdatedAt = time.Now(), time.Now()
	if err := repo.Create(acc); err != nil {
		t.Fatalf("seeding account through the repository: %v", err)
	}
	return acc.ID
}

func checkingAccount(profileID, name string, initial, current float64) *bankaccount.BankAccount {
	return &bankaccount.BankAccount{
		ProfileID: profileID, Name: name, Type: bankaccount.AccountTypeChecking,
		InitialBalance: initial, CurrentBalance: current,
	}
}

func cardAccount(profileID, name string, limit float64) *bankaccount.BankAccount {
	closing, due := 27, 3
	return &bankaccount.BankAccount{
		ProfileID: profileID, Name: name, Type: bankaccount.AccountTypeCreditCard,
		CreditLimit: &limit, ClosingDay: &closing, DueDay: &due,
	}
}
