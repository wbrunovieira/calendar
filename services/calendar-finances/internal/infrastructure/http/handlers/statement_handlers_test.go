package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

type importAccountRepo struct {
	accounts map[string]*bankaccount.BankAccount
}

func (r *importAccountRepo) Create(*bankaccount.BankAccount) error { return nil }
func (r *importAccountRepo) FindByID(string) (*bankaccount.BankAccount, error) {
	return nil, nil
}
func (r *importAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (r *importAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) { return nil, nil }
func (r *importAccountRepo) Update(*bankaccount.BankAccount) error        { return nil }
func (r *importAccountRepo) Delete(string) error                          { return nil }
func (r *importAccountRepo) UpdateDisplayOrders([]bankaccount.DisplayOrderUpdate) error {
	return nil
}
func (r *importAccountRepo) FindByProviderAccountID(id string) (*bankaccount.BankAccount, error) {
	return r.accounts[id], nil
}

type recordingStatementRepo struct{ lines []*statement.Line }

func (r *recordingStatementRepo) UpsertMany(lines []*statement.Line) (int, int, error) {
	r.lines = append(r.lines, lines...)
	return len(lines), 0, nil
}
func (r *recordingStatementRepo) FindByExternalID(string, statement.Provider, string) (*statement.Line, error) {
	return nil, nil
}
func (r *recordingStatementRepo) List(statement.ListFilter) ([]*statement.Line, error) {
	return r.lines, nil
}
func (r *recordingStatementRepo) Update(*statement.Line) error { return nil }

func importHandler(accounts *importAccountRepo, lines *recordingStatementRepo) *StatementHandlers {
	return NewStatementHandlers(accounts, usecases.NewImportStatementUseCase(lines))
}

func postImport(t *testing.T, h *StatementHandlers, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.Import(rec, httptest.NewRequest(http.MethodPost, "/api/v1/statements/import", strings.NewReader(body)))
	return rec
}

// The caller says which account at the BANK the statement came from, and this service
// resolves it. Letting the caller name an account here would put one account's
// spending on another's statement with nothing to catch it.
func TestStatementImport_ResolvesTheAccountFromTheProvidersID(t *testing.T) {
	accounts := &importAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"prov-card": {ID: "card-1", Name: "Cartão MP", Type: bankaccount.AccountTypeCreditCard, Currency: "BRL"},
	}}
	lines := &recordingStatementRepo{}

	rec := postImport(t, importHandler(accounts, lines), `{
		"providerAccountId":"prov-card",
		"payload":{"results":[{"id":"x1","date":"2026-09-05T12:00:00.000Z",
			"description":"Compra","amount":"40.00","currencyCode":"BRL","type":"DEBIT","status":"POSTED"}]}}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body.String())
	}
	if len(lines.lines) != 1 || lines.lines[0].AccountID != "card-1" {
		t.Fatalf("the line landed on the wrong account: %+v", lines.lines)
	}
	// A card: the purchase leaves money, whatever the statement's own convention.
	if lines.lines[0].AmountMinor != -4000 {
		t.Errorf("expected -4000, got %d", lines.lines[0].AmountMinor)
	}
}

// An unmapped provider account is answered with a 404 that names it, so the fix is
// obvious. Importing into a guessed account is the one outcome worse than not
// importing at all.
func TestStatementImport_AnUnmappedAccountIsRefusedByName(t *testing.T) {
	rec := postImport(t, importHandler(&importAccountRepo{accounts: map[string]*bankaccount.BankAccount{}}, &recordingStatementRepo{}),
		`{"providerAccountId":"desconhecida-123","payload":{"results":[]}}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "desconhecida-123") {
		t.Errorf("the message must name the account nobody mapped: %q", rec.Body.String())
	}
}

// A checking account reads the same DEBIT the other way round, and getting this wrong
// inverts every card-payment reconciliation.
func TestStatementImport_AChequingAccountKeepsTheProvidersSign(t *testing.T) {
	accounts := &importAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"prov-conta": {ID: "conta-1", Name: "Nubank", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
	}}
	lines := &recordingStatementRepo{}

	postImport(t, importHandler(accounts, lines), `{
		"providerAccountId":"prov-conta",
		"payload":{"results":[{"id":"y1","date":"2026-09-05T12:00:00.000Z",
			"description":"Pagamento de fatura","amount":"-60.00","currencyCode":"BRL","type":"DEBIT","status":"POSTED"}]}}`)

	if len(lines.lines) != 1 {
		t.Fatalf("expected one line, got %d", len(lines.lines))
	}
	if lines.lines[0].AmountMinor != -6000 {
		t.Errorf("a checking statement already speaks in balance: expected -6000, got %d", lines.lines[0].AmountMinor)
	}
}

// A rejected line is reported in the response, so whoever runs the cron sees it.
func TestStatementImport_RejectionsComeBackInTheResponse(t *testing.T) {
	accounts := &importAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"prov-card": {ID: "card-1", Type: bankaccount.AccountTypeCreditCard, Currency: "BRL"},
	}}

	rec := postImport(t, importHandler(accounts, &recordingStatementRepo{}), `{
		"providerAccountId":"prov-card",
		"payload":{"results":[{"id":"","date":"2026-09-05T12:00:00.000Z",
			"description":"Sem id","amount":"10.00","currencyCode":"BRL","type":"DEBIT","status":"POSTED"}]}}`)

	var body struct {
		Data struct {
			Inserted int `json:"inserted"`
			Rejected []struct {
				Reason string `json:"reason"`
			} `json:"rejected"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Inserted != 0 || len(body.Data.Rejected) != 1 {
		t.Fatalf("the rejection must be visible: %s", rec.Body.String())
	}
	if body.Data.Rejected[0].Reason == "" {
		t.Error("a rejection with no reason is not a report")
	}
}

// The uniqueness key includes the provider, so a typo does not fail — it opens a
// second namespace and imports the whole statement again.
func TestStatementImport_AnUnknownProviderIsRefused(t *testing.T) {
	accounts := &importAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"prov-card": {ID: "card-1", Type: bankaccount.AccountTypeCreditCard, Currency: "BRL"},
	}}
	rec := postImport(t, importHandler(accounts, &recordingStatementRepo{}),
		`{"providerAccountId":"prov-card","provider":"pluggi","payload":{"results":[]}}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

// A quiet day is not a failure. The cron pulls every morning.
func TestStatementImport_AnEmptyWindowSucceeds(t *testing.T) {
	accounts := &importAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"prov-card": {ID: "card-1", Type: bankaccount.AccountTypeCreditCard, Currency: "BRL"},
	}}
	rec := postImport(t, importHandler(accounts, &recordingStatementRepo{}),
		`{"providerAccountId":"prov-card","payload":{"results":[]}}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body.String())
	}
}
