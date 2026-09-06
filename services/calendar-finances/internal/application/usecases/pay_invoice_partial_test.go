package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
)

// The real case: during a cash squeeze the Nubank Juridica bill of R$ 1.489,22 was
// paid with R$ 1.018,18 on 2026-08-03. Marking it settled would have erased
// R$ 471,04 of debt.

type sumStubRepo struct {
	fakeTransactionRepo
	total float64
}

func (r *sumStubRepo) SumByInvoiceID(invoiceID string) (float64, error) {
	return r.total, nil
}

func payFixture(t *testing.T, storedAmount, authoritativeTotal float64) (*PayInvoiceUseCaseV2, *fakeAccountRepo, string) {
	t.Helper()
	cardID, checkingID := "card-1", "checking-1"
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}}
	invRepo.invoices["inv-1"] = &invoice.Invoice{
		ID: "inv-1", BankAccountID: cardID, Amount: storedAmount,
		Status:      invoice.StatusClosed,
		OpeningDate: time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
		ClosingDate: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		DueDate:     time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC),
	}
	linked := checkingID
	accRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		cardID:     {ID: cardID, Type: bankaccount.AccountTypeCreditCard, LinkedAccountID: &linked, Name: "Nubank Juridica Cartao", Currency: "BRL"},
		checkingID: {ID: checkingID, Type: bankaccount.AccountTypeChecking, ProfileID: "p1", Currency: "BRL", CurrentBalance: 5000},
	}}
	txRepo := &sumStubRepo{total: authoritativeTotal}
	return NewPayInvoiceUseCaseV2(invRepo, accRepo, txRepo), accRepo, "inv-1"
}

func TestPayV2_PartialPaymentLeavesTheDebtVisible(t *testing.T) {
	uc, _, id := payFixture(t, 1489.22, 1489.22)

	inv, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 1018.18, PaidAt: "2026-08-03"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != invoice.StatusPartiallyPaid {
		t.Errorf("status = %s, want %s", inv.Status, invoice.StatusPartiallyPaid)
	}
	if got, want := inv.AmountRemaining(), 471.04; got < want-0.005 || got > want+0.005 {
		t.Errorf("amountRemaining = %.2f, want %.2f", got, want)
	}
}

func TestPayV2_SecondPaymentSettlesTheBill(t *testing.T) {
	uc, _, id := payFixture(t, 1489.22, 1489.22)

	if _, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 1018.18, PaidAt: "2026-08-03"}); err != nil {
		t.Fatalf("first payment: %v", err)
	}
	inv, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 471.04, PaidAt: "2026-08-20"})
	if err != nil {
		t.Fatalf("a partially paid bill must accept another payment: %v", err)
	}
	if inv.Status != invoice.StatusPaid {
		t.Errorf("status = %s, want %s", inv.Status, invoice.StatusPaid)
	}
}

func TestPayV2_StaleStoredAmountDoesNotSettleTheBill(t *testing.T) {
	// The stored Amount is a cached sum. Here it is stale and far too low, so
	// deciding against it would settle a bill that is nowhere near covered.
	uc, _, id := payFixture(t, 100.00, 1489.22)

	inv, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 1018.18, PaidAt: "2026-08-03"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != invoice.StatusPartiallyPaid {
		t.Errorf("status = %s, want %s — the authoritative total is 1489.22, not the stored 100.00",
			inv.Status, invoice.StatusPartiallyPaid)
	}
}

func TestPayV2_SettledBillRejectsFurtherPayment(t *testing.T) {
	uc, _, id := payFixture(t, 100, 100)
	if _, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 100, PaidAt: "2026-08-03"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 10, PaidAt: "2026-08-04"}); err != ErrInvoiceAlreadyPaid {
		t.Errorf("err = %v, want ErrInvoiceAlreadyPaid", err)
	}
}

func TestPayV2_EachPartialPaymentDebitsTheFundingAccountOnce(t *testing.T) {
	uc, accRepo, id := payFixture(t, 1489.22, 1489.22)

	if _, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 1018.18, PaidAt: "2026-08-03"}); err != nil {
		t.Fatalf("first payment: %v", err)
	}
	if _, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 471.04, PaidAt: "2026-08-20"}); err != nil {
		t.Fatalf("second payment: %v", err)
	}

	// The money really left the account twice: 5000 - 1018.18 - 471.04.
	got := accRepo.accounts["checking-1"].CurrentBalance
	want := 3510.78
	if got < want-0.005 || got > want+0.005 {
		t.Errorf("funding balance = %.2f, want %.2f — each partial payment must debit once", got, want)
	}
}

// An invoice whose charges are not linked by invoice_id sums to zero. Settling it on
// any payment would be wrong, and writing that zero back would destroy the only
// record of what was billed.
func TestPayV2_UnlinkedChargesKeepTheStoredTotal(t *testing.T) {
	uc, _, id := payFixture(t, 1489.22, 0)

	inv, err := uc.Execute(PayInvoiceInput{InvoiceID: id, PaidAmount: 500, PaidAt: "2026-08-03"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != invoice.StatusPartiallyPaid {
		t.Errorf("status = %s, want %s — a zero sum must not settle the bill", inv.Status, invoice.StatusPartiallyPaid)
	}
	if inv.Amount != 1489.22 {
		t.Errorf("amount = %.2f, want the stored 1489.22 preserved", inv.Amount)
	}
}
