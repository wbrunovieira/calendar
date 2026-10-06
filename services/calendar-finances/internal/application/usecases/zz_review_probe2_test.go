package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// PROBE 6: the CANONICAL card-side shape. A payment made on 03/08 is filed by the
// date rule as a LINE of the August bill (opening 27/07) and it SETTLES the July bill
// (closing 27/07). Execute detaches it from August and never restates August.
func TestProbe_TheBillItWasALineOfIsNeverRestated(t *testing.T) {
	const cardID = "card"
	card := creditCardAccountWith("wb", cardID, 27, 3)

	julho := &invoice.Invoice{ID: "julho", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27),
		DueDate: day(2026, 8, 6), ReferenceDate: day(2026, 7, 1),
		Status: invoice.StatusClosed, Amount: 1000.00}
	agosto := &invoice.Invoice{ID: "agosto", BankAccountID: cardID,
		OpeningDate: day(2026, 7, 27), ClosingDate: day(2026, 8, 27),
		DueDate: day(2026, 9, 3), ReferenceDate: day(2026, 8, 1),
		Status: invoice.StatusClosed, Amount: 300.00}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"julho": julho, "agosto": agosto}}

	compraJulho := &transaction.Transaction{ID: "cj", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1000.00, OccurredOn: day(2026, 7, 10), InvoiceID: &julho.ID}
	compraAgosto := &transaction.Transaction{ID: "ca", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1300.00, OccurredOn: day(2026, 8, 10), InvoiceID: &agosto.ID}
	// the payment of the July bill, dated 03/08 -> the date rule filed it on AGOSTO,
	// where SumByInvoiceID counts it as -1000. 1300 - 1000 = 300 = agosto.Amount.
	pagamento := &transaction.Transaction{ID: "pg", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 1000.00, OccurredOn: day(2026, 8, 3), InvoiceID: &agosto.ID}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{compraJulho, compraAgosto, pagamento}}

	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}}, txRepo, invRepo)

	if err := uc.Execute("pg", "julho"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	a := invRepo.invoices["agosto"]
	lines, _ := txRepo.SumByInvoiceID("agosto")
	t.Logf("AGOSTO stored amount=%.2f, its lines now sum to %.2f  (drift %.2f)", a.Amount, lines, lines-a.Amount)
	j := invRepo.invoices["julho"]
	var paid float64
	if j.PaidAmount != nil {
		paid = *j.PaidAmount
	}
	t.Logf("JULHO amount=%.2f paid=%.2f status=%s", j.Amount, paid, j.Status)
}

// PROBE 7: a funding leg whose invoice_id points at ANOTHER bill of the card.
func TestProbe_FundingLegDetachedFromAnotherBillSilently(t *testing.T) {
	const cardID, contaID = "card", "conta"
	card := creditCardAccountWith("wb", cardID, 27, 3)
	conta := &bankaccount.BankAccount{ID: contaID, ProfileID: "wb",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}

	julho := &invoice.Invoice{ID: "julho", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27),
		DueDate: day(2026, 8, 6), ReferenceDate: day(2026, 7, 1),
		Status: invoice.StatusClosed, Amount: 1000.00}
	agosto := &invoice.Invoice{ID: "agosto", BankAccountID: cardID,
		OpeningDate: day(2026, 7, 27), ClosingDate: day(2026, 8, 27),
		DueDate: day(2026, 9, 3), ReferenceDate: day(2026, 8, 1),
		Status: invoice.StatusClosed, Amount: 2000.00}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"julho": julho, "agosto": agosto}}

	cj := &transaction.Transaction{ID: "cj", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1000.00, OccurredOn: day(2026, 7, 10), InvoiceID: &julho.ID}
	ca := &transaction.Transaction{ID: "ca", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1000.00, OccurredOn: day(2026, 8, 10), InvoiceID: &agosto.ID}
	// a leg carrying a stale invoice_id on AGOSTO (counted +1000 there: 1000+1000=2000)
	perna := &transaction.Transaction{ID: "perna", ProfileID: "wb", BankAccountID: contaID,
		DestinationAccountID: strPtr(cardID), InvoiceID: &agosto.ID,
		Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount:               1000.00, OccurredOn: day(2026, 8, 3)}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{cj, ca, perna}}

	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card, contaID: conta}},
		txRepo, invRepo)

	if err := uc.Execute("perna", "julho"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	a := invRepo.invoices["agosto"]
	lines, _ := txRepo.SumByInvoiceID("agosto")
	t.Logf("AGOSTO stored=%.2f lines=%.2f drift=%.2f", a.Amount, lines, lines-a.Amount)
}

// PROBE 8: legitimate partial payments must still be allowed (minimum, then the rest).
func TestProbe_LegitimatePartialsStillAllowed(t *testing.T) {
	const cardID, contaID = "card", "conta"
	card := creditCardAccountWith("wb", cardID, 27, 3)
	conta := &bankaccount.BankAccount{ID: contaID, ProfileID: "wb",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	bill := &invoice.Invoice{ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27),
		DueDate: day(2026, 8, 6), ReferenceDate: day(2026, 7, 1),
		Status: invoice.StatusClosed, Amount: 1018.18}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	compra := &transaction.Transaction{ID: "compra", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1018.18, OccurredOn: day(2026, 7, 10), InvoiceID: &bill.ID}
	minimo := &transaction.Transaction{ID: "minimo", ProfileID: "wb", BankAccountID: contaID,
		DestinationAccountID: strPtr(cardID), Type: transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed, Amount: 200.00, OccurredOn: day(2026, 8, 3)}
	resto := &transaction.Transaction{ID: "resto", ProfileID: "wb", BankAccountID: contaID,
		DestinationAccountID: strPtr(cardID), Type: transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed, Amount: 818.18, OccurredOn: day(2026, 8, 20)}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{compra, minimo, resto}}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card, contaID: conta}},
		txRepo, invRepo)
	e1 := uc.Execute("minimo", "inv")
	b := invRepo.invoices["inv"]
	t.Logf("after minimum: err=%v status=%s paid=%v", e1, b.Status, b.PaidAmount)
	e2 := uc.Execute("resto", "inv")
	b = invRepo.invoices["inv"]
	t.Logf("after rest:    err=%v status=%s paid=%v", e2, b.Status, b.PaidAmount)
}

// PROBE 9: reversing a funding leg -- does the bill stop reading PAID?
func TestProbe_ReversedFundingLegRestatesTheBill(t *testing.T) {
	const cardID, contaID = "card", "conta"
	card := creditCardAccountWith("wb", cardID, 27, 3)
	conta := &bankaccount.BankAccount{ID: contaID, ProfileID: "wb",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	bill := &invoice.Invoice{ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27),
		DueDate: day(2026, 8, 6), ReferenceDate: day(2026, 7, 1),
		Status: invoice.StatusClosed, Amount: 1018.18}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	compra := &transaction.Transaction{ID: "compra", ProfileID: "wb", BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1018.18, OccurredOn: day(2026, 7, 10), InvoiceID: &bill.ID}
	perna := &transaction.Transaction{ID: "perna", ProfileID: "wb", BankAccountID: contaID,
		DestinationAccountID: strPtr(cardID), Type: transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed, Amount: 1018.18, OccurredOn: day(2026, 8, 3)}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{compra, perna}}
	accRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card, contaID: conta}}
	uc := NewMarkInvoicePaymentUseCase(accRepo, txRepo, invRepo)
	if err := uc.Execute("perna", "inv"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	t.Logf("after marking: status=%s paid=%v", invRepo.invoices["inv"].Status, invRepo.invoices["inv"].PaidAmount)

	del := NewDeleteTransactionUseCase(txRepo, accRepo, nil)
	del.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))
	if err := del.ExecuteWithReason(ReverseTransactionInput{
		ID: "perna", Reason: transaction.ReasonNeverHappened, By: "review", Note: "probe"}); err != nil {
		t.Fatalf("reversing: %v", err)
	}
	b := invRepo.invoices["inv"]
	t.Logf("after reversing the leg: status=%s paid=%v amount=%.2f remaining=%.2f",
		b.Status, b.PaidAmount, b.Amount, b.AmountRemaining())
}
