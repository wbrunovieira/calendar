package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Embedding the domain interfaces keeps these fakes honest: an unexpected call
// panics instead of returning a silent zero.

type invariantsAccountRepo struct {
	bankaccount.Repository
	accounts []*bankaccount.BankAccount
}

func (f *invariantsAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) {
	return f.accounts, nil
}

type invariantsTxRepo struct {
	transaction.Repository
	balance float64
}

func (f *invariantsTxRepo) CalculateBalanceByBankAccountID(string) (float64, error) {
	return f.balance, nil
}

type invariantsInvoiceRepo struct{ invoice.Repository }

func newInvariantsHandlerFor(stored, ledger float64) *InvariantsHandlers {
	accounts := &invariantsAccountRepo{accounts: []*bankaccount.BankAccount{{
		ID:             "acc-1",
		Name:           "Mercado Pago",
		Type:           bankaccount.AccountTypeChecking,
		InitialBalance: 0,
		CurrentBalance: stored,
	}}}
	uc := usecases.NewCheckInvariantsUseCase(
		accounts,
		&invariantsTxRepo{balance: ledger},
		&invariantsInvoiceRepo{},
	)
	return NewInvariantsHandlers(uc)
}

func TestInvariantsCheck_AnswersOKWhenEveryBalanceMatchesItsLedger(t *testing.T) {
	rec := httptest.NewRecorder()
	newInvariantsHandlerFor(250, 250).Check(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health/invariants", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var body struct {
		Data usecases.CheckInvariantsResult `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the report: %v", err)
	}
	if !body.Data.OK {
		t.Error("ok = false, want true")
	}
	if body.Data.CheckedAccounts != 1 {
		t.Errorf("checkedAccounts = %d, want 1", body.Data.CheckedAccounts)
	}
}

// The production health-check cron has no jq, so it reads the status code.
// A drift has to be visible without parsing the body.
func TestInvariantsCheck_AnswersConflictWhenALedgerDisagrees(t *testing.T) {
	rec := httptest.NewRecorder()
	newInvariantsHandlerFor(1000, 912.35).Check(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health/invariants", nil))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 so a drift is visible without parsing the body", rec.Code)
	}

	var body struct {
		Data usecases.CheckInvariantsResult `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the report: %v", err)
	}
	if body.Data.OK {
		t.Error("ok = true, want false")
	}
	if len(body.Data.AccountDrifts) != 1 {
		t.Fatalf("expected the drifting account in the body, got %d", len(body.Data.AccountDrifts))
	}
	if body.Data.AccountDrifts[0].Drift != 87.65 {
		t.Errorf("drift = %v, want 87.65", body.Data.AccountDrifts[0].Drift)
	}
}
