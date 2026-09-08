//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"encoding/json"
	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	httpHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/statement"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Against a real database, because what is being tested is the interaction: the
// UNIQUE that stops one entry being claimed twice, and whether a second run finds
// nothing left to do.
//
// The scenario is the one that happened by hand: a statement carrying a charge the
// ledger has, and one it does not. Finding the second is the reason this exists.
func TestE2E_ReconcileFindsWhatTheLedgerIsMissing(t *testing.T) {
	db := testDB(t)
	seedStatementAccount(t, db)

	lines := persistence.NewStatementRepository(db)
	known := lineFor(t, "conhecida", -5558, "BRL", nil) // the Cloudflare entry exists
	unknown := lineFor(t, "ausente", -5390, "BRL", nil) // nothing in the ledger for this
	if _, _, err := lines.UpsertMany([]*statement.Line{known, unknown}); err != nil {
		t.Fatalf("import: %v", err)
	}

	status, body := reconcile(t, db, stAccountID)

	// 409 because there IS something to look at — that is the point of the run.
	if status != http.StatusConflict {
		t.Fatalf("got %d: %s", status, body)
	}

	var out struct {
		Data struct {
			Matched int `json:"matched"`
			Missing []struct {
				AmountMinor int64  `json:"amountMinor"`
				Description string `json:"description"`
			} `json:"missing"`
			Ambiguous []struct{} `json:"ambiguous"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode: %v — %s", err, body)
	}
	if out.Data.Matched != 1 {
		t.Errorf("the charge the ledger has must be linked, got %d", out.Data.Matched)
	}
	if len(out.Data.Missing) != 1 || out.Data.Missing[0].AmountMinor != -5390 {
		t.Fatalf("the charge the ledger lacks must be named: %+v", out.Data.Missing)
	}
	if len(out.Data.Ambiguous) != 0 {
		t.Errorf("nothing here is ambiguous: %+v", out.Data.Ambiguous)
	}

	// The morning cron runs over an overlapping window, so a second pass must find
	// nothing new — and must not claim the same entry twice.
	_, second := reconcile(t, db, stAccountID)
	var again struct {
		Data struct {
			Matched int `json:"matched"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(second), &again)
	if again.Data.Matched != 0 {
		t.Errorf("the second run matched %d more", again.Data.Matched)
	}

	var live int
	if err := db.QueryRow(`
		SELECT count(*) FROM finance.reconciliation_matches
		WHERE transaction_id = $1 AND unmatched_at IS NULL`, stTxID).Scan(&live); err != nil {
		t.Fatalf("count matches: %v", err)
	}
	if live != 1 {
		t.Fatalf("the entry must be claimed exactly once, found %d", live)
	}
}

// reconcile builds the handler from the REAL repositories and calls it, so the test
// exercises the same graph production wires — not a convenient subset of it.
func reconcile(t *testing.T, db *sql.DB, accountID string) (int, string) {
	t.Helper()
	accounts := persistence.NewBankAccountRepository(db)
	handler := httpHandlers.NewStatementHandlers(
		accounts,
		usecases.NewImportStatementUseCase(persistence.NewStatementRepository(db)),
		usecases.NewReconcileStatementUseCase(
			persistence.NewStatementRepository(db),
			persistence.NewTransactionRepository(db),
			persistence.NewMatchRepository(db),
			accounts,
		),
	)

	r := mux.NewRouter()
	r.PathPrefix("/api/v1").Subrouter().
		HandleFunc("/bank-accounts/{id}/statement/reconcile", handler.Reconcile).Methods("POST")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/api/v1/bank-accounts/"+accountID+"/statement/reconcile", strings.NewReader("{}")))
	return rec.Code, rec.Body.String()
}

const (
	payProfileID  = "e2e00000-0000-0000-0000-0000000000b1"
	payCheckingID = "e2e00000-0000-0000-0000-0000000000b2"
	payCardID     = "e2e00000-0000-0000-0000-0000000000b3"
	payTxID       = "e2e00000-0000-0000-0000-0000000000b4"
)

// A card bill payment is ONE ledger row that the bank prints on TWO statements: it
// leaves the checking account and lands on the card. Reconciling one side must not
// consume it for the other, or whichever account runs second reports money missing
// that is not missing — and this system exists to be believed when it says that.
//
// Against a real database on purpose: the account scope lives in a SQL JOIN, and a
// fake agreeing with itself is what let the global version ship.
func TestE2E_ABillPaymentReconcilesOnBothStatements(t *testing.T) {
	db := testDB(t)
	seedPaymentAccounts(t, db)

	lines := persistence.NewStatementRepository(db)
	if _, _, err := lines.UpsertMany([]*statement.Line{
		paymentLine(t, "saida", payCheckingID, -101818),
		paymentLine(t, "entrada", payCardID, 101818),
	}); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Order matters to the bug: the first account to run is the one that used to win.
	_, checking := reconcile(t, db, payCheckingID)
	_, card := reconcile(t, db, payCardID)

	assertOneMatchNothingMissing(t, "conta corrente", checking)
	assertOneMatchNothingMissing(t, "cartão", card)

	var live int
	if err := db.QueryRow(`
		SELECT count(*) FROM finance.reconciliation_matches
		WHERE transaction_id = $1 AND unmatched_at IS NULL`, payTxID).Scan(&live); err != nil {
		t.Fatalf("count matches: %v", err)
	}
	if live != 2 {
		t.Fatalf("the payment must be claimed once per statement, found %d", live)
	}

	// And both lines must say so in storage, or the next run walks the whole history
	// again to rediscover what it already knows.
	var unmatched int
	if err := db.QueryRow(`
		SELECT count(*) FROM finance.bank_statement_lines
		WHERE account_id IN ($1,$2) AND status <> 'MATCHED'`,
		payCheckingID, payCardID).Scan(&unmatched); err != nil {
		t.Fatalf("count lines: %v", err)
	}
	if unmatched != 0 {
		t.Fatalf("%d reconciled lines are still stored as unmatched", unmatched)
	}
}

func assertOneMatchNothingMissing(t *testing.T, side, body string) {
	t.Helper()
	var out struct {
		Data struct {
			Matched int `json:"matched"`
			Missing []struct {
				AmountMinor int64 `json:"amountMinor"`
			} `json:"missing"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("%s: decode: %v — %s", side, err, body)
	}
	if out.Data.Matched != 1 || len(out.Data.Missing) != 0 {
		t.Fatalf("%s reported money missing that is not: %s", side, body)
	}
}

func seedPaymentAccounts(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.reconciliation_matches WHERE transaction_id = $1`, payTxID)
		db.Exec(`DELETE FROM finance.bank_statement_lines WHERE account_id IN ($1,$2)`, payCheckingID, payCardID)
		db.Exec(`DELETE FROM finance.transactions WHERE id = $1`, payTxID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id IN ($1,$2)`, payCheckingID, payCardID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, payProfileID)
	})
	exec(t, db, `INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1,$2,'E2E Pagamento','BUSINESS') ON CONFLICT (id) DO NOTHING`, payProfileID, "e2e-bill-payment")

	checking := checkingAccount(payProfileID, "Conta Pagadora", 0, 0)
	checking.ID = payCheckingID
	seedAccountThroughRepository(t, db, checking)

	card := cardAccount(payProfileID, "Cartão Pago", 3650)
	card.ID = payCardID
	seedAccountThroughRepository(t, db, card)

	exec(t, db, `INSERT INTO finance.transactions
		(id, profile_id, bank_account_id, destination_account_id, type, status,
		 amount, currency, description, occurred_on)
		VALUES ($1,$2,$3,$4,'TRANSFER','CONFIRMED',1018.18,'BRL','Pagamento fatura','2026-09-03')
		ON CONFLICT (id) DO NOTHING`, payTxID, payProfileID, payCheckingID, payCardID)
}

func paymentLine(t *testing.T, externalID, accountID string, amountMinor int64) *statement.Line {
	t.Helper()
	l, err := statement.New(statement.CreateParams{
		AccountID:       accountID,
		Provider:        statement.ProviderPluggy,
		ExternalID:      externalID,
		BookedDate:      time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
		AmountMinor:     amountMinor,
		Currency:        "BRL",
		AccountCurrency: "BRL",
		Description:     "Pagamento de fatura",
		Raw:             json.RawMessage(`{"id":"` + externalID + `","source":"test"}`),
	})
	if err != nil {
		t.Fatalf("building the line: %v", err)
	}
	return l
}

// An account that does not exist is not a malformed request, and a database that is
// down is not one either. The cron reads the status to decide whether to retry: 400
// tells it to give up and fix its payload, which is wrong for both.
func TestE2E_ReconcileAnswers404ForAnAccountThatDoesNotExist(t *testing.T) {
	db := testDB(t)

	status, body := reconcile(t, db, "e2e00000-0000-0000-0000-0000000000ff")

	if status != http.StatusNotFound {
		t.Fatalf("got %d, want 404: %s", status, body)
	}
}
