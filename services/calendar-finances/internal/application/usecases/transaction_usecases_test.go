package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type fakeProfileRepo struct {
	profiles map[string]*profile.Profile
}

func (f *fakeProfileRepo) Create(*profile.Profile) error { return nil }
func (f *fakeProfileRepo) FindByCalendarID(string) (*profile.Profile, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeProfileRepo) FindAll() ([]*profile.Profile, error) { return nil, nil }
func (f *fakeProfileRepo) Update(*profile.Profile) error        { return nil }
func (f *fakeProfileRepo) Delete(string) error                  { return nil }
func (f *fakeProfileRepo) FindByID(id string) (*profile.Profile, error) {
	if p, ok := f.profiles[id]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

type fakeAccountRepo struct {
	accounts map[string]*bankaccount.BankAccount
}

func (f *fakeAccountRepo) Create(*bankaccount.BankAccount) error { return nil }
func (f *fakeAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (f *fakeAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) { return nil, nil }
func (f *fakeAccountRepo) Update(*bankaccount.BankAccount) error        { return nil }
func (f *fakeAccountRepo) Delete(string) error                          { return nil }
func (f *fakeAccountRepo) FindByID(id string) (*bankaccount.BankAccount, error) {
	if acc, ok := f.accounts[id]; ok {
		return acc, nil
	}
	return nil, errors.New("not found")
}

type fakeCategoryRepo struct {
	categories map[string]*category.Category
}

func (f *fakeCategoryRepo) Create(*category.Category) error { return nil }
func (f *fakeCategoryRepo) Update(*category.Category) error { return nil }
func (f *fakeCategoryRepo) FindByID(id string) (*category.Category, error) {
	if cat, ok := f.categories[id]; ok {
		return cat, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeCategoryRepo) ListByProfile(profileID string) ([]*category.Category, error) {
	var list []*category.Category
	for _, cat := range f.categories {
		if cat.ProfileID == profileID {
			list = append(list, cat)
		}
	}
	return list, nil
}
func (f *fakeCategoryRepo) Deactivate(string) error { return nil }

type fakeTransactionRepo struct {
	created []*transaction.Transaction
}

func (f *fakeTransactionRepo) Create(tx *transaction.Transaction) error {
	f.created = append(f.created, tx)
	return nil
}

func (f *fakeTransactionRepo) GetByID(id string) (*transaction.Transaction, error) {
	for _, tx := range f.created {
		if tx.ID == id {
			return tx, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeTransactionRepo) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	return f.created, nil
}

func (f *fakeTransactionRepo) UpdateStatus(string, transaction.Status, time.Time, *string) error {
	return nil
}
func (f *fakeTransactionRepo) Delete(string) error { return nil }

func (f *fakeTransactionRepo) SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error) {
	result := make(map[string]float64)
	for _, tx := range f.created {
		if tx.ProfileID != profileID {
			continue
		}
		if tx.CategoryID == nil {
			continue
		}
		if tx.Type != transaction.TypeExpense {
			continue
		}
		result[*tx.CategoryID] += tx.Amount
	}
	return result, nil
}

func TestCreateTransactionUseCaseExpense(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "WB Digital",
			Type:       profile.ProfileTypeBusiness,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta Corrente",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Software",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}

	txRepo := &fakeTransactionRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo)
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Amount:        200,
		Currency:      "BRL",
		Description:   "Assinatura SaaS",
		OccurredOn:    "2025-02-01",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Type != transaction.TypeExpense {
		t.Fatalf("expected expense type, got %s", tx.Type)
	}

	if len(txRepo.created) != 1 {
		t.Fatalf("expected transaction persisted, got %d", len(txRepo.created))
	}
}

func TestCreateTransactionUseCaseCreditLimitExceeded(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-cc"

	now := time.Now()
	limit := 300.0
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "WB Digital",
			Type:       profile.ProfileTypeBusiness,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Cartão",
			Type:           bankaccount.AccountTypeCreditCard,
			InitialBalance: 0,
			CurrentBalance: 0,
			Currency:       "BRL",
			IsActive:       true,
			CreditLimit:    &limit,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo)
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        400,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2025-02-01",
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("expected credit limit error, got nil")
	}

	if !errors.Is(err, ErrCreditLimitExceeded) {
		t.Fatalf("expected ErrCreditLimitExceeded, got %v", err)
	}
}
