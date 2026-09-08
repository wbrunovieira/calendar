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
	stmt      *fakeStatementRepo
	createErr error
}

// Enforces both unique indexes, because a fake that accepts what the database refuses
// makes the race untestable — and the race is the whole reason the indexes exist.
func (f *fakeMatchRepo) Create(m *statement.Match) error {
	if f.createErr != nil {
		return f.createErr
	}
	for _, existing := range f.matches {
		if existing.UnmatchedAt != nil {
			continue
		}
		if existing.LineID == m.LineID && existing.TransactionID == m.TransactionID {
			return statement.ErrLineAlreadyMatched
		}
		if existing.TransactionID == m.TransactionID &&
			f.accountOf(existing.LineID) == f.accountOf(m.LineID) {
			return statement.ErrAlreadyClaimedOnAccount
		}
	}
	f.matches = append(f.matches, m)
	return nil
}

func (f *fakeMatchRepo) accountOf(lineID string) string {
	if f.stmt == nil {
		return ""
	}
	for _, l := range f.stmt.lines {
		if l.ID == lineID {
			return l.AccountID
		}
	}
	return ""
}

// Returns undone matches too, exactly as the real ByLine does. Filtering them here
// made the caller's own UnmatchedAt check untestable — and an undone match read as a
// live one is a line that never comes back.
func (f *fakeMatchRepo) ByLine(lineID string) ([]*statement.Match, error) {
	out := []*statement.Match{}
	for _, m := range f.matches {
		if m.LineID == lineID {
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
func (f *fakeMatchRepo) Unmatch(id, reason string) error {
	for _, m := range f.matches {
		if m.ID == id {
			return m.Unmatch(reason, time.Now())
		}
	}
	return errors.New("no such match")
}

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

// foreignLine is a card charge the bank printed in its original currency, with the
// converted figure alongside. Reconciling against the face value compares dollars to
// reais: US$ 107,54 against R$ 107,54, off by the exchange rate.
func foreignLine(t *testing.T, id, accountID string, faceMinor, accountMinor int64, day int) *statement.Line {
	t.Helper()
	line, err := statement.New(statement.CreateParams{
		AccountID: accountID, Provider: statement.ProviderPluggy, ExternalID: id,
		BookedDate:         time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC),
		AmountMinor:        faceMinor,
		AmountAccountMinor: &accountMinor,
		Currency:           "USD", AccountCurrency: "BRL",
		Description: "Anthropic* Claude Sub",
	})
	if err != nil {
		t.Fatalf("build line: %v", err)
	}
	line.ID = "line-" + id
	return line
}

// The converted figure is the one that means anything on a BRL account. This is the
// Anthropic charge: US$ 107,54 that cost R$ 580,00. Matching must find the R$ 580,00
// entry — and comparing face values would instead go looking for R$ 107,54.
func TestReconcile_AForeignLineIsMatchedByItsConvertedValue(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{foreignLine(t, "a", "acc", -10754, -58000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 580, 5)}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Matched != 1 {
		t.Fatalf("the converted value is the one that matches: %+v", out)
	}
}

// The ledger side has no such guard. An entry booked in dollars on a BRL account would
// otherwise match a real R$ 107,54 charge by face value alone — the reconciler
// declaring an unrelated charge settled, at 5.2 times the wrong figure. That is not a
// hypothetical: five dollar charges were once posted as if they were reais here.
func TestReconcile_AnEntryInAnotherCurrencyIsNeverACandidate(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -10754, 5)}
	dollars := systemCharge("tx1", "acc", 107.54, 5)
	dollars.Currency = "USD"
	txns.created = []*transaction.Transaction{dollars}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Matched != 0 {
		t.Fatalf("dollars matched a real charge: %+v", out)
	}
	if len(out.Missing) != 1 {
		t.Fatalf("and the charge must still be reported: %+v", out)
	}
}

// The bank restates a posted line — a foreign charge re-converted, a value corrected.
// The import puts it back to unmatched, but the match row it was reconciled against is
// still live. Skipping the line because a match exists makes it vanish: not matched,
// not missing, not even counted, on every run from then on. The ledger says R$ 40 and
// the bank now says R$ 90, and the report says nothing at all.
func TestReconcile_ARestatedLineIsUnmatchedAndLookedAtAgain(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	line := statementLine(t, "a", "acc", -4000, 5)
	lines.lines = []*statement.Line{line}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	if out, _ := uc.Execute("acc"); out.Matched != 1 {
		t.Fatalf("setup: %+v", out)
	}

	// The bank changes its mind, and the import reflects it.
	line.AmountMinor = -9000
	line.MarkUnmatched()

	out, err := uc.Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Checked != 1 {
		t.Fatalf("the restated line was not even looked at: %+v", out)
	}
	if len(out.Missing) != 1 || out.Missing[0].AmountMinor != -9000 {
		t.Fatalf("the bank now charges 90,00 and nothing in the ledger covers it: %+v", out)
	}
	if matches.matches[0].UnmatchedAt == nil {
		t.Error("the old match still stands against an amount the bank no longer claims")
	}
}

// A reversed entry moved no money — balances derive from CONFIRMED rows. If its bank
// line stays MATCHED, the line is never listed again and the reconciler keeps saying
// the charge is accounted for. That is the phantom this feature exists to catch,
// pointing the other way.
func TestReconcile_AReversedEntryReleasesItsLine(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	charge := systemCharge("tx1", "acc", 40, 5)
	txns.created = []*transaction.Transaction{charge}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	if out, _ := uc.Execute("acc"); out.Matched != 1 {
		t.Fatalf("setup: %+v", out)
	}

	// Reversed: the row is kept as history and stops counting.
	charge.Status = transaction.StatusReversed

	out, err := uc.Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Missing) != 1 {
		t.Fatalf("the bank charged 40,00 and the ledger no longer has it: %+v", out)
	}
	if matches.matches[0].UnmatchedAt == nil {
		t.Error("the line is still reconciled against an entry that was undone")
	}
	if len(lines.updates) == 0 {
		t.Fatal("the release was never persisted: the next run reads the stored status")
	}
	if lines.updates[len(lines.updates)-1].Status != statement.StatusUnmatched {
		t.Errorf("the line is stored as %s", lines.updates[len(lines.updates)-1].Status)
	}
}

// A match that still stands must be left exactly where it is: not re-counted, not
// re-created, and not undone. Re-matching on every run would multiply the rows and
// re-counting would inflate the report.
func TestReconcile_AMatchThatStillStandsIsLeftAlone(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	uc.Execute("acc")
	out, err := uc.Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Matched != 0 || out.Checked != 0 || len(out.Missing) != 0 {
		t.Fatalf("a settled line gave the second run work to do: %+v", out)
	}
	if len(matches.matches) != 1 || matches.matches[0].UnmatchedAt != nil {
		t.Fatalf("the standing match was disturbed: %+v", matches.matches)
	}
}

// Two forecasts of the same value near the same date, and the bank paid one of them.
// Calling that "money the system does not have" is false twice over: the entries are
// sitting right there, and the report sends whoever reads it hunting a charge that was
// never missing. Which of the two the bank settled is not this service's call.
func TestReconcile_TwoForecastsThatFitAreAmbiguousNotMissing(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -50000, 5)}
	first, second := systemCharge("tx1", "acc", 500, 5), systemCharge("tx2", "acc", 500, 6)
	first.Status, second.Status = transaction.StatusPlanned, transaction.StatusPlanned
	txns.created = []*transaction.Transaction{first, second}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Missing) != 0 {
		t.Fatalf("nothing is missing — both entries exist: %+v", out)
	}
	if len(out.Ambiguous) != 1 {
		t.Fatalf("two forecasts fit and neither may be chosen: %+v", out)
	}
	if len(out.Ambiguous[0].CandidateIDs) != 2 {
		t.Errorf("both candidates must be named: %+v", out.Ambiguous[0])
	}
	if len(matches.matches) != 0 {
		t.Errorf("and nothing may be matched: %+v", matches.matches)
	}
}

// The tolerance is two days, and the boundary is the whole point of it: wide enough to
// absorb purchase date against posting date, narrow enough that two different charges
// of the same value in the same week do not become each other. Nothing pinned the
// edge, so widening it to 5 or 10 would have passed silently.
func TestReconcile_TheDateToleranceHasAnEdge(t *testing.T) {
	for _, tc := range []struct {
		name      string
		day       int
		wantMatch bool
	}{
		{"two days apart is the same movement", 7, true},
		{"three days apart is not", 8, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines, txns, matches, accounts := reconcileFixture(t)
			lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
			txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, tc.day)}

			out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
			if (out.Matched == 1) != tc.wantMatch {
				t.Fatalf("matched=%d, want match=%v", out.Matched, tc.wantMatch)
			}
		})
	}
}

// Money is compared in cents, and the conversion has to round rather than truncate:
// 1.15 is 114.99999999999999 in float64, so dropping the rounding turns R$ 1,15 into
// 114 cents and the entry stops matching the line the bank sent.
func TestReconcile_CentsAreRoundedNotTruncated(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -115, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 1.15, 5)}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if out.Matched != 1 {
		t.Fatalf("R$ 1,15 did not match 115 cents: %+v", out)
	}
}

// One account cannot hold two profiles, but the query says so and nothing tested it.
// Dropping the profile filter would put the other profile's entries in the candidate
// set — personal money answering for a company charge.
func TestReconcile_AnotherProfilesEntryIsNeverACandidate(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	other := systemCharge("tx1", "acc", 40, 5)
	other.ProfileID = "p2"
	txns.created = []*transaction.Transaction{other}

	out, _ := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if out.Matched != 0 || len(out.Missing) != 1 {
		t.Fatalf("another profile's entry answered for this charge: %+v", out)
	}
}

// Money ARRIVING in a checking account is positive, and it is the destination leg of a
// transfer that says so. Without the account check in signedMinorFor, a transfer would
// be read as leaving whichever account was asking.
func TestReconcile_MoneyArrivingByTransferIsPositive(t *testing.T) {
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"origem":  {ID: "origem", ProfileID: "p1", Name: "Origem", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
		"destino": {ID: "destino", ProfileID: "p1", Name: "Destino", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
	}}
	lines := &fakeStatementRepo{}
	txns := &fakeTransactionRepo{}
	matches := &fakeMatchRepo{stmt: lines}

	lines.lines = []*statement.Line{statementLine(t, "entrada", "destino", 117000, 5)}
	txns.created = []*transaction.Transaction{{
		ID: "tx1", ProfileID: "p1", BankAccountID: "origem",
		DestinationAccountID: strPtr("destino"),
		Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 1170, Currency: "BRL", Description: "Dinheiro retirado",
		OccurredOn: time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC),
	}}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("destino")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Matched != 1 {
		t.Fatalf("money arriving was not recognised: %+v", out)
	}
}

// Banks print informational lines worth nothing at all. A match has to cover a
// positive amount, so building one for a zero line fails — and that failure used to
// abort the whole run, throwing away every finding already computed. One meaningless
// line must not cost the reconciliation.
func TestReconcile_AZeroValueLineDoesNotAbortTheRun(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	zero := statementLine(t, "zero", "acc", 0, 5)
	lines.lines = []*statement.Line{zero, statementLine(t, "real", "acc", -5390, 5)}
	// A legacy zero-amount entry is what turns the zero line into a candidate pair,
	// and building a match for it is what fails.
	txns.created = []*transaction.Transaction{systemCharge("tx0", "acc", 0, 5)}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("one meaningless line killed the run: %v", err)
	}
	if len(out.Missing) != 1 || out.Missing[0].AmountMinor != -5390 {
		t.Fatalf("the real charge must still be reported: %+v", out)
	}
	if len(matches.matches) != 0 {
		t.Errorf("nothing was owed on a line worth nothing: %+v", matches.matches)
	}
}

// Revalidation has to ask the same question that made the match. Checking only "was
// it reversed" leaves three ways for the ledger to move out from under a standing
// match — and a line whose match still stands is never looked at again, so the
// account reports clean while the money is somewhere else.
func TestReconcile_TheLedgerMovingUnderAMatchReleasesTheLine(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(*transaction.Transaction)
	}{
		{"the entry is moved to another account", func(tx *transaction.Transaction) {
			tx.BankAccountID = "outra"
		}},
		{"the entry's amount is corrected", func(tx *transaction.Transaction) {
			tx.Amount = 45
		}},
		{"the entry's date is moved months away", func(tx *transaction.Transaction) {
			tx.OccurredOn = time.Date(2026, time.December, 5, 0, 0, 0, 0, time.UTC)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines, txns, matches, accounts := reconcileFixture(t)
			accounts.accounts["outra"] = &bankaccount.BankAccount{
				ID: "outra", ProfileID: "p1", Name: "Outra", Type: bankaccount.AccountTypeChecking, Currency: "BRL",
			}
			lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
			charge := systemCharge("tx1", "acc", 40, 5)
			txns.created = []*transaction.Transaction{charge}

			uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
			if out, _ := uc.Execute("acc"); out.Matched != 1 {
				t.Fatalf("setup: %+v", out)
			}

			tc.apply(charge)

			out, err := uc.Execute("acc")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Missing) != 1 {
				t.Fatalf("the bank charged 40,00 and nothing in the ledger covers it now: %+v", out)
			}
			if matches.matches[0].UnmatchedAt == nil {
				t.Error("the match still stands on criteria that no longer hold")
			}
		})
	}
}

// Not knowing is not the same as "it was reversed". A read that failed says nothing
// about the entry, and recording TRANSACTION_REVERSED writes a reason that is simply
// untrue — permanently, since a match refuses a second undo — while the report claims
// money is missing, which is the one answer that gets acted on.
func TestReconcile_AFailedLookupIsNotAReversal(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	if out, _ := uc.Execute("acc"); out.Matched != 1 {
		t.Fatalf("setup: %+v", out)
	}

	txns.created = nil // gone from this account's live entries
	txns.getByIDErr = errors.New("connection reset by peer")

	_, err := uc.Execute("acc")
	if err == nil {
		t.Fatal("a failed read was treated as an answer")
	}
	if matches.matches[0].UnmatchedAt != nil {
		t.Errorf("the match was undone on the strength of a failed read: %v", *matches.matches[0].UnmatchedReason)
	}
}

// A line deliberately set aside is a decision someone made, with a reason the schema
// requires. Reporting it as missing money on every run makes the report noise, and
// matching it destroys both the status and the reason with nothing left to say a
// decision was ever taken.
func TestReconcile_AnIgnoredLineIsLeftAlone(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	ignored := statementLine(t, "a", "acc", -120000, 5)
	if err := ignored.MarkIgnored("perna de rolagem, contabilizada do outro lado"); err != nil {
		t.Fatalf("ignore: %v", err)
	}
	lines.lines = []*statement.Line{ignored}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 1200, 5)}

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Missing) != 0 || out.Matched != 0 {
		t.Fatalf("a line set aside on purpose was put back in the report: %+v", out)
	}
	if ignored.Status != statement.StatusIgnored || ignored.IgnoredReason == nil {
		t.Fatalf("the decision and its reason were overwritten: status=%s reason=%v",
			ignored.Status, ignored.IgnoredReason)
	}
}

// A match a person undid — "WRONG_MATCH", say — must put the line back in the report.
// Reading undone matches as live is how a line disappears for good after somebody
// corrects a mistake, which is the opposite of what correcting it was for.
func TestReconcile_AnUndoneMatchDoesNotHoldTheLine(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}

	uc := NewReconcileStatementUseCase(lines, txns, matches, accounts)
	if out, _ := uc.Execute("acc"); out.Matched != 1 {
		t.Fatalf("setup: %+v", out)
	}
	if err := matches.Unmatch(matches.matches[0].ID, "WRONG_MATCH"); err != nil {
		t.Fatalf("undo: %v", err)
	}

	out, err := uc.Execute("acc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Matched != 1 {
		t.Fatalf("the line was not looked at again after its match was undone: %+v", out)
	}
}

// Losing the race for an entry does not settle the line. The winner claimed the entry
// from a DIFFERENT line of this account, so this charge is still unaccounted for, and
// dropping it from every bucket makes it vanish from the report — the run's own
// arithmetic (checked = matched + missing + ambiguous + ready + pending) stops adding
// up, and nobody is told a thing.
func TestReconcile_LosingTheEntryStillLeavesTheChargeToReport(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}
	matches.createErr = statement.ErrAlreadyClaimedOnAccount

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("losing a race is not an error: %v", err)
	}
	if len(out.Missing) != 1 {
		t.Fatalf("the charge has no entry left to answer for it: %+v", out)
	}
	if out.Checked != out.Matched+len(out.Missing)+len(out.Ambiguous)+len(out.ReadyToConfirm)+out.Pending {
		t.Errorf("a line fell out of the report: %+v", out)
	}
}

// The other race means the opposite: the winner matched THIS line to THIS entry, so
// the work is done and there is nothing to report.
func TestReconcile_LosingToTheSamePairIsWorkAlreadyDone(t *testing.T) {
	lines, txns, matches, accounts := reconcileFixture(t)
	lines.lines = []*statement.Line{statementLine(t, "a", "acc", -4000, 5)}
	txns.created = []*transaction.Transaction{systemCharge("tx1", "acc", 40, 5)}
	matches.createErr = statement.ErrLineAlreadyMatched

	out, err := NewReconcileStatementUseCase(lines, txns, matches, accounts).Execute("acc")
	if err != nil {
		t.Fatalf("losing a race is not an error: %v", err)
	}
	if len(out.Missing) != 0 {
		t.Fatalf("the line is reconciled, by the other run: %+v", out)
	}
}
