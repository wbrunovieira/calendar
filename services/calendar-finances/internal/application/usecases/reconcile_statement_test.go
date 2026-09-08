package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type fakeMatchRepo struct {
	matches []*statement.Match
}

func (f *fakeMatchRepo) Create(m *statement.Match) error {
	f.matches = append(f.matches, m)
	return nil
}
func (f *fakeMatchRepo) ByLine(lineID string) ([]*statement.Match, error) {
	out := []*statement.Match{}
	for _, m := range f.matches {
		if m.LineID == lineID && m.UnmatchedAt == nil {
			out = append(out, m)
		}
	}
	return out, nil
}
func (f *fakeMatchRepo) ByTransaction(txID string) ([]*statement.Match, error) {
	out := []*statement.Match{}
	for _, m := range f.matches {
		if m.TransactionID == txID && m.UnmatchedAt == nil {
			out = append(out, m)
		}
	}
	return out, nil
}
func (f *fakeMatchRepo) Unmatch(id, reason string) error { return nil }

func statementLine(t *testing.T, id, accountID string, minor int64, day int) *statement.Line {
	t.Helper()
	line, err := statement.New(statement.CreateParams{
		AccountID: accountID, Provider: statement.ProviderPluggy, ExternalID: id,
		BookedDate:  time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC),
		AmountMinor: minor, Currency: "BRL", AccountCurrency: "BRL",
		Description: "linha " + id,
	})
	if err != nil {
		t.Fatalf("build line: %v", err)
	}
	line.ID = "line-" + id
	return line
}

func systemCharge(id, accountID string, amount float64, day int) *transaction.Transaction {
	return &transaction.Transaction{
		ID: id, ProfileID: "p1", BankAccountID: accountID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: amount, Currency: "BRL", Description: "lançamento " + id,
		OccurredOn: time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC),
	}
}

func reconcileFixture(t *testing.T) (*fakeStatementRepo, *fakeTransactionRepo, *fakeMatchRepo, *fakeAccountRepo) {
	t.Helper()
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc": {ID: "acc", ProfileID: "p1", Name: "Conta", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
	}}
	return &fakeStatementRepo{}, &fakeTransactionRepo{}, &fakeMatchRepo{}, accounts
}

// One line, one entry of the same value on the same account, a couple of days apart:
// that is the everyday case, and matching it is what stops a reconciliation being done
// by hand.
func TestReconcile_OneCandidateIsMatched(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 6)}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Matched != 1 {
		t.Fatalf("expected one match, got %d (%+v)", out.Matched, out)
	}
	if len(matches.matches) != 1 || matches.matches[0].TransactionID != "tx1" {
		t.Fatalf("matched the wrong thing: %+v", matches.matches)
	}
	if matches.matches[0].Method != statement.MethodDeterministic {
		t.Errorf("a match on account, amount and date is deterministic, got %s", matches.matches[0].Method)
	}
}

// Nothing in the system for a line the bank charged: that is money missing, and it is
// the whole point. It must be REPORTED, never invented.
func TestReconcile_ALineWithNoCandidateIsReportedAsMissing(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -5390, 5)}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")

	if out.Matched != 0 || len(out.Missing) != 1 {
		t.Fatalf("the bank charged something the system does not have: %+v", out)
	}
	if out.Missing[0].AmountMinor != -5390 {
		t.Errorf("got %d", out.Missing[0].AmountMinor)
	}
	if len(matches.matches) != 0 {
		t.Error("nothing may be matched when there is nothing to match to")
	}
}

// Three charges of R$40 in the same month are indistinguishable by value. Picking one
// is exactly how a legitimate Cloudflare charge was once deleted as a phantom.
func TestReconcile_AmbiguityIsNeverGuessed(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{
		systemCharge("tx1", "acc", 40, 5),
		systemCharge("tx2", "acc", 40, 6),
	}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")

	if out.Matched != 0 {
		t.Fatal("two equally good candidates must not be resolved by picking one")
	}
	if len(out.Ambiguous) != 1 || len(out.Ambiguous[0].CandidateIDs) != 2 {
		t.Fatalf("the ambiguity must be reported with its candidates: %+v", out.Ambiguous)
	}
}

// A date one or two days apart is the norm, not a discrepancy: the bank prints the
// purchase date and the entry often carries the posting date.
func TestReconcile_ADayOrTwoApartStillMatches(t *testing.T) {
	for _, day := range []int{3, 4, 5, 6, 7} {
		lines, txns, matches, accounts := reconcileFixture(t)
		lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
		txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, day)}

		out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
		if out.Matched != 1 {
			t.Errorf("day %d: expected a match, got %+v", day, out)
		}
	}
}

// Far enough apart and it is a different movement that happens to cost the same.
func TestReconcile_TooFarApartIsNotTheSameMovement(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 20)}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if out.Matched != 0 || len(out.Missing) != 1 {
		t.Fatalf("fifteen days apart is not the same charge: %+v", out)
	}
}

// A pending authorisation still changes its amount and date when it settles. Matching
// it asserts a check the next sync invalidates.
func TestReconcile_APendingLineIsLeftAlone(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	line := statementLine(t, "a", "acc", -4000, 5)
	line.ProviderStatus = statement.ProviderStatusPending
	lines.lines = []*statement.Line{line}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if out.Matched != 0 {
		t.Error("a pending line must wait until the bank settles it")
	}
	if len(out.Missing) != 0 {
		t.Error("and it is not missing either — it is simply not ready")
	}
}

// An entry already matched to another line must not be claimed twice: two bank lines
// of the same value would otherwise both point at one entry, and the second charge
// would silently look accounted for.
func TestReconcile_AnEntryIsNotMatchedTwice(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{
		statementLine(t, "a", "acc", -4000, 5),
		statementLine(t, "b", "acc", -4000, 6),
	}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")

	if out.Matched != 1 {
		t.Fatalf("only one line can claim the single entry: %+v", out)
	}
	if len(out.Missing) != 1 {
		t.Fatalf("the other is a charge the system does not have: %+v", out)
	}
}

// Running it again changes nothing: the cron reconciles every morning over an
// overlapping window.
func TestReconcile_RunningTwiceMatchesOnce(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	uc.Execute("acc")
	out, _ := uc.Execute("acc")

	if out.Matched != 0 {
		t.Errorf("the second run has nothing left to match, got %d", out.Matched)
	}
	if len(matches.matches) != 1 {
		t.Errorf("and must not record the same match twice: %d", len(matches.matches))
	}
}

func cardFixture(t *testing.T) (*fakeStatementRepo, *fakeTransactionRepo, *fakeMatchRepo, *fakeAccountRepo) {
	t.Helper()
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": {ID: "card", ProfileID: "p1", Name: "Cartao", Type: bankaccount.AccountTypeCreditCard, Currency: "BRL"},
	}}
	return &fakeStatementRepo{}, &fakeTransactionRepo{}, &fakeMatchRepo{}, accounts
}

// A card statement speaks in debt and the ledger speaks in money. A purchase leaves,
// whatever the statement's own convention — and getting this backwards would match
// every purchase against nothing and every payment against the wrong thing.
func TestReconcile_APurchaseOnACardIsMoneyLeaving(t *testing.T) {
	lines, txns, matches, accounts := cardFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "card", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "card", 40, 5)}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("card")
	if out.Matched != 1 {
		t.Fatalf("expected the purchase to match: %+v", out)
	}
}

// The payment is the one movement that appears on two statements at once, so it is the
// one that most needs its sign right. On the card it ARRIVES: it pays the debt down.
func TestReconcile_APaymentOnACardIsMoneyArriving(t *testing.T) {
	lines, txns, matches, accounts := cardFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "pg", "card", 101818, 3)}

	payment := &transaction.Transaction{
		ID: "pay1", ProfileID: "p1", BankAccountID: "checking",
		DestinationAccountID: strPtr("card"),
		Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 1018.18, Currency: "BRL", Description: "Pagamento fatura",
		OccurredOn: time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
	}
	txns.created = []*transaction.Transaction{payment}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("card")
	if out.Matched != 1 {
		t.Fatalf("the payment leg must match the card's credit line: %+v", out)
	}
}

// Cents are compared as integers, never as floats: 0.1 + 0.2 is the reason a ledger
// uses minor units, and a tolerance on value is how a real difference becomes
// "acceptable rounding" and vanishes.
func TestReconcile_CentsAreComparedExactly(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -5558, 5)}
	txns.created = []*transaction.Transaction{
		systemCharge("quase", "acc", 55.57, 5),
		systemCharge("exato", "acc", 55.58, 5),
	}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if out.Matched != 1 {
		t.Fatalf("exactly one entry has this value: %+v", out)
	}
	if matches.matches[0].TransactionID != "exato" {
		t.Errorf("matched %s — one cent apart is a different charge", matches.matches[0].TransactionID)
	}
}
