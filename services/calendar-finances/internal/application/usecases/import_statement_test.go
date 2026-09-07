package usecases

import (
	"os"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

type fakeStatementRepo struct {
	lines    []*statement.Line
	inserted int
	updated  int
	err      error
}

func (f *fakeStatementRepo) UpsertMany(lines []*statement.Line) (int, int, error) {
	if f.err != nil {
		return 0, 0, f.err
	}
	for _, l := range lines {
		known := false
		for _, existing := range f.lines {
			if existing.ExternalID == l.ExternalID && existing.AccountID == l.AccountID {
				known = true
			}
		}
		if known {
			f.updated++
			continue
		}
		f.lines = append(f.lines, l)
		f.inserted++
	}
	return f.inserted, f.updated, nil
}

func (f *fakeStatementRepo) FindByExternalID(accountID string, provider statement.Provider, externalID string) (*statement.Line, error) {
	for _, l := range f.lines {
		if l.AccountID == accountID && l.ExternalID == externalID {
			return l, nil
		}
	}
	return nil, nil
}
func (f *fakeStatementRepo) List(statement.ListFilter) ([]*statement.Line, error) {
	return f.lines, nil
}
func (f *fakeStatementRepo) Update(*statement.Line) error { return nil }

func pluggyPayload(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../domain/statement/testdata/pluggy_transactions.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return raw
}

func cardImport(repo statement.Repository) *ImportStatementUseCase {
	return NewImportStatementUseCase(repo)
}

func cardInput(payload []byte) ImportStatementInput {
	return ImportStatementInput{
		AccountID:       "card-1",
		Provider:        statement.ProviderPluggy,
		AccountKind:     statement.AccountKindCard,
		AccountCurrency: "BRL",
		Payload:         payload,
	}
}

// The line charged in dollars must be stored by what it cost in reais.
//
// This is the Anthropic bug in its original form: the statement shows 108.00 because
// the card was charged in USD, and reading that as reais turned a R$ 576,03
// subscription into a R$ 108,00 one — then into an invented billing dispute, and a
// support ticket opened for nothing.
func TestImportStatement_AForeignLineIsStoredByWhatItCostInReais(t *testing.T) {
	repo := &fakeStatementRepo{}
	out, err := cardImport(repo).Execute(cardInput(pluggyPayload(t)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Inserted != 4 {
		t.Fatalf("expected the four fixture lines, got %d (%v)", out.Inserted, out.Rejected)
	}

	var anthropic *statement.Line
	for _, l := range repo.lines {
		if l.Description == "Anthropic* Claude Sub" {
			anthropic = l
		}
	}
	if anthropic == nil {
		t.Fatal("the dollar line did not make it in")
	}
	if anthropic.Currency != "USD" {
		t.Errorf("the currency the bank charged in must survive, got %q", anthropic.Currency)
	}
	if anthropic.AmountAccountMinor == nil {
		t.Fatal("a foreign line with no converted value is exactly what must not be stored")
	}
	// -57603: negative because a purchase takes money from the holder, whatever the
	// card statement's own convention says.
	if got := *anthropic.AmountAccountMinor; got != -57603 {
		t.Errorf("expected -57603 centavos in the account's currency, got %d", got)
	}
	if anthropic.AmountMinor != -10800 {
		t.Errorf("the face value stays in its own currency: got %d", anthropic.AmountMinor)
	}
}

// The card statement speaks in debt, the holder thinks in money. A purchase leaves,
// a payment arrives — and the payment is the one movement that shows up on both
// accounts at once, so it is the one that most needs its sign right.
func TestImportStatement_APaymentOnTheCardArrives(t *testing.T) {
	repo := &fakeStatementRepo{}
	if _, err := cardImport(repo).Execute(cardInput(pluggyPayload(t))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, l := range repo.lines {
		if l.Description != "Pagamento recebido" {
			continue
		}
		if l.AmountMinor <= 0 {
			t.Errorf("a payment pays the debt down: money in, got %d", l.AmountMinor)
		}
		return
	}
	t.Fatal("the payment line did not make it in")
}

// Pluggy timestamps are UTC and this system runs in Sao Paulo. A purchase at 21h on
// the 30th is still the 30th here; treating it as the 1st makes it missing from one
// month and extra in the next — the shape of a phantom.
func TestImportStatement_TheDateIsTheOneTheIssuerWouldPrint(t *testing.T) {
	repo := &fakeStatementRepo{}
	payload := []byte(`{"results":[{"id":"late-night","date":"2026-08-31T02:30:00.000Z",
		"description":"Compra tarde","amount":"10.00","currencyCode":"BRL","type":"DEBIT","status":"POSTED"}]}`)

	if _, err := cardImport(repo).Execute(cardInput(payload)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := repo.lines[0].BookedDate
	if got.Day() != 30 || got.Month() != time.August {
		t.Errorf("02:30 UTC on the 31st is 23:30 on the 30th in Sao Paulo, got %s", got.Format("2006-01-02"))
	}
}

// Pulling the same window twice is the normal case, not the exception: the cron runs
// every morning over an overlapping range.
func TestImportStatement_ReimportingTheSameWindowAddsNothing(t *testing.T) {
	repo := &fakeStatementRepo{}
	uc := cardImport(repo)
	payload := pluggyPayload(t)

	first, _ := uc.Execute(cardInput(payload))
	repo.inserted, repo.updated = 0, 0
	second, err := uc.Execute(cardInput(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Inserted != 4 {
		t.Fatalf("first import should bring four, got %d", first.Inserted)
	}
	if second.Inserted != 0 {
		t.Errorf("the second import must add nothing, got %d", second.Inserted)
	}
	if second.Updated != 4 {
		t.Errorf("but it must refresh what the bank now says, got %d", second.Updated)
	}
}

// A line the importer cannot trust is REPORTED, never dropped. Silence here is how a
// charge goes missing for weeks — which is precisely what happened to the R$ 53,90
// YouTube and the R$ 116,03 Tim.
func TestImportStatement_ALineItCannotTrustIsReportedNotSwallowed(t *testing.T) {
	repo := &fakeStatementRepo{}
	payload := []byte(`{"results":[
		{"id":"ok","date":"2026-08-05T12:00:00.000Z","description":"Boa","amount":"10.00","currencyCode":"BRL","type":"DEBIT","status":"POSTED"},
		{"id":"","date":"2026-08-06T12:00:00.000Z","description":"Sem id","amount":"20.00","currencyCode":"BRL","type":"DEBIT","status":"POSTED"},
		{"id":"sem-cambio","date":"2026-08-07T12:00:00.000Z","description":"Dolar sem conversao","amount":"30.00","currencyCode":"USD","type":"DEBIT","status":"POSTED"}
	]}`)

	out, err := cardImport(repo).Execute(cardInput(payload))
	if err != nil {
		t.Fatalf("a bad line must not fail the whole import: %v", err)
	}
	if out.Inserted != 1 {
		t.Errorf("the good line must land, got %d", out.Inserted)
	}
	if len(out.Rejected) != 2 {
		t.Fatalf("both bad lines must be reported, got %v", out.Rejected)
	}
	for _, r := range out.Rejected {
		if r.Reason == "" {
			t.Error("a rejection with no reason is not a report")
		}
	}
}

// What the bank said is kept verbatim. Pluggy rewrites transactions — categories
// change, PENDING becomes POSTED, rows vanish on a reversal — and without the original
// that reads as this system losing data.
func TestImportStatement_TheProviderPayloadIsKeptVerbatim(t *testing.T) {
	repo := &fakeStatementRepo{}
	if _, err := cardImport(repo).Execute(cardInput(pluggyPayload(t))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, l := range repo.lines {
		if len(l.Raw) == 0 {
			t.Fatalf("line %q kept no raw payload", l.Description)
		}
	}
}

// A pending authorisation is not a settled charge, and saying so is what stops every
// normal settlement from reading as an anomaly later.
func TestImportStatement_ThePendingFlagSurvives(t *testing.T) {
	repo := &fakeStatementRepo{}
	if _, err := cardImport(repo).Execute(cardInput(pluggyPayload(t))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, l := range repo.lines {
		if l.Description == "Registrobr" && l.ProviderStatus != statement.ProviderStatusPending {
			t.Errorf("the fixture's PENDING line came in as %q", l.ProviderStatus)
		}
	}
}
