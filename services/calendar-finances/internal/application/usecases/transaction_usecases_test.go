package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
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
	accounts      map[string]*bankaccount.BankAccount
	updateCalled  bool
	lastUpdatedID string
}

func (f *fakeAccountRepo) Create(*bankaccount.BankAccount) error { return nil }
func (f *fakeAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (f *fakeAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) { return nil, nil }
func (f *fakeAccountRepo) Update(acc *bankaccount.BankAccount) error {
	f.updateCalled = true
	f.lastUpdatedID = acc.ID
	// Update the account in the map to reflect the change
	if f.accounts != nil {
		f.accounts[acc.ID] = acc
	}
	return nil
}
func (f *fakeAccountRepo) Delete(string) error { return nil }
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
func (f *fakeCategoryRepo) GetDescendantIDs(id string) ([]string, error) {
	// Simple implementation: find the category and all children recursively
	ids := []string{id}
	for _, cat := range f.categories {
		if cat.ParentID != nil && *cat.ParentID == id {
			childIDs, _ := f.GetDescendantIDs(cat.ID)
			ids = append(ids, childIDs...)
		}
	}
	return ids, nil
}

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

func (f *fakeTransactionRepo) Update(tx *transaction.Transaction) error {
	for i, existing := range f.created {
		if existing.ID == tx.ID {
			f.created[i] = tx
			return nil
		}
	}
	return errors.New("not found")
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

func (f *fakeTransactionRepo) SumByInvoiceID(invoiceID string) (float64, error) {
	var total float64
	for _, tx := range f.created {
		if tx.InvoiceID != nil && *tx.InvoiceID == invoiceID {
			total += tx.Amount
		}
	}
	return total, nil
}

type fakeInvoiceRepo struct {
	invoices map[string]*invoice.Invoice
}

func (f *fakeInvoiceRepo) Create(inv *invoice.Invoice) error {
	if f.invoices == nil {
		f.invoices = make(map[string]*invoice.Invoice)
	}
	f.invoices[inv.ID] = inv
	return nil
}

func (f *fakeInvoiceRepo) FindByID(id string) (*invoice.Invoice, error) {
	if inv, ok := f.invoices[id]; ok {
		return inv, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeInvoiceRepo) FindByBankAccountID(bankAccountID string) ([]*invoice.Invoice, error) {
	var list []*invoice.Invoice
	for _, inv := range f.invoices {
		if inv.BankAccountID == bankAccountID {
			list = append(list, inv)
		}
	}
	return list, nil
}

func (f *fakeInvoiceRepo) FindOpenByBankAccountID(bankAccountID string) (*invoice.Invoice, error) {
	for _, inv := range f.invoices {
		if inv.BankAccountID == bankAccountID && inv.Status == invoice.StatusOpen {
			return inv, nil
		}
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) FindByBankAccountAndDate(bankAccountID string, txDate time.Time) (*invoice.Invoice, error) {
	for _, inv := range f.invoices {
		if inv.BankAccountID == bankAccountID && inv.ContainsDate(txDate) {
			return inv, nil
		}
	}
	return nil, nil
}

func (f *fakeInvoiceRepo) Update(inv *invoice.Invoice) error {
	if f.invoices == nil {
		return errors.New("not found")
	}
	f.invoices[inv.ID] = inv
	return nil
}

func (f *fakeInvoiceRepo) Delete(id string) error {
	delete(f.invoices, id)
	return nil
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
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
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
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	confirmedStatus := "CONFIRMED"
	// Use today's date so balance validation runs (historical dates skip validation)
	today := time.Now().Format("2006-01-02")
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        400,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    today,
		Status:        &confirmedStatus, // Balance validation only runs for CONFIRMED transactions
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("expected credit limit error, got nil")
	}

	if !errors.Is(err, ErrCreditLimitExceeded) {
		t.Fatalf("expected ErrCreditLimitExceeded, got %v", err)
	}
}

func TestUpdateTransactionUseCaseSuccess(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"
	newCategoryID := "cat-2"
	txID := "tx-1"

	now := time.Now()

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
			Name:      "Alimentação",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		newCategoryID: {
			ID:        newCategoryID,
			ProfileID: profileID,
			Name:      "iFood",
			Type:      category.TypeExpense,
			ParentID:  &categoryID,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Almoço",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo)
	input := UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &newCategoryID,
		Type:          "EXPENSE",
		Amount:        150,
		Currency:      "BRL",
		Description:   "iFood Delivery",
		OccurredOn:    "2025-02-01",
	}

	tx, err := useCase.Execute(txID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Amount != 150 {
		t.Fatalf("expected amount 150, got %v", tx.Amount)
	}

	if tx.Description != "iFood Delivery" {
		t.Fatalf("expected description 'iFood Delivery', got %s", tx.Description)
	}

	if tx.CategoryID == nil || *tx.CategoryID != newCategoryID {
		t.Fatalf("expected category to be updated to %s", newCategoryID)
	}
}

func TestUpdateTransactionUseCaseNotFound(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{}}

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo)
	input := UpdateTransactionInput{
		BankAccountID: "account-1",
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "Test",
		OccurredOn:    "2025-02-01",
	}

	_, err := useCase.Execute("non-existent", input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrTransactionNotFound) {
		t.Fatalf("expected ErrTransactionNotFound, got %v", err)
	}
}

func TestUpdateTransactionUseCaseWithStatusChange(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	txID := "tx-1"

	now := time.Now()

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

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo)
	confirmedStatus := "CONFIRMED"
	input := UpdateTransactionInput{
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2025-02-01",
	}

	tx, err := useCase.Execute(txID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != transaction.StatusConfirmed {
		t.Fatalf("expected status CONFIRMED, got %s", tx.Status)
	}
}

func TestUpdateTransactionUseCaseInvalidCategory(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	txID := "tx-1"
	wrongCategoryID := "wrong-cat"

	now := time.Now()

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

	// Category belongs to different profile
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		wrongCategoryID: {
			ID:        wrongCategoryID,
			ProfileID: "other-profile",
			Name:      "Other Category",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Test",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo)
	input := UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &wrongCategoryID,
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "Test",
		OccurredOn:    "2025-02-01",
	}

	_, err := useCase.Execute(txID, input)
	if err == nil {
		t.Fatal("expected error for category from different profile")
	}

	if !errors.Is(err, ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound, got %v", err)
	}
}

func TestCreateTransactionUseCaseWithStatusConfirmed(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Bruno Pessoal",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Nubank",
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
			Name:      "Alimentação",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}

	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        26.42,
		Currency:      "BRL",
		Description:   "iFood",
		OccurredOn:    "2026-01-06",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != transaction.StatusConfirmed {
		t.Fatalf("expected status CONFIRMED, got %s", tx.Status)
	}

	if tx.Amount != 26.42 {
		t.Fatalf("expected amount 26.42, got %v", tx.Amount)
	}
}

func TestCreateTransactionUseCaseDefaultStatusPlanned(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Bruno Pessoal",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Nubank",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// No status provided - should default to PLANNED
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        50.00,
		Currency:      "BRL",
		Description:   "Compra futura",
		OccurredOn:    "2026-01-10",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != transaction.StatusPlanned {
		t.Fatalf("expected default status PLANNED, got %s", tx.Status)
	}
}

func TestCreateTransactionUseCaseInvalidStatus(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	invalidStatus := "INVALID_STATUS"
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &invalidStatus,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Test",
		OccurredOn:    "2026-01-06",
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateTransactionPlannedSkipsBalanceValidation(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	// Account with only R$100, but we'll try to create R$1000 PLANNED expense
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 100,
			CurrentBalance: 100,
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
			Name:      "Equipamentos",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// PLANNED status should skip balance validation
	plannedStatus := "PLANNED"
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &plannedStatus,
		Amount:        1000, // More than current balance of 100
		Currency:      "BRL",
		Description:   "GPS de Voo - Flylimit (4/4)",
		OccurredOn:    "2026-01-21",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("PLANNED transaction should skip balance validation, got error: %v", err)
	}

	if tx.Status != transaction.StatusPlanned {
		t.Fatalf("expected status PLANNED, got %s", tx.Status)
	}

	if tx.Amount != 1000 {
		t.Fatalf("expected amount 1000, got %f", tx.Amount)
	}
}

func TestCreateTransactionConfirmedValidatesBalance(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	// Account with only R$100
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 100,
			CurrentBalance: 100,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// CONFIRMED status should validate balance (use today's date, historical dates skip validation)
	confirmedStatus := "CONFIRMED"
	today := time.Now().Format("2006-01-02")
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        1000, // More than current balance of 100
		Currency:      "BRL",
		Description:   "Compra grande",
		OccurredOn:    today,
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("CONFIRMED transaction should validate balance, expected error")
	}

	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestCreateTransactionDefaultStatusSkipsBalanceValidation(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	// Account with only R$100
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 100,
			CurrentBalance: 100,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// No status provided - defaults to PLANNED, should skip balance validation
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        1000, // More than current balance of 100
		Currency:      "BRL",
		Description:   "Despesa futura",
		OccurredOn:    "2026-01-21",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("Default PLANNED status should skip balance validation, got error: %v", err)
	}

	if tx.Status != transaction.StatusPlanned {
		t.Fatalf("expected default status PLANNED, got %s", tx.Status)
	}
}

func TestCreateTransactionWithReminderOn(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Bruno Pessoal",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 0,
			CurrentBalance: 0,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	reminderOn := "2025-03-10" // 10 days before occurredOn
	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "INCOME",
		Amount:        700,
		Currency:      "BRL",
		Description:   "Devolucao de curso Vagner Santa Rita",
		OccurredOn:    "2025-03-20",
		ReminderOn:    &reminderOn,
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.ReminderOn == nil {
		t.Fatal("expected ReminderOn to be set")
	}

	expectedReminder := time.Date(2025, time.March, 10, 0, 0, 0, 0, time.UTC)
	if !tx.ReminderOn.Equal(expectedReminder) {
		t.Fatalf("expected ReminderOn %v, got %v", expectedReminder, *tx.ReminderOn)
	}
}

func TestCreateTransactionWithoutReminderOn(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra sem lembrete",
		OccurredOn:    "2025-03-20",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.ReminderOn != nil {
		t.Fatalf("expected ReminderOn to be nil when not provided, got %v", *tx.ReminderOn)
	}
}

// =============================================================================
// Historical Transaction Balance Validation Tests (TDD)
// =============================================================================

func TestCreateTransaction_HistoricalConfirmed_ShouldSkipBalanceValidation(t *testing.T) {
	// Given: An account with R$100 balance
	// When: Creating a CONFIRMED expense of R$800 for a past date (yesterday)
	// Then: Should succeed without balance validation (historical transaction)

	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	// Account with only R$100
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 100,
			CurrentBalance: 100,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// Yesterday's date (historical)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	confirmedStatus := "CONFIRMED"

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        800, // More than current balance of 100
		Currency:      "BRL",
		Description:   "Residencial Mae",
		OccurredOn:    yesterday,
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("Historical CONFIRMED transaction should skip balance validation, got error: %v", err)
	}

	if tx.Status != transaction.StatusConfirmed {
		t.Fatalf("expected status CONFIRMED, got %s", tx.Status)
	}

	if tx.Amount != 800 {
		t.Fatalf("expected amount 800, got %.2f", tx.Amount)
	}
}

func TestCreateTransaction_TodayConfirmed_ShouldValidateBalance(t *testing.T) {
	// Given: An account with R$100 balance
	// When: Creating a CONFIRMED expense of R$800 for today
	// Then: Should fail with insufficient balance (current/future transaction)

	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	// Account with only R$100
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 100,
			CurrentBalance: 100,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// Today's date
	today := time.Now().Format("2006-01-02")
	confirmedStatus := "CONFIRMED"

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        800, // More than current balance of 100
		Currency:      "BRL",
		Description:   "Compra grande",
		OccurredOn:    today,
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("Today's CONFIRMED transaction should validate balance, expected error")
	}

	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestCreateTransaction_FutureConfirmed_ShouldValidateBalance(t *testing.T) {
	// Given: An account with R$100 balance
	// When: Creating a CONFIRMED expense of R$800 for tomorrow
	// Then: Should fail with insufficient balance (future transaction)

	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	// Account with only R$100
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 100,
			CurrentBalance: 100,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	// Tomorrow's date
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	confirmedStatus := "CONFIRMED"

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        800, // More than current balance of 100
		Currency:      "BRL",
		Description:   "Compra futura",
		OccurredOn:    tomorrow,
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("Future CONFIRMED transaction should validate balance, expected error")
	}

	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

// =============================================================================
// Balance Update Tests (TDD)
// =============================================================================

func TestCreateTransaction_ConfirmedExpense_ShouldDecreaseBalance(t *testing.T) {
	// Given: An account with R$1000 balance
	// When: Creating a CONFIRMED expense of R$300
	// Then: Account balance should decrease to R$700

	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        300,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    yesterday,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the account was updated
	if !accountRepo.updateCalled {
		t.Fatal("expected accountRepo.Update to be called")
	}

	// Verify the balance was decreased
	account := accountRepo.accounts[accountID]
	expectedBalance := 700.0 // 1000 - 300
	if account.CurrentBalance != expectedBalance {
		t.Fatalf("expected balance %.2f, got %.2f", expectedBalance, account.CurrentBalance)
	}
}

func TestCreateTransaction_ConfirmedIncome_ShouldIncreaseBalance(t *testing.T) {
	// Given: An account with R$500 balance
	// When: Creating a CONFIRMED income of R$1000
	// Then: Account balance should increase to R$1500

	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 500,
			CurrentBalance: 500,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "INCOME",
		Status:        &confirmedStatus,
		Amount:        1000,
		Currency:      "BRL",
		Description:   "Salario",
		OccurredOn:    yesterday,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the account was updated
	if !accountRepo.updateCalled {
		t.Fatal("expected accountRepo.Update to be called")
	}

	// Verify the balance was increased
	account := accountRepo.accounts[accountID]
	expectedBalance := 1500.0 // 500 + 1000
	if account.CurrentBalance != expectedBalance {
		t.Fatalf("expected balance %.2f, got %.2f", expectedBalance, account.CurrentBalance)
	}
}

func TestCreateTransaction_PlannedTransaction_ShouldNotUpdateBalance(t *testing.T) {
	// Given: An account with R$1000 balance
	// When: Creating a PLANNED expense of R$500
	// Then: Account balance should remain R$1000 (no update)

	profileID := "profile-1"
	accountID := "account-1"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Test",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)

	plannedStatus := "PLANNED"
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &plannedStatus,
		Amount:        500,
		Currency:      "BRL",
		Description:   "Compra futura",
		OccurredOn:    tomorrow,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the account was NOT updated (PLANNED transactions don't affect balance)
	if accountRepo.updateCalled {
		t.Fatal("expected accountRepo.Update NOT to be called for PLANNED transactions")
	}

	// Verify the balance remains unchanged
	account := accountRepo.accounts[accountID]
	if account.CurrentBalance != 1000 {
		t.Fatalf("expected balance 1000, got %.2f", account.CurrentBalance)
	}
}
