package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	rebuild := usecases.NewRebuildInvoiceCyclesUseCase(accounts, &invariantsTxRepo{balance: ledger}, &invariantsInvoiceRepo{})
	return NewInvariantsHandlers(uc, rebuild, nil, nil)
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

// A reconciliation that cannot reach the bank must NOT answer like a clean run.
// "No divergences" during an outage is exactly the silence this check exists to
// break, so both the unconfigured and the unreachable case answer 503.
type stubReconciler struct {
	report usecases.ReconciliationReport
	err    error
}

func (s *stubReconciler) Execute() (usecases.ReconciliationReport, error) {
	return s.report, s.err
}

func TestBankReconciliationAnswers503WhenTheBankCannotBeReached(t *testing.T) {
	h := &InvariantsHandlers{}
	h.SetBankReconciler(&stubReconciler{err: errors.New("open finance answered 409: not_connected")})
	rec := httptest.NewRecorder()

	h.BankReconciliation(rec, httptest.NewRequest(http.MethodGet, "/health/bank-reconciliation", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not_connected") {
		t.Errorf("body = %q, want the provider's reason", rec.Body.String())
	}
}

func TestBankReconciliationAnswers503WhenItIsNotConfigured(t *testing.T) {
	rec := httptest.NewRecorder()

	(&InvariantsHandlers{}).BankReconciliation(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 — nothing was compared", rec.Code)
	}
}

func TestBankReconciliationReportsTheDivergences(t *testing.T) {
	h := &InvariantsHandlers{}
	h.SetBankReconciler(&stubReconciler{report: usecases.ReconciliationReport{
		Checked: 2,
		Diverged: []usecases.Divergence{{
			Name: "Cartão Mercado Pago", System: 4743.42, Bank: 4225.97, Difference: 517.45,
		}},
	}})
	rec := httptest.NewRecorder()

	h.BankReconciliation(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	// 409, not 200. The health-check cron has no jq and reads the status code,
	// so a 200 carrying the divergence in its body is a finding nobody is told
	// about. The first version of this test asserted 200 and would have locked
	// that in.
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 so the cron can see it", rec.Code)
	}
	for _, want := range []string{"Cartão Mercado Pago", "517.45", "4225.97"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body does not carry %q: %s", want, rec.Body.String())
		}
	}
}

// A clean run still answers 200, or the cron would alert every day.
func TestBankReconciliationAnswers200WhenEverythingAgrees(t *testing.T) {
	h := &InvariantsHandlers{}
	h.SetBankReconciler(&stubReconciler{report: usecases.ReconciliationReport{OK: true, Checked: 6, Total: 13}})
	rec := httptest.NewRecorder()

	h.BankReconciliation(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 on a clean run", rec.Code)
	}
}

// A run that compared nothing arrives here with OK=false and must not be 200.
func TestBankReconciliationDoesNotReport200ForAnEmptyComparison(t *testing.T) {
	h := &InvariantsHandlers{}
	h.SetBankReconciler(&stubReconciler{report: usecases.ReconciliationReport{OK: false, Checked: 0, Total: 13}})
	rec := httptest.NewRecorder()

	h.BankReconciliation(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code == http.StatusOK {
		t.Error("a comparison of nothing answered 200")
	}
}
