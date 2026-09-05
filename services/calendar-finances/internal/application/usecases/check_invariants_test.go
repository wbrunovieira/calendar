package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// The fakes below embed the domain interface on purpose: any method the use case
// starts calling that the test did not set up panics loudly instead of quietly
// returning a zero value. A silent zero is exactly how a money bug hides.

type invariantAccountRepo struct {
	bankaccount.Repository
	accounts []*bankaccount.BankAccount
	err      error
}

func (f *invariantAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) {
	return f.accounts, f.err
}

type invariantTxRepo struct {
	transaction.Repository
	balances    map[string]float64
	invoiceSums map[string]float64
}

func (f *invariantTxRepo) CalculateBalanceByBankAccountID(accountID string) (float64, error) {
	return f.balances[accountID], nil
}

func (f *invariantTxRepo) SumByInvoiceID(invoiceID string) (float64, error) {
	return f.invoiceSums[invoiceID], nil
}

type invariantInvoiceRepo struct {
	invoice.Repository
	byAccount map[string][]*invoice.Invoice
	asked     []string
}

func (f *invariantInvoiceRepo) FindByBankAccountID(accountID string) ([]*invoice.Invoice, error) {
	f.asked = append(f.asked, accountID)
	return f.byAccount[accountID], nil
}

func invariantCheckingAccount(id string, initial, current float64) *bankaccount.BankAccount {
	return &bankaccount.BankAccount{
		ID:             id,
		Name:           "Conta " + id,
		Type:           bankaccount.AccountTypeChecking,
		InitialBalance: initial,
		CurrentBalance: current,
		IsActive:       true,
	}
}

func TestCheckInvariants_ReportsNoDriftWhenTheStoredBalanceMatchesTheLedger(t *testing.T) {
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{
		invariantCheckingAccount("acc-1", 100, 350),
	}}
	txs := &invariantTxRepo{balances: map[string]float64{"acc-1": 250}}
	invoices := &invariantInvoiceRepo{}

	uc := NewCheckInvariantsUseCase(accounts, txs, invoices)
	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CheckedAccounts != 1 {
		t.Errorf("CheckedAccounts = %d, want 1", result.CheckedAccounts)
	}
	if len(result.AccountDrifts) != 0 {
		t.Errorf("expected no drift, got %+v", result.AccountDrifts)
	}
	if !result.OK {
		t.Error("OK = false, want true when every balance matches")
	}
}

func TestCheckInvariants_ReportsTheExactDriftOfACheckingAccount(t *testing.T) {
	// Stored says 1000.00, the ledger says 100 + 812.35 = 912.35 → the account
	// is 87.65 richer than the transactions justify.
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{
		invariantCheckingAccount("acc-1", 100, 1000),
	}}
	txs := &invariantTxRepo{balances: map[string]float64{"acc-1": 812.35}}

	uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.AccountDrifts) != 1 {
		t.Fatalf("expected 1 drift, got %d", len(result.AccountDrifts))
	}
	got := result.AccountDrifts[0]
	if got.ComputedBalance != 912.35 {
		t.Errorf("ComputedBalance = %v, want 912.35", got.ComputedBalance)
	}
	if got.Drift != 87.65 {
		t.Errorf("Drift = %v, want 87.65", got.Drift)
	}
	if got.Note != "" {
		t.Errorf("a checking account should carry no note, got %q", got.Note)
	}
	if result.OK {
		t.Error("OK = true, want false when a checking account drifts")
	}
}

// Balances are money: rounding both sides to cents is what absorbs the float
// noise of summing amounts. A tenth of a cent is not a missing transaction.
func TestCheckInvariants_IgnoresSubCentNoise(t *testing.T) {
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{
		invariantCheckingAccount("acc-1", 0, 10.0),
	}}
	txs := &invariantTxRepo{balances: map[string]float64{"acc-1": 10.001}}

	uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
	result, _ := uc.Execute()

	if len(result.AccountDrifts) != 0 {
		t.Errorf("a tenth of a cent is float noise, not drift: %+v", result.AccountDrifts)
	}
}

func TestCheckInvariants_MarketValueAccountsAreReportedButDoNotFailTheCheck(t *testing.T) {
	// An INVESTMENT account stores quotas × price, not initial + transactions.
	// Until that moves to its own column the difference is expected, so it must
	// be visible without turning the whole check red.
	for _, accType := range []bankaccount.AccountType{
		bankaccount.AccountTypeInvestment,
		bankaccount.AccountTypeExchange,
		bankaccount.AccountTypeWallet,
	} {
		t.Run(string(accType), func(t *testing.T) {
			acc := invariantCheckingAccount("acc-inv", 0, 5000)
			acc.Type = accType
			accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{acc}}
			txs := &invariantTxRepo{balances: map[string]float64{"acc-inv": 3000}}

			uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
			result, _ := uc.Execute()

			if len(result.AccountDrifts) != 1 {
				t.Fatalf("the difference must stay visible, got %d entries", len(result.AccountDrifts))
			}
			if result.AccountDrifts[0].Note == "" {
				t.Error("a market-value account must explain why it differs")
			}
			if !result.OK {
				t.Error("OK = false: an expected difference must not fail the check")
			}
		})
	}
}

// A card sitting at exactly zero has never been written by anything: every
// write path guards cards out of the balance. That difference is a known gap,
// not a regression.
func TestCheckInvariants_ACardNothingEverWroteDoesNotFailTheCheck(t *testing.T) {
	card := invariantCheckingAccount("card-1", 0, 0)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}
	txs := &invariantTxRepo{balances: map[string]float64{"card-1": -430.12}}

	uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
	result, _ := uc.Execute()

	if len(result.AccountDrifts) != 1 {
		t.Fatalf("expected the card to be reported, got %d entries", len(result.AccountDrifts))
	}
	if result.AccountDrifts[0].Note == "" {
		t.Error("a card at zero must explain why its balance was never written")
	}
	if !result.OK {
		t.Error("OK = false: a card nothing ever wrote is a known gap, not a regression")
	}
}

// But PayInvoiceUseCaseV2 does write a card's balance, and it recalculates the
// card afterwards. So a card holding a non-zero balance IS maintained by the
// same invariant as any other account, and a difference there is a real defect.
// Excusing it would let the endpoint bless the first bug it was built to catch.
func TestCheckInvariants_ACardWithAMaintainedBalanceFailsTheCheck(t *testing.T) {
	card := invariantCheckingAccount("card-1", 0, -3160.58)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}
	txs := &invariantTxRepo{balances: map[string]float64{"card-1": -3466.33}}

	uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
	result, _ := uc.Execute()

	if len(result.AccountDrifts) != 1 {
		t.Fatalf("expected the card to be reported, got %d entries", len(result.AccountDrifts))
	}
	got := result.AccountDrifts[0]
	if got.Note != "" {
		t.Errorf("a maintained card must carry no excuse, got %q", got.Note)
	}
	if got.Drift != 305.75 {
		t.Errorf("Drift = %v, want 305.75", got.Drift)
	}
	if result.OK {
		t.Error("OK = true while a maintained card balance disagrees with its ledger")
	}
}

func TestCheckInvariants_ReportsInvoicesWhoseStoredTotalDoesNotMatchTheirTransactions(t *testing.T) {
	card := invariantCheckingAccount("card-1", 0, 0)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}

	ref := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	invoices := &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{
		"card-1": {
			{ID: "inv-stale", BankAccountID: "card-1", ReferenceDate: ref, Amount: 0},
			{ID: "inv-ok", BankAccountID: "card-1", ReferenceDate: ref, Amount: 126.74},
		},
	}}
	txs := &invariantTxRepo{
		balances: map[string]float64{"card-1": 0},
		invoiceSums: map[string]float64{
			"inv-stale": 923.04,
			"inv-ok":    126.74,
		},
	}

	uc := NewCheckInvariantsUseCase(accounts, txs, invoices)
	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CheckedInvoices != 2 {
		t.Errorf("CheckedInvoices = %d, want 2", result.CheckedInvoices)
	}
	if len(result.InvoiceDrifts) != 1 {
		t.Fatalf("expected only the stale invoice, got %+v", result.InvoiceDrifts)
	}
	got := result.InvoiceDrifts[0]
	if got.InvoiceID != "inv-stale" {
		t.Errorf("InvoiceID = %q, want inv-stale", got.InvoiceID)
	}
	if got.StoredAmount != 0 || got.ComputedAmount != 923.04 {
		t.Errorf("stored/computed = %v/%v, want 0/923.04", got.StoredAmount, got.ComputedAmount)
	}
	if got.Drift != -923.04 {
		t.Errorf("Drift = %v, want -923.04 (stored minus computed)", got.Drift)
	}
	// credit_card_invoices.amount has no writer: the read paths recompute it in
	// memory without persisting, and the one route that does persist it refuses
	// PAID invoices. Every stored value is stale by construction, so this can
	// never be brought to zero. Failing the check on it would mean an alarm
	// nobody can clear, and an alarm nobody can clear stops being read.
	if got.Note == "" {
		t.Error("an invoice total must explain that it is derived, not stored")
	}
	if !result.OK {
		t.Error("OK = false: a stale invoice total is a known gap with no action available")
	}
}

func TestCheckInvariants_OnlyCreditCardsAreScannedForInvoices(t *testing.T) {
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{
		invariantCheckingAccount("acc-1", 0, 0),
	}}
	// The checking account has invoices on file. Nothing may look at them: an
	// empty fake here would pass with or without the guard, which is no test.
	invoices := &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{
		"acc-1": {{ID: "inv-1", BankAccountID: "acc-1", Amount: 500}},
	}}
	txs := &invariantTxRepo{invoiceSums: map[string]float64{"inv-1": 1}}

	uc := NewCheckInvariantsUseCase(accounts, txs, invoices)
	result, _ := uc.Execute()

	if len(invoices.asked) != 0 {
		t.Errorf("invoices were fetched for a non-card account: %v", invoices.asked)
	}
	if result.CheckedInvoices != 0 {
		t.Errorf("CheckedInvoices = %d, want 0 for a checking account", result.CheckedInvoices)
	}
	if len(result.InvoiceDrifts) != 0 {
		t.Errorf("a checking account has no invoices to drift: %+v", result.InvoiceDrifts)
	}
}
