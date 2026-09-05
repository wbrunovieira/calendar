package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/gorilla/mux"
)

// fakeSellAccountRepo is an in-memory bankaccount.Repository for the sell e2e.
type fakeSellAccountRepo struct {
	accounts map[string]*bankaccount.BankAccount
}

func (r *fakeSellAccountRepo) Create(a *bankaccount.BankAccount) error {
	r.accounts[a.ID] = a
	return nil
}
func (r *fakeSellAccountRepo) FindByID(id string) (*bankaccount.BankAccount, error) {
	if a, ok := r.accounts[id]; ok {
		return a, nil
	}
	return nil, usecases.ErrBankAccountNotFound
}
func (r *fakeSellAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (r *fakeSellAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) { return nil, nil }
func (r *fakeSellAccountRepo) Update(a *bankaccount.BankAccount) error {
	r.accounts[a.ID] = a
	return nil
}
func (r *fakeSellAccountRepo) Delete(string) error                                        { return nil }
func (r *fakeSellAccountRepo) UpdateDisplayOrders([]bankaccount.DisplayOrderUpdate) error { return nil }

// sellTestServer wires the real use case + handler behind a mux router, so the
// request exercises routing, path-var extraction, the handler and the use case.
func sellTestServer(t *testing.T, repo *fakeSellAccountRepo) http.Handler {
	t.Helper()
	uc := usecases.NewSellPositionUseCase(repo, &FakeTransactionRepository{})
	h := NewBankAccountHandlers(nil, nil, nil, nil, nil, nil, nil, nil, nil, uc)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/bank-accounts/{id}/sell", h.Sell).Methods("POST")
	return router
}

func newSellFixture() (*fakeSellAccountRepo, *bankaccount.BankAccount, *bankaccount.BankAccount) {
	cash, _ := bankaccount.NewBankAccount("profile-1", "Clear", bankaccount.AccountTypeChecking, 182.04, "BRL")
	pos, _ := bankaccount.NewBankAccount("profile-1", "SNAG11", bankaccount.AccountTypeInvestment, 0, "BRL")
	_ = pos.SetQuotasFromPrice(120, 9.84)
	_ = pos.SetLinkedAccount(cash)
	repo := &fakeSellAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		pos.ID:  pos,
		cash.ID: cash,
	}}
	return repo, pos, cash
}

func TestSellRoute_FullSell_Returns200AndUpdatesState(t *testing.T) {
	repo, pos, cash := newSellFixture()
	server := sellTestServer(t, repo)

	body := `{"quantity":120,"unitPrice":9.84,"occurredOn":"2026-07-30"}`
	req := httptest.NewRequest("POST", "/api/v1/bank-accounts/"+pos.ID+"/sell", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Account     *bankaccount.BankAccount `json:"account"`
			Transaction *transaction.Transaction `json:"transaction"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Data.Account == nil || resp.Data.Account.NumberOfQuotas == nil || *resp.Data.Account.NumberOfQuotas != 0 {
		t.Errorf("position quotas not zeroed: %+v", resp.Data.Account)
	}
	if resp.Data.Transaction == nil || resp.Data.Transaction.Type != transaction.TypeTransfer {
		t.Fatalf("expected a TRANSFER transaction, got %+v", resp.Data.Transaction)
	}
	if math.Abs(resp.Data.Transaction.Amount-1180.80) > 1e-6 {
		t.Errorf("tx amount = %.2f, want 1180.80", resp.Data.Transaction.Amount)
	}
	// Cash account credited with the proceeds.
	if math.Abs(cash.CurrentBalance-(182.04+1180.80)) > 1e-6 {
		t.Errorf("cash balance = %.2f, want %.2f", cash.CurrentBalance, 182.04+1180.80)
	}
}

func TestSellRoute_UnknownAccount_Returns404(t *testing.T) {
	repo, _, _ := newSellFixture()
	server := sellTestServer(t, repo)

	body := `{"quantity":1,"unitPrice":9.84,"occurredOn":"2026-07-30"}`
	req := httptest.NewRequest("POST", "/api/v1/bank-accounts/does-not-exist/sell", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestSellRoute_Oversell_Returns400(t *testing.T) {
	repo, pos, _ := newSellFixture()
	server := sellTestServer(t, repo)

	body := `{"quantity":999,"unitPrice":9.84,"occurredOn":"2026-07-30"}`
	req := httptest.NewRequest("POST", "/api/v1/bank-accounts/"+pos.ID+"/sell", strings.NewReader(body))
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func (r *FakeTransactionRepository) DeleteMany(ids []string) error {
	for _, id := range ids {
		if err := r.Delete(id); err != nil {
			return err
		}
	}
	return nil
}
