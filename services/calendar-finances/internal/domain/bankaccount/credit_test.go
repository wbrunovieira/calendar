package bankaccount

import "testing"

func creditCard(limit float64) *BankAccount {
	return &BankAccount{Type: AccountTypeCreditCard, CreditLimit: &limit}
}

// What a card commits against its limit is everything still owed on it. The caller
// supplies that figure: it comes from the unpaid invoices, not from the account's
// balance, which this system only moves when an invoice is paid.
func TestCreditUsageFor_TheRealCase(t *testing.T) {
	usage := creditCard(400).CreditUsageFor(380.23)

	if !almost(usage.Outstanding, 380.23) {
		t.Errorf("expected outstanding 380.23, got %v", usage.Outstanding)
	}
	if !almost(usage.Available, 19.77) {
		t.Errorf("expected available 19.77, got %v", usage.Available)
	}
	if usage.UsagePercent < 95 || usage.UsagePercent > 95.1 {
		t.Errorf("expected usage ~95%%, got %v", usage.UsagePercent)
	}
}

func TestCreditUsageFor_NoDebtIsFullyAvailable(t *testing.T) {
	usage := creditCard(400).CreditUsageFor(0)

	if !almost(usage.Available, 400) || usage.UsagePercent != 0 {
		t.Fatalf("expected the whole limit available, got %+v", usage)
	}
}

// A refund landing after the bill was paid can leave a negative outstanding; it
// commits nothing and must not read as negative usage.
func TestCreditUsageFor_ACreditCommitsNothing(t *testing.T) {
	usage := creditCard(400).CreditUsageFor(-25)

	if usage.Outstanding != 0 {
		t.Errorf("expected outstanding clamped to 0, got %v", usage.Outstanding)
	}
	if usage.UsagePercent != 0 {
		t.Errorf("expected 0%% usage, got %v", usage.UsagePercent)
	}
}

func TestCreditUsageFor_OverTheLimit(t *testing.T) {
	usage := creditCard(400).CreditUsageFor(450)

	if !almost(usage.Available, -50) {
		t.Errorf("expected -50 available, got %v", usage.Available)
	}
	if usage.UsagePercent <= 100 {
		t.Errorf("expected usage above 100%%, got %v", usage.UsagePercent)
	}
}

func TestCreditUsageFor_NoLimitDoesNotDivideByZero(t *testing.T) {
	usage := (&BankAccount{Type: AccountTypeCreditCard}).CreditUsageFor(120)

	if usage.UsagePercent != 0 {
		t.Errorf("expected 0%% with no limit configured, got %v", usage.UsagePercent)
	}
}

// Money figures elsewhere in this service are rounded to cents; an agent or report
// printing the raw field should not see 19.769999999999982.
func TestCreditUsageFor_RoundsToCents(t *testing.T) {
	usage := creditCard(400).CreditUsageFor(380.23)

	if usage.Available != 19.77 {
		t.Fatalf("expected exactly 19.77, got %v", usage.Available)
	}
}

func almost(a, b float64) bool {
	d := a - b
	return d < 0.005 && d > -0.005
}
