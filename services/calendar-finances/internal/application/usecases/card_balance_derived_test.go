package usecases

import (
	"testing"

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
