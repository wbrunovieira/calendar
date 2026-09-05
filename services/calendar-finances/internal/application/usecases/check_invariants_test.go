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

// A credit card's balance is a frozen snapshot of the last invoice payment:
// create, update and delete all skip cards entirely, and only the invoice
// payment ever writes one. So drift on a card is the purchases made since that
// payment, with no missing transaction behind it — whatever the balance reads.
// Failing the check on it would re-arm itself with the next coffee.
func TestCheckInvariants_NoCreditCardFailsTheCheckWhileItsBalanceIsASnapshot(t *testing.T) {
	for _, tc := range []struct {
		name          string
		storedBalance float64
		ledger        float64
		expectedDrift float64
	}{
		{"never paid", 0, -741.48, 741.48},
		{"paid once, used since", -3160.58, -3466.33, 305.75},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := invariantCheckingAccount("card-1", 0, tc.storedBalance)
			card.Type = bankaccount.AccountTypeCreditCard
			accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}
			txs := &invariantTxRepo{balances: map[string]float64{"card-1": tc.ledger}}

			uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
			result, _ := uc.Execute()

			if len(result.AccountDrifts) != 1 {
				t.Fatalf("expected the card to be reported, got %d entries", len(result.AccountDrifts))
			}
			got := result.AccountDrifts[0]
			if got.Note == "" {
				t.Error("a card must explain that its balance is a snapshot")
			}
			if got.Drift != tc.expectedDrift {
				t.Errorf("Drift = %v, want %v", got.Drift, tc.expectedDrift)
			}
			if !result.OK {
				t.Error("OK = false: a card's staleness is the design, not a transaction to find")
			}
		})
	}
}

// A CLOSED invoice whose stored total is double the sum of its transactions is
// the signature of a duplicated or lost charge — and one call to
// POST /invoices/{id}/recalculate persists exactly this sum. It is actionable,
// so it must fail the check.
func TestCheckInvariants_AnOpenOrClosedInvoiceDriftFailsTheCheck(t *testing.T) {
	for _, status := range []invoice.Status{invoice.StatusOpen, invoice.StatusClosed} {
		t.Run(string(status), func(t *testing.T) {
			card := invariantCheckingAccount("card-1", 0, 0)
			card.Type = bankaccount.AccountTypeCreditCard
			accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}

			ref := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
			invoices := &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{
				"card-1": {{ID: "inv-doubled", BankAccountID: "card-1", ReferenceDate: ref, Amount: 49.80, Status: status}},
			}}
			txs := &invariantTxRepo{
				balances:    map[string]float64{"card-1": 0},
				invoiceSums: map[string]float64{"inv-doubled": 24.90},
			}

			uc := NewCheckInvariantsUseCase(accounts, txs, invoices)
			result, err := uc.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.InvoiceDrifts) != 1 {
				t.Fatalf("expected the invoice to be reported, got %+v", result.InvoiceDrifts)
			}
			got := result.InvoiceDrifts[0]
			if got.Drift != 24.90 {
				t.Errorf("Drift = %v, want 24.90 (stored minus computed)", got.Drift)
			}
			if got.Note != "" {
				t.Errorf("an invoice a route can recalculate carries no excuse, got %q", got.Note)
			}
			if result.OK {
				t.Error("OK = true while an invoice a single POST would fix is out of line")
			}
		})
	}
}

// A PAID invoice is the one case with no way back: the recalculate route
// refuses it. Reported, but it cannot gate an alarm nobody can clear.
func TestCheckInvariants_APaidInvoiceDriftIsReportedWithoutFailingTheCheck(t *testing.T) {
	card := invariantCheckingAccount("card-1", 0, 0)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}

	ref := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	invoices := &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{
		"card-1": {
			{ID: "inv-paid", BankAccountID: "card-1", ReferenceDate: ref, Amount: 897.43, Status: invoice.StatusPaid},
			{ID: "inv-ok", BankAccountID: "card-1", ReferenceDate: ref, Amount: 126.74, Status: invoice.StatusClosed},
		},
	}}
	txs := &invariantTxRepo{
		balances: map[string]float64{"card-1": 0},
		invoiceSums: map[string]float64{
			"inv-paid": 0,
			"inv-ok":   126.74,
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
		t.Fatalf("expected only the paid invoice, got %+v", result.InvoiceDrifts)
	}
	got := result.InvoiceDrifts[0]
	if got.InvoiceID != "inv-paid" {
		t.Errorf("InvoiceID = %q, want inv-paid", got.InvoiceID)
	}
	if got.Drift != 897.43 {
		t.Errorf("Drift = %v, want 897.43", got.Drift)
	}
	if got.Note == "" {
		t.Error("a paid invoice must explain that it can no longer be recalculated")
	}
	if !result.OK {
		t.Error("OK = false on the one invoice state no route can fix")
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

// Drift runs both ways and the check has to catch both. A stored balance ABOVE
// the ledger is money the account claims and cannot justify; a stored balance
// BELOW it is a payment that landed and was never recorded, or an expense
// recorded twice. Every account test used to sit on the same side of zero, so
// `drift <= 0` passed the whole suite.
func TestCheckInvariants_CatchesDriftInBothDirections(t *testing.T) {
	for _, tc := range []struct {
		name          string
		stored        float64
		ledger        float64
		expectedDrift float64
	}{
		{"stored above the ledger", 1000, 812.35, 187.65},
		{"stored below the ledger", 812.35, 1000, -187.65},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{
				invariantCheckingAccount("acc-1", 0, tc.stored),
			}}
			txs := &invariantTxRepo{balances: map[string]float64{"acc-1": tc.ledger}}

			uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
			result, _ := uc.Execute()

			if len(result.AccountDrifts) != 1 {
				t.Fatalf("expected 1 drift, got %d", len(result.AccountDrifts))
			}
			if got := result.AccountDrifts[0].Drift; got != tc.expectedDrift {
				t.Errorf("Drift = %v, want %v", got, tc.expectedDrift)
			}
			if result.OK {
				t.Error("OK = true while a checking account disagrees with its ledger")
			}
		})
	}
}

// The same, for invoices: a stored total below the sum of its charges hides a
// bill, and one above it invents one.
func TestCheckInvariants_CatchesInvoiceDriftInBothDirections(t *testing.T) {
	for _, tc := range []struct {
		name          string
		stored        float64
		computed      float64
		expectedDrift float64
	}{
		{"stored above the charges", 49.80, 24.90, 24.90},
		{"stored below the charges", 0, 923.04, -923.04},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := invariantCheckingAccount("card-1", 0, 0)
			card.Type = bankaccount.AccountTypeCreditCard
			accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}

			invoices := &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{
				"card-1": {{ID: "inv-1", BankAccountID: "card-1", Amount: tc.stored, Status: invoice.StatusClosed}},
			}}
			txs := &invariantTxRepo{
				balances:    map[string]float64{"card-1": 0},
				invoiceSums: map[string]float64{"inv-1": tc.computed},
			}

			uc := NewCheckInvariantsUseCase(accounts, txs, invoices)
			result, _ := uc.Execute()

			if len(result.InvoiceDrifts) != 1 {
				t.Fatalf("expected 1 invoice drift, got %d", len(result.InvoiceDrifts))
			}
			if got := result.InvoiceDrifts[0].Drift; got != tc.expectedDrift {
				t.Errorf("Drift = %v, want %v", got, tc.expectedDrift)
			}
			if result.OK {
				t.Error("OK = true while a closed invoice disagrees with its charges")
			}
		})
	}
}

// A closed account's balance should still equal its ledger, so it keeps gating
// the check. Carrying the flag in the report is what lets a reader tell at a
// glance whether a drift is worth chasing — which only works if the flag
// reports the account's real state, both ways.
func TestCheckInvariants_ReportsWhetherTheAccountIsStillActive(t *testing.T) {
	for _, active := range []bool{true, false} {
		name := "active"
		if !active {
			name = "deactivated"
		}
		t.Run(name, func(t *testing.T) {
			account := invariantCheckingAccount("acc-1", 0, 500)
			account.IsActive = active
			accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{account}}
			txs := &invariantTxRepo{balances: map[string]float64{"acc-1": 100}}

			uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{})
			result, _ := uc.Execute()

			if len(result.AccountDrifts) != 1 {
				t.Fatalf("expected the account to be reported, got %d", len(result.AccountDrifts))
			}
			if result.AccountDrifts[0].IsActive != active {
				t.Errorf("IsActive = %v, want %v", result.AccountDrifts[0].IsActive, active)
			}
			if result.OK {
				t.Error("OK = true: a balance must match its ledger whether or not the account is open")
			}
		})
	}
}
