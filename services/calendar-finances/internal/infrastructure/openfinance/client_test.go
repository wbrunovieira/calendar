package openfinance

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The figure to trust on a card is limit minus available. `usedAmount` is not
// filled by every issuer, and the account `balance` means different things per
// connector — some report the open bill, others the whole revolving debt — so
// comparing against it would manufacture a difference every day.
func TestOwedOnACardIsLimitMinusAvailable(t *testing.T) {
	a := Account{Type: "CREDIT", Balance: "4237.87"}
	a.Credit = &struct {
		CreditLimit          string `json:"creditLimit"`
		AvailableCreditLimit string `json:"availableCreditLimit"`
		UsedAmount           string `json:"usedAmount"`
	}{CreditLimit: "4400.00", AvailableCreditLimit: "162.13", UsedAmount: ""}

	owed, ok := a.Owed()

	if !ok {
		t.Fatal("expected the card to report what it owes")
	}
	if owed < 4237.86 || owed > 4237.88 {
		t.Errorf("owed = %v, want 4237.87", owed)
	}
}

// Some issuers fill usedAmount and not the pair; that is the fallback.
func TestOwedFallsBackToUsedAmount(t *testing.T) {
	a := Account{Type: "CREDIT"}
	a.Credit = &struct {
		CreditLimit          string `json:"creditLimit"`
		AvailableCreditLimit string `json:"availableCreditLimit"`
		UsedAmount           string `json:"usedAmount"`
	}{UsedAmount: "3269.80"}

	owed, ok := a.Owed()

	if !ok || owed < 3269.79 || owed > 3269.81 {
		t.Errorf("owed = %v ok=%v, want 3269.80", owed, ok)
	}
}

// A card that reports neither must say so rather than answer zero: zero reads as
// "this card owes nothing", which is a lie that reconciles.
func TestOwedRefusesToGuessOnACardWithNoFigures(t *testing.T) {
	a := Account{Type: "CREDIT", Balance: "999.00"}
	a.Credit = &struct {
		CreditLimit          string `json:"creditLimit"`
		AvailableCreditLimit string `json:"availableCreditLimit"`
		UsedAmount           string `json:"usedAmount"`
	}{}

	if _, ok := a.Owed(); ok {
		t.Error("a card with no limit figures reported a number anyway")
	}
}

func TestOwedOnACheckingAccountIsItsBalance(t *testing.T) {
	owed, ok := Account{Type: "BANK", Balance: "718.36"}.Owed()

	if !ok || owed < 718.35 || owed > 718.37 {
		t.Errorf("owed = %v ok=%v, want 718.36", owed, ok)
	}
}

func TestOwedRefusesAnEmptyBalance(t *testing.T) {
	if _, ok := (Account{Type: "BANK", Balance: ""}).Owed(); ok {
		t.Error("an empty balance was read as a number")
	}
}

func TestListAccountsParsesTheProviderPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer chave" {
			t.Errorf("Authorization = %q", got)
		}
		w.Write([]byte(`{"results":[
			{"id":"a1","type":"BANK","name":"Mercado Pago","balance":"718.36"},
			{"id":"a2","type":"CREDIT","name":"MP","balance":"0","creditData":{"creditLimit":"4400.00","availableCreditLimit":"162.13"}}
		]}`))
	}))
	defer srv.Close()
	c := NewClient("chave")
	c.baseURL = srv.URL

	accounts, err := c.ListAccounts()

	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("got %d accounts, want 2", len(accounts))
	}
	if owed, _ := accounts[1].Owed(); owed < 4237.86 || owed > 4237.88 {
		t.Errorf("card owed = %v, want 4237.87", owed)
	}
}

// The provider's own reason — a stale consent, a rate limit — has to reach the
// caller. A daily job that swallows it leaves nobody able to act on it.
func TestListAccountsCarriesTheProvidersReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":{"code":"not_connected","message":"reconnect required"}}`))
	}))
	defer srv.Close()
	c := NewClient("chave")
	c.baseURL = srv.URL

	_, err := c.ListAccounts()

	if err == nil {
		t.Fatal("expected an error")
	}
	if !contains(err.Error(), "not_connected") {
		t.Errorf("error = %q, want it to carry the provider's reason", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// An account with no readable figure must be ABSENT from the map, not zero.
// Zero would compare as "the bank says this account owes nothing", which
// reconciles against an empty ledger and hides a broken connector.
func TestOwedByProviderIDLeavesOutWhatItCannotRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[
			{"id":"bom","type":"BANK","balance":"718.36"},
			{"id":"sem-saldo","type":"BANK","balance":""}
		]}`))
	}))
	defer srv.Close()
	c := NewClient("chave")
	c.baseURL = srv.URL

	owed, err := c.OwedByProviderID()

	if err != nil {
		t.Fatalf("OwedByProviderID: %v", err)
	}
	if _, present := owed["sem-saldo"]; present {
		t.Error("an unreadable account was entered as a number")
	}
	if v := owed["bom"]; v < 718.35 || v > 718.37 {
		t.Errorf("owed = %v, want 718.36", v)
	}
}
