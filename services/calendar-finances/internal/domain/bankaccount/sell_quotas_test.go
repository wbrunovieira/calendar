package bankaccount

import (
	"math"
	"testing"
)

// approxEqual compares floats within a cent's worth of tolerance, since raw
// share×price products carry floating-point noise that the money layer rounds.
func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// newQuotaAccount returns an investment account holding `quotas` shares priced
// at `price` each, ready for SellQuotas tests.
func newQuotaAccount(t *testing.T, quotas, price float64) *BankAccount {
	t.Helper()
	acc, err := NewBankAccount("profile-1", "SNAG11", AccountTypeInvestment, 0, "BRL")
	if err != nil {
		t.Fatalf("NewBankAccount: %v", err)
	}
	if err := acc.SetQuotasFromPrice(quotas, price); err != nil {
		t.Fatalf("SetQuotasFromPrice: %v", err)
	}
	return acc
}

func TestSellQuotas_FullSellClosesPosition(t *testing.T) {
	acc := newQuotaAccount(t, 120, 9.84)

	proceeds, err := acc.SellQuotas(120, 9.84)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !approxEqual(proceeds, 120*9.84) {
		t.Errorf("proceeds = %.2f, want %.2f", proceeds, 120*9.84)
	}
	// Position closed: zero shares (NOT nil, to avoid panicking the price sync).
	if acc.NumberOfQuotas == nil {
		t.Fatal("NumberOfQuotas is nil after full sell; expected pointer to 0")
	}
	if *acc.NumberOfQuotas != 0 {
		t.Errorf("NumberOfQuotas = %v, want 0", *acc.NumberOfQuotas)
	}
	if acc.HasQuotas() {
		t.Error("HasQuotas() should be false after closing the position")
	}
	if acc.CurrentBalance != 0 {
		t.Errorf("CurrentBalance = %.2f, want 0", acc.CurrentBalance)
	}
}

func TestSellQuotas_PartialSellKeepsRemainderMarkedAtMarket(t *testing.T) {
	acc := newQuotaAccount(t, 120, 9.84) // market price 9.84

	// Sell 73 at a slightly worse price; remainder stays marked at market 9.84.
	proceeds, err := acc.SellQuotas(73, 9.80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !approxEqual(proceeds, 73*9.80) {
		t.Errorf("proceeds = %.2f, want %.2f", proceeds, 73*9.80)
	}
	if acc.NumberOfQuotas == nil || *acc.NumberOfQuotas != 47 {
		t.Fatalf("remaining quotas = %v, want 47", acc.NumberOfQuotas)
	}
	if !approxEqual(acc.CurrentBalance, 47*9.84) {
		t.Errorf("CurrentBalance = %.4f, want %.4f (remainder marked at market)", acc.CurrentBalance, 47*9.84)
	}
}

func TestSellQuotas_CannotSellMoreThanHeld(t *testing.T) {
	acc := newQuotaAccount(t, 10, 100)

	if _, err := acc.SellQuotas(11, 100); err == nil {
		t.Fatal("expected error selling more than held, got nil")
	}
	// Position must be untouched after a rejected sell.
	if acc.NumberOfQuotas == nil || *acc.NumberOfQuotas != 10 {
		t.Errorf("quotas mutated on rejected sell: %v", acc.NumberOfQuotas)
	}
}

func TestSellQuotas_RejectsNonPositiveQuantity(t *testing.T) {
	acc := newQuotaAccount(t, 10, 100)

	if _, err := acc.SellQuotas(0, 100); err == nil {
		t.Error("expected error for zero quantity")
	}
	if _, err := acc.SellQuotas(-5, 100); err == nil {
		t.Error("expected error for negative quantity")
	}
}

func TestSellQuotas_RejectsNegativePrice(t *testing.T) {
	acc := newQuotaAccount(t, 10, 100)

	if _, err := acc.SellQuotas(1, -1); err == nil {
		t.Error("expected error for negative unit price")
	}
}

func TestSellQuotas_ErrorsWhenNoQuotas(t *testing.T) {
	acc, err := NewBankAccount("profile-1", "Checking", AccountTypeChecking, 500, "BRL")
	if err != nil {
		t.Fatalf("NewBankAccount: %v", err)
	}

	if _, err := acc.SellQuotas(1, 10); err == nil {
		t.Error("expected error selling from an account with no quotas")
	}
}
