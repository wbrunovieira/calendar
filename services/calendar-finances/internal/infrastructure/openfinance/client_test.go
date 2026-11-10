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
		// The REAL envelope, as the provider sends it and as the skill documents
		// it: {"ok":true,"tool":"...","result":{...,"results":[...]}}. The first
		// version of this fake invented a bare {"results":[...]}, so the client
		// read zero accounts from production with a nil error and every test here
		// passed.
		w.Write([]byte(`{"ok":true,"tool":"openfinance_list_accounts","result":{"total":2,"results":[
			{"id":"a1","type":"BANK","name":"Mercado Pago","balance":"718.36"},
			{"id":"a2","type":"CREDIT","name":"MP","balance":"0","creditData":{"creditLimit":"4400.00","availableCreditLimit":"162.13"}}
		]}}`))
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
func TestFiguresByProviderIDLeavesOutWhatItCannotRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"result":{"results":[
			{"id":"bom","type":"BANK","balance":"718.36"},
			{"id":"sem-saldo","type":"BANK","balance":""}
		]}}`))
	}))
	defer srv.Close()
	c := NewClient("chave")
	c.baseURL = srv.URL

	owed, kinds, err := c.FiguresByProviderID()

	if err != nil {
		t.Fatalf("FiguresByProviderID: %v", err)
	}
	if _, present := owed["sem-saldo"]; present {
		t.Error("an unreadable account was entered as a number")
	}
	if v := owed["bom"]; v < 718.35 || v > 718.37 {
		t.Errorf("owed = %v, want 718.36", v)
	}
	// The kind survives even for the account with no readable figure: pointing a
	// card at a checking account is worth naming either way.
	if kinds["sem-saldo"] != "BANK" || kinds["bom"] != "BANK" {
		t.Errorf("kinds = %v, want both BANK", kinds)
	}
}

// What the provider says each account IS has to come back, or the cross-check
// that catches a transposed mapping has nothing to compare against.
func TestFiguresByProviderIDReportsWhatEachAccountIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"result":{"results":[
			{"id":"conta","type":"BANK","balance":"0.51"},
			{"id":"cartao","type":"CREDIT","balance":"0","creditData":{"creditLimit":"1000.00","availableCreditLimit":"634.50"}}
		]}}`))
	}))
	defer srv.Close()
	c := NewClient("chave")
	c.baseURL = srv.URL

	owed, kinds, err := c.FiguresByProviderID()

	if err != nil {
		t.Fatalf("FiguresByProviderID: %v", err)
	}
	if kinds["conta"] != "BANK" || kinds["cartao"] != "CREDIT" {
		t.Errorf("kinds = %v, want conta BANK and cartao CREDIT", kinds)
	}
	if v := owed["cartao"]; v < 365.49 || v > 365.51 {
		t.Errorf("card owed = %v, want 365.50", v)
	}
}

// A bare payload still works, so a provider that stops wrapping does not empty
// this out silently.
func TestListAccountsAlsoAcceptsAnUnwrappedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"id":"a1","type":"BANK","balance":"10.00"}]}`))
	}))
	defer srv.Close()
	c := NewClient("chave")
	c.baseURL = srv.URL

	accounts, err := c.ListAccounts()

	if err != nil || len(accounts) != 1 {
		t.Fatalf("accounts=%d err=%v, want 1 account", len(accounts), err)
	}
}

// An answer carrying no accounts is an ERROR, never an empty list. An empty list
// reconciles clean against everything, so a changed envelope would be
// indistinguishable from a healthy bank — which is what shipped first.
func TestListAccountsRefusesAnAnswerWithNoAccounts(t *testing.T) {
	for _, payload := range []string{
		`{"ok":true,"tool":"x","result":{"total":0,"results":[]}}`,
		`{"ok":true,"tool":"x","result":{}}`,
		`{"algo":"inesperado"}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(payload))
		}))
		c := NewClient("chave")
		c.baseURL = srv.URL

		_, err := c.ListAccounts()

		if err == nil {
			t.Errorf("payload %s was accepted as a healthy empty answer", payload)
		}
		srv.Close()
	}
}

// Sscanf accepted a prefix and read comma decimals a thousandfold wrong, and
// both still reconcile against something.
func TestParseAmountRefusesAnythingItCannotReadWhole(t *testing.T) {
	for _, bad := range []string{"718.36abc", "1.234,56", "718,36", "1,234.56", "R$ 718,36", "abc", "", "NaN", "Inf", "-Inf"} {
		if v, ok := parseAmount(bad); ok {
			t.Errorf("parseAmount(%q) = %v, true — want a refusal", bad, v)
		}
	}
}

func TestParseAmountReadsWhatTheProviderActuallySends(t *testing.T) {
	for in, want := range map[string]float64{"718.36": 718.36, "4400.00": 4400, "0": 0, "-12.5": -12.5, " 162.13 ": 162.13} {
		v, ok := parseAmount(in)
		if !ok || v < want-0.001 || v > want+0.001 {
			t.Errorf("parseAmount(%q) = %v, %v; want %v", in, v, ok, want)
		}
	}
}
