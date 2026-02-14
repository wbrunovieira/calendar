package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// --- fake repo for reorder + create auto-order tests ---

type fakeBankAccountRepoForReorder struct {
	accounts      []*bankaccount.BankAccount
	updatedOrders []bankaccount.DisplayOrderUpdate
	updateErr     error
	created       *bankaccount.BankAccount
}

func (f *fakeBankAccountRepoForReorder) Create(acc *bankaccount.BankAccount) error {
	f.created = acc
	return nil
}
func (f *fakeBankAccountRepoForReorder) FindByID(id string) (*bankaccount.BankAccount, error) {
	for _, a := range f.accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeBankAccountRepoForReorder) FindByProfileID(profileID string) ([]*bankaccount.BankAccount, error) {
	var result []*bankaccount.BankAccount
	for _, a := range f.accounts {
		if a.ProfileID == profileID {
			result = append(result, a)
		}
	}
	return result, nil
}
func (f *fakeBankAccountRepoForReorder) FindAll() ([]*bankaccount.BankAccount, error) {
	return f.accounts, nil
}
func (f *fakeBankAccountRepoForReorder) Update(*bankaccount.BankAccount) error { return nil }
func (f *fakeBankAccountRepoForReorder) Delete(string) error                   { return nil }
func (f *fakeBankAccountRepoForReorder) UpdateDisplayOrders(updates []bankaccount.DisplayOrderUpdate) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updatedOrders = updates
	return nil
}

// --- Reorder Use Case Tests ---

func TestReorderBankAccounts_Success(t *testing.T) {
	repo := &fakeBankAccountRepoForReorder{}
	uc := NewReorderBankAccountsUseCase(repo)

	items := []ReorderItem{
		{ID: "acc-1", DisplayOrder: 2},
		{ID: "acc-2", DisplayOrder: 1},
		{ID: "acc-3", DisplayOrder: 3},
	}

	err := uc.Execute(items)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.updatedOrders) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(repo.updatedOrders))
	}

	// Verify mapping
	for i, item := range items {
		if repo.updatedOrders[i].ID != item.ID {
			t.Errorf("update[%d].ID = %s, want %s", i, repo.updatedOrders[i].ID, item.ID)
		}
		if repo.updatedOrders[i].DisplayOrder != item.DisplayOrder {
			t.Errorf("update[%d].DisplayOrder = %d, want %d", i, repo.updatedOrders[i].DisplayOrder, item.DisplayOrder)
		}
	}
}

func TestReorderBankAccounts_EmptyList(t *testing.T) {
	repo := &fakeBankAccountRepoForReorder{}
	uc := NewReorderBankAccountsUseCase(repo)

	err := uc.Execute([]ReorderItem{})
	if err != nil {
		t.Fatalf("expected no error for empty list, got %v", err)
	}
}

func TestReorderBankAccounts_RepoError(t *testing.T) {
	repo := &fakeBankAccountRepoForReorder{
		updateErr: errors.New("db connection failed"),
	}
	uc := NewReorderBankAccountsUseCase(repo)

	err := uc.Execute([]ReorderItem{{ID: "acc-1", DisplayOrder: 1}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db connection failed" {
		t.Errorf("expected 'db connection failed', got '%s'", err.Error())
	}
}

// --- Create Bank Account: Auto DisplayOrder Tests ---

func TestCreateBankAccount_AutoDisplayOrder_FirstAccount(t *testing.T) {
	repo := &fakeBankAccountRepoForReorder{
		accounts: []*bankaccount.BankAccount{}, // no existing accounts
	}
	uc := NewCreateBankAccountUseCase(repo)

	result, err := uc.Execute(CreateBankAccountInput{
		ProfileID: "profile-1",
		Name:      "Nubank",
		Type:      "CHECKING",
		Currency:  "BRL",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.DisplayOrder == nil {
		t.Fatal("expected DisplayOrder to be auto-assigned, got nil")
	}
	if *result.DisplayOrder != 1 {
		t.Errorf("expected DisplayOrder=1 for first account, got %d", *result.DisplayOrder)
	}
}

func TestCreateBankAccount_AutoDisplayOrder_WithExistingAccounts(t *testing.T) {
	order1 := 1
	order2 := 2
	repo := &fakeBankAccountRepoForReorder{
		accounts: []*bankaccount.BankAccount{
			{ID: "existing-1", ProfileID: "profile-1", DisplayOrder: &order1},
			{ID: "existing-2", ProfileID: "profile-1", DisplayOrder: &order2},
		},
	}
	uc := NewCreateBankAccountUseCase(repo)

	result, err := uc.Execute(CreateBankAccountInput{
		ProfileID: "profile-1",
		Name:      "Inter",
		Type:      "CHECKING",
		Currency:  "BRL",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.DisplayOrder == nil {
		t.Fatal("expected DisplayOrder to be auto-assigned, got nil")
	}
	if *result.DisplayOrder != 3 {
		t.Errorf("expected DisplayOrder=3 (max+1), got %d", *result.DisplayOrder)
	}
}

func TestCreateBankAccount_AutoDisplayOrder_OnlyCountsSameProfile(t *testing.T) {
	order1 := 1
	order5 := 5
	repo := &fakeBankAccountRepoForReorder{
		accounts: []*bankaccount.BankAccount{
			{ID: "other-profile", ProfileID: "profile-2", DisplayOrder: &order5},
			{ID: "same-profile", ProfileID: "profile-1", DisplayOrder: &order1},
		},
	}
	uc := NewCreateBankAccountUseCase(repo)

	result, err := uc.Execute(CreateBankAccountInput{
		ProfileID: "profile-1",
		Name:      "Bradesco",
		Type:      "CHECKING",
		Currency:  "BRL",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be max of profile-1 orders (1) + 1 = 2, NOT 6
	if result.DisplayOrder == nil {
		t.Fatal("expected DisplayOrder to be auto-assigned, got nil")
	}
	if *result.DisplayOrder != 2 {
		t.Errorf("expected DisplayOrder=2 (profile max+1), got %d", *result.DisplayOrder)
	}
}

func TestCreateBankAccount_ExplicitDisplayOrder_IsPreserved(t *testing.T) {
	repo := &fakeBankAccountRepoForReorder{
		accounts: []*bankaccount.BankAccount{},
	}
	uc := NewCreateBankAccountUseCase(repo)

	explicitOrder := 42
	result, err := uc.Execute(CreateBankAccountInput{
		ProfileID:    "profile-1",
		Name:         "Custom Order",
		Type:         "SAVINGS",
		Currency:     "BRL",
		DisplayOrder: &explicitOrder,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.DisplayOrder == nil || *result.DisplayOrder != 42 {
		t.Errorf("expected explicit DisplayOrder=42 to be preserved, got %v", result.DisplayOrder)
	}
}
