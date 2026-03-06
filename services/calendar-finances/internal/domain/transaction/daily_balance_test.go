package transaction

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func ptr(s string) *string { return &s }

func TestCalculateDailyBalances_SimpleExpensesAndIncome(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 1000, OccurredOn: date(2026, 1, 1)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 200, OccurredOn: date(2026, 1, 1)},
		{ID: "3", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 300, OccurredOn: date(2026, 1, 2)},
	}

	// currentBalance = 1000 - 200 - 300 = 500
	result := CalculateDailyBalances(txs, 500, "acc-1")

	if len(result) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result))
	}

	// Day 1: +1000 -200 = +800, balance = 800
	if result[0].Date != "2026-01-01" {
		t.Errorf("expected date 2026-01-01, got %s", result[0].Date)
	}
	if result[0].Balance != 800 {
		t.Errorf("expected balance 800, got %.2f", result[0].Balance)
	}
	if result[0].DayTotal != 800 {
		t.Errorf("expected dayTotal 800, got %.2f", result[0].DayTotal)
	}

	// Day 2: -300, balance = 500
	if result[1].Date != "2026-01-02" {
		t.Errorf("expected date 2026-01-02, got %s", result[1].Date)
	}
	if result[1].Balance != 500 {
		t.Errorf("expected balance 500, got %.2f", result[1].Balance)
	}
	if result[1].DayTotal != -300 {
		t.Errorf("expected dayTotal -300, got %.2f", result[1].DayTotal)
	}
}

func TestCalculateDailyBalances_LastDayMatchesCurrentBalance(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 5000, OccurredOn: date(2026, 2, 10)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 3795.01, OccurredOn: date(2026, 2, 11)},
	}

	currentBalance := 1204.99
	result := CalculateDailyBalances(txs, currentBalance, "acc-1")

	lastDay := result[len(result)-1]
	if lastDay.Balance != currentBalance {
		t.Errorf("last day balance should match currentBalance, got %.2f want %.2f", lastDay.Balance, currentBalance)
	}
}

func TestCalculateDailyBalances_PlannedTransactionsExcludedFromDerivation(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 1000, OccurredOn: date(2026, 3, 1)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 500, OccurredOn: date(2026, 3, 5)},
		{ID: "3", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusPlanned, Amount: 2000, OccurredOn: date(2026, 3, 18)},
		{ID: "4", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusPlanned, Amount: 700, OccurredOn: date(2026, 3, 20)},
	}

	// currentBalance reflects only confirmed: 1000 - 500 = 500
	currentBalance := 500.0
	result := CalculateDailyBalances(txs, currentBalance, "acc-1")

	if len(result) != 4 {
		t.Fatalf("expected 4 days, got %d", len(result))
	}

	// Last confirmed day (03-05) should equal currentBalance
	if result[1].Date != "2026-03-05" {
		t.Errorf("expected 2026-03-05, got %s", result[1].Date)
	}
	if result[1].Balance != 500 {
		t.Errorf("last confirmed day should be 500, got %.2f", result[1].Balance)
	}

	// Planned days project from currentBalance
	// 03-18: 500 - 2000 = -1500
	if result[2].Balance != -1500 {
		t.Errorf("planned expense day should be -1500, got %.2f", result[2].Balance)
	}

	// 03-20: -1500 + 700 = -800
	if result[3].Balance != -800 {
		t.Errorf("planned income day should be -800, got %.2f", result[3].Balance)
	}
}

func TestCalculateDailyBalances_CancelledTransactionsIgnored(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 1000, OccurredOn: date(2026, 1, 1)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusCancelled, Amount: 9999, OccurredOn: date(2026, 1, 1)},
	}

	currentBalance := 1000.0
	result := CalculateDailyBalances(txs, currentBalance, "acc-1")

	if len(result) != 1 {
		t.Fatalf("expected 1 day, got %d", len(result))
	}
	if result[0].Balance != 1000 {
		t.Errorf("cancelled should be ignored, expected 1000 got %.2f", result[0].Balance)
	}
	if result[0].DayTotal != 1000 {
		t.Errorf("cancelled should not affect dayTotal, expected 1000 got %.2f", result[0].DayTotal)
	}
}

func TestCalculateDailyBalances_TransferSubtractsFromSource(t *testing.T) {
	dest := "acc-other"
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 5000, OccurredOn: date(2026, 1, 1)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeTransfer, Status: StatusConfirmed, Amount: 3000, OccurredOn: date(2026, 1, 2), DestinationAccountID: &dest},
	}

	// currentBalance = 5000 - 3000 = 2000
	result := CalculateDailyBalances(txs, 2000, "acc-1")

	if result[0].Balance != 5000 {
		t.Errorf("day 1 balance should be 5000, got %.2f", result[0].Balance)
	}
	if result[1].Balance != 2000 {
		t.Errorf("day 2 after transfer should be 2000, got %.2f", result[1].Balance)
	}
	if result[1].DayTotal != -3000 {
		t.Errorf("transfer dayTotal should be -3000, got %.2f", result[1].DayTotal)
	}
}

func TestCalculateDailyBalances_IncomingTransferAddsToDestination(t *testing.T) {
	source := "acc-source"
	dest := "acc-1"
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 1000, OccurredOn: date(2026, 1, 1)},
		{ID: "2", BankAccountID: source, Type: TypeTransfer, Status: StatusConfirmed, Amount: 4000, OccurredOn: date(2026, 1, 2), DestinationAccountID: &dest},
	}

	// currentBalance = 1000 + 4000 = 5000
	result := CalculateDailyBalances(txs, 5000, "acc-1")

	if len(result) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result))
	}
	if result[0].Balance != 1000 {
		t.Errorf("day 1 balance should be 1000, got %.2f", result[0].Balance)
	}
	// Incoming transfer: +4000
	if result[1].Balance != 5000 {
		t.Errorf("day 2 after incoming transfer should be 5000, got %.2f", result[1].Balance)
	}
	if result[1].DayTotal != 4000 {
		t.Errorf("incoming transfer dayTotal should be +4000, got %.2f", result[1].DayTotal)
	}
}

func TestCalculateDailyBalances_EmptyTransactions(t *testing.T) {
	result := CalculateDailyBalances(nil, 1000, "acc-1")

	if len(result) != 0 {
		t.Errorf("expected 0 days for empty transactions, got %d", len(result))
	}
}

func TestCalculateDailyBalances_MultipleDaysSameDate(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 100, OccurredOn: date(2026, 1, 5)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 30, OccurredOn: date(2026, 1, 5)},
		{ID: "3", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 20, OccurredOn: date(2026, 1, 5)},
	}

	// currentBalance = 100 - 30 - 20 = 50
	result := CalculateDailyBalances(txs, 50, "acc-1")

	if len(result) != 1 {
		t.Fatalf("expected 1 day (all same date), got %d", len(result))
	}
	if result[0].Balance != 50 {
		t.Errorf("balance should be 50, got %.2f", result[0].Balance)
	}
	if result[0].DayTotal != 50 {
		t.Errorf("dayTotal should be 50 (+100-30-20), got %.2f", result[0].DayTotal)
	}
}

func TestCalculateDailyBalances_ExpenseWithDestinationShouldNotAffectDestBalance(t *testing.T) {
	dest := "acc-1"
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 1000, OccurredOn: date(2026, 1, 1)},
		// This expense is FROM another account but has destination_account_id = acc-1.
		// It should NOT affect acc-1's balance (only transfers do).
		{ID: "2", BankAccountID: "acc-other", Type: TypeExpense, Status: StatusConfirmed, Amount: 500, OccurredOn: date(2026, 1, 2), DestinationAccountID: &dest},
	}

	// currentBalance = 1000 (the expense from acc-other doesn't affect acc-1)
	result := CalculateDailyBalances(txs, 1000, "acc-1")

	if len(result) != 1 {
		t.Fatalf("expected 1 day (expense from other account ignored), got %d", len(result))
	}
	if result[0].Balance != 1000 {
		t.Errorf("expense from other account should not affect balance, expected 1000 got %.2f", result[0].Balance)
	}
}

func TestCalculateDailyBalances_PlannedBeforeLastConfirmedShouldNotAffectBalance(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 1000, OccurredOn: date(2026, 1, 1)},
		// Planned in the past (before last confirmed) - should NOT shift the balance
		{ID: "2", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusPlanned, Amount: 184, OccurredOn: date(2026, 1, 15)},
		{ID: "3", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusConfirmed, Amount: 300, OccurredOn: date(2026, 2, 1)},
		// Planned in the future (after last confirmed) - SHOULD project forward
		{ID: "4", BankAccountID: "acc-1", Type: TypeExpense, Status: StatusPlanned, Amount: 500, OccurredOn: date(2026, 3, 1)},
	}

	// currentBalance = 1000 - 300 = 700 (only confirmed)
	result := CalculateDailyBalances(txs, 700, "acc-1")

	// Last confirmed day (02-01) must equal currentBalance
	var lastConfirmedBalance float64
	for _, d := range result {
		if d.Date == "2026-02-01" {
			lastConfirmedBalance = d.Balance
		}
	}
	if lastConfirmedBalance != 700 {
		t.Errorf("last confirmed day should equal currentBalance 700, got %.2f", lastConfirmedBalance)
	}

	// Future planned projects: 700 - 500 = 200
	var futureBalance float64
	for _, d := range result {
		if d.Date == "2026-03-01" {
			futureBalance = d.Balance
		}
	}
	if futureBalance != 200 {
		t.Errorf("future planned should project from currentBalance, expected 200 got %.2f", futureBalance)
	}
}

func TestCalculateDailyBalances_FloatingPointPrecision(t *testing.T) {
	txs := []*Transaction{
		{ID: "1", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 0.1, OccurredOn: date(2026, 1, 1)},
		{ID: "2", BankAccountID: "acc-1", Type: TypeIncome, Status: StatusConfirmed, Amount: 0.2, OccurredOn: date(2026, 1, 1)},
	}

	result := CalculateDailyBalances(txs, 0.3, "acc-1")

	if result[0].Balance != 0.3 {
		t.Errorf("expected 0.3 (rounded), got %.20f", result[0].Balance)
	}
}
