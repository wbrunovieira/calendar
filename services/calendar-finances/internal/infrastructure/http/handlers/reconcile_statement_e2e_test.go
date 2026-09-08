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
