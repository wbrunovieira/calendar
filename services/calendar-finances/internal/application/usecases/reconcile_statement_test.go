package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type fakeMatchRepo struct {
	matches []*statement.Match
	// stmt is how the fake resolves a match back to its account, the way the real
	// query joins bank_statement_lines. Held as a pointer so it sees lines added
	// after construction.
	stmt *fakeStatementRepo
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

// Faithful to the real query: scoped to the account, via the line the match points at.
// A global answer here is what let one transfer's two statement sides fight over the
// single row that represents it.
func (f *fakeMatchRepo) ClaimedOnAccount(txID, accountID string) (bool, error) {
	for _, m := range f.matches {
		if m.TransactionID != txID || m.UnmatchedAt != nil {
			continue
		}
		for _, l := range f.stmt.lines {
			if l.ID == m.LineID && l.AccountID == accountID {
				return true, nil
			}
		}
	}
	return false, nil
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
	lines := &fakeStatementRepo{}
	return lines, &fakeTransactionRepo{}, &fakeMatchRepo{stmt: lines}, accounts
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
	lines := &fakeStatementRepo{}
	return lines, &fakeTransactionRepo{}, &fakeMatchRepo{stmt: lines}, accounts
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

// A planned entry the bank has now paid is not a match — it is a nudge.
//
// Matching it would assert that a forecast is a fact. Ignoring it would leave the
// money looking missing while the entry sits there waiting. Naming it separately is
// the only honest answer, and it is what closes the loop for entries posted ahead of
// time: the CRM records an expected payment as PLANNED, the bank says it arrived, and
// this is the signal to confirm it.
func TestReconcile_APlannedEntryThatArrivedIsReadyToConfirm(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", 125000, 5)}

	expected := &transaction.Transaction{
		ID: "tx1", ProfileID: "p1", BankAccountID: "acc",
		Type: transaction.TypeIncome, Status: transaction.StatusPlanned,
		Amount: 1250, Currency: "BRL", Description: "Gomez Studio - parcela 1/2",
		OccurredOn: time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC),
	}
	txns.created = []*transaction.Transaction{expected}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Matched != 0 {
		t.Error("a forecast is not a fact; it must not be matched")
	}
	if len(out.Missing) != 0 {
		t.Error("and the money is not missing either — the entry is right there")
	}
	if len(out.ReadyToConfirm) != 1 {
		t.Fatalf("it must be named as ready to confirm: %+v", out)
	}
	if out.ReadyToConfirm[0].TransactionID != "tx1" {
		t.Errorf("pointed at %s", out.ReadyToConfirm[0].TransactionID)
	}
	if out.ReadyToConfirm[0].AmountMinor != 125000 {
		t.Errorf("got %d", out.ReadyToConfirm[0].AmountMinor)
	}
}

// A confirmed entry beats a planned one for the same line: the money already moved and
// the forecast is stale. Proposing both would ask a human to choose between a fact and
// a guess.
func TestReconcile_AConfirmedEntryWinsOverAPlannedOne(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", 125000, 5)}
	txns.created = []*transaction.Transaction{
		{ID: "planejado", ProfileID: "p1", BankAccountID: "acc", Type: transaction.TypeIncome,
			Status: transaction.StatusPlanned, Amount: 1250, Currency: "BRL", Description: "previsto",
			OccurredOn: time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)},
		{ID: "real", ProfileID: "p1", BankAccountID: "acc", Type: transaction.TypeIncome,
			Status: transaction.StatusConfirmed, Amount: 1250, Currency: "BRL", Description: "recebido",
			OccurredOn: time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)},
	}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")

	if out.Matched != 1 {
		t.Fatalf("the confirmed entry is the match: %+v", out)
	}
	if matches.matches[0].TransactionID != "real" {
		t.Errorf("matched %s", matches.matches[0].TransactionID)
	}
	if len(out.ReadyToConfirm) != 0 {
		t.Error("the stale forecast must not also be proposed")
	}
}

type failingMatchRepo struct {
	fakeMatchRepo
	failByTransaction bool
	failByLine        bool
}

func (f *failingMatchRepo) ClaimedOnAccount(id, accountID string) (bool, error) {
	if f.failByTransaction {
		return false, errors.New("database unavailable")
	}
	return f.fakeMatchRepo.ClaimedOnAccount(id, accountID)
}

func (f *failingMatchRepo) ByLine(id string) ([]*statement.Match, error) {
	if f.failByLine {
		return nil, errors.New("database unavailable")
	}
	return f.fakeMatchRepo.ByLine(id)
}

// A database that cannot answer "is this entry already spoken for?" must stop the run,
// not answer no.
//
// The unique index guards the PAIR (line, transaction), so it catches one line claiming
// one entry twice — and does nothing about two different lines claiming the same entry.
// That second case is the one that matters: the second bank charge would silently look
// accounted for, which is the failure this whole design exists to prevent. Swallowing
// the error made a database hiccup produce exactly it.
func TestReconcile_ADatabaseFailureStopsTheRunInsteadOfGuessing(t *testing.T) {
	lines, txns, _, accounts := reconcileFixture(t)
	matches := &failingMatchRepo{failByTransaction: true}
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	_, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err == nil {
		t.Fatal("a reconciler that cannot check must refuse to match, not match anyway")
	}
	if len(matches.matches) != 0 {
		t.Error("and must record nothing")
	}
}

func TestReconcile_AFailureReadingTheLinesMatchesAlsoStops(t *testing.T) {
	lines, txns, _, accounts := reconcileFixture(t)
	matches := &failingMatchRepo{failByLine: true}
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	if _, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc"); err == nil {
		t.Fatal("not knowing whether the line was already reconciled is not the same as it not being")
	}
}

// The payment of a card bill must reconcile on BOTH statements.
//
// It is one row: the checking account is the origin and the card is the destination,
// so it appears on the bank's statement for each. Claiming it globally means whichever
// account is reconciled first wins and the other reports money missing that is not —
// the exact phantom this code exists to prevent, and order-dependent, so the same data
// gives different answers depending on the order of the calls.
func TestReconcile_APaymentReconcilesOnBothStatements(t *testing.T) {
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"corrente": {ID: "corrente", ProfileID: "p1", Name: "Conta", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
		"cartao":   {ID: "cartao", ProfileID: "p1", Name: "Cartao", Type: bankaccount.AccountTypeCreditCard, Currency: "BRL"},
	}}
	lines := &fakeStatementRepo{}
	txns := &fakeTransactionRepo{}
	matches := &fakeMatchRepo{stmt: lines}

	// The bank shows it on both: leaving the checking account, arriving on the card.
	saida := statementLine(t, "saida", "corrente", -101818, 3)
	entrada := statementLine(t, "entrada", "cartao", 101818, 3)
	lines.lines = []*statement.Line{saida, entrada}

	txns.created = []*transaction.Transaction{{
		ID: "pay1", ProfileID: "p1", BankAccountID: "corrente",
		DestinationAccountID: strPtr("cartao"),
		Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 1018.18, Currency: "BRL", Description: "Pagamento fatura",
		OccurredOn: time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
	}}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)

	corrente, err := uc.Execute("corrente")
	if err != nil {
		t.Fatalf("corrente: %v", err)
	}
	cartao, err := uc.Execute("cartao")
	if err != nil {
		t.Fatalf("cartao: %v", err)
	}

	if corrente.Matched != 1 || len(corrente.Missing) != 0 {
		t.Errorf("checking side: %+v", corrente)
	}
	if cartao.Matched != 1 || len(cartao.Missing) != 0 {
		t.Fatalf("card side reported money missing that is not: %+v", cartao)
	}
}

// The same entry must still not be claimed twice on the SAME account: two lines of the
// same value there are two charges, and letting both point at one entry makes the
// second silently look accounted for.
func TestReconcile_TheSameEntryIsStillClaimedOncePerAccount(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{
		statementLine(t, "a", "acc", -4000, 5),
		statementLine(t, "b", "acc", -4000, 6),
	}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	uc.Execute("acc")
	out, _ := uc.Execute("acc")

	if out.Matched != 0 {
		t.Errorf("the second run has nothing to match: %+v", out)
	}
	if len(matches.matches) != 1 {
		t.Errorf("the entry was claimed %d times", len(matches.matches))
	}
}

// One forecast cannot settle two real charges. Proposing it for both says "just
// confirm this" twice, and confirming once makes the other charge vanish from the
// report — the hole then appears a day later with nothing pointing at why.
func TestReconcile_AForecastIsProposedForOneLineOnly(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{
		statementLine(t, "a", "acc", -50000, 5),
		statementLine(t, "b", "acc", -50000, 6),
	}
	txns.created = []*transaction.Transaction{{
		ID: "previsto", ProfileID: "p1", BankAccountID: "acc",
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 500, Currency: "BRL", Description: "previsto",
		OccurredOn: time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC),
	}}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")

	if len(out.ReadyToConfirm) != 1 {
		t.Fatalf("one forecast settles one charge: %+v", out.ReadyToConfirm)
	}
	if len(out.Missing) != 1 {
		t.Fatalf("and the other charge is genuinely missing: %+v", out.Missing)
	}
}

// A matched line must SAY it is matched, in storage. Leaving every line UNMATCHED
// forever means each run re-examines the whole history, and anything reading the
// line's own status — a listing, a screen, a later query — is told the opposite of
// what happened.
func TestReconcile_AMatchedLineIsSavedAsMatched(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 6)}

	if _, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(lines.updates) != 1 {
		t.Fatalf("the line's status was never persisted: %d updates", len(lines.updates))
	}
	if lines.updates[0].Status != statement.StatusMatched {
		t.Errorf("saved as %s, want MATCHED", lines.updates[0].Status)
	}
}

// If the match is recorded and saving the line's status fails, the run must say so.
// Reporting success there leaves the two halves disagreeing with nobody told.
func TestReconcile_AFailureToSaveTheLineStatusIsReported(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 6)}
	lines.updateErr = errors.New("database unavailable")

	_, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err == nil {
		t.Fatal("the line status could not be saved and the run reported success")
	}
}

// A database that blinked is not a bad request. The morning cron reads the status code
// to decide whether to retry, so the failure has to arrive labelled as this service's
// problem and not the caller's.
func TestReconcile_AFailedReadIsLabelledAsStorage(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.listErr = errors.New("connection reset")

	_, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if !errors.Is(err, ErrReconcileStorage) {
		t.Fatalf("got %v, want it wrapped in ErrReconcileStorage", err)
	}
}
