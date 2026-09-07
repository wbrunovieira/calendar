package invoice

import (
	"testing"
	"time"
)

func settledBill(t *testing.T) *Invoice {
	t.Helper()
	paidAt := time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC)
	paid := 799.57
	return &Invoice{
		ID: "inv-1", BankAccountID: "card", Status: StatusPaid,
		Amount: 799.57, PaidAmount: &paid, PaidAt: &paidAt,
		OpeningDate: time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC),
		ClosingDate: time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC),
		DueDate:     time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
	}
}

// Restating a bill's payments to zero must not make it reopenable.
//
// Reopen() refuses a bill that was ever settled, because putting an old cycle back to
// OPEN is how it starts accepting charges from a later one — the merged-cycle failure.
// The guard reads PaidAt and PaidAmount, and RestatePayments used to clear both: after
// reversing a payment the guard silently stopped applying, and its comment described a
// protection that no longer existed.
func TestRestatePayments_ABillThatWasSettledStillCannotReopen(t *testing.T) {
	inv := settledBill(t)

	inv.RestatePayments(0)

	if inv.Status != StatusClosed {
		t.Fatalf("a bill with nothing paid is closed, got %s", inv.Status)
	}
	if inv.PaidAmount != nil {
		t.Errorf("what was paid must be restated to nothing, got %v", *inv.PaidAmount)
	}
	if inv.PaidAt == nil {
		t.Fatal("the fact that this bill was once settled must survive: Reopen reads it")
	}
	if err := inv.Reopen(); err == nil {
		t.Error("a bill that was settled must not reopen, even after its payment is reversed")
	}
}

// The bill is still payable, which is what actually matters for recovery — the money
// came back and the bill owes again.
func TestRestatePayments_AnUnsettledBillCanBePaidAgain(t *testing.T) {
	inv := settledBill(t)
	inv.RestatePayments(0)

	if err := inv.Pay(799.57, time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("the bill must accept payment again: %v", err)
	}
	if inv.Status != StatusPaid {
		t.Errorf("paying it in full settles it, got %s", inv.Status)
	}
}

// Restating downwards to a partial amount says so.
func TestRestatePayments_APartialTotalLeavesTheBillPartiallyPaid(t *testing.T) {
	inv := settledBill(t)

	inv.RestatePayments(119.94)

	if inv.Status != StatusPartiallyPaid {
		t.Fatalf("got %s", inv.Status)
	}
	if inv.PaidAmount == nil || *inv.PaidAmount != 119.94 {
		t.Errorf("what was paid must be replaced, not accumulated: %v", inv.PaidAmount)
	}
}
