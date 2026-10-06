package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type invoiceAdjustmentSpy struct {
	calls []string
	ids   []string
	from  []float64
	to    []float64
}

func (s *invoiceAdjustmentSpy) Record(id string, before, after float64, reason, by string) error {
	s.calls = append(s.calls, reason)
	s.ids = append(s.ids, id)
	s.from = append(s.from, before)
	s.to = append(s.to, after)
	return nil
}

// settledBillWithStaleTotal builds the real deadlock: eight PAID bills on the two
// Nubank cards store a total their own lines contradict -- 1.204,61 against lines of
// 1.204,64, 814,92 against 940,93 -- so marking their payment is refused for being
// out of sync, and the route that would resync refused them for being paid.
func settledBillWithStaleTotal(stored, charges float64, paid float64) (*RecalculateInvoiceAmountUseCase, *fakeInvoiceRepo) {
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "card",
		OpeningDate: day(2026, 3, 27), ClosingDate: day(2026, 4, 27), DueDate: day(2026, 5, 4),
		ReferenceDate: day(2026, 4, 1), Status: invoice.StatusPaid, Amount: stored, PaidAmount: &paid}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "compra", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: charges, Currency: "BRL",
			Description: "compras do ciclo", OccurredOn: day(2026, 4, 10), InvoiceID: strPtr("inv")},
	}}
	return NewRecalculateInvoiceAmountUseCase(invRepo, txRepo), invRepo
}

// A bill's Amount is a CACHE of the sum of its own lines. Recomputing it is the
// golden rule, not a breach of it -- nothing is invented, the transactions are the
// source. Refusing it on a settled bill made a stale cache permanent, and deadlocked
// the one repair that needed it: marking the payment refuses an out-of-sync bill, and
// resyncing refused a paid one.
func TestRecalculateInvoice_ASettledBillsCacheIsStillDerivable(t *testing.T) {
	uc, invRepo := settledBillWithStaleTotal(1204.61, 1204.64, 1204.64)

	out, err := uc.Execute("inv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Amount != 1204.64 {
		t.Fatalf("amount = %.2f, want 1204.64", out.Amount)
	}
	if invRepo.invoices["inv"].Amount != 1204.64 {
		t.Fatal("the recalculated total was not saved")
	}
	// Still covered by what was paid, so it stays settled.
	if out.Status != invoice.StatusPaid {
		t.Errorf("status = %s, want PAID", out.Status)
	}
}

// If the recomputed total turns out HIGHER than what was paid, the bill owes again.
// That is information, not damage: it is how a charge that arrived late on a settled
// cycle becomes visible instead of hiding behind a PAID flag.
func TestRecalculateInvoice_ASettledBillThatOwesMoreStopsClaimingToBePaid(t *testing.T) {
	uc, _ := settledBillWithStaleTotal(814.92, 940.93, 814.92)

	out, err := uc.Execute("inv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Amount != 940.93 {
		t.Fatalf("amount = %.2f, want 940.93", out.Amount)
	}
	if out.Status == invoice.StatusPaid {
		t.Fatal("a bill of 940.93 settled by 814.92 is still claiming to be paid")
	}
	if out.PaidAmount == nil || *out.PaidAmount != 814.92 {
		t.Fatalf("paidAmount = %v, want it untouched at 814.92: recomputing the total records no payment", out.PaidAmount)
	}
}

// A correction with no trail is the erasure of proof -- the reason the balance
// recalculation carries one. The same applies here, where the number being replaced
// came off a bank statement.
func TestRecalculateInvoice_TheCorrectionLeavesATrail(t *testing.T) {
	uc, _ := settledBillWithStaleTotal(1204.61, 1204.64, 1204.64)
	spy := &invoiceAdjustmentSpy{}
	uc.SetAdjustmentLog(spy)

	if _, err := uc.Execute("inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.calls) != 1 {
		t.Fatalf("recorded %d corrections, want 1", len(spy.calls))
	}
	if spy.from[0] != 1204.61 || spy.to[0] != 1204.64 {
		t.Errorf("recorded %.2f -> %.2f, want 1204.61 -> 1204.64", spy.from[0], spy.to[0])
	}
	if spy.ids[0] != "card" {
		t.Errorf("recorded against %q, want the card the bill belongs to", spy.ids[0])
	}
}

// Recalculating to the same number changes nothing and must not fill the trail with
// no-ops: a log of non-events is a log nobody reads.
func TestRecalculateInvoice_NoTrailWhenNothingMoved(t *testing.T) {
	uc, _ := settledBillWithStaleTotal(1204.64, 1204.64, 1204.64)
	spy := &invoiceAdjustmentSpy{}
	uc.SetAdjustmentLog(spy)

	if _, err := uc.Execute("inv"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.calls) != 0 {
		t.Fatalf("recorded %d corrections, want none: nothing moved", len(spy.calls))
	}
}

// An unsettled bill keeps working exactly as before.
func TestRecalculateInvoice_AnUnsettledBillIsUnaffected(t *testing.T) {
	bill := &invoice.Invoice{ID: "inv", BankAccountID: "card",
		OpeningDate: day(2026, 3, 27), ClosingDate: day(2026, 4, 27), DueDate: day(2026, 5, 4),
		ReferenceDate: day(2026, 4, 1), Status: invoice.StatusClosed, Amount: 100}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{"inv": bill}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{ID: "c", ProfileID: "p", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 250, Currency: "BRL", Description: "compra",
			OccurredOn: day(2026, 4, 10), InvoiceID: strPtr("inv")},
	}}
	uc := NewRecalculateInvoiceAmountUseCase(invRepo, txRepo)

	out, err := uc.Execute("inv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Amount != 250 {
		t.Fatalf("amount = %.2f, want 250", out.Amount)
	}
	_ = bankaccount.AccountTypeCreditCard
}
