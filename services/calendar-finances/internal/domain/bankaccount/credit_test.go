package bankaccount

import "testing"

func creditCard(limit, balance float64) *BankAccount {
	return &BankAccount{
		Type:           AccountTypeCreditCard,
		CurrentBalance: balance,
		CreditLimit:    &limit,
	}
}

// The real case that exposed this: a R$400 card owing R$380,23, R$301,74 of it in an
// installment plan. Availability must come from everything still owed, not from the
// open invoice — that is what the bank blocks against the limit.
func TestAvailableCredit_UsesTheWholeOutstandingDebt(t *testing.T) {
	card := creditCard(400, -380.23)

	if got := card.Outstanding(); !almost(got, 380.23) {
		t.Fatalf("expected outstanding 380.23, got %v", got)
	}
	if got := card.AvailableCredit(); !almost(got, 19.77) {
		t.Fatalf("expected available 19.77, got %v", got)
	}
	if got := card.CreditUsagePercent(); got < 95 || got > 95.1 {
		t.Fatalf("expected usage ~95%%, got %v", got)
	}
}

func TestAvailableCredit_CardWithNoDebtIsFullyAvailable(t *testing.T) {
	card := creditCard(400, 0)

	if got := card.Outstanding(); got != 0 {
		t.Fatalf("expected no outstanding, got %v", got)
	}
	if got := card.AvailableCredit(); !almost(got, 400) {
		t.Fatalf("expected the whole limit available, got %v", got)
	}
}

// A refund landing after the bill was paid leaves the card with a credit balance.
// That commits nothing, and usage must not go negative.
func TestAvailableCredit_CreditBalanceCommitsNothing(t *testing.T) {
	card := creditCard(400, 25)

	if got := card.Outstanding(); got != 0 {
		t.Fatalf("expected outstanding clamped to 0, got %v", got)
	}
	if got := card.CreditUsagePercent(); got != 0 {
		t.Fatalf("expected 0%% usage, got %v", got)
	}
}

func TestAvailableCredit_OverTheLimitGoesNegative(t *testing.T) {
	card := creditCard(400, -450)

	if got := card.AvailableCredit(); !almost(got, -50) {
		t.Fatalf("expected -50 available, got %v", got)
	}
	if got := card.CreditUsagePercent(); got <= 100 {
		t.Fatalf("expected usage above 100%%, got %v", got)
	}
}

func TestAvailableCredit_NoLimitConfiguredDoesNotDivideByZero(t *testing.T) {
	card := &BankAccount{Type: AccountTypeCreditCard, CurrentBalance: -120}

	if got := card.CreditUsagePercent(); got != 0 {
		t.Fatalf("expected 0%% when no limit is set, got %v", got)
	}
	if got := card.AvailableCredit(); !almost(got, -120) {
		t.Fatalf("expected -120 available, got %v", got)
	}
}

// A checking account has no limit to consume; asking is meaningless, not an error.
func TestAvailableCredit_NonCardHasNoUsage(t *testing.T) {
	acc := &BankAccount{Type: AccountTypeChecking, CurrentBalance: -500}

	if got := acc.Outstanding(); got != 0 {
		t.Fatalf("expected 0 outstanding on a non-card, got %v", got)
	}
	if got := acc.CreditUsagePercent(); got != 0 {
		t.Fatalf("expected 0%% on a non-card, got %v", got)
	}
}

func almost(a, b float64) bool {
	d := a - b
	return d < 0.005 && d > -0.005
}
