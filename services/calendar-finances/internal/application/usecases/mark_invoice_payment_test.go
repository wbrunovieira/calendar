package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// The fixture is the shape the Nubank PF cycle 27/11..27/12 actually has in
// production: charges of 2.693,73 and, filed as a LINE of the same bill, the credit
// of 2.693,73 that paid it. The two cancel out, so the bill stores 0,00 -- a bill
// claiming nothing is owed, on a cycle that was charged 2.693,73 and paid 2.693,73.
func paymentFixture(t *testing.T) (*MarkInvoicePaymentUseCase, *fakeTransactionRepo, *fakeInvoiceRepo) {
	t.Helper()
	const cardID, profileID = "card", "personal"
	card := creditCardAccountWith(profileID, cardID, 27, 3)

	bill := &invoice.Invoice{
		ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2025, 11, 27), ClosingDate: day(2025, 12, 27),
		DueDate: day(2026, 1, 5), ReferenceDate: day(2025, 12, 1),
		Status: invoice.StatusPaid, Amount: 0,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{bill.ID: bill}}

	// The payment, filed as a line of the bill it settled. It predates
	// paid_invoice_id being written, so nothing in the row says what it is.
	pagamento := &transaction.Transaction{
		ID: "pagamento", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 2693.73, Currency: "BRL", Description: "Pagamento fatura Cartao Pessoal Nubank",
		OccurredOn: day(2025, 12, 3), InvoiceID: &bill.ID,
	}
	// The cycle's real charges, which must come through untouched.
	compra := &transaction.Transaction{
		ID: "compra", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 2566.39, Currency: "BRL", Description: "compras do ciclo",
		OccurredOn: day(2025, 12, 10), InvoiceID: &bill.ID,
	}
	outra := &transaction.Transaction{
		ID: "outra", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 127.34, Currency: "BRL", Description: "iFood",
		OccurredOn: day(2025, 12, 11), InvoiceID: &bill.ID,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{pagamento, compra, outra}}

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
	// The bill is worth what was CHARGED on it. Counting the payment as a line is
	// what made it store 0,00 for a cycle that was charged 2.693,73.
	if bill.Amount != 2693.73 {
		t.Fatalf("bill amount = %.2f, want 2693.73", bill.Amount)
	}
	if bill.Status != invoice.StatusPaid {
		t.Fatalf("status = %s, want PAID", bill.Status)
	}
}

// Marking must not swallow the bill's real contents. The purchase stays.
func TestMarkPayment_LeavesTheRealLinesAlone(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, txn := range txRepo.created {
		if txn.ID == "compra" || txn.ID == "outra" {
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
	if got := invRepo.invoices["inv"].Amount; got != 2693.73 {
		t.Fatalf("bill amount = %.2f, want 2693.73: a reversed charge is being billed", got)
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
	if bill.Amount != 0 {
		t.Fatalf("bill amount = %.2f, want the original 0.00", bill.Amount)
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

// A pin says "file this LINE on that bill". Once the credit is a payment it is not a
// line of anything, so the instruction has no subject left. What protects it from a
// later re-derivation is paid_invoice_id, which UpdateTransaction now exempts.
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
	// With the payment filed on the NEXT cycle, this one stores only its charges and
	// that one stores the credit sitting in it -- each agreeing with its own lines,
	// which is what restating requires.
	invRepo.invoices["inv"].Amount = 2693.73
	next.Amount = -2693.73

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
	// Its own lines never changed, so its total must not either.
	if got := invRepo.invoices["inv"].Amount; got != 2693.73 {
		t.Fatalf("the bill it settled reads %.2f, want 2693.73", got)
	}
	if id := txRepo.created[0].InvoiceID; id == nil || *id != "proxima" {
		t.Fatalf("invoiceID = %v, want proxima", id)
	}
}

// Re-pointing a payment at the bill it really settled must take it off the bill it
// no longer settles. Restating only the new one left the old bill reading PAID with
// no payment behind it -- R$ 2.693,73 of debt simply gone from the ledger, and
// GetCreditUsage handing back that much phantom limit.
func TestMarkPayment_RepointTakesItOffTheBillItNoLongerSettles(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)
	previous := &invoice.Invoice{
		ID: "anterior", BankAccountID: "card",
		OpeningDate: day(2025, 10, 27), ClosingDate: day(2025, 11, 27),
		DueDate: day(2025, 12, 3), ReferenceDate: day(2025, 11, 1),
		Status: invoice.StatusClosed, Amount: 2693.73,
	}
	invRepo.invoices[previous.ID] = previous
	txRepo.created = append(txRepo.created, &transaction.Transaction{
		ID: "compra-anterior", ProfileID: "personal", BankAccountID: "card",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 2693.73, Currency: "BRL", Description: "compras do ciclo anterior",
		OccurredOn: day(2025, 11, 10), InvoiceID: strPtr("anterior"),
	})

	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if err := uc.Execute("pagamento", "anterior"); err != nil {
		t.Fatalf("re-pointing: %v", err)
	}

	if got := invRepo.invoices["anterior"].PaidAmount; got == nil || *got != 2693.73 {
		t.Fatalf("the bill it settles reads paid %v, want 2693.73", got)
	}
	if got := invRepo.invoices["inv"].PaidAmount; got != nil {
		t.Fatalf("the bill it no longer settles still reads paid %v", got)
	}
	if got := invRepo.invoices["inv"].Status; got == invoice.StatusPaid {
		t.Fatal("the bill it no longer settles is still PAID with nothing behind it")
	}
}

// A payment that has not happened cannot settle anything. SumByInvoiceID counts a
// PLANNED row while SumLivePaymentsByInvoiceID does not, so marking one raises the
// bill's total and records nothing as paid: the bill flips out of PAID and owes the
// whole amount again, from money that never moved.
func TestMarkPayment_RefusesAPaymentThatHasNotHappened(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	for _, st := range []transaction.Status{
		transaction.StatusPlanned, transaction.StatusCancelled, transaction.StatusReversed,
	} {
		txRepo.created[0].Status = st
		if err := uc.Execute("pagamento", "inv"); !errors.Is(err, ErrPaymentNotConfirmed) {
			t.Fatalf("status %s: err = %v, want ErrPaymentNotConfirmed", st, err)
		}
	}
}

// A cycle that is still accruing has no settled amount to record. RederiveStatus
// returns early on OPEN, so writing paid_amount there leaves the status untouched and
// AmountRemaining reporting 0 for a bill that is still collecting charges.
func TestMarkPayment_RefusesAnOpenCycle(t *testing.T) {
	uc, _, invRepo := paymentFixture(t)
	invRepo.invoices["inv"].Status = invoice.StatusOpen

	if err := uc.Execute("pagamento", "inv"); !errors.Is(err, ErrInvoiceStillOpen) {
		t.Fatalf("err = %v, want ErrInvoiceStillOpen", err)
	}
}

// The same payment reaching the bill twice -- once as the funding account's TRANSFER
// leg created by /invoices/{id}/pay, once as the card-side credit imported from the
// statement -- would record the bill as paid twice over and hand back that much
// phantom limit.
func TestMarkPayment_RefusesPayingMoreThanTheBillIsWorth(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	// The V2 payment leg already settles this bill in full.
	txRepo.created = append(txRepo.created, &transaction.Transaction{
		ID: "perna-v2", ProfileID: "personal", BankAccountID: "conta-corrente",
		Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 2693.73, Currency: "BRL", Description: "Pagamento fatura",
		OccurredOn: day(2025, 12, 3), PaidInvoiceID: strPtr("inv"),
	})
	if err := uc.Execute("pagamento", "inv"); !errors.Is(err, ErrPaymentExceedsInvoice) {
		t.Fatalf("err = %v, want ErrPaymentExceedsInvoice", err)
	}
}

// Restating a bill's TOTAL is only safe while its stored figure already agrees with
// the lines linked to it. On the Mercado Pago card a bill read 2.647,97 against the
// 2.702,66 the statement charged, because four late fees were filed on the wrong
// cycle -- overwriting the stored figure there replaces the statement number with the
// smaller one, on a settled historical bill, with no route back.
func TestMarkPayment_RefusesABillWhoseStoredTotalDisagreesWithItsLines(t *testing.T) {
	uc, _, invRepo := paymentFixture(t)
	invRepo.invoices["inv"].Amount = 54.69 // its lines sum to 0,00

	err := uc.Execute("pagamento", "inv")
	if !errors.Is(err, ErrInvoiceAmountOutOfSync) {
		t.Fatalf("err = %v, want ErrInvoiceAmountOutOfSync", err)
	}
	// And it refuses BEFORE writing: a half-applied repair is worse than none.
	if invRepo.invoices["inv"].Amount != 54.69 {
		t.Fatal("the bill was rewritten by a call that refused")
	}
}

// Release writes invoice_id, and SumByInvoiceID counts anything that is not INCOME as
// a charge -- so releasing a TRANSFER would add its amount to a bill as a purchase.
func TestMarkPayment_ReleaseRefusesWhatIsNotACredit(t *testing.T) {
	uc, txRepo, _ := paymentFixture(t)
	txRepo.created[0].Type = transaction.TypeTransfer
	txRepo.created[0].PaidInvoiceID = strPtr("inv")
	txRepo.created[0].InvoiceID = nil

	if err := uc.Release("pagamento"); !errors.Is(err, ErrNotAnInvoicePayment) {
		t.Fatalf("err = %v, want ErrNotAnInvoicePayment", err)
	}
}

// A marked payment must survive an ordinary edit. It is not a line of any bill, so
// there is no bill for its date to re-derive onto: re-filing it as a line while
// paid_invoice_id still names the bill it settled double-counts it -- the total drops
// by the payment and paid_amount keeps it. Marking clears the pin, so before this the
// payment carried no exemption at all and a one-day nudge to the date was enough.
func TestMarkPayment_SurvivesAnEditThatMovesItsDate(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)
	if err := uc.Execute("pagamento", "inv"); err != nil {
		t.Fatalf("marking: %v", err)
	}

	updater := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
			"card": creditCardAccountWith("personal", "card", 27, 3),
		}},
		&fakeCategoryRepo{},
		txRepo,
		invRepo,
		&noopBalanceRecalculator{},
	)
	// One day later -- nothing else about the row changes.
	if _, err := updater.Execute("pagamento", UpdateTransactionInput{
		BankAccountID: "card",
		Type:          string(transaction.TypeIncome),
		Status:        strPtr(string(transaction.StatusConfirmed)),
		Amount:        2693.73,
		Currency:      "BRL",
		Description:   "Pagamento fatura Cartao Pessoal Nubank",
		OccurredOn:    "2025-12-04",
	}); err != nil {
		t.Fatalf("editing: %v", err)
	}

	after := txRepo.created[0]
	if after.InvoiceID != nil {
		t.Fatalf("the payment was re-filed as a line of bill %v while still settling one", *after.InvoiceID)
	}
	if after.PaidInvoiceID == nil || *after.PaidInvoiceID != "inv" {
		t.Fatalf("paidInvoiceID = %v, want inv", after.PaidInvoiceID)
	}
}

// fundingLegFixture is the CANONICAL shape, and the one 24 of the 31 real payments
// use: a single TRANSFER on the funding account whose destination is the card. One
// row debits the account and credits the card, and stays out of income and expense --
// which is why PayInvoiceV2 creates it instead of an EXPENSE/INCOME pair.
//
// 21 of those 24 carry no paid_invoice_id, because they were posted by hand as plain
// transfers. That is why bills read "pago 0,00" while the bank reports them PAID.
func fundingLegFixture(t *testing.T) (*MarkInvoicePaymentUseCase, *fakeTransactionRepo, *fakeInvoiceRepo) {
	t.Helper()
	const cardID, contaID, profileID = "card", "conta", "wb"
	card := creditCardAccountWith(profileID, cardID, 27, 3)
	conta := &bankaccount.BankAccount{ID: contaID, ProfileID: profileID, Name: "Nubank Juridica",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}

	bill := &invoice.Invoice{
		ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27),
		DueDate: day(2026, 8, 6), ReferenceDate: day(2026, 7, 1),
		Status: invoice.StatusClosed, Amount: 1018.18,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{bill.ID: bill}}

	perna := &transaction.Transaction{
		ID: "perna", ProfileID: profileID, BankAccountID: contaID,
		DestinationAccountID: strPtr(cardID),
		Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 1018.18, Currency: "BRL", Description: "Pagamento fatura Nubank Juridica Cartão",
		OccurredOn: day(2026, 8, 3),
	}
	compra := &transaction.Transaction{
		ID: "compra", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 1018.18, Currency: "BRL", Description: "compras do ciclo",
		OccurredOn: day(2026, 7, 10), InvoiceID: &bill.ID,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{perna, compra}}

	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card, contaID: conta}},
		txRepo, invRepo,
	)
	return uc, txRepo, invRepo
}

func TestMarkPayment_LinksTheFundingLegToTheBillItPaid(t *testing.T) {
	uc, txRepo, invRepo := fundingLegFixture(t)

	if err := uc.Execute("perna", "inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := txRepo.created[0]
	if p.PaidInvoiceID == nil || *p.PaidInvoiceID != "inv" {
		t.Fatalf("paidInvoiceID = %v, want inv", p.PaidInvoiceID)
	}
	// It was never a line of the bill and must not become one: SumByInvoiceID counts
	// anything that is not INCOME as a charge, so filing a TRANSFER in would add its
	// amount to the bill as a purchase.
	if p.InvoiceID != nil {
		t.Fatalf("the funding leg became a line of the bill: %v", *p.InvoiceID)
	}
	bill := invRepo.invoices["inv"]
	if bill.PaidAmount == nil || *bill.PaidAmount != 1018.18 {
		t.Fatalf("paidAmount = %v, want 1018.18", bill.PaidAmount)
	}
	if bill.Status != invoice.StatusPaid {
		t.Fatalf("status = %s, want PAID", bill.Status)
	}
	// The bill is worth what was charged, unchanged by learning who paid it.
	if bill.Amount != 1018.18 {
		t.Fatalf("bill amount = %.2f, want 1018.18", bill.Amount)
	}
}

// A transfer between two ordinary accounts has no bill to settle. Only a leg whose
// DESTINATION is the card does.
func TestMarkPayment_RefusesATransferThatDoesNotReachACard(t *testing.T) {
	uc, txRepo, _ := fundingLegFixture(t)
	txRepo.created[0].DestinationAccountID = nil

	if err := uc.Execute("perna", "inv"); !errors.Is(err, ErrNotACreditCard) {
		t.Fatalf("err = %v, want ErrNotACreditCard", err)
	}
}

// The destination card must be the one whose bill this is, or the leg would settle a
// bill it never reached.
func TestMarkPayment_RefusesALegThatReachesAnotherCard(t *testing.T) {
	uc, txRepo, invRepo := fundingLegFixture(t)
	outro := creditCardAccountWith("wb", "outro-cartao", 27, 3)
	uc.accountRepo.(*fakeAccountRepo).accounts["outro-cartao"] = outro
	txRepo.created[0].DestinationAccountID = strPtr("outro-cartao")
	_ = invRepo

	if err := uc.Execute("perna", "inv"); !errors.Is(err, ErrInvoiceNotThisCard) {
		t.Fatalf("err = %v, want ErrInvoiceNotThisCard", err)
	}
}

// Releasing a funding leg must only unlink it. Handing it to the date rule would file
// a TRANSFER as a line of a bill, which SumByInvoiceID counts as a charge.
func TestMarkPayment_ReleasingAFundingLegDoesNotFileItAsALine(t *testing.T) {
	uc, txRepo, invRepo := fundingLegFixture(t)
	// The cycle that CONTAINS 03/08, the day the leg moved: a bill closing 27/07 is
	// paid in the cycle that opens 27/07, so a bill for the leg's date always exists.
	// Without it the date rule had nothing to find and the test passed either way.
	invRepo.invoices["proxima"] = &invoice.Invoice{
		ID: "proxima", BankAccountID: "card",
		OpeningDate: day(2026, 7, 27), ClosingDate: day(2026, 8, 27),
		DueDate: day(2026, 9, 3), ReferenceDate: day(2026, 8, 1),
		Status: invoice.StatusClosed, Amount: 799.57,
	}
	if err := uc.Execute("perna", "inv"); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if err := uc.Release("perna"); err != nil {
		t.Fatalf("releasing: %v", err)
	}
	p := txRepo.created[0]
	if p.PaidInvoiceID != nil {
		t.Fatal("still recorded as settling the bill")
	}
	if p.InvoiceID != nil {
		t.Fatalf("the funding leg was filed as a line of bill %v", *p.InvoiceID)
	}
	if got := invRepo.invoices["inv"].PaidAmount; got != nil {
		t.Fatalf("the bill still reads paid %v", got)
	}
}

// The duplicate this model exists to prevent: the funding leg and a card-side credit
// for the SAME payment, both linked, recording the bill as paid twice over.
func TestMarkPayment_RefusesTheSecondLegOfOnePayment(t *testing.T) {
	uc, txRepo, _ := fundingLegFixture(t)
	if err := uc.Execute("perna", "inv"); err != nil {
		t.Fatalf("marking the funding leg: %v", err)
	}
	// The card-side "Pagamento recebido" of the very same payment.
	txRepo.created = append(txRepo.created, &transaction.Transaction{
		ID: "credito-cartao", ProfileID: "wb", BankAccountID: "card",
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 1018.18, Currency: "BRL", Description: "Pagamento recebido",
		OccurredOn: day(2026, 8, 3),
	})

	if err := uc.Execute("credito-cartao", "inv"); !errors.Is(err, ErrPaymentExceedsInvoice) {
		t.Fatalf("err = %v, want ErrPaymentExceedsInvoice: the bill is now paid twice", err)
	}
}

// The bill a payment LEAVES is almost never the bill it settles: the date rule files
// a payment made on 03/08 into the cycle that opens 27/07. Restating only the settled
// bill left that one stating a total its own lines no longer support -- 300,00 against
// lines of 1.300,00 -- which GetCreditUsage then reads as 1.000,00 of credit consumed.
func TestMarkPayment_RestatesTheBillThePaymentLeaves(t *testing.T) {
	card := creditCardAccountWith("p", "card", 27, 3)
	julho := &invoice.Invoice{ID: "julho", BankAccountID: "card",
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 3),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 1000}
	agosto := &invoice.Invoice{ID: "agosto", BankAccountID: "card",
		OpeningDate: day(2026, 7, 27), ClosingDate: day(2026, 8, 27), DueDate: day(2026, 9, 3),
		ReferenceDate: day(2026, 8, 1), Status: invoice.StatusClosed, Amount: 300}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"julho": julho, "agosto": agosto}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "cj", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "compras julho",
			OccurredOn: day(2026, 7, 10), InvoiceID: strPtr("julho")},
		{ID: "ca", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 1300, Currency: "BRL", Description: "compras agosto",
			OccurredOn: day(2026, 8, 10), InvoiceID: strPtr("agosto")},
		{ID: "pg", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeIncome,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "Pagamento fatura",
			OccurredOn: day(2026, 8, 3), InvoiceID: strPtr("agosto")},
	}}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card}}, txRepo, invRepo)

	if err := uc.Execute("pg", "julho"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	soma, _ := txRepo.SumByInvoiceID("agosto")
	if got := invRepo.invoices["agosto"].Amount; got != soma {
		t.Fatalf("the bill it left stores %.2f and its lines sum to %.2f", got, soma)
	}
	if got := invRepo.invoices["julho"].PaidAmount; got == nil || *got != 1000 {
		t.Fatalf("the bill it settles records %v paid, want 1000", got)
	}
}

// Detaching a row from a bill's lines changes the total by its CONTRIBUTION: a credit
// contributes -amount, a transfer +amount. Adding the amount in both cases let a bill
// of 1.000,00 be recorded as paid 2.000,00.
func TestMarkPayment_RefusesWhenDetachingTheLegLowersTheBill(t *testing.T) {
	card := creditCardAccountWith("p", "card", 27, 3)
	conta := &bankaccount.BankAccount{ID: "conta", ProfileID: "p", Name: "Nubank",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	pago := 1000.0
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "card",
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 3),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 2000, PaidAmount: &pago}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "perna", ProfileID: "p", BankAccountID: "conta", DestinationAccountID: strPtr("card"),
			Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed, Amount: 1000,
			Currency: "BRL", Description: "Pagamento fatura", OccurredOn: day(2026, 8, 3),
			InvoiceID: strPtr("inv")},
		{ID: "compra", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "compra",
			OccurredOn: day(2026, 7, 10), InvoiceID: strPtr("inv")},
		{ID: "credito", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeIncome,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "pago antes",
			OccurredOn: day(2026, 7, 20), PaidInvoiceID: strPtr("inv")},
	}}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card, "conta": conta}}, txRepo, invRepo)

	if err := uc.Execute("perna", "inv"); !errors.Is(err, ErrPaymentExceedsInvoice) {
		t.Fatalf("err = %v, want ErrPaymentExceedsInvoice", err)
	}
}

// A cross-profile payment is ONE movement of money written as two mutually linked
// rows. The money comparison alone cannot see the pair whenever the bill is at least
// twice the payment, and two such pairs already exist on the WB card.
func TestMarkPayment_RefusesTheOtherHalfOfTheSamePayment(t *testing.T) {
	card := creditCardAccountWith("p", "card", 27, 3)
	conta := &bankaccount.BankAccount{ID: "conta", ProfileID: "outro", Name: "MP",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "card",
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 3),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 2000}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "perna", ProfileID: "outro", BankAccountID: "conta", DestinationAccountID: strPtr("card"),
			Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed, Amount: 1000,
			Currency: "BRL", Description: "Pagamento fatura", OccurredOn: day(2026, 8, 3),
			LinkedTransactionID: strPtr("credito")},
		{ID: "credito", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeIncome,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "Pagamento recebido",
			OccurredOn: day(2026, 8, 3), LinkedTransactionID: strPtr("perna")},
		{ID: "compra", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 2000, Currency: "BRL", Description: "compra",
			OccurredOn: day(2026, 7, 10), InvoiceID: strPtr("inv")},
	}}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card, "conta": conta}}, txRepo, invRepo)

	if err := uc.Execute("perna", "inv"); err != nil {
		t.Fatalf("the first half: %v", err)
	}
	if err := uc.Execute("credito", "inv"); !errors.Is(err, ErrPaymentAlreadyRecorded) {
		t.Fatalf("err = %v, want ErrPaymentAlreadyRecorded: the bill is now paid twice on 1000 of cash", err)
	}
}

// A paid_amount written by inv.Pay() has no row behind it, so it cannot be
// re-derived. Linking a smaller payment must not silently drop the difference and
// turn a settled bill into an owed one.
func TestMarkPayment_RefusesToLowerWhatTheBillRecordsAsPaid(t *testing.T) {
	uc, txRepo, invRepo := paymentFixture(t)
	maior := 2980.62
	invRepo.invoices["inv"].PaidAmount = &maior
	_ = txRepo

	err := uc.Execute("pagamento", "inv")
	if !errors.Is(err, ErrWouldLowerRecordedPayment) {
		t.Fatalf("err = %v, want ErrWouldLowerRecordedPayment", err)
	}
	if got := invRepo.invoices["inv"].PaidAmount; got == nil || *got != 2980.62 {
		t.Fatalf("paidAmount = %v, want it untouched at 2980.62", got)
	}
}

// A card's own balance does not move as cash, so a transfer out of one cannot fund a
// bill. Accepted silently, it let one card's bill read PAID off another's credit line.
func TestMarkPayment_RefusesOneCardFundingAnothersBill(t *testing.T) {
	cardA := creditCardAccountWith("p", "cardA", 27, 3)
	cardB := creditCardAccountWith("p", "cardB", 27, 3)
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "cardB",
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 3),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 1000}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "entre", ProfileID: "p", BankAccountID: "cardA", DestinationAccountID: strPtr("cardB"),
			Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed, Amount: 1000,
			Currency: "BRL", Description: "entre cartoes", OccurredOn: day(2026, 8, 3)},
	}}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"cardA": cardA, "cardB": cardB}},
		txRepo, &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}})

	if err := uc.Execute("entre", "inv"); !errors.Is(err, ErrCardCannotFundAPayment) {
		t.Fatalf("err = %v, want ErrCardCannotFundAPayment", err)
	}
}

// Only a TRANSFER names its destination as the card. A row of another type carrying
// one must still be read on its own account, or the type half of the rule is just
// decoration -- and the mutation that drops it left the whole suite green.
func TestMarkPayment_OnlyATransferResolvesTheCardFromItsDestination(t *testing.T) {
	conta := &bankaccount.BankAccount{ID: "conta", ProfileID: "p", Name: "Nubank",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	card := creditCardAccountWith("p", "card", 27, 3)
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "card",
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 3),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 1000}
	// An INCOME on the CHECKING account that happens to name the card. Nothing about
	// it settles a card bill: the money landed in the checking account.
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "entrada", ProfileID: "p", BankAccountID: "conta", DestinationAccountID: strPtr("card"),
			Type: transaction.TypeIncome, Status: transaction.StatusConfirmed, Amount: 1000,
			Currency: "BRL", Description: "entrada na conta", OccurredOn: day(2026, 8, 3)},
	}}
	uc := NewMarkInvoicePaymentUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"conta": conta, "card": card}},
		txRepo, &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}})

	if err := uc.Execute("entrada", "inv"); !errors.Is(err, ErrNotACreditCard) {
		t.Fatalf("err = %v, want ErrNotACreditCard", err)
	}
}
