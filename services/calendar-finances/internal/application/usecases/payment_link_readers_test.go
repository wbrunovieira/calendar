package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A settlement is identified by the LINK, not by how it was described.
//
// Reading the description alone counted a real payment as consumption whenever it was
// worded differently -- "Pro-labore parcial - pagamento minimo do cartao pessoal
// Nubank" is a genuine row on the personal card, and it was being spent twice in every
// expense and cashflow report. The description stays as a fallback, because 21 funding
// legs in production carry no link yet and dropping it would make all of them count.
func TestIsInvoiceSettlement_IdentifiedByTheLinkNotTheWording(t *testing.T) {
	cases := []struct {
		name string
		txn  *transaction.Transaction
		want bool
	}{
		{"linked, worded differently",
			&transaction.Transaction{Description: "Pro-labore parcial - pagamento minimo do cartao pessoal Nubank",
				PaidInvoiceID: strPtr("inv")}, true},
		{"linked, no description at all",
			&transaction.Transaction{Description: "", PaidInvoiceID: strPtr("inv")}, true},
		{"not linked yet, but described as one",
			&transaction.Transaction{Description: "Pagamento fatura Nubank Juridica Cartão"}, true},
		{"not linked, described as one, different case and padding",
			&transaction.Transaction{Description: "  PAGAMENTO FATURA cartao  "}, true},
		{"an ordinary purchase",
			&transaction.Transaction{Description: "iFood"}, false},
		{"a purchase that merely mentions a bill",
			&transaction.Transaction{Description: "Taxa da fatura"}, false},
	}
	for _, c := range cases {
		if got := isInvoiceSettlement(c.txn); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// How much a bill has been paid is a question about the BILL, not about a profile's
// transaction list. Asking it through a profile-filtered List made a cross-profile
// funding leg invisible -- and that is exactly the shape that causes a double
// payment, so the one report that would catch it could not see it.
func TestCheckInvariants_SeesACrossProfilePaymentLeg(t *testing.T) {
	const cardID, cardProfile, payerProfile = "card", "wb", "pessoal"
	card := creditCardAccountWith(cardProfile, cardID, 27, 3)
	bill := &invoice.Invoice{ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 6),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 1000}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "compra", ProfileID: cardProfile, BankAccountID: cardID, Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "compra",
			OccurredOn: day(2026, 7, 10), InvoiceID: strPtr("inv")},
		// Bruno pays the company card from his personal account: the leg belongs to
		// the PERSONAL profile while the bill belongs to the company's.
		{ID: "perna", ProfileID: payerProfile, BankAccountID: "conta-pessoal",
			DestinationAccountID: strPtr(cardID),
			Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
			Amount: 1500, Currency: "BRL", Description: "Pagamento fatura",
			OccurredOn: day(2026, 8, 3), PaidInvoiceID: strPtr("inv")},
	}}

	uc := NewCheckInvariantsUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo,
		&fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}},
	)
	out, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.PaymentDrifts) != 1 {
		t.Fatalf("drifts = %d, want 1: a cross-profile leg paying 1500 on a bill worth 1000 is invisible", len(out.PaymentDrifts))
	}
	if got := out.PaymentDrifts[0].Excess; got != 500 {
		t.Errorf("excess = %.2f, want 500", got)
	}
}

// A planned payment has not moved money, so it cannot make a bill look overpaid.
func TestCheckInvariants_APlannedPaymentIsNotCountedAsPaid(t *testing.T) {
	const cardID, profileID = "card", "wb"
	card := creditCardAccountWith(profileID, cardID, 27, 3)
	bill := &invoice.Invoice{ID: "inv", BankAccountID: cardID,
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 6),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusClosed, Amount: 1000}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "compra", ProfileID: profileID, BankAccountID: cardID, Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 1000, Currency: "BRL", Description: "compra",
			OccurredOn: day(2026, 7, 10), InvoiceID: strPtr("inv")},
		{ID: "planejado", ProfileID: profileID, BankAccountID: "conta", DestinationAccountID: strPtr(cardID),
			Type: transaction.TypeTransfer, Status: transaction.StatusPlanned,
			Amount: 5000, Currency: "BRL", Description: "Pagamento fatura",
			OccurredOn: day(2026, 8, 3), PaidInvoiceID: strPtr("inv")},
	}}
	uc := NewCheckInvariantsUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo, &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}})

	out, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.PaymentDrifts) != 0 {
		t.Fatalf("drifts = %d, want 0: money that has not moved is not paid", len(out.PaymentDrifts))
	}
}

// paymentLinkFixture: a funding leg that settles a bill, plus the updater.
func paymentLinkFixture(t *testing.T) (*UpdateTransactionUseCase, *fakeTransactionRepo, *fakeInvoiceRepo) {
	t.Helper()
	card := creditCardAccountWith("p", "card", 27, 3)
	conta := &bankaccount.BankAccount{ID: "conta", ProfileID: "p", Name: "Nubank",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true}
	pago := 1018.18
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "card",
		OpeningDate: day(2026, 6, 27), ClosingDate: day(2026, 7, 27), DueDate: day(2026, 8, 6),
		ReferenceDate: day(2026, 7, 1), Status: invoice.StatusPaid, Amount: 1018.18, PaidAmount: &pago}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "compra", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 1018.18, Currency: "BRL", Description: "compras",
			OccurredOn: day(2026, 7, 10), InvoiceID: strPtr("inv")},
		{ID: "perna", ProfileID: "p", BankAccountID: "conta", DestinationAccountID: strPtr("card"),
			Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed, Amount: 1018.18,
			Currency: "BRL", Description: "Pagamento fatura", OccurredOn: day(2026, 8, 3),
			PaidInvoiceID: strPtr("inv")},
	}}
	uc := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card, "conta": conta}},
		&fakeCategoryRepo{}, txRepo, invRepo, &noopBalanceRecalculator{})
	return uc, txRepo, invRepo
}

// Changing a linked payment's AMOUNT must restate the bill it settles.
//
// Nothing did, so a leg corrected from 1.018,18 to 900,00 left the bill still
// recording 1.018,18 paid: 118,18 of debt invisible, and that much phantom limit.
func TestUpdateTransaction_RestatesTheBillWhenALinkedPaymentChangesAmount(t *testing.T) {
	uc, _, invRepo := paymentLinkFixture(t)

	if _, err := uc.Execute("perna", UpdateTransactionInput{
		BankAccountID:        "conta",
		DestinationAccountID: strPtr("card"),
		Type:                 string(transaction.TypeTransfer),
		Status:               strPtr(string(transaction.StatusConfirmed)),
		Amount:               900,
		Currency:             "BRL",
		Description:          "Pagamento fatura",
		OccurredOn:           "2026-08-03",
	}); err != nil {
		t.Fatalf("editing: %v", err)
	}
	bill := invRepo.invoices["inv"]
	if bill.PaidAmount == nil || *bill.PaidAmount != 900 {
		t.Fatalf("paidAmount = %v, want 900", bill.PaidAmount)
	}
	if bill.Status == invoice.StatusPaid {
		t.Error("a bill of 1018.18 settled by 900.00 is still claiming to be paid")
	}
}

// An edit that makes the row unable to settle anything must release the link, not
// strand it.
//
// Turning a linked leg into an EXPENSE left type=EXPENSE, no destination and
// paid_invoice_id still set. SumLivePaymentsByInvoiceID has no type filter, so the
// bill kept counting it, and Release answers ErrNotACreditCard -- no route left to
// unlink it, bill PAID forever.
func TestUpdateTransaction_ReleasesTheLinkWhenTheRowCanNoLongerSettleABill(t *testing.T) {
	uc, txRepo, invRepo := paymentLinkFixture(t)

	if _, err := uc.Execute("perna", UpdateTransactionInput{
		BankAccountID: "conta",
		Type:          string(transaction.TypeExpense),
		Status:        strPtr(string(transaction.StatusConfirmed)),
		Amount:        1018.18,
		Currency:      "BRL",
		Description:   "era um pagamento, virou uma despesa",
		OccurredOn:    "2026-08-03",
	}); err != nil {
		t.Fatalf("editing: %v", err)
	}
	after := txRepo.created[1]
	if after.PaidInvoiceID != nil {
		t.Fatalf("the link is stranded on a row that cannot settle a bill: %v", *after.PaidInvoiceID)
	}
	bill := invRepo.invoices["inv"]
	if bill.PaidAmount != nil {
		t.Fatalf("the bill still records %v paid by a row that is no longer a payment", bill.PaidAmount)
	}
}

// Reversing a linked payment already restates the bill; confirming that an ordinary
// edit that changes nothing relevant leaves the bill exactly as it was.
func TestUpdateTransaction_AnUnrelatedEditLeavesTheBillAlone(t *testing.T) {
	uc, _, invRepo := paymentLinkFixture(t)

	if _, err := uc.Execute("perna", UpdateTransactionInput{
		BankAccountID:        "conta",
		DestinationAccountID: strPtr("card"),
		Type:                 string(transaction.TypeTransfer),
		Status:               strPtr(string(transaction.StatusConfirmed)),
		Amount:               1018.18,
		Currency:             "BRL",
		Description:          "Pagamento fatura Nubank Juridica Cartão (descricao corrigida)",
		OccurredOn:           "2026-08-03",
	}); err != nil {
		t.Fatalf("editing: %v", err)
	}
	bill := invRepo.invoices["inv"]
	if bill.PaidAmount == nil || *bill.PaidAmount != 1018.18 {
		t.Fatalf("paidAmount = %v, want it untouched at 1018.18", bill.PaidAmount)
	}
	if bill.Status != invoice.StatusPaid {
		t.Errorf("status = %s, want it to stay PAID", bill.Status)
	}
}
