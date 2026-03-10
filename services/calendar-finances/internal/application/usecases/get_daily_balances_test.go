package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func TestGetDailyBalances_ShouldIncludeInitialBalance(t *testing.T) {
	// Setup: account with initial_balance = 100
	accountID := "acc-1"
	profileID := "profile-1"
	initialBalance := 100.0

	accountRepo := &fakeBankAccountRepoForBalance{
		accounts: map[string]*bankaccount.BankAccount{
			accountID: {
				ID:             accountID,
				ProfileID:      profileID,
				InitialBalance: initialBalance,
				CurrentBalance: 900, // 100 + 1000 - 200
			},
		},
	}

	txRepo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{
			{
				ID:            "tx1",
				ProfileID:     profileID,
				BankAccountID: accountID,
				Type:          transaction.TypeIncome,
				Status:        transaction.StatusConfirmed,
				Amount:        1000,
				OccurredOn:    time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:            "tx2",
				ProfileID:     profileID,
				BankAccountID: accountID,
				Type:          transaction.TypeExpense,
				Status:        transaction.StatusConfirmed,
				Amount:        200,
				OccurredOn:    time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	useCase := NewGetDailyBalancesUseCase(txRepo, accountRepo)

	result, err := useCase.Execute(GetDailyBalancesInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result))
	}

	// Day 1 (Jan 5): startBalance=100, +1000 = 1100
	if result[0].Balance != 1100 {
		t.Errorf("day 1 balance should be 1100 (initial 100 + income 1000), got %.2f", result[0].Balance)
	}

	// Day 2 (Jan 10): 1100 - 200 = 900 (matches current_balance)
	if result[1].Balance != 900 {
		t.Errorf("day 2 balance should be 900 (1100 - 200), got %.2f", result[1].Balance)
	}
}

// Fake bank account repo for daily balance tests
type fakeBankAccountRepoForBalance struct {
	accounts map[string]*bankaccount.BankAccount
}

func (f *fakeBankAccountRepoForBalance) Create(account *bankaccount.BankAccount) error {
	return nil
}
func (f *fakeBankAccountRepoForBalance) FindByID(id string) (*bankaccount.BankAccount, error) {
	if acc, ok := f.accounts[id]; ok {
		return acc, nil
	}
	return nil, nil
}
func (f *fakeBankAccountRepoForBalance) FindByProfileID(profileID string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (f *fakeBankAccountRepoForBalance) FindAll() ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (f *fakeBankAccountRepoForBalance) Update(account *bankaccount.BankAccount) error {
	return nil
}
func (f *fakeBankAccountRepoForBalance) Delete(id string) error {
	return nil
}
func (f *fakeBankAccountRepoForBalance) UpdateDisplayOrders(updates []bankaccount.DisplayOrderUpdate) error {
	return nil
}
