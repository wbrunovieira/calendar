package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

func makeAccountWithCurrency(id, currency string) *bankaccount.BankAccount {
	now := time.Now()
	return &bankaccount.BankAccount{
		ID:             id,
		ProfileID:      "profile-1",
		Name:           id,
		Type:           bankaccount.AccountTypeChecking,
		InitialBalance: 0,
		CurrentBalance: 0,
		Currency:       currency,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func baseUpdateInput(name, currency string) UpdateBankAccountInput {
	return UpdateBankAccountInput{
		ProfileID: "profile-1",
		Name:      name,
		Type:      string(bankaccount.AccountTypeChecking),
		Currency:  currency,
		IsActive:  true,
	}
}

// Given: account with BRL currency, update sends empty currency
// When: Execute
// Then: currency must remain BRL (not be overwritten with empty string)
func TestUpdateBankAccount_EmptyCurrencyKeepsExisting(t *testing.T) {
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-1": makeAccountWithCurrency("acc-1", "BRL"),
	}}

	uc := NewUpdateBankAccountUseCase(repo)
	_, err := uc.Execute("acc-1", baseUpdateInput("acc-1", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.accounts["acc-1"].Currency != "BRL" {
		t.Fatalf("expected currency to remain BRL, got %q", repo.accounts["acc-1"].Currency)
	}
}

// Given: account with BRL currency, update sends invalid currency
// When: Execute
// Then: returns error
func TestUpdateBankAccount_InvalidCurrencyReturnsError(t *testing.T) {
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-1": makeAccountWithCurrency("acc-1", "BRL"),
	}}

	uc := NewUpdateBankAccountUseCase(repo)
	_, err := uc.Execute("acc-1", baseUpdateInput("acc-1", "XYZ"))
	if err == nil {
		t.Fatal("expected error for invalid currency, got nil")
	}
}

// Given: account linked to another account, update sends payload without linkedAccountId
// When: Execute
// Then: linkedAccountId is preserved (not zeroed)
func TestUpdateBankAccount_NilLinkedAccountIDPreservesExisting(t *testing.T) {
	linkedID := "parent-account-id"
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-1": {
			ID:              "acc-1",
			ProfileID:       "profile-1",
			Name:            "acc-1",
			Type:            bankaccount.AccountTypeInvestment,
			Currency:        "BRL",
			IsActive:        true,
			LinkedAccountID: &linkedID,
		},
	}}

	uc := NewUpdateBankAccountUseCase(repo)
	input := baseUpdateInput("acc-1", "BRL")
	// LinkedAccountID is nil (omitted from payload)
	_, err := uc.Execute("acc-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := repo.accounts["acc-1"].LinkedAccountID
	if got == nil || *got != linkedID {
		t.Fatalf("expected linkedAccountId %q to be preserved, got %v", linkedID, got)
	}
}

// Given: account with BRL, update sends USD (valid)
// When: Execute
// Then: currency is updated to USD
func TestUpdateBankAccount_ValidCurrencyUpdates(t *testing.T) {
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-1": makeAccountWithCurrency("acc-1", "BRL"),
	}}

	uc := NewUpdateBankAccountUseCase(repo)
	_, err := uc.Execute("acc-1", baseUpdateInput("acc-1", "USD"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.accounts["acc-1"].Currency != "USD" {
		t.Fatalf("expected currency USD, got %q", repo.accounts["acc-1"].Currency)
	}
}

// Given: an account whose seed balance was cadastrado wrong
// When: Execute is called without initialBalance in the payload
// Then: the seed is preserved (nil means "not sent")
func TestUpdateBankAccount_NilInitialBalancePreservesExisting(t *testing.T) {
	acc := makeAccountWithCurrency("acc-1", "BRL")
	acc.InitialBalance = 3428.08
	acc.CurrentBalance = 48.01
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-1": acc}}

	uc := NewUpdateBankAccountUseCase(repo)
	input := baseUpdateInput("acc-1", "BRL")
	input.CurrentBalance = 48.01
	if _, err := uc.Execute("acc-1", input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := repo.accounts["acc-1"].InitialBalance; got != 3428.08 {
		t.Fatalf("expected initial balance to be preserved at 3428.08, got %v", got)
	}
	if got := repo.accounts["acc-1"].CurrentBalance; got != 48.01 {
		t.Fatalf("expected current balance untouched at 48.01, got %v", got)
	}
}

// Given: an account seeded 35.70 above the real opening balance
// When: the seed is corrected downwards
// Then: the current balance shifts by the same delta, because
//
//	balance = initialBalance + sum(transactions) and the transactions did not change
func TestUpdateBankAccount_InitialBalanceShiftsCurrentBalance(t *testing.T) {
	acc := makeAccountWithCurrency("acc-1", "BRL")
	acc.InitialBalance = 3428.08
	acc.CurrentBalance = 48.01
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-1": acc}}

	corrected := 3392.38
	uc := NewUpdateBankAccountUseCase(repo)
	input := baseUpdateInput("acc-1", "BRL")
	input.CurrentBalance = 48.01
	input.InitialBalance = &corrected

	if _, err := uc.Execute("acc-1", input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := repo.accounts["acc-1"]
	if updated.InitialBalance != corrected {
		t.Fatalf("expected initial balance %v, got %v", corrected, updated.InitialBalance)
	}
	// 48.01 - 35.70 = 12.31
	if diff := updated.CurrentBalance - 12.31; diff > 0.005 || diff < -0.005 {
		t.Fatalf("expected current balance 12.31 after the seed correction, got %v", updated.CurrentBalance)
	}
}

// Given: an account being updated for an unrelated reason (a rename)
// When: initialBalance is sent unchanged
// Then: the current balance is not shifted
func TestUpdateBankAccount_UnchangedInitialBalanceDoesNotShiftCurrentBalance(t *testing.T) {
	acc := makeAccountWithCurrency("acc-1", "BRL")
	acc.InitialBalance = 100
	acc.CurrentBalance = 250
	repo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-1": acc}}

	same := 100.0
	uc := NewUpdateBankAccountUseCase(repo)
	input := baseUpdateInput("renamed", "BRL")
	input.CurrentBalance = 250
	input.InitialBalance = &same

	if _, err := uc.Execute("acc-1", input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := repo.accounts["acc-1"].CurrentBalance; got != 250 {
		t.Fatalf("expected current balance to stay at 250, got %v", got)
	}
}
