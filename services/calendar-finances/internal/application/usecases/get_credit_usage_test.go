package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type usageInvoiceRepo struct {
	fakeInvoiceRepo
	list []*invoice.Invoice
}

func (r *usageInvoiceRepo) FindByBankAccountID(string) ([]*invoice.Invoice, error) {
	return r.list, nil
}

type usageTxRepo struct {
	mockTransactionRepo
	txs        []*transaction.Transaction
	lastFilter transaction.ListFilter
}

func (r *usageTxRepo) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	r.lastFilter = filter
	return r.txs, nil
}

// Mirrors the real repository: the bill is worth what its transactions say, not what
// the invoice row's amount column happens to hold.
func (r *usageTxRepo) SumByInvoiceID(invoiceID string) (float64, error) {
	total := 0.0
	for _, tx := range r.txs {
		if tx.InvoiceID != nil && *tx.InvoiceID == invoiceID && tx.Status == transaction.StatusConfirmed {
			total += tx.Amount
		}
	}
	return total, nil
}

func card(limit float64) *bankaccount.BankAccount {
	return &bankaccount.BankAccount{
		ID: "card", ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard, CreditLimit: &limit,
	}
}

// The real card that started this: R$400 limit, one closed bill part-paid and rolled
// into instalments, and the open cycle. The bank shows R$19,77 available; the balance
// field would have said something else entirely, because nothing updates it as
// purchases happen.
func TestGetCreditUsage_SumsWhatIsStillOwedOnTheInvoices(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card(400)}}
	invoiceRepo := &usageInvoiceRepo{list: []*invoice.Invoice{
		{ID: "closed", BankAccountID: "card", Status: invoice.StatusClosed},
		{ID: "open", BankAccountID: "card", Status: invoice.StatusOpen},
		{ID: "old", BankAccountID: "card", Status: invoice.StatusPaid},
	}}
	txRepo := &usageTxRepo{txs: []*transaction.Transaction{
		charge("closed", 253.48), charge("open", 126.75), charge("old", 2702.66),
	}}

	uc := NewGetCreditUsageUseCase(accountRepo, invoiceRepo, txRepo)

	usage, err := uc.Execute("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.Outstanding != 380.23 {
		t.Errorf("expected 380.23 owed across the unpaid invoices, got %.2f", usage.Outstanding)
	}
	if usage.Available != 19.77 {
		t.Errorf("expected 19.77 available, got %.2f", usage.Available)
	}
}

// A purchase that never joined an invoice is still owed. Two such charges exist on
// the real Mercado Pago card; leaving them out understates the debt.
func TestGetCreditUsage_CountsChargesWithNoInvoice(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card(1000)}}
	invoiceRepo := &usageInvoiceRepo{list: []*invoice.Invoice{
		{ID: "open", BankAccountID: "card", Status: invoice.StatusOpen},
	}}
	txRepo := &usageTxRepo{txs: []*transaction.Transaction{
		{BankAccountID: "card", Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
			Amount: 116.03, Description: "Aporte WB Digital (via cartao MP)", OccurredOn: time.Now()},
		charge("open", 100),
	}}

	uc := NewGetCreditUsageUseCase(accountRepo, invoiceRepo, txRepo)

	usage, err := uc.Execute("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.Outstanding != 216.03 {
		t.Fatalf("expected 100 of invoiced debt plus the 116.03 orphan charge, got %.2f", usage.Outstanding)
	}
}

func TestGetCreditUsage_RejectsAnAccountThatIsNotACard(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"checking": {ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking},
	}}

	uc := NewGetCreditUsageUseCase(accountRepo, &usageInvoiceRepo{}, &usageTxRepo{})

	if _, err := uc.Execute("checking"); err == nil {
		t.Fatal("expected an error: a checking account has no credit limit to report on")
	}
}

func TestGetCreditUsage_UnknownAccount(t *testing.T) {
	uc := NewGetCreditUsageUseCase(&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{}},
		&usageInvoiceRepo{}, &usageTxRepo{})

	if _, err := uc.Execute("nope"); err == nil {
		t.Fatal("expected an error for an unknown account")
	}
}

func charge(invoiceID string, amount float64) *transaction.Transaction {
	return &transaction.Transaction{
		BankAccountID: "card", Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: amount, InvoiceID: strPtr(invoiceID), Description: "Compra", OccurredOn: time.Now(),
	}
}

// The repository refuses a query with no profile. Nothing asserted this, so the route
// returned 500 on every call while the suite stayed green.
func TestGetCreditUsage_QueriesWithinTheProfile(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card(400)}}
	txRepo := &usageTxRepo{}

	uc := NewGetCreditUsageUseCase(accountRepo, &usageInvoiceRepo{}, txRepo)
	if _, err := uc.Execute("card"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if txRepo.lastFilter.ProfileID != "p1" {
		t.Errorf("expected the account's profile in the filter, got %q", txRepo.lastFilter.ProfileID)
	}
	if txRepo.lastFilter.BankAccountID == nil || *txRepo.lastFilter.BankAccountID != "card" {
		t.Error("expected the query scoped to this card")
	}
}

// Only confirmed spending is owed. A planned charge may still be cancelled, and money
// arriving on the card reduces the debt rather than adding to it.
func TestGetCreditUsage_IgnoresPlannedAndIncomingOnOrphanCharges(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card(1000)}}
	txRepo := &usageTxRepo{txs: []*transaction.Transaction{
		{BankAccountID: "card", Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
			Amount: 500, Description: "Assinatura futura", OccurredOn: time.Now()},
		{BankAccountID: "card", Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
			Amount: 300, Description: "Estorno", OccurredOn: time.Now()},
	}}

	uc := NewGetCreditUsageUseCase(accountRepo, &usageInvoiceRepo{}, txRepo)
	usage, err := uc.Execute("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.Outstanding != 0 {
		t.Fatalf("expected nothing owed from a planned charge or a refund, got %.2f", usage.Outstanding)
	}
}

// A charge already counted through its invoice must not be counted again.
func TestGetCreditUsage_DoesNotDoubleCountAnInvoicedCharge(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card(1000)}}
	invoiceRepo := &usageInvoiceRepo{list: []*invoice.Invoice{
		{ID: "open", BankAccountID: "card", Status: invoice.StatusOpen},
	}}
	txRepo := &usageTxRepo{txs: []*transaction.Transaction{charge("open", 200)}}

	uc := NewGetCreditUsageUseCase(accountRepo, invoiceRepo, txRepo)
	usage, err := uc.Execute("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.Outstanding != 200 {
		t.Fatalf("expected the charge counted once, got %.2f", usage.Outstanding)
	}
}
