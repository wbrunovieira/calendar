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
