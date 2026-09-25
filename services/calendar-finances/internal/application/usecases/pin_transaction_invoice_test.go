package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func pinFixture(t *testing.T) (*PinTransactionInvoiceUseCase, *fakeTransactionRepo, *fakeInvoiceRepo) {
	t.Helper()
	const cardID, profileID = "card", "personal"
	card := creditCardAccountWith(profileID, cardID, 9, 14)

	august := &invoice.Invoice{
		ID: "inv-august", BankAccountID: cardID,
		OpeningDate: day(2026, 7, 10), ClosingDate: day(2026, 8, 9),
		DueDate: day(2026, 8, 14), ReferenceDate: day(2026, 8, 1), Status: invoice.StatusPaid,
	}
	september := &invoice.Invoice{
		ID: "inv-september", BankAccountID: cardID,
		OpeningDate: day(2026, 8, 9), ClosingDate: day(2026, 9, 9),
		DueDate: day(2026, 9, 14), ReferenceDate: day(2026, 9, 1), Status: invoice.StatusOpen,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		august.ID: august, september.ID: september}}

	// A late fee posted on 10/08 — the first day of the September cycle — that the
	// issuer charged on the AUGUST bill, because that is the bill that generated it.
	fee := &transaction.Transaction{
		ID: "fee", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 36.23, Currency: "BRL", Description: "Multa por atraso",
		OccurredOn: day(2026, 8, 10), InvoiceID: &september.ID,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{fee}}

	uc := NewPinTransactionInvoiceUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo, invRepo,
	)
	return uc, txRepo, invRepo
}

// The whole point: a charge the issuer billed on a different cycle than its date
// implies can be filed where it was actually billed, WITHOUT touching the date.
func TestPin_MovesTheChargeToTheBillItWasActuallyOn(t *testing.T) {
	uc, txRepo, _ := pinFixture(t)

	if err := uc.Execute("fee", "inv-august"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := txRepo.created[0]
	if got.InvoiceID == nil || *got.InvoiceID != "inv-august" {
		t.Fatalf("invoiceID = %v, want inv-august", got.InvoiceID)
	}
	if !got.InvoicePinned {
		t.Fatal("the choice was not marked as explicit: the next date-based derivation will undo it")
	}
	if !got.OccurredOn.Equal(day(2026, 8, 10)) {
		t.Fatal("the date was changed; the date came from the statement and was never wrong")
	}
}

// A pin that any later edit silently undoes is not a pin.
func TestPin_SurvivesADateBasedReassignment(t *testing.T) {
	uc, txRepo, invRepo := pinFixture(t)
	if err := uc.Execute("fee", "inv-august"); err != nil {
		t.Fatalf("pin: %v", err)
	}

	update := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
			"card": creditCardAccountWith("personal", "card", 9, 14)}},
		&fakeCategoryRepo{}, txRepo, invRepo, &noopBalanceRecalculator{},
	)
	// Same card, date moved by a day — enough to trigger the reassignment.
	if _, err := update.Execute("fee", UpdateTransactionInput{
		BankAccountID: "card", Type: "EXPENSE", Amount: 36.23, Currency: "BRL",
		Description: "Multa por atraso", OccurredOn: "2026-08-11",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := txRepo.created[0]; got.InvoiceID == nil || *got.InvoiceID != "inv-august" {
		t.Fatalf("invoiceID = %v after an edit: the pin was silently undone", got.InvoiceID)
	}
}

// Unpinning hands the row back to the date rule.
func TestPin_CanBeReleased(t *testing.T) {
	uc, txRepo, _ := pinFixture(t)
	if err := uc.Execute("fee", "inv-august"); err != nil {
		t.Fatalf("pin: %v", err)
	}
	if err := uc.Release("fee"); err != nil {
		t.Fatalf("release: %v", err)
	}
	got := txRepo.created[0]
	if got.InvoicePinned {
		t.Fatal("still pinned")
	}
	if got.InvoiceID == nil || *got.InvoiceID != "inv-september" {
		t.Fatalf("invoiceID = %v; releasing must hand the row back to the date rule", got.InvoiceID)
	}
}

// A bill from another card is not a choice, it is a mistake.
func TestPin_RefusesAnInvoiceFromAnotherCard(t *testing.T) {
	uc, _, invRepo := pinFixture(t)
	invRepo.invoices["inv-other"] = &invoice.Invoice{
		ID: "inv-other", BankAccountID: "outro-cartao",
		OpeningDate: day(2026, 7, 10), ClosingDate: day(2026, 8, 9),
		DueDate: day(2026, 8, 14), ReferenceDate: day(2026, 8, 1), Status: invoice.StatusOpen,
	}
	if err := uc.Execute("fee", "inv-other"); err == nil {
		t.Fatal("a charge was filed onto another card's bill")
	}
}

// An invoice payment is not a line of a bill, and pinning one would file it in.
func TestPin_RefusesAnInvoicePayment(t *testing.T) {
	uc, txRepo, _ := pinFixture(t)
	paid := "inv-august"
	txRepo.created[0].PaidInvoiceID = &paid

	if err := uc.Execute("fee", "inv-august"); err == nil {
		t.Fatal("a payment was filed into a bill: the bill now reads smaller instead of paid")
	}
}

func TestPin_RefusesWhatIsNotOnACard(t *testing.T) {
	const profileID = "personal"
	checking := &bankaccount.BankAccount{ID: "checking", ProfileID: profileID, Name: "MP",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	tx := &transaction.Transaction{ID: "t", ProfileID: profileID, BankAccountID: "checking",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 10, Currency: "BRL", Description: "conta", OccurredOn: day(2026, 8, 10)}
	uc := NewPinTransactionInvoiceUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"checking": checking}},
		&fakeTransactionRepo{created: []*transaction.Transaction{tx}},
		&fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}},
	)
	if err := uc.Execute("t", "qualquer"); !errors.Is(err, ErrNotACreditCard) {
		t.Fatalf("err = %v, want ErrNotACreditCard", err)
	}
}
