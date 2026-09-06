package invoice

import (
	"encoding/json"
	"testing"
	"time"
)

// The Nubank Juridica card was paid in parts during a cash squeeze: R$ 1.018,18
// against a bill of R$ 1.489,22, and R$ 119,94 against one of R$ 1.429,22. Pay()
// marked both PAID regardless of the amount, which would have erased R$ 1.780,32
// of real debt from the ledger.

func billOf(amount float64) *Invoice {
	return &Invoice{
		ID:            "inv-1",
		BankAccountID: "card-1",
		Amount:        amount,
		Status:        StatusClosed,
	}
}

func day(d int) time.Time {
	return time.Date(2026, time.September, d, 0, 0, 0, 0, time.UTC)
}

func TestPay_PartialPaymentDoesNotSettleTheBill(t *testing.T) {
	inv := billOf(1489.22)

	if err := inv.Pay(1018.18, day(3)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.Status != StatusPartiallyPaid {
		t.Errorf("status = %s, want %s", inv.Status, StatusPartiallyPaid)
	}
	if inv.PaidAmount == nil || *inv.PaidAmount != 1018.18 {
		t.Errorf("paidAmount = %v, want 1018.18", inv.PaidAmount)
	}
	if got, want := inv.AmountRemaining(), 471.04; !nearly(got, want) {
		t.Errorf("amountRemaining = %.2f, want %.2f — this is the debt that must not disappear", got, want)
	}
	if inv.IsPaid() {
		t.Error("a partially paid bill must not report as paid")
	}
}

func TestPay_SecondPaymentAccumulatesAndSettles(t *testing.T) {
	inv := billOf(1489.22)

	if err := inv.Pay(1018.18, day(3)); err != nil {
		t.Fatalf("first payment: %v", err)
	}
	if err := inv.Pay(471.04, day(20)); err != nil {
		t.Fatalf("second payment must be allowed on a partially paid bill: %v", err)
	}

	if inv.Status != StatusPaid {
		t.Errorf("status = %s, want %s once the total is covered", inv.Status, StatusPaid)
	}
	if inv.PaidAmount == nil || !nearly(*inv.PaidAmount, 1489.22) {
		t.Errorf("paidAmount = %v, want the accumulated 1489.22", inv.PaidAmount)
	}
	if got := inv.AmountRemaining(); !nearly(got, 0) {
		t.Errorf("amountRemaining = %.2f, want 0", got)
	}
	if inv.PaidAt == nil || !inv.PaidAt.Equal(day(20)) {
		t.Errorf("paidAt = %v, want the date it was settled (20th)", inv.PaidAt)
	}
}

func TestPay_ExactPaymentSettlesInOne(t *testing.T) {
	inv := billOf(769.51)
	if err := inv.Pay(769.51, day(5)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != StatusPaid {
		t.Errorf("status = %s, want %s", inv.Status, StatusPaid)
	}
}

func TestPay_OverpaymentSettlesAndNeverGoesNegative(t *testing.T) {
	inv := billOf(100)
	if err := inv.Pay(150, day(5)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != StatusPaid {
		t.Errorf("status = %s, want %s", inv.Status, StatusPaid)
	}
	if got := inv.AmountRemaining(); got != 0 {
		t.Errorf("amountRemaining = %.2f, want 0 — an overpayment is not negative debt", got)
	}
}

func TestPay_AlreadySettledBillIsRejected(t *testing.T) {
	inv := billOf(100)
	if err := inv.Pay(100, day(5)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := inv.Pay(10, day(6)); err == nil {
		t.Error("paying a settled bill must be rejected")
	}
}

func TestPay_RejectsNonPositiveAmounts(t *testing.T) {
	for _, amount := range []float64{0, -5} {
		inv := billOf(100)
		if err := inv.Pay(amount, day(5)); err == nil {
			t.Errorf("Pay(%.2f) must be rejected", amount)
		}
	}
}

func TestPay_CentRoundingStillSettles(t *testing.T) {
	// Three payments that sum to the total only within float tolerance.
	inv := billOf(100.00)
	for _, part := range []float64{33.33, 33.33, 33.34} {
		if err := inv.Pay(part, day(5)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if inv.Status != StatusPaid {
		t.Errorf("status = %s, want %s — float dust must not leave a bill unsettled", inv.Status, StatusPaid)
	}
}

func TestAmountRemaining_UnpaidBillOwesTheWholeAmount(t *testing.T) {
	if got := billOf(250).AmountRemaining(); !nearly(got, 250) {
		t.Errorf("amountRemaining = %.2f, want 250", got)
	}
}

func nearly(a, b float64) bool {
	d := a - b
	return d < 0.005 && d > -0.005
}

func TestInvoiceJSON_ExposesTheOutstandingDebt(t *testing.T) {
	inv := billOf(1489.22)
	if err := inv.Pay(1018.18, day(3)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	remaining, ok := got["amountRemaining"]
	if !ok {
		t.Fatal("amountRemaining must be in the payload, or a partial payment looks settled to every consumer")
	}
	if v, _ := remaining.(float64); !nearly(v, 471.04) {
		t.Errorf("amountRemaining = %v, want 471.04", remaining)
	}
	if got["status"] != string(StatusPartiallyPaid) {
		t.Errorf("status = %v, want %s", got["status"], StatusPartiallyPaid)
	}
	// The stored fields must still be there for existing consumers.
	if v, _ := got["paidAmount"].(float64); !nearly(v, 1018.18) {
		t.Errorf("paidAmount = %v, want 1018.18", got["paidAmount"])
	}
}
