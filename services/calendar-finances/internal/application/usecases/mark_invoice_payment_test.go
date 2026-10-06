package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func paymentFixture(t *testing.T) (*MarkInvoicePaymentUseCase, *fakeTransactionRepo, *fakeInvoiceRepo) {
	t.Helper()
	const cardID, profileID = "card", "personal"
	card := creditCardAccountWith(profileID, cardID, 27, 3)

	bill := &invoice.Invoice{
		ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2025, 11, 27), ClosingDate: day(2025, 12, 27),
		DueDate: day(2026, 1, 3), ReferenceDate: day(2025, 12, 1),
		Status: invoice.StatusPaid, Amount: -2566.39,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{bill.ID: bill}}

	// A payment that was filed as a LINE of the bill, which is what made the total
	// negative. It predates PaidInvoiceID being written, so nothing says what it is.
	pagamento := &transaction.Transaction{
		ID: "pagamento", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 2693.73, Currency: "BRL", Description: "Pagamento fatura Cartao Pessoal Nubank",
		OccurredOn: day(2025, 12, 3), InvoiceID: &bill.ID,
	}
	compra := &transaction.Transaction{
		ID: "compra", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 127.34, Currency: "BRL", Description: "iFood",
		OccurredOn: day(2025, 12, 10), InvoiceID: &bill.ID,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{pagamento, compra}}

	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo, invRepo,
	)
	return uc, txRepo, invRepo
}

// The whole point: a payment settles a bill, it is not a line of one. Marking it
// moves it OUT of the bill's contents and INTO what the bill has been paid.
func TestMarkPayment_MovesItOutOfTheBillAndIntoWhatWasPaid(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)

	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := txRepo.created[0]
	if p.PaidInvoiceID == nil || *p.PaidInvoiceID != "inv" {
		t.Fatalf("paidInvoiceID = %v, want inv", p.PaidInvoiceID)
	}
	if p.InvoiceID != nil {
		t.Fatal("still filed as a line of the bill: the total keeps counting it as a credit")
	}
	bill := invRepo.invoices["inv"]
	if bill.PaidAmount == nil || *bill.PaidAmount != 2693.73 {
		t.Fatalf("paidAmount = %v, want 2693.73", bill.PaidAmount)
	}
	// The bill is worth what was charged on it. Counting the payment as a line is
	// what made it -2.566,39 instead of the 127,34 actually spent.
	if bill.Amount != 127.34 {
		t.Fatalf("bill amount = %.2f, want 127.34", bill.Amount)
	}
}

// Marking must not swallow the bill's real contents. The purchase stays.
func TestMarkPayment_LeavesTheRealLinesAlone(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, txn := range txRepo.created {
		if txn.ID == "compra" {
			if txn.InvoiceID == nil || *txn.InvoiceID != "inv" {
				t.Fatal("a purchase was detached from the bill it belongs to")
			}
			if txn.PaidInvoiceID != nil {
				t.Fatal("a purchase was marked as a payment")
			}
		}
	}
}

// A payment credits the card. An expense is a purchase, and calling one a payment
// would take a real charge off the bill.
func TestMarkPayment_RefusesAnExpense(t *testing.T) {
	uc, _, _ := paymentFixture(t)
	if err := uc.Execute("compra", "inv"); err == nil {
		t.Fatal("a purchase was accepted as a payment: the bill just lost a real charge")
	}
}

// A bill from another card is not a choice, it is a mistake.
func TestMarkPayment_RefusesABillFromAnotherCard(t *testing.T) {
	uc, _, invRepo := paymentFixture(t)
	invRepo.invoices["outra"] = &invoice.Invoice{
		ID: "outra", BankAccountID: "outro-cartao",
		OpeningDate: day(2025, 11, 27), ClosingDate: day(2025, 12, 27),
		DueDate: day(2026, 1, 3), ReferenceDate: day(2025, 12, 1), Status: invoice.StatusOpen,
	}
	if err := uc.Execute("pagamento", "outra"); err == nil {
		t.Fatal("a payment was pointed at another card's bill")
	}
}

func TestMarkPayment_RefusesWhatIsNotOnACard(t *testing.T) {
	const profileID = "personal"
	checking := &bankaccount.BankAccount{ID: "conta", ProfileID: profileID, Name: "MP",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	tx := &transaction.Transaction{ID: "t", ProfileID: profileID, BankAccountID: "conta",
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 10, Currency: "BRL", Description: "x", OccurredOn: day(2026, 1, 1)}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"conta": checking}},
		&fakeTransactionRepo{created: []*transaction.Transaction{tx}},
		&fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}},
	)
	if err := uc.Execute("t", "inv"); !errors.Is(err, ErrNotACreditCard) {
		t.Fatalf("err = %v, want ErrNotACreditCard", err)
	}
}

// Running it twice must change nothing the second time: the repair has to be safe
// to re-run after a partial failure.
func TestMarkPayment_IsIdempotent(t *testing.T) {
	uc, _, invRepo := paymentFixture(t)
	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("first: %v", err)
	}
	first := invRepo.invoices["inv"].PaidAmount
	if first == nil {
		t.Fatal("nothing was recorded as paid")
	}
	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("second: %v", err)
	}
	got := invRepo.invoices["inv"].PaidAmount
	if got == nil || *got != *first {
		t.Fatalf("paidAmount went from %.2f to %v: the payment was counted twice", *first, got)
	}
}

// Restating the total must not count a reversed charge. The fake used to drop only
// CANCELLED while the repository drops REVERSED too, so this rule held in the
// database and not in the tests that claimed to check it.
func TestMarkPayment_ReversedChargesDoNotCountTowardsTheBill(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)
	txRepo.created = append(txRepo.created, &transaction.Transaction{
		ID: "estornada", ProfileID: "personal", BankAccountID: "card",
		Type: transaction.TypeExpense, Status: transaction.StatusReversed,
		Amount: 999.99, Currency: "BRL", Description: "compra estornada",
		OccurredOn: day(2025, 12, 11), InvoiceID: strPtr("inv"),
	})

	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := invRepo.invoices["inv"].Amount; got != 127.34 {
		t.Fatalf("bill amount = %.2f, want 127.34: a reversed charge is being billed", got)
	}
}

// A refund marked as a payment by mistake has to be able to go back to being a line
// of the bill without anyone editing a column by hand.
func TestMarkPayment_ReleaseFilesItBackOntoTheBillForItsDate(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)
	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if err := uc.Release("pagamento"); err != nil {
		t.Fatalf("releasing: %v", err)
	}

	p := txRepo.created[0]
	if p.PaidInvoiceID != nil {
		t.Fatal("still recorded as settling the bill")
	}
	if p.InvoiceID == nil || *p.InvoiceID != "inv" {
		t.Fatalf("invoiceID = %v, want the bill covering 03/12", p.InvoiceID)
	}
	bill := invRepo.invoices["inv"]
	if bill.PaidAmount != nil {
		t.Fatalf("paidAmount = %v, want nothing: the payment was taken back", bill.PaidAmount)
	}
	if bill.Amount != -2566.39 {
		t.Fatalf("bill amount = %.2f, want the original -2566.39", bill.Amount)
	}
}

// Releasing something that was never marked is not an error, and must not go on to
// rewrite the bill it happens to sit on.
func TestMarkPayment_ReleaseOfAnUnmarkedChargeChangesNothing(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	before := txRepo.updates
	if err := uc.Release("compra"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txRepo.updates != before {
		t.Fatal("an unmarked charge was rewritten")
	}
}

// A pin says "file this line on that bill". Once the credit is a payment, the
// instruction has no subject left, and a surviving pin would make a later re-derive
// refuse to touch it.
func TestMarkPayment_ClearsAPinLeftOnTheCredit(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	txRepo.created[0].InvoicePinned = true

	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txRepo.created[0].InvoicePinned {
		t.Fatal("the credit is still pinned as a line of a bill it no longer belongs to")
	}
}

// The normal case, not an edge one: a bill closes on 27/12 and is paid on 03/01,
// which falls inside the NEXT cycle. So the bill a payment settles and the bill its
// date lands on are routinely two different bills, and releasing it has to restate
// both — the one that loses the payment and the one that gains the line.
func TestMarkPayment_ReleaseRestatesBothBillsWhenTheyDiffer(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)
	next := &invoice.Invoice{
		ID: "proxima", BankAccountID: "card",
		OpeningDate: day(2025, 12, 27), ClosingDate: day(2026, 1, 27),
		DueDate: day(2026, 2, 3), ReferenceDate: day(2026, 1, 1),
		Status: invoice.StatusOpen,
	}
	invRepo.invoices[next.ID] = next
	// Paid on 03/01: inside the next cycle, settling the one that closed on 27/12.
	txRepo.created[0].OccurredOn = day(2026, 1, 3)
	txRepo.created[0].InvoiceID = strPtr("proxima")

	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if got := invRepo.invoices["inv"].PaidAmount; got == nil || *got != 2693.73 {
		t.Fatalf("settled bill paidAmount = %v, want 2693.73", got)
	}

	if err := uc.Release("pagamento"); err != nil {
		t.Fatalf("releasing: %v", err)
	}
	if got := invRepo.invoices["inv"].PaidAmount; got != nil {
		t.Fatalf("the bill it settled still reads paid %v", got)
	}
	if got := invRepo.invoices["proxima"].Amount; got != -2693.73 {
		t.Fatalf("the bill for the date reads %.2f, want -2693.73: it never got restated", got)
	}
	if id := txRepo.created[0].InvoiceID; id == nil || *id != "proxima" {
		t.Fatalf("invoiceID = %v, want proxima", id)
	}
}
