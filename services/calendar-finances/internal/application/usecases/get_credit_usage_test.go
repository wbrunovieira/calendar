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
	txs []*transaction.Transaction
}

func (r *usageTxRepo) List(transaction.ListFilter) ([]*transaction.Transaction, error) {
	return r.txs, nil
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
		{ID: "closed", BankAccountID: "card", Status: invoice.StatusClosed, Amount: 253.48},
		{ID: "open", BankAccountID: "card", Status: invoice.StatusOpen, Amount: 126.75},
		{ID: "old", BankAccountID: "card", Status: invoice.StatusPaid, Amount: 2702.66},
	}}

	uc := NewGetCreditUsageUseCase(accountRepo, invoiceRepo, &usageTxRepo{})

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
		{ID: "open", BankAccountID: "card", Status: invoice.StatusOpen, Amount: 100},
	}}
	txRepo := &usageTxRepo{txs: []*transaction.Transaction{
		{BankAccountID: "card", Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
			Amount: 116.03, Description: "Aporte WB Digital (via cartao MP)", OccurredOn: time.Now()},
		{BankAccountID: "card", Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
			Amount: 50, InvoiceID: strPtr("open"), Description: "Mercado", OccurredOn: time.Now()},
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
