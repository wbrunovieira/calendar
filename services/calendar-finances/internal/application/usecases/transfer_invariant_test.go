package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Money that leaves one account has to arrive somewhere. Nothing checked that.
//
// The invariants report covers accounts, invoices, cycles, instalments and
// payments — five checks, and none of them looks at whether a movement has both
// of its sides. That is the double-entry property, and its absence is the shape
// of #1248 (deleting a TRANSFER does not reverse the destination), #2622 (a
// cross-profile credit landing outside every bill) and #1319 (a contribution
// crediting an account that never received the money).
//
// The balance arithmetic already counts both legs of a same-profile transfer —
// source minus, destination plus — so a transfer pointing at nothing drains the
// source and credits no one, and every existing check stays green.

func transferRow(id string, over func(*transaction.Transaction)) *transaction.Transaction {
	dest := "conta-b"
	t := &transaction.Transaction{
		ID: id, ProfileID: "p1", BankAccountID: "conta-a", DestinationAccountID: &dest,
		Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 100, Description: "Transferencia",
	}
	if over != nil {
		over(t)
	}
	return t
}

func invariantsFor(txs []*transaction.Transaction, accounts map[string]*bankaccount.BankAccount) *CheckInvariantsResult {
	return checkTransfers(txs, accounts)
}

func contas() map[string]*bankaccount.BankAccount {
	return map[string]*bankaccount.BankAccount{
		"conta-a": {ID: "conta-a", ProfileID: "p1", Name: "Conta A"},
		"conta-b": {ID: "conta-b", ProfileID: "p1", Name: "Conta B"},
	}
}

func TestATransferWithBothSidesIsFine(t *testing.T) {
	r := invariantsFor([]*transaction.Transaction{transferRow("t1", nil)}, contas())

	if len(r.TransferDrifts) != 0 {
		t.Errorf("reported %+v on a healthy transfer", r.TransferDrifts)
	}
}

// The money leaves and arrives nowhere. The source's balance drops, no account
// gains, and every other invariant stays green.
func TestATransferWithNoDestinationIsReported(t *testing.T) {
	r := invariantsFor([]*transaction.Transaction{
		transferRow("t1", func(t *transaction.Transaction) { t.DestinationAccountID = nil }),
	}, contas())

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("drifts = %+v, want the transfer with no destination", r.TransferDrifts)
	}
	if r.OK {
		t.Error("a transfer that lands nowhere passed the check")
	}
}

func TestATransferPointingAtAnAccountThatDoesNotExistIsReported(t *testing.T) {
	fantasma := "conta-que-nao-existe"
	r := invariantsFor([]*transaction.Transaction{
		transferRow("t1", func(t *transaction.Transaction) { t.DestinationAccountID = &fantasma }),
	}, contas())

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("drifts = %+v, want the transfer to a missing account", r.TransferDrifts)
	}
}

// A cross-profile transfer is stored as TWO linked rows — an expense on one
// profile and an income on the other. If one of them is gone, money crossed a
// profile boundary in one direction only.
func parLigado() []*transaction.Transaction {
	a, b := "t-saida", "t-entrada"
	return []*transaction.Transaction{
		{ID: a, ProfileID: "p1", BankAccountID: "conta-a", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 770, Description: "Aporte WB",
			LinkedTransactionID: &b},
		{ID: b, ProfileID: "p2", BankAccountID: "conta-b", Type: transaction.TypeIncome,
			Status: transaction.StatusConfirmed, Amount: 770, Description: "Aporte WB",
			LinkedTransactionID: &a},
	}
}

func TestALinkedPairWithBothSidesIsFine(t *testing.T) {
	if r := invariantsFor(parLigado(), contas()); len(r.TransferDrifts) != 0 {
		t.Errorf("reported %+v on a healthy pair", r.TransferDrifts)
	}
}

func TestALinkedTransactionWhoseSiblingIsGoneIsReported(t *testing.T) {
	par := parLigado()[:1] // a entrada sumiu

	r := invariantsFor(par, contas())

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("drifts = %+v, want the orphaned leg", r.TransferDrifts)
	}
}

// One side confirmed and the other cancelled is the same hole with a different
// cause: the money moved on one book and not on the other.
func TestALinkedPairWithOneSideCancelledIsReported(t *testing.T) {
	par := parLigado()
	par[1].Status = transaction.StatusCancelled

	r := invariantsFor(par, contas())

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("drifts = %+v, want the half-cancelled pair", r.TransferDrifts)
	}
}

func TestALinkedPairThatDisagreesOnTheAmountIsReported(t *testing.T) {
	par := parLigado()
	par[1].Amount = 700

	r := invariantsFor(par, contas())

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("drifts = %+v, want the pair that disagrees on the amount", r.TransferDrifts)
	}
}

// Both sides cancelled is not a hole: nothing moved on either book.
func TestAPairCancelledOnBothSidesIsNotReported(t *testing.T) {
	par := parLigado()
	par[0].Status = transaction.StatusCancelled
	par[1].Status = transaction.StatusCancelled

	if r := invariantsFor(par, contas()); len(r.TransferDrifts) != 0 {
		t.Errorf("reported %+v on a pair cancelled on both sides", r.TransferDrifts)
	}
}

// A one-sided link is still a hole: A says it is paired with B, B says nothing.
func TestALinkThatIsNotMutualIsReported(t *testing.T) {
	par := parLigado()
	par[1].LinkedTransactionID = nil

	r := invariantsFor(par, contas())

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("drifts = %+v, want the one-sided link", r.TransferDrifts)
	}
}

// A cancelled transfer moved nothing, so its destination does not matter.
func TestACancelledTransferWithNoDestinationIsNotReported(t *testing.T) {
	r := invariantsFor([]*transaction.Transaction{
		transferRow("t1", func(t *transaction.Transaction) {
			t.DestinationAccountID = nil
			t.Status = transaction.StatusCancelled
		}),
	}, contas())

	if len(r.TransferDrifts) != 0 {
		t.Errorf("reported %+v on a cancelled transfer", r.TransferDrifts)
	}
}

// The checks above exercise the rule. These exercise the WIRING: that the report
// actually runs it, fails because of it, and gathers both profiles.
//
// Both were untested and both mutants survived — the helper was right and
// nothing proved the report called it properly, which is the same gap that let a
// card-balance regression through two PRs ago.

type invariantsTxFake struct {
	transaction.Repository
	porPerfil map[string][]*transaction.Transaction
}

func (f *invariantsTxFake) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	return f.porPerfil[filter.ProfileID], nil
}
func (f *invariantsTxFake) CalculateBalanceByBankAccountID(string) (float64, error) { return 0, nil }

func TestTheReportFailsWhenMoneyHasOnlyOneSide(t *testing.T) {
	contas := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"conta-a": {ID: "conta-a", ProfileID: "p1", Name: "Conta A", Type: bankaccount.AccountTypeChecking},
	}}
	txs := &invariantsTxFake{porPerfil: map[string][]*transaction.Transaction{
		"p1": {transferRow("t1", func(t *transaction.Transaction) { t.DestinationAccountID = nil })},
	}}

	r, err := NewCheckInvariantsUseCase(contas, txs, &fakeInvoiceRepo{}).Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("transferDrifts = %+v, want the one-sided transfer", r.TransferDrifts)
	}
	if r.OK {
		t.Error("ok = true with money that left and arrived nowhere — the daily check answers 200 and says nothing")
	}
}

// A cross-profile pair has one leg in each book. A report that walked one
// profile at a time would call every healthy pair half-written — and, worse,
// would miss a genuinely broken one whose surviving leg lives in the profile it
// never looked at.
func TestTheReportGathersBothProfilesBeforeJudgingAPair(t *testing.T) {
	contas := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"conta-a": {ID: "conta-a", ProfileID: "p1", Name: "Pessoal", Type: bankaccount.AccountTypeChecking},
		"conta-b": {ID: "conta-b", ProfileID: "p2", Name: "WB", Type: bankaccount.AccountTypeChecking},
	}}
	par := parLigado()
	txs := &invariantsTxFake{porPerfil: map[string][]*transaction.Transaction{
		"p1": {par[0]},
		"p2": {par[1]},
	}}

	r, err := NewCheckInvariantsUseCase(contas, txs, &fakeInvoiceRepo{}).Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(r.TransferDrifts) != 0 {
		t.Errorf("a healthy cross-profile pair was reported as broken: %+v", r.TransferDrifts)
	}
}

// And a pair that really is broken across profiles is still caught.
func TestAPairBrokenAcrossProfilesIsStillCaught(t *testing.T) {
	contas := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"conta-a": {ID: "conta-a", ProfileID: "p1", Name: "Pessoal", Type: bankaccount.AccountTypeChecking},
		"conta-b": {ID: "conta-b", ProfileID: "p2", Name: "WB", Type: bankaccount.AccountTypeChecking},
	}}
	par := parLigado()
	par[1].Status = transaction.StatusCancelled
	txs := &invariantsTxFake{porPerfil: map[string][]*transaction.Transaction{
		"p1": {par[0]},
		"p2": {par[1]},
	}}

	r, _ := NewCheckInvariantsUseCase(contas, txs, &fakeInvoiceRepo{}).Execute()

	if len(r.TransferDrifts) != 1 {
		t.Fatalf("transferDrifts = %+v, want the half-cancelled cross-profile pair", r.TransferDrifts)
	}
}
