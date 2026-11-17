package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A credit card's balance is DERIVED from its bills, on every read, with one
// owner — decided on 16/11/2026 (issue #1277).
//
// What it used to mean: "the sum of the ledger the last time someone paid a
// bill". Six write paths skip credit cards deliberately and only the
// pay-invoice path writes, so between bills the number stands still while
// purchases happen. Measured the same day, per card: stored -455,70, derived
// 371,50, raw sum of every confirmed row 6.652,93 — three numbers, and only the
// derived one matched the bank to the cent.
//
// The tell was the card created two days earlier: on it all three agreed. The
// divergence is accumulated damage, not a formula error, so it cannot be fixed
// by a one-off recalculation.

func cardWithBills(stored float64) (*fakeAccountRepo, *usageInvoiceRepo, *usageTxRepo) {
	limit := 400.0
	accounts := map[string]*bankaccount.BankAccount{
		"card": {
			ID: "card", ProfileID: "p1", Name: "Cartao",
			Type: bankaccount.AccountTypeCreditCard, CreditLimit: &limit,
			CurrentBalance: stored, IsActive: true,
		},
		"conta": {
			ID: "conta", ProfileID: "p1", Name: "Conta",
			Type: bankaccount.AccountTypeChecking, CurrentBalance: 718.36, IsActive: true,
		},
	}
	invoices := &usageInvoiceRepo{list: []*invoice.Invoice{
		{ID: "aberta", BankAccountID: "card", Status: invoice.StatusOpen},
	}}
	inv := "aberta"
	txs := &usageTxRepo{txs: []*transaction.Transaction{
		{ID: "t1", InvoiceID: &inv, Amount: 371.50, Status: transaction.StatusConfirmed},
	}}
	return &fakeAccountRepo{accounts: accounts}, invoices, txs
}

func TestListReportsACardBalanceDerivedFromItsBills(t *testing.T) {
	repo, invoices, txs := cardWithBills(-455.70)
	uc := NewListBankAccountsUseCase(repo, NewCardOutstanding(invoices, txs))

	accounts, err := uc.Execute("p1")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	byID := map[string]*bankaccount.BankAccount{}
	for _, a := range accounts {
		byID[a.ID] = a
	}
	if got := byID["card"].CurrentBalance; got < -371.51 || got > -371.49 {
		t.Errorf("card balance = %v, want -371.50 derived from the bills (stored was -455.70)", got)
	}
}

// Debt stays NEGATIVE. Every total that adds balances up — net worth, the
// account list, the month close — keeps working only if the sign survives, and
// flipping it would turn debt into money without a single test going red.
func TestADerivedCardBalanceKeepsDebtNegative(t *testing.T) {
	repo, invoices, txs := cardWithBills(0)
	uc := NewListBankAccountsUseCase(repo, NewCardOutstanding(invoices, txs))

	accounts, _ := uc.Execute("p1")
	for _, a := range accounts {
		if a.ID == "card" && a.CurrentBalance >= 0 {
			t.Errorf("card balance = %v, want a negative number for money owed", a.CurrentBalance)
		}
	}
}

func TestListLeavesAccountsThatAreNotCardsAlone(t *testing.T) {
	repo, invoices, txs := cardWithBills(-455.70)
	uc := NewListBankAccountsUseCase(repo, NewCardOutstanding(invoices, txs))

	accounts, _ := uc.Execute("p1")
	for _, a := range accounts {
		if a.ID == "conta" && (a.CurrentBalance < 718.35 || a.CurrentBalance > 718.37) {
			t.Errorf("checking balance = %v, want the stored 718.36 untouched", a.CurrentBalance)
		}
	}
}

func TestGetReportsACardBalanceDerivedFromItsBills(t *testing.T) {
	repo, invoices, txs := cardWithBills(-455.70)
	uc := NewGetBankAccountUseCase(repo, NewCardOutstanding(invoices, txs))

	account, err := uc.Execute("card")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := account.CurrentBalance; got < -371.51 || got > -371.49 {
		t.Errorf("card balance = %v, want -371.50", got)
	}
}

// A card with nothing owed reports zero, not the stale stored figure.
func TestASettledCardReportsZeroAndNotTheStoredNumber(t *testing.T) {
	repo, invoices, _ := cardWithBills(-455.70)
	uc := NewGetBankAccountUseCase(repo, NewCardOutstanding(invoices, &usageTxRepo{}))

	account, _ := uc.Execute("card")

	if account.CurrentBalance != 0 {
		t.Errorf("card balance = %v, want 0 — nothing is owed", account.CurrentBalance)
	}
}

// The derivation is the SAME one credit-usage already answers with. Two owners
// would be two numbers again, which is the whole defect.
func TestTheDerivedBalanceAgreesWithCreditUsage(t *testing.T) {
	repo, invoices, txs := cardWithBills(-455.70)
	outstanding := NewCardOutstanding(invoices, txs)

	account, _ := NewGetBankAccountUseCase(repo, outstanding).Execute("card")
	usage, err := NewGetCreditUsageUseCase(repo, invoices, txs).Execute("card")
	if err != nil {
		t.Fatalf("credit usage: %v", err)
	}

	if -account.CurrentBalance != usage.Outstanding {
		t.Errorf("balance %v and credit-usage %v disagree", account.CurrentBalance, usage.Outstanding)
	}
}

// A card's BALANCE is what it owes today; its COMMITTED total, which is what the
// available limit is made of, is a different number and keeps its own name.
//
// The gap is the instalment plan. `SumByInvoiceID` counts PLANNED rows and an
// instalment writes one PLANNED charge per future bill, so a balance that walks
// every invoice reports the whole plan as if it were due. Measured on the real
// Mercado Pago card on 16/11/2026: 4.502,45 committed, 1.358,65 owed. The
// planning screens subtract the balance from today's cash, so the difference is
// not cosmetic.
func cardWithAnInstalmentPlan() (*fakeAccountRepo, *usageInvoiceRepo, *usageTxRepo, time.Time) {
	hoje := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	limite := 4400.0
	accounts := map[string]*bankaccount.BankAccount{
		"card": {
			ID: "card", ProfileID: "p1", Name: "Cartao",
			Type: bankaccount.AccountTypeCreditCard, CreditLimit: &limite,
			CurrentBalance: -1185.02, IsActive: true,
		},
	}
	dia := func(d int, m time.Month) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	invoices := &usageInvoiceRepo{list: []*invoice.Invoice{
		// Ja fechada e quitada.
		{ID: "nov", BankAccountID: "card", Status: invoice.StatusPaid, ClosingDate: dia(9, time.November)},
		// Ciclo ABERTO: fecha em dezembro, e o que esta correndo hoje.
		{ID: "dez", BankAccountID: "card", Status: invoice.StatusOpen, ClosingDate: dia(9, time.December)},
		// Ciclos que ainda nao abriram: so carregam parcelas.
		{ID: "jan", BankAccountID: "card", Status: invoice.StatusOpen, ClosingDate: time.Date(2027, 1, 9, 0, 0, 0, 0, time.UTC)},
		{ID: "fev", BankAccountID: "card", Status: invoice.StatusOpen, ClosingDate: time.Date(2027, 2, 9, 0, 0, 0, 0, time.UTC)},
	}}
	ref := func(s string) *string { return &s }
	txs := &usageTxRepo{txs: []*transaction.Transaction{
		{ID: "a", InvoiceID: ref("dez"), Amount: 1053.45, Status: transaction.StatusConfirmed, Type: transaction.TypeExpense},
		{ID: "b", InvoiceID: ref("dez"), Amount: 305.20, Status: transaction.StatusPlanned, Type: transaction.TypeExpense},
		{ID: "c", InvoiceID: ref("jan"), Amount: 938.55, Status: transaction.StatusPlanned, Type: transaction.TypeExpense},
		{ID: "d", InvoiceID: ref("fev"), Amount: 938.55, Status: transaction.StatusPlanned, Type: transaction.TypeExpense},
	}}
	return &fakeAccountRepo{accounts: accounts}, invoices, txs, hoje
}

func TestOwedNowLeavesOutCyclesThatHaveNotOpened(t *testing.T) {
	repo, invoices, txs, hoje := cardWithAnInstalmentPlan()

	owed, err := NewCardOutstanding(invoices, txs).OwedNow(repo.accounts["card"], hoje)
	if err != nil {
		t.Fatalf("OwedNow: %v", err)
	}

	if owed < 1358.64 || owed > 1358.66 {
		t.Errorf("owed now = %v, want 1358.65 — the open cycle only", owed)
	}
}

func TestCommittedStillCountsTheWholeInstalmentPlan(t *testing.T) {
	repo, invoices, txs, _ := cardWithAnInstalmentPlan()

	committed, err := NewCardOutstanding(invoices, txs).For(repo.accounts["card"])
	if err != nil {
		t.Fatalf("For: %v", err)
	}

	// 1.053,45 + 305,20 + 938,55 + 938,55. The issuer reserves all of it against
	// the limit, so the available-limit answer must not shrink.
	if committed < 3235.74 || committed > 3235.76 {
		t.Errorf("committed = %v, want 3235.75 — every unpaid bill", committed)
	}
}

func TestTheBalanceReportedIsWhatIsOwedNow(t *testing.T) {
	repo, invoices, txs, _ := cardWithAnInstalmentPlan()

	account, err := NewGetBankAccountUseCase(repo, NewCardOutstanding(invoices, txs)).Execute("card")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Negative: the sign convention every total depends on.
	if account.CurrentBalance < -1358.66 || account.CurrentBalance > -1358.64 {
		t.Errorf("balance = %v, want -1358.65 and not the committed total", account.CurrentBalance)
	}
}

// An INCOME line on a bill SUBTRACTS — a refund lowers what is owed. The fake
// used to add it, so this had never been exercised anywhere.
func TestARefundOnABillLowersWhatIsOwed(t *testing.T) {
	repo, invoices, txs, hoje := cardWithAnInstalmentPlan()
	ref := "dez"
	txs.txs = append(txs.txs, &transaction.Transaction{
		ID: "estorno", InvoiceID: &ref, Amount: 53.45, Status: transaction.StatusConfirmed, Type: transaction.TypeIncome,
	})

	owed, _ := NewCardOutstanding(invoices, txs).OwedNow(repo.accounts["card"], hoje)

	if owed < 1305.19 || owed > 1305.21 {
		t.Errorf("owed now = %v, want 1305.20 — the refund came off", owed)
	}
}

// A charge that has not joined a bill yet belongs to the cycle in progress, so
// it is owed now. It must not be double counted against a future bill either.
func TestAChargeWithNoBillYetCountsAsOwedNow(t *testing.T) {
	repo, invoices, txs, hoje := cardWithAnInstalmentPlan()
	txs.txs = append(txs.txs, &transaction.Transaction{
		ID: "solta", Amount: 89.84, Status: transaction.StatusConfirmed, Type: transaction.TypeExpense,
	})

	owed, _ := NewCardOutstanding(invoices, txs).OwedNow(repo.accounts["card"], hoje)

	if owed < 1448.48 || owed > 1448.50 {
		t.Errorf("owed now = %v, want 1448.49", owed)
	}
}

// A card whose bills cannot be read FAILS the request. It does not quietly serve
// the stored number.
//
// The alternative was graceful degradation, as the bank reconciliation chose —
// but that report has an `Unreadable` field to say so. The account list has no
// way to express "this balance is unknown", and a screen full of balances where
// one is silently stale is worse than a screen that refuses to load. Pinned here
// because a mutant that swallowed this error passed the whole suite.
func TestAnUnreadableCardFailsTheListInsteadOfServingAStaleNumber(t *testing.T) {
	repo, _, txs := cardWithBills(-455.70)
	quebrado := &brokenInvoiceRepo{}
	uc := NewListBankAccountsUseCase(repo, NewCardOutstanding(quebrado, txs))

	accounts, err := uc.Execute("p1")

	if err == nil {
		t.Fatalf("expected a failure; got %d accounts with balances nobody verified", len(accounts))
	}
}

func TestAnUnreadableCardAlsoFailsTheSingleAccountRead(t *testing.T) {
	repo, _, txs := cardWithBills(-455.70)
	uc := NewGetBankAccountUseCase(repo, NewCardOutstanding(&brokenInvoiceRepo{}, txs))

	if _, err := uc.Execute("card"); err == nil {
		t.Error("expected a failure reading a card whose bills cannot be read")
	}
}

type brokenInvoiceRepo struct {
	fakeInvoiceRepo
}

func (r *brokenInvoiceRepo) FindByBankAccountID(string) ([]*invoice.Invoice, error) {
	return nil, errors.New("bills unavailable")
}

// The unfiltered path — the one every main screen uses, with no profileId — must
// derive too. A mutant deleting the FindAll branch left the suite green.
func TestListWithNoProfileAlsoDerivesTheCardBalance(t *testing.T) {
	repo, invoices, txs := cardWithBills(-455.70)
	uc := NewListBankAccountsUseCase(repo, NewCardOutstanding(invoices, txs))

	accounts, err := uc.Execute("")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatal("the unfiltered list came back empty")
	}
	for _, a := range accounts {
		if a.ID == "card" && (a.CurrentBalance < -371.51 || a.CurrentBalance > -371.49) {
			t.Errorf("card balance = %v on the unfiltered path, want -371.50", a.CurrentBalance)
		}
	}
}
