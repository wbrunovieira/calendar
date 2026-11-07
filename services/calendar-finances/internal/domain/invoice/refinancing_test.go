package invoice

import (
	"math"
	"testing"
	"time"
)

// near compares money. Subtracting two decimals in binary floating point does not
// land on the decimal you wrote down, so an exact compare fails on arithmetic
// rather than on behaviour.
func near(got, want float64) bool { return math.Abs(got-want) < 0.005 }

// A bank can settle a bill without anyone paying it: it credits the outstanding
// balance and charges the amount again as future instalments. The ledger had no
// word for that. The bill stayed CLOSED and unpaid forever while the instalments
// it had already become were also recorded — the same debt counted twice, and a
// dashboard reporting R$ 1.742,31 owed on a card the bank considered settled.
//
// Recording it as a payment is not an option: no money left any account, and
// doing so would debit a balance that reconciles to the cent against the bank.

func settledCandidate(t *testing.T, amount float64) *Invoice {
	t.Helper()
	inv, err := New(CreateParams{
		BankAccountID: "card-1",
		ClosingDay:    9,
		DueDay:        14,
		ReferenceDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	inv.Amount = amount
	if err := inv.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return inv
}

func TestRefinanceSettlesTheBillWithoutRecordingAPayment(t *testing.T) {
	inv := settledCandidate(t, 1649.35)
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

	if err := inv.Refinance("txn-credit", at); err != nil {
		t.Fatalf("Refinance: %v", err)
	}

	if inv.Status != StatusRefinanced {
		t.Errorf("status = %q, want %q", inv.Status, StatusRefinanced)
	}
	if got := inv.AmountRemaining(); got != 0 {
		t.Errorf("amount remaining = %.2f, want 0", got)
	}
	// The distinction that matters: nothing was paid. A PaidAmount here would
	// flow into every report as money that left an account, and none did.
	if inv.PaidAmount != nil {
		t.Errorf("paid amount = %v, want nil: refinancing is not a payment", *inv.PaidAmount)
	}
	if inv.PaidAt != nil {
		t.Errorf("paid at = %v, want nil", *inv.PaidAt)
	}
	if inv.RefinancedBy == nil || *inv.RefinancedBy != "txn-credit" {
		t.Errorf("refinanced by = %v, want txn-credit", inv.RefinancedBy)
	}
	if inv.RefinancedAt == nil || !inv.RefinancedAt.Equal(at) {
		t.Errorf("refinanced at = %v, want %v", inv.RefinancedAt, at)
	}
}

func TestRefinanceSettlesOnlyWhatIsLeftAfterAPartialPayment(t *testing.T) {
	inv := settledCandidate(t, 3292.96)
	if err := inv.Pay(3200.00, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if got := inv.AmountRemaining(); !near(got, 92.96) {
		t.Fatalf("precondition: remaining = %v, want 92.96", got)
	}

	if err := inv.Refinance("txn-credit", time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Refinance: %v", err)
	}

	if got := inv.AmountRemaining(); got != 0 {
		t.Errorf("amount remaining = %.2f, want 0", got)
	}
	// The partial payment is history and must survive: it really happened.
	if inv.PaidAmount == nil || !near(*inv.PaidAmount, 3200.00) {
		t.Errorf("paid amount = %v, want 3200.00 preserved", inv.PaidAmount)
	}
}

func TestRefinanceRequiresATransaction(t *testing.T) {
	inv := settledCandidate(t, 100)

	if err := inv.Refinance("", time.Now()); err == nil {
		t.Fatal("expected refinancing with no transaction to be refused")
	}
	if inv.Status != StatusClosed {
		t.Errorf("status = %q, want the bill left untouched", inv.Status)
	}
}

func TestRefinanceRefusesABillThatOwesNothing(t *testing.T) {
	inv := settledCandidate(t, 500)
	if err := inv.Pay(500, time.Now()); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	if err := inv.Refinance("txn-credit", time.Now()); err == nil {
		t.Fatal("expected a settled bill to refuse refinancing")
	}
	if inv.Status != StatusPaid {
		t.Errorf("status = %q, want PAID preserved", inv.Status)
	}
}

// An open cycle is still collecting charges; its total is not yet known, so
// there is no balance to convert.
func TestRefinanceRefusesAnOpenCycle(t *testing.T) {
	inv, err := New(CreateParams{
		BankAccountID: "card-1", ClosingDay: 9, DueDay: 14,
		ReferenceDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	inv.Amount = 800

	if err := inv.Refinance("txn-credit", time.Now()); err == nil {
		t.Fatal("expected an open cycle to refuse refinancing")
	}
	if inv.Status != StatusOpen {
		t.Errorf("status = %q, want OPEN preserved", inv.Status)
	}
}

func TestRefinanceIsNotRepeatable(t *testing.T) {
	inv := settledCandidate(t, 1000)
	if err := inv.Refinance("txn-a", time.Now()); err != nil {
		t.Fatalf("Refinance: %v", err)
	}

	if err := inv.Refinance("txn-b", time.Now()); err == nil {
		t.Fatal("expected a refinanced bill to refuse a second conversion")
	}
	if *inv.RefinancedBy != "txn-a" {
		t.Errorf("refinanced by = %q, want the first conversion kept", *inv.RefinancedBy)
	}
}

// Same reasoning as a paid bill: letting a converted cycle reopen puts an old
// cycle back to OPEN, where it starts accepting charges from a later one.
func TestARefinancedBillCannotReopen(t *testing.T) {
	inv := settledCandidate(t, 1000)
	if err := inv.Refinance("txn-a", time.Now()); err != nil {
		t.Fatalf("Refinance: %v", err)
	}

	if err := inv.Reopen(); err == nil {
		t.Fatal("expected a refinanced bill to refuse reopening")
	}
}

func TestRefinancedBillReportsNoOutstandingDebt(t *testing.T) {
	inv := settledCandidate(t, 4905.86)
	if err := inv.Refinance("txn-credit", time.Now()); err != nil {
		t.Fatalf("Refinance: %v", err)
	}

	if !inv.IsSettled() {
		t.Error("IsSettled() = false, want true for a refinanced bill")
	}
	if inv.IsPartiallyPaid() {
		t.Error("IsPartiallyPaid() = true, want false")
	}
}

// A paid bill is settled too; the predicate must not become a synonym for
// "refinanced".
func TestIsSettledCoversPaidBillsAsWell(t *testing.T) {
	paid := settledCandidate(t, 100)
	if err := paid.Pay(100, time.Now()); err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if !paid.IsSettled() {
		t.Error("IsSettled() = false for a PAID bill, want true")
	}

	owing := settledCandidate(t, 100)
	if owing.IsSettled() {
		t.Error("IsSettled() = true for a CLOSED unpaid bill, want false")
	}
}
