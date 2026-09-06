package usecases

import (
	"errors"
	"strings"
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
	updateCount   int
}

func (f *fakeAccountRepo) Create(*bankaccount.BankAccount) error { return nil }
func (f *fakeAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (f *fakeAccountRepo) FindAll() ([]*bankaccount.BankAccount, error) { return nil, nil }
func (f *fakeAccountRepo) Update(acc *bankaccount.BankAccount) error {
	f.updateCalled = true
	f.lastUpdatedID = acc.ID
	f.updateCount++
	// Update the account in the map to reflect the change
	if f.accounts != nil {
		f.accounts[acc.ID] = acc
	}
	return nil
}
func (f *fakeAccountRepo) Delete(string) error                                        { return nil }
func (f *fakeAccountRepo) UpdateDisplayOrders([]bankaccount.DisplayOrderUpdate) error { return nil }
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

func strPtr(s string) *string { return &s }

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

func (f *fakeTransactionRepo) UpdateStatus(id string, status transaction.Status, occurredOn time.Time, notes *string) error {
	for _, tx := range f.created {
		if tx.ID == id {
			tx.Status = status
			tx.OccurredOn = occurredOn
			tx.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("not found")
}
func (f *fakeTransactionRepo) Delete(id string) error {
	for i, tx := range f.created {
		if tx.ID == id {
			f.created = append(f.created[:i], f.created[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

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

// invoiceSigned mirrors the SQL the repository runs: an invoice total is charges
// minus credits, so an INCOME linked to an invoice reduces it. Keeping the fakes
// on the same rule stops a unit test from agreeing with a bug the database
// would reject.
func invoiceSigned(tx *transaction.Transaction) float64 {
	if tx.Type == transaction.TypeIncome {
		return -tx.Amount
	}
	return tx.Amount
}

// The repository skips CANCELLED rows; so must this, or a unit test can pass on
// a total the database would never return.
func (f *fakeTransactionRepo) SumByInvoiceID(invoiceID string) (float64, error) {
	var total float64
	for _, tx := range f.created {
		if tx.InvoiceID != nil && *tx.InvoiceID == invoiceID && tx.Status != transaction.StatusCancelled {
			total += invoiceSigned(tx)
		}
	}
	return total, nil
}

func (f *fakeTransactionRepo) SumByInvoiceIDByStatus(invoiceID string, status transaction.Status) (float64, error) {
	var total float64
	for _, tx := range f.created {
		if tx.InvoiceID != nil && *tx.InvoiceID == invoiceID && tx.Status == status {
			total += invoiceSigned(tx)
		}
	}
	return total, nil
}

func (f *fakeTransactionRepo) CalculateBalanceByBankAccountID(bankAccountID string) (float64, error) {
	var balance float64
	for _, tx := range f.created {
		if tx.Status != transaction.StatusConfirmed {
			continue
		}
		if tx.BankAccountID == bankAccountID {
			switch tx.Type {
			case transaction.TypeIncome:
				balance += tx.Amount
			case transaction.TypeExpense:
				balance -= tx.Amount
			case transaction.TypeTransfer:
				balance -= tx.Amount
			}
		}
		if tx.DestinationAccountID != nil && *tx.DestinationAccountID == bankAccountID && tx.Type == transaction.TypeTransfer {
			balance += tx.Amount
		}
	}
	return balance, nil
}

func (f *fakeTransactionRepo) FindByExternalID(externalID string) (*transaction.Transaction, error) {
	return nil, nil
}

type fakeInvoiceRepo struct {
	invoices map[string]*invoice.Invoice
}

func (f *fakeInvoiceRepo) Create(inv *invoice.Invoice) error {
	if f.invoices == nil {
		f.invoices = make(map[string]*invoice.Invoice)
	}
	// Check for duplicate reference month + bank account (like real DB constraint)
	for _, existing := range f.invoices {
		if existing.BankAccountID == inv.BankAccountID &&
			existing.ReferenceDate.Year() == inv.ReferenceDate.Year() &&
			existing.ReferenceDate.Month() == inv.ReferenceDate.Month() {
			return errors.New("pq: duplicate key value violates unique constraint \"uq_invoice_account_reference\"")
		}
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

func (f *fakeInvoiceRepo) FindOpenPastClosingDate(now time.Time) ([]*invoice.Invoice, error) {
	var list []*invoice.Invoice
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, inv := range f.invoices {
		if inv.Status == invoice.StatusOpen {
			closingDate := time.Date(inv.ClosingDate.Year(), inv.ClosingDate.Month(), inv.ClosingDate.Day(), 0, 0, 0, 0, time.UTC)
			if nowDate.After(closingDate) {
				list = append(list, inv)
			}
		}
	}
	return list, nil
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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
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

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)
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

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)
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

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)
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

	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)
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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

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

// =============================================================================
// Credit Card Balance Tests (TDD)
// =============================================================================

func TestCreateTransaction_CreditCardExpense_ShouldNotUpdateBalance(t *testing.T) {
	// Given: A credit card with R$0 balance and R$2000 limit
	// When: Creating a CONFIRMED expense of R$500 on the credit card
	// Then: Credit card balance should remain R$0 (no balance update for credit cards)
	//       Only the invoice amount should be updated

	profileID := "profile-1"
	creditCardID := "cc-1"
	checkingID := "checking-1"

	now := time.Now()
	closingDay := 9
	dueDay := 14

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

	limit := 2000.0
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		creditCardID: {
			ID:              creditCardID,
			ProfileID:       profileID,
			Name:            "Cartão Mercado Pago",
			Type:            bankaccount.AccountTypeCreditCard,
			InitialBalance:  0,
			CurrentBalance:  0,
			Currency:        "BRL",
			IsActive:        true,
			CreditLimit:     &limit,
			ClosingDay:      &closingDay,
			DueDay:          &dueDay,
			LinkedAccountID: &checkingID,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		checkingID: {
			ID:             checkingID,
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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: creditCardID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        500,
		Currency:      "BRL",
		Description:   "Compra no cartão",
		OccurredOn:    yesterday,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the credit card balance was NOT updated
	creditCard := accountRepo.accounts[creditCardID]
	if creditCard.CurrentBalance != 0 {
		t.Fatalf("Credit card balance should remain 0, got %.2f", creditCard.CurrentBalance)
	}

	// Verify the checking account balance was also NOT updated (payment hasn't happened yet)
	checking := accountRepo.accounts[checkingID]
	if checking.CurrentBalance != 1000 {
		t.Fatalf("Checking account balance should remain 1000, got %.2f", checking.CurrentBalance)
	}
}

func TestCreateTransaction_CreditCardIncome_ShouldNotUpdateBalance(t *testing.T) {
	// Given: A credit card
	// When: Creating a CONFIRMED income (refund) on the credit card
	// Then: Credit card balance should remain unchanged

	profileID := "profile-1"
	creditCardID := "cc-1"

	now := time.Now()
	closingDay := 9
	dueDay := 14

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

	limit := 2000.0
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		creditCardID: {
			ID:             creditCardID,
			ProfileID:      profileID,
			Name:           "Cartão",
			Type:           bankaccount.AccountTypeCreditCard,
			InitialBalance: 0,
			CurrentBalance: 0,
			Currency:       "BRL",
			IsActive:       true,
			CreditLimit:    &limit,
			ClosingDay:     &closingDay,
			DueDay:         &dueDay,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: creditCardID,
		Type:          "INCOME",
		Status:        &confirmedStatus,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Estorno",
		OccurredOn:    yesterday,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the credit card balance was NOT updated
	creditCard := accountRepo.accounts[creditCardID]
	if creditCard.CurrentBalance != 0 {
		t.Fatalf("Credit card balance should remain 0, got %.2f", creditCard.CurrentBalance)
	}
}

// ── Delete Transaction tests ────────────────────────────────────────

func TestDeleteTransaction_ConfirmedExpense_ShouldRestoreBalance(t *testing.T) {
	// Given: An account with R$1000, a confirmed expense of R$300 was created (balance = R$700)
	// When: Deleting the transaction
	// Then: Balance should be restored to R$1000

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

	// Create a confirmed expense
	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	txn, err := createUC.Execute(CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        300,
		Currency:      "BRL",
		Description:   "Cafe",
		OccurredOn:    yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %v", err)
	}

	// Verify balance decreased to 700
	account := accountRepo.accounts[accountID]
	if account.CurrentBalance != 700 {
		t.Fatalf("expected balance 700 after expense, got %.2f", account.CurrentBalance)
	}

	// Reset tracking
	accountRepo.updateCalled = false

	// Delete the transaction
	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err = deleteUC.Execute(txn.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting transaction: %v", err)
	}

	// Verify balance was restored to 1000
	account = accountRepo.accounts[accountID]
	if account.CurrentBalance != 1000 {
		t.Fatalf("expected balance 1000 after delete, got %.2f", account.CurrentBalance)
	}

	if !accountRepo.updateCalled {
		t.Fatal("expected accountRepo.Update to be called on delete")
	}
}

func TestDeleteTransaction_ConfirmedIncome_ShouldReverseBalance(t *testing.T) {
	// Given: An account with R$500, a confirmed income of R$1000 was created (balance = R$1500)
	// When: Deleting the transaction
	// Then: Balance should be restored to R$500

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
			Name:           "Nubank",
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

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	txn, err := createUC.Execute(CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "INCOME",
		Status:        &confirmedStatus,
		Amount:        1000,
		Currency:      "BRL",
		Description:   "Salario",
		OccurredOn:    yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %v", err)
	}

	if accountRepo.accounts[accountID].CurrentBalance != 1500 {
		t.Fatalf("expected balance 1500 after income, got %.2f", accountRepo.accounts[accountID].CurrentBalance)
	}

	accountRepo.updateCalled = false

	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err = deleteUC.Execute(txn.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting transaction: %v", err)
	}

	account := accountRepo.accounts[accountID]
	if account.CurrentBalance != 500 {
		t.Fatalf("expected balance 500 after delete, got %.2f", account.CurrentBalance)
	}
}

func TestDeleteTransaction_PlannedTransaction_ShouldNotChangeBalance(t *testing.T) {
	// Given: An account with R$1000, a PLANNED expense of R$300 (balance stays R$1000)
	// When: Deleting the transaction
	// Then: Balance should remain R$1000, accountRepo.Update should NOT be called

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

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// PLANNED is the default status (no Status field passed)
	txn, err := createUC.Execute(CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        300,
		Currency:      "BRL",
		Description:   "Compra futura",
		OccurredOn:    yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %v", err)
	}

	// Balance should not change for planned
	if accountRepo.accounts[accountID].CurrentBalance != 1000 {
		t.Fatalf("expected balance 1000 for planned, got %.2f", accountRepo.accounts[accountID].CurrentBalance)
	}

	accountRepo.updateCalled = false

	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err = deleteUC.Execute(txn.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting transaction: %v", err)
	}

	// Balance should remain 1000
	if accountRepo.accounts[accountID].CurrentBalance != 1000 {
		t.Fatalf("expected balance 1000 after delete, got %.2f", accountRepo.accounts[accountID].CurrentBalance)
	}

	// accountRepo.Update should NOT be called for planned transactions
	if accountRepo.updateCalled {
		t.Fatal("accountRepo.Update should not be called when deleting a PLANNED transaction")
	}
}

func TestDeleteTransaction_CreditCard_ShouldNotChangeBalance(t *testing.T) {
	// Given: A credit card account, a confirmed expense was created
	// When: Deleting the transaction
	// Then: Balance should NOT change (credit cards don't update balance)

	profileID := "profile-1"
	creditCardID := "cc-1"

	now := time.Now()
	closingDay := 5
	dueDay := 15
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
		creditCardID: {
			ID:             creditCardID,
			ProfileID:      profileID,
			Name:           "Cartão Nubank",
			Type:           bankaccount.AccountTypeCreditCard,
			InitialBalance: 0,
			CurrentBalance: 0,
			Currency:       "BRL",
			IsActive:       true,
			ClosingDay:     &closingDay,
			DueDay:         &dueDay,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	txn, err := createUC.Execute(CreateTransactionInput{
		ProfileID:     profileID,
		BankAccountID: creditCardID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        200,
		Currency:      "BRL",
		Description:   "Compra cartao",
		OccurredOn:    yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %v", err)
	}

	accountRepo.updateCalled = false

	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err = deleteUC.Execute(txn.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting transaction: %v", err)
	}

	// Credit card balance should remain 0
	cc := accountRepo.accounts[creditCardID]
	if cc.CurrentBalance != 0 {
		t.Fatalf("expected credit card balance 0, got %.2f", cc.CurrentBalance)
	}

	if accountRepo.updateCalled {
		t.Fatal("accountRepo.Update should not be called for credit card transactions")
	}
}

func TestDeleteTransaction_NotFound_ShouldReturnError(t *testing.T) {
	txRepo := &fakeTransactionRepo{}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{}}

	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err := deleteUC.Execute("non-existent-id")

	if err == nil {
		t.Fatal("expected error when deleting non-existent transaction")
	}
}

// =============================================================================
// TRANSFER Destination Account Tests (TDD)
// =============================================================================

func TestCreateTransaction_ConfirmedTransfer_ShouldCreditDestination(t *testing.T) {
	// Given: Source account with R$5000, destination account with R$1000
	// When: Creating a CONFIRMED TRANSFER of R$3000
	// Then: Source balance should decrease to R$2000, destination should increase to R$4000

	profileID := "profile-1"
	sourceID := "source-1"
	destID := "dest-1"

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
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 5000,
			CurrentBalance: 5000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		destID: {
			ID:             destID,
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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		Type:                 "TRANSFER",
		Status:               &confirmedStatus,
		Amount:               3000,
		Currency:             "BRL",
		Description:          "Transferência para Nubank",
		OccurredOn:           yesterday,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify source was debited
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 2000 {
		t.Fatalf("expected source balance 2000, got %.2f", source.CurrentBalance)
	}

	// Verify destination was credited
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 4000 {
		t.Fatalf("expected destination balance 4000, got %.2f", dest.CurrentBalance)
	}

	// Verify both accounts were updated
	if accountRepo.updateCount != 2 {
		t.Fatalf("expected 2 account updates (source + destination), got %d", accountRepo.updateCount)
	}
}

func TestCreateTransaction_PlannedTransfer_ShouldNotUpdateAnyBalance(t *testing.T) {
	// Given: Source account with R$5000, destination account with R$1000
	// When: Creating a PLANNED TRANSFER of R$3000
	// Then: Both balances should remain unchanged

	profileID := "profile-1"
	sourceID := "source-1"
	destID := "dest-1"

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
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 5000,
			CurrentBalance: 5000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		destID: {
			ID:             destID,
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

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		Type:                 "TRANSFER",
		Amount:               3000,
		Currency:             "BRL",
		Description:          "Transferência planejada",
		OccurredOn:           yesterday,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both balances should remain unchanged
	if accountRepo.accounts[sourceID].CurrentBalance != 5000 {
		t.Fatalf("expected source balance 5000, got %.2f", accountRepo.accounts[sourceID].CurrentBalance)
	}
	if accountRepo.accounts[destID].CurrentBalance != 1000 {
		t.Fatalf("expected destination balance 1000, got %.2f", accountRepo.accounts[destID].CurrentBalance)
	}

	if accountRepo.updateCalled {
		t.Fatal("expected no account updates for PLANNED transfer")
	}
}

func TestDeleteTransaction_ConfirmedTransfer_ShouldReverseBothAccounts(t *testing.T) {
	// Given: Source R$5000, Dest R$1000, CONFIRMED TRANSFER of R$3000 created
	//        (Source now R$2000, Dest now R$4000)
	// When: Deleting the transfer
	// Then: Source should be restored to R$5000, Dest should be restored to R$1000

	profileID := "profile-1"
	sourceID := "source-1"
	destID := "dest-1"

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
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 5000,
			CurrentBalance: 5000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		destID: {
			ID:             destID,
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

	// Create a confirmed transfer
	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)
	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	txn, err := createUC.Execute(CreateTransactionInput{
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		Type:                 "TRANSFER",
		Status:               &confirmedStatus,
		Amount:               3000,
		Currency:             "BRL",
		Description:          "Transferência",
		OccurredOn:           yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error creating transfer: %v", err)
	}

	// Reset tracking
	accountRepo.updateCalled = false
	accountRepo.updateCount = 0

	// Delete the transfer
	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err = deleteUC.Execute(txn.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting transfer: %v", err)
	}

	// Source should be restored
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 5000 {
		t.Fatalf("expected source balance 5000 after delete, got %.2f", source.CurrentBalance)
	}

	// Destination should be restored
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 1000 {
		t.Fatalf("expected destination balance 1000 after delete, got %.2f", dest.CurrentBalance)
	}

	// Both accounts should have been updated
	if accountRepo.updateCount != 2 {
		t.Fatalf("expected 2 account updates on delete, got %d", accountRepo.updateCount)
	}
}

func TestUpdateTransactionStatus_TransferConfirm_ShouldCreditDestination(t *testing.T) {
	// Given: A PLANNED transfer between two accounts
	// When: Confirming the transfer
	// Then: Source should be debited, destination should be credited

	profileID := "profile-1"
	sourceID := "source-1"
	destID := "dest-1"

	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 5000,
			CurrentBalance: 5000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		destID: {
			ID:             destID,
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

	existingTx := &transaction.Transaction{
		ID:                   "tx-transfer-1",
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		Type:                 transaction.TypeTransfer,
		Status:               transaction.StatusPlanned,
		Amount:               3000,
		Currency:             "BRL",
		Description:          "Transferência",
		OccurredOn:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}

	useCase := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err := useCase.Execute("tx-transfer-1", UpdateTransactionStatusInput{
		Status:     "CONFIRMED",
		OccurredOn: &yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source should be debited
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 2000 {
		t.Fatalf("expected source balance 2000, got %.2f", source.CurrentBalance)
	}

	// Destination should be credited
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 4000 {
		t.Fatalf("expected destination balance 4000, got %.2f", dest.CurrentBalance)
	}
}

func TestUpdateTransactionStatus_TransferCancel_ShouldReverseDestination(t *testing.T) {
	// Given: A CONFIRMED transfer (source debited, dest credited)
	// When: Cancelling the transfer
	// Then: Source should be restored, destination should be reversed

	profileID := "profile-1"
	sourceID := "source-1"
	destID := "dest-1"

	now := time.Now()

	// Balances after the confirmed transfer: source 2000, dest 4000
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 5000,
			CurrentBalance: 2000, // After transfer of 3000
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		destID: {
			ID:             destID,
			ProfileID:      profileID,
			Name:           "Nubank",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000,
			CurrentBalance: 4000, // After receiving transfer of 3000
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:                   "tx-transfer-2",
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		Type:                 transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed,
		Amount:               3000,
		Currency:             "BRL",
		Description:          "Transferência",
		OccurredOn:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}

	useCase := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)

	_, err := useCase.Execute("tx-transfer-2", UpdateTransactionStatusInput{
		Status: "CANCELLED",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source should be restored
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 5000 {
		t.Fatalf("expected source balance 5000 after cancel, got %.2f", source.CurrentBalance)
	}

	// Destination should be reversed
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 1000 {
		t.Fatalf("expected destination balance 1000 after cancel, got %.2f", dest.CurrentBalance)
	}
}

// =============================================================================
// Cross-Profile Transfer Tests (TDD)
// =============================================================================

func TestCreateTransaction_CrossProfileTransfer_ShouldCreatePairedTransactions(t *testing.T) {
	// Given: Source account in profile WB, destination account in profile Bruno
	// When: Creating a CONFIRMED TRANSFER of R$1500 (salary)
	// Then: Should create EXPENSE in WB + INCOME in Bruno, linked by LinkedTransactionID

	profileWB := "profile-wb"
	profileBruno := "profile-bruno"
	sourceID := "nubank-pj"
	destID := "nubank-pessoal"
	sourceCatID := "cat-salary-expense"
	destCatID := "cat-salary-income"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileWB: {
			ID:         profileWB,
			CalendarID: "cal-1",
			Name:       "WB Digital",
			Type:       profile.ProfileTypeBusiness,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		profileBruno: {
			ID:         profileBruno,
			CalendarID: "cal-2",
			Name:       "Bruno Pessoal",
			Type:       profile.ProfileTypePersonal,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileWB,
			Name:           "Nubank PJ",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 10000,
			CurrentBalance: 10000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		destID: {
			ID:             destID,
			ProfileID:      profileBruno,
			Name:           "Nubank Pessoal",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 500,
			CurrentBalance: 500,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		sourceCatID: {
			ID:        sourceCatID,
			ProfileID: profileWB,
			Name:      "Salário",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		destCatID: {
			ID:        destCatID,
			ProfileID: profileBruno,
			Name:      "Salário",
			Type:      category.TypeIncome,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}

	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:             profileWB,
		BankAccountID:         sourceID,
		DestinationAccountID:  &destID,
		CategoryID:            &sourceCatID,
		DestinationCategoryID: &destCatID,
		Type:                  "TRANSFER",
		Status:                &confirmedStatus,
		Amount:                1500,
		Currency:              "BRL",
		Description:           "Salário Bruno",
		OccurredOn:            yesterday,
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created 2 transactions
	if len(txRepo.created) != 2 {
		t.Fatalf("expected 2 transactions (paired), got %d", len(txRepo.created))
	}

	// First tx (returned) should be EXPENSE in source profile
	if tx.Type != transaction.TypeExpense {
		t.Fatalf("expected source tx type EXPENSE, got %s", tx.Type)
	}
	if tx.ProfileID != profileWB {
		t.Fatalf("expected source tx profile %s, got %s", profileWB, tx.ProfileID)
	}
	if tx.BankAccountID != sourceID {
		t.Fatalf("expected source tx account %s, got %s", sourceID, tx.BankAccountID)
	}
	if tx.CategoryID == nil || *tx.CategoryID != sourceCatID {
		t.Fatal("expected source tx to have source category")
	}

	// Second tx should be INCOME in destination profile
	destTx := txRepo.created[1]
	if destTx.Type != transaction.TypeIncome {
		t.Fatalf("expected dest tx type INCOME, got %s", destTx.Type)
	}
	if destTx.ProfileID != profileBruno {
		t.Fatalf("expected dest tx profile %s, got %s", profileBruno, destTx.ProfileID)
	}
	if destTx.BankAccountID != destID {
		t.Fatalf("expected dest tx account %s, got %s", destID, destTx.BankAccountID)
	}
	if destTx.CategoryID == nil || *destTx.CategoryID != destCatID {
		t.Fatal("expected dest tx to have destination category")
	}

	// Both should be linked to each other
	if tx.LinkedTransactionID == nil {
		t.Fatal("expected source tx to have LinkedTransactionID")
	}
	if destTx.LinkedTransactionID == nil {
		t.Fatal("expected dest tx to have LinkedTransactionID")
	}
	if *tx.LinkedTransactionID != destTx.ID {
		t.Fatalf("expected source linked to dest ID %s, got %s", destTx.ID, *tx.LinkedTransactionID)
	}
	if *destTx.LinkedTransactionID != tx.ID {
		t.Fatalf("expected dest linked to source ID %s, got %s", tx.ID, *destTx.LinkedTransactionID)
	}

	// Both balances should be updated
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 8500 {
		t.Fatalf("expected source balance 8500, got %.2f", source.CurrentBalance)
	}
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 2000 {
		t.Fatalf("expected dest balance 2000, got %.2f", dest.CurrentBalance)
	}
}

func TestCreateTransaction_CrossProfileTransfer_RequiresDestinationCategory(t *testing.T) {
	// Given: Cross-profile transfer WITHOUT destinationCategoryId
	// Then: Should return ErrDestinationCategoryRequired

	profileWB := "profile-wb"
	profileBruno := "profile-bruno"
	sourceID := "nubank-pj"
	destID := "nubank-pessoal"
	sourceCatID := "cat-salary-expense"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileWB:    {ID: profileWB, CalendarID: "cal-1", Name: "WB", Type: profile.ProfileTypeBusiness, IsActive: true, CreatedAt: now, UpdatedAt: now},
		profileBruno: {ID: profileBruno, CalendarID: "cal-2", Name: "Bruno", Type: profile.ProfileTypePersonal, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {ID: sourceID, ProfileID: profileWB, Name: "Nubank PJ", Type: bankaccount.AccountTypeChecking, CurrentBalance: 10000, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
		destID:   {ID: destID, ProfileID: profileBruno, Name: "Nubank", Type: bankaccount.AccountTypeChecking, CurrentBalance: 500, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		sourceCatID: {ID: sourceCatID, ProfileID: profileWB, Name: "Salário", Type: category.TypeExpense, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:            profileWB,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		CategoryID:           &sourceCatID,
		// DestinationCategoryID intentionally omitted
		Type:        "TRANSFER",
		Status:      &confirmedStatus,
		Amount:      1500,
		Currency:    "BRL",
		Description: "Salário",
		OccurredOn:  yesterday,
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Fatal("expected error for missing destination category")
	}
	if !errors.Is(err, ErrDestinationCategoryRequired) {
		t.Fatalf("expected ErrDestinationCategoryRequired, got %v", err)
	}
}

func TestCreateTransaction_SameProfileTransfer_StillWorksAsBefore(t *testing.T) {
	// Given: Source and destination in the SAME profile
	// When: Creating a TRANSFER
	// Then: Should create a single TRANSFER transaction (not paired)

	profileID := "profile-1"
	sourceID := "mercado-pago"
	destID := "caixinha"

	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {ID: profileID, CalendarID: "cal-1", Name: "Bruno", Type: profile.ProfileTypePersonal, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {ID: sourceID, ProfileID: profileID, Name: "Mercado Pago", Type: bankaccount.AccountTypeChecking, CurrentBalance: 5000, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
		destID:   {ID: destID, ProfileID: profileID, Name: "Caixinha", Type: bankaccount.AccountTypeChecking, CurrentBalance: 1000, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := CreateTransactionInput{
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &destID,
		Type:                 "TRANSFER",
		Status:               &confirmedStatus,
		Amount:               3000,
		Currency:             "BRL",
		Description:          "Transferência para caixinha",
		OccurredOn:           yesterday,
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create only 1 transaction (not paired)
	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction for same-profile transfer, got %d", len(txRepo.created))
	}

	// Should be TRANSFER type, not EXPENSE
	if tx.Type != transaction.TypeTransfer {
		t.Fatalf("expected TRANSFER type for same-profile, got %s", tx.Type)
	}

	// Should NOT have LinkedTransactionID
	if tx.LinkedTransactionID != nil {
		t.Fatal("same-profile transfer should not have LinkedTransactionID")
	}
}

func TestDeleteTransaction_CrossProfileLinked_ShouldDeleteBoth(t *testing.T) {
	// Given: Two linked transactions (EXPENSE in WB, INCOME in Bruno)
	// When: Deleting the source (EXPENSE)
	// Then: Both should be deleted, both balances reversed

	profileWB := "profile-wb"
	profileBruno := "profile-bruno"
	sourceID := "nubank-pj"
	destID := "nubank-pessoal"

	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {ID: sourceID, ProfileID: profileWB, Name: "Nubank PJ", Type: bankaccount.AccountTypeChecking, CurrentBalance: 8500, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
		destID:   {ID: destID, ProfileID: profileBruno, Name: "Nubank", Type: bankaccount.AccountTypeChecking, CurrentBalance: 2000, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	destTxID := "dest-tx-1"
	sourceTxID := "source-tx-1"

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{
			ID:                  sourceTxID,
			ProfileID:           profileWB,
			BankAccountID:       sourceID,
			Type:                transaction.TypeExpense,
			Status:              transaction.StatusConfirmed,
			Amount:              1500,
			Currency:            "BRL",
			Description:         "Salário Bruno",
			LinkedTransactionID: &destTxID,
			OccurredOn:          now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
		{
			ID:                  destTxID,
			ProfileID:           profileBruno,
			BankAccountID:       destID,
			Type:                transaction.TypeIncome,
			Status:              transaction.StatusConfirmed,
			Amount:              1500,
			Currency:            "BRL",
			Description:         "Salário Bruno",
			LinkedTransactionID: &sourceTxID,
			OccurredOn:          now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}}

	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	err := deleteUC.Execute(sourceTxID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both transactions should be deleted
	if len(txRepo.created) != 0 {
		t.Fatalf("expected 0 transactions after delete, got %d", len(txRepo.created))
	}

	// Source balance should be restored (8500 + 1500 = 10000)
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 10000 {
		t.Fatalf("expected source balance 10000, got %.2f", source.CurrentBalance)
	}

	// Dest balance should be reversed (2000 - 1500 = 500)
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 500 {
		t.Fatalf("expected dest balance 500, got %.2f", dest.CurrentBalance)
	}
}

func TestUpdateTransactionStatus_CrossProfileConfirm_ShouldUpdateBoth(t *testing.T) {
	// Given: Two PLANNED linked transactions
	// When: Confirming the source
	// Then: Both should be confirmed, both balances updated

	profileWB := "profile-wb"
	profileBruno := "profile-bruno"
	sourceID := "nubank-pj"
	destID := "nubank-pessoal"

	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {ID: sourceID, ProfileID: profileWB, Name: "Nubank PJ", Type: bankaccount.AccountTypeChecking, CurrentBalance: 10000, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
		destID:   {ID: destID, ProfileID: profileBruno, Name: "Nubank", Type: bankaccount.AccountTypeChecking, CurrentBalance: 500, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	destTxID := "dest-tx-1"
	sourceTxID := "source-tx-1"

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{
			ID:                  sourceTxID,
			ProfileID:           profileWB,
			BankAccountID:       sourceID,
			Type:                transaction.TypeExpense,
			Status:              transaction.StatusPlanned,
			Amount:              1500,
			Currency:            "BRL",
			Description:         "Salário Bruno",
			LinkedTransactionID: &destTxID,
			OccurredOn:          now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
		{
			ID:                  destTxID,
			ProfileID:           profileBruno,
			BankAccountID:       destID,
			Type:                transaction.TypeIncome,
			Status:              transaction.StatusPlanned,
			Amount:              1500,
			Currency:            "BRL",
			Description:         "Salário Bruno",
			LinkedTransactionID: &sourceTxID,
			OccurredOn:          now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}}

	useCase := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err := useCase.Execute(sourceTxID, UpdateTransactionStatusInput{
		Status:     "CONFIRMED",
		OccurredOn: &yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source balance should be debited (10000 - 1500 = 8500)
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 8500 {
		t.Fatalf("expected source balance 8500, got %.2f", source.CurrentBalance)
	}

	// Dest balance should be credited (500 + 1500 = 2000)
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 2000 {
		t.Fatalf("expected dest balance 2000, got %.2f", dest.CurrentBalance)
	}

	// Linked transaction should also be confirmed
	linkedTx, _ := txRepo.GetByID(destTxID)
	if linkedTx.Status != transaction.StatusConfirmed {
		t.Fatalf("expected linked tx status CONFIRMED, got %s", linkedTx.Status)
	}
}

func TestCreateTransaction_CrossProfilePlanned_ShouldNotUpdateBalances(t *testing.T) {
	// Given: A cross-profile transfer with PLANNED status
	// When: Created
	// Then: Neither source nor destination balances should change

	profileWB := "profile-wb"
	profileBruno := "profile-bruno"
	sourceID := "nubank-pj"
	destID := "mercado-pago"

	now := time.Now()

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileWB: {ID: profileWB, Name: "WB Digital Solutions"},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {ID: sourceID, ProfileID: profileWB, Name: "Nubank PJ", Type: bankaccount.AccountTypeChecking, CurrentBalance: 5000, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
		destID:   {ID: destID, ProfileID: profileBruno, Name: "Mercado Pago", Type: bankaccount.AccountTypeChecking, CurrentBalance: 300, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}
	catRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		"cat-expense": {ID: "cat-expense", ProfileID: profileWB, Name: "Produtividade", Type: category.TypeExpense},
		"cat-income":  {ID: "cat-income", ProfileID: profileBruno, Name: "Reembolso", Type: category.TypeIncome},
	}}
	txRepo := &fakeTransactionRepo{}
	invRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, catRepo, txRepo, invRepo, nil, nil)

	plannedStatus := "PLANNED"
	destAccountID := destID
	catExpense := "cat-expense"
	catIncome := "cat-income"
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	_, err := useCase.Execute(CreateTransactionInput{
		ProfileID:             profileWB,
		BankAccountID:         sourceID,
		DestinationAccountID:  &destAccountID,
		CategoryID:            &catExpense,
		DestinationCategoryID: &catIncome,
		Type:                  "TRANSFER",
		Status:                &plannedStatus,
		Amount:                27.05,
		Currency:              "BRL",
		Description:           "Obsidian - reembolso",
		OccurredOn:            yesterday,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source balance should NOT change (still 5000)
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 5000 {
		t.Fatalf("expected source balance 5000, got %.2f", source.CurrentBalance)
	}

	// Destination balance should NOT change (still 300)
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 300 {
		t.Fatalf("expected dest balance 300, got %.2f", dest.CurrentBalance)
	}

	// Should have created 2 transactions (both PLANNED)
	if len(txRepo.created) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txRepo.created))
	}
	if txRepo.created[0].Status != transaction.StatusPlanned {
		t.Fatalf("expected source PLANNED, got %s", txRepo.created[0].Status)
	}
	if txRepo.created[1].Status != transaction.StatusPlanned {
		t.Fatalf("expected dest PLANNED, got %s", txRepo.created[1].Status)
	}
}

func TestUpdateTransactionStatus_CrossProfileCancel_ShouldReverseBoth(t *testing.T) {
	// Given: Two CONFIRMED linked transactions
	// When: Cancelling the source
	// Then: Both should be cancelled, both balances reversed

	profileWB := "profile-wb"
	profileBruno := "profile-bruno"
	sourceID := "nubank-pj"
	destID := "mercado-pago"

	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {ID: sourceID, ProfileID: profileWB, Name: "Nubank PJ", Type: bankaccount.AccountTypeChecking, CurrentBalance: 8500, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
		destID:   {ID: destID, ProfileID: profileBruno, Name: "Mercado Pago", Type: bankaccount.AccountTypeChecking, CurrentBalance: 550, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	destTxID := "dest-tx-cancel"
	sourceTxID := "source-tx-cancel"

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{
			ID:                  sourceTxID,
			ProfileID:           profileWB,
			BankAccountID:       sourceID,
			Type:                transaction.TypeExpense,
			Status:              transaction.StatusConfirmed,
			Amount:              50,
			Currency:            "BRL",
			Description:         "Reembolso teste",
			LinkedTransactionID: &destTxID,
			OccurredOn:          now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
		{
			ID:                  destTxID,
			ProfileID:           profileBruno,
			BankAccountID:       destID,
			Type:                transaction.TypeIncome,
			Status:              transaction.StatusConfirmed,
			Amount:              50,
			Currency:            "BRL",
			Description:         "Reembolso teste",
			LinkedTransactionID: &sourceTxID,
			OccurredOn:          now,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}}

	useCase := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)

	_, err := useCase.Execute(sourceTxID, UpdateTransactionStatusInput{
		Status: "CANCELLED",
		Reason: strPtr("Cancelado"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source balance should be restored (8500 + 50 = 8550) — EXPENSE reversal
	source := accountRepo.accounts[sourceID]
	if source.CurrentBalance != 8550 {
		t.Fatalf("expected source balance 8550, got %.2f", source.CurrentBalance)
	}

	// Dest balance should be reversed (550 - 50 = 500) — INCOME reversal
	dest := accountRepo.accounts[destID]
	if dest.CurrentBalance != 500 {
		t.Fatalf("expected dest balance 500, got %.2f", dest.CurrentBalance)
	}

	// Both should be cancelled
	sourceTx, _ := txRepo.GetByID(sourceTxID)
	if sourceTx.Status != transaction.StatusCancelled {
		t.Fatalf("expected source CANCELLED, got %s", sourceTx.Status)
	}
	linkedTx, _ := txRepo.GetByID(destTxID)
	if linkedTx.Status != transaction.StatusCancelled {
		t.Fatalf("expected linked CANCELLED, got %s", linkedTx.Status)
	}
}

// ===== UpdateTransaction Balance Adjustment Tests (TDD) =====

func TestUpdateTransaction_AmountChange_AdjustsBalance(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta Corrente",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 800, // Started at 1000, had 200 expense confirmed
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
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200, // Old amount
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	confirmedStatus := "CONFIRMED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        300, // New amount: 100 more
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Balance should go from 800 to 700 (reverse +200, apply -300 = net -100)
	acc := accountRepo.accounts[accountID]
	if acc.CurrentBalance != 700 {
		t.Fatalf("expected balance 700, got %.2f", acc.CurrentBalance)
	}
}

func TestUpdateTransaction_TypeChange_AdjustsBalance(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta Corrente",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 900, // Started at 1000, had 100 expense
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Reembolso",
			Type:      category.TypeIncome,
			IsActive:  true,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	confirmedStatus := "CONFIRMED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "INCOME", // Changed from EXPENSE to INCOME
		Status:        &confirmedStatus,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Reembolso",
		OccurredOn:    now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Balance: 900 + 100 (reverse expense) + 100 (apply income) = 1100
	acc := accountRepo.accounts[accountID]
	if acc.CurrentBalance != 1100 {
		t.Fatalf("expected balance 1100, got %.2f", acc.CurrentBalance)
	}
}

func TestUpdateTransaction_AccountSwitch_MovesBalance(t *testing.T) {
	profileID := "profile-1"
	accountA := "account-a"
	accountB := "account-b"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountA: {
			ID:             accountA,
			ProfileID:      profileID,
			Name:           "Nubank",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 800, // Had 200 expense
			Currency:       "BRL",
			IsActive:       true,
		},
		accountB: {
			ID:             accountB,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 500,
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Software",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountA,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200,
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	confirmedStatus := "CONFIRMED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountB, // Moved to account B
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        200,
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Account A: 800 + 200 (reverse) = 1000
	accA := accountRepo.accounts[accountA]
	if accA.CurrentBalance != 1000 {
		t.Fatalf("expected account A balance 1000, got %.2f", accA.CurrentBalance)
	}

	// Account B: 500 - 200 (apply) = 300
	accB := accountRepo.accounts[accountB]
	if accB.CurrentBalance != 300 {
		t.Fatalf("expected account B balance 300, got %.2f", accB.CurrentBalance)
	}
}

func TestUpdateTransaction_CreditCard_NoBalanceChange(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-cc"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()
	limit := 5000.0

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Cartao",
			Type:           bankaccount.AccountTypeCreditCard,
			CurrentBalance: 0,
			Currency:       "BRL",
			IsActive:       true,
			CreditLimit:    &limit,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Software",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	confirmedStatus := "CONFIRMED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        300, // Changed amount
		Currency:      "BRL",
		Description:   "Compra maior",
		OccurredOn:    now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Credit card balance should NOT change
	acc := accountRepo.accounts[accountID]
	if acc.CurrentBalance != 0 {
		t.Fatalf("expected credit card balance 0, got %.2f", acc.CurrentBalance)
	}
}

func TestUpdateTransaction_PlannedToConfirmed_AppliesBalance(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta Corrente",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 1000,
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Software",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned, // Was PLANNED
		Amount:        200,
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	confirmedStatus := "CONFIRMED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus, // Now CONFIRMED
		Amount:        200,
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Balance: 1000 - 200 = 800 (PLANNED had no impact, CONFIRMED applies)
	acc := accountRepo.accounts[accountID]
	if acc.CurrentBalance != 800 {
		t.Fatalf("expected balance 800, got %.2f", acc.CurrentBalance)
	}
}

func TestUpdateTransaction_ConfirmedToPlanned_ReversesBalance(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Conta Corrente",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 800, // 1000 - 200 expense
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Software",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200,
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	plannedStatus := "PLANNED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID,
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Status:        &plannedStatus, // Changed to PLANNED
		Amount:        200,
		Currency:      "BRL",
		Description:   "SaaS",
		OccurredOn:    now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Balance: 800 + 200 (reverse confirmed expense) = 1000
	acc := accountRepo.accounts[accountID]
	if acc.CurrentBalance != 1000 {
		t.Fatalf("expected balance 1000, got %.2f", acc.CurrentBalance)
	}
}

func TestUpdateTransaction_TransferDestinationChange_AdjustsBothDests(t *testing.T) {
	profileID := "profile-1"
	sourceID := "account-src"
	oldDestID := "account-old-dest"
	newDestID := "account-new-dest"
	categoryID := "cat-1"
	txID := "tx-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceID: {
			ID:             sourceID,
			ProfileID:      profileID,
			Name:           "Nubank",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 800, // 1000 - 200 transfer
			Currency:       "BRL",
			IsActive:       true,
		},
		oldDestID: {
			ID:             oldDestID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 700, // 500 + 200 transfer
			Currency:       "BRL",
			IsActive:       true,
		},
		newDestID: {
			ID:             newDestID,
			ProfileID:      profileID,
			Name:           "Clear",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 300,
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Transferencia",
			Type:      category.TypeTransfer,
			IsActive:  true,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:                   txID,
		ProfileID:            profileID,
		BankAccountID:        sourceID,
		DestinationAccountID: &oldDestID,
		CategoryID:           &categoryID,
		Type:                 transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed,
		Amount:               200,
		Currency:             "BRL",
		Description:          "Transferencia",
		OccurredOn:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil)

	confirmedStatus := "CONFIRMED"
	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID:        sourceID,
		DestinationAccountID: &newDestID, // Changed destination
		CategoryID:           &categoryID,
		Type:                 "TRANSFER",
		Status:               &confirmedStatus,
		Amount:               200,
		Currency:             "BRL",
		Description:          "Transferencia",
		OccurredOn:           now.Format("2006-01-02"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Source stays the same: 800 (reverse +200, apply -200 = net 0)
	src := accountRepo.accounts[sourceID]
	if src.CurrentBalance != 800 {
		t.Fatalf("expected source balance 800, got %.2f", src.CurrentBalance)
	}

	// Old dest: 700 - 200 (reverse credit) = 500
	oldDest := accountRepo.accounts[oldDestID]
	if oldDest.CurrentBalance != 500 {
		t.Fatalf("expected old dest balance 500, got %.2f", oldDest.CurrentBalance)
	}

	// New dest: 300 + 200 (apply credit) = 500
	newDest := accountRepo.accounts[newDestID]
	if newDest.CurrentBalance != 500 {
		t.Fatalf("expected new dest balance 500, got %.2f", newDest.CurrentBalance)
	}
}

// ===== Invoice Reassignment on Update Tests (TDD) =====

func TestUpdateTransaction_CreditCardAccountChange_ReassignsInvoice(t *testing.T) {
	profileID := "profile-1"
	ccAccountA := "cc-account-a"
	ccAccountB := "cc-account-b"
	categoryID := "cat-1"
	txID := "tx-1"
	invoiceA := "invoice-a"
	invoiceB := "invoice-b"
	now := time.Now()
	closingDay := 24
	dueDay := 1

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		ccAccountA: {
			ID:         ccAccountA,
			ProfileID:  profileID,
			Name:       "Cartão Nubank",
			Type:       bankaccount.AccountTypeCreditCard,
			Currency:   "BRL",
			IsActive:   true,
			ClosingDay: &closingDay,
			DueDay:     &dueDay,
		},
		ccAccountB: {
			ID:         ccAccountB,
			ProfileID:  profileID,
			Name:       "Cartão Mercado Pago",
			Type:       bankaccount.AccountTypeCreditCard,
			Currency:   "BRL",
			IsActive:   true,
			ClosingDay: &closingDay,
			DueDay:     &dueDay,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Alimentação",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	invoiceRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceA: {
			ID:            invoiceA,
			BankAccountID: ccAccountA,
			ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			OpeningDate:   time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC),
			ClosingDate:   time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			DueDate:       time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			Status:        invoice.StatusOpen,
		},
		invoiceB: {
			ID:            invoiceB,
			BankAccountID: ccAccountB,
			ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			OpeningDate:   time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC),
			ClosingDate:   time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			DueDate:       time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			Status:        invoice.StatusOpen,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: ccAccountA,
		CategoryID:    &categoryID,
		InvoiceID:     &invoiceA,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Totalpass",
		OccurredOn:    time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, invoiceRepo, nil)

	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: ccAccountB, // Changed to different credit card
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "Totalpass",
		OccurredOn:    "2026-03-15",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Transaction should now have invoiceB (from ccAccountB), not invoiceA
	updatedTx, _ := txRepo.GetByID(txID)
	if updatedTx.InvoiceID == nil {
		t.Fatal("expected invoice to be reassigned, got nil")
	}
	if *updatedTx.InvoiceID != invoiceB {
		t.Fatalf("expected invoice %s, got %s", invoiceB, *updatedTx.InvoiceID)
	}
}

func TestUpdateTransaction_CreditCardToRegularAccount_ClearsInvoice(t *testing.T) {
	profileID := "profile-1"
	ccAccountID := "cc-account"
	checkingID := "checking-account"
	categoryID := "cat-1"
	txID := "tx-1"
	invoiceID := "invoice-1"
	now := time.Now()
	closingDay := 24
	dueDay := 1

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		ccAccountID: {
			ID:         ccAccountID,
			ProfileID:  profileID,
			Name:       "Cartão Nubank",
			Type:       bankaccount.AccountTypeCreditCard,
			Currency:   "BRL",
			IsActive:   true,
			ClosingDay: &closingDay,
			DueDay:     &dueDay,
		},
		checkingID: {
			ID:             checkingID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 500,
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Alimentação",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	invoiceRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {
			ID:            invoiceID,
			BankAccountID: ccAccountID,
			ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			OpeningDate:   time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC),
			ClosingDate:   time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			DueDate:       time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			Status:        invoice.StatusOpen,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: ccAccountID,
		CategoryID:    &categoryID,
		InvoiceID:     &invoiceID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, invoiceRepo, nil)

	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: checkingID, // Changed to regular account
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2026-03-15",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updatedTx, _ := txRepo.GetByID(txID)
	if updatedTx.InvoiceID != nil {
		t.Fatalf("expected invoice to be cleared, got %s", *updatedTx.InvoiceID)
	}
}

func TestUpdateTransaction_RegularToCreditCard_AssignsInvoice(t *testing.T) {
	profileID := "profile-1"
	checkingID := "checking-account"
	ccAccountID := "cc-account"
	categoryID := "cat-1"
	txID := "tx-1"
	invoiceID := "invoice-1"
	now := time.Now()
	closingDay := 24
	dueDay := 1

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		checkingID: {
			ID:             checkingID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 500,
			Currency:       "BRL",
			IsActive:       true,
		},
		ccAccountID: {
			ID:         ccAccountID,
			ProfileID:  profileID,
			Name:       "Cartão Nubank",
			Type:       bankaccount.AccountTypeCreditCard,
			Currency:   "BRL",
			IsActive:   true,
			ClosingDay: &closingDay,
			DueDay:     &dueDay,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Alimentação",
			Type:      category.TypeExpense,
			IsActive:  true,
		},
	}}

	invoiceRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {
			ID:            invoiceID,
			BankAccountID: ccAccountID,
			ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			OpeningDate:   time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC),
			ClosingDate:   time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			DueDate:       time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			Status:        invoice.StatusOpen,
		},
	}}

	existingTx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     profileID,
		BankAccountID: checkingID,
		CategoryID:    &categoryID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{existingTx}}
	useCase := NewUpdateTransactionUseCase(accountRepo, categoryRepo, txRepo, invoiceRepo, nil)

	_, err := useCase.Execute(txID, UpdateTransactionInput{
		BankAccountID: ccAccountID, // Changed to credit card
		CategoryID:    &categoryID,
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2026-03-15",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updatedTx, _ := txRepo.GetByID(txID)
	if updatedTx.InvoiceID == nil {
		t.Fatal("expected invoice to be assigned, got nil")
	}
	if *updatedTx.InvoiceID != invoiceID {
		t.Fatalf("expected invoice %s, got %s", invoiceID, *updatedTx.InvoiceID)
	}
}

// ===== RecalculateBalance Tests (TDD) =====

func TestRecalculateBalance_SumsConfirmedTransactions(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	otherAccountID := "account-2"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Mercado Pago",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 999, // Wrong balance (drifted)
			Currency:       "BRL",
			IsActive:       true,
		},
		otherAccountID: {
			ID:             otherAccountID,
			ProfileID:      profileID,
			Name:           "Nubank",
			Type:           bankaccount.AccountTypeChecking,
			CurrentBalance: 500,
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{
			ID:            "tx-1",
			ProfileID:     profileID,
			BankAccountID: accountID,
			Type:          transaction.TypeIncome,
			Status:        transaction.StatusConfirmed,
			Amount:        1000,
			OccurredOn:    now,
		},
		{
			ID:            "tx-2",
			ProfileID:     profileID,
			BankAccountID: accountID,
			Type:          transaction.TypeExpense,
			Status:        transaction.StatusConfirmed,
			Amount:        300,
			OccurredOn:    now,
		},
		{
			ID:            "tx-3",
			ProfileID:     profileID,
			BankAccountID: accountID,
			Type:          transaction.TypeExpense,
			Status:        transaction.StatusPlanned, // Should be ignored
			Amount:        9999,
			OccurredOn:    now,
		},
		{
			ID:                   "tx-4",
			ProfileID:            profileID,
			BankAccountID:        otherAccountID,
			DestinationAccountID: &accountID,
			Type:                 transaction.TypeTransfer,
			Status:               transaction.StatusConfirmed,
			Amount:               200, // Incoming transfer
			OccurredOn:           now,
		},
	}}

	useCase := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	result, err := useCase.Execute(accountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 0 (initial) + 1000 (income) - 300 (expense) + 200 (incoming transfer) = 900
	if result.NewBalance != 900 {
		t.Fatalf("expected new balance 900, got %.2f", result.NewBalance)
	}

	if result.OldBalance != 999 {
		t.Fatalf("expected old balance 999, got %.2f", result.OldBalance)
	}

	// Account should be updated
	acc := accountRepo.accounts[accountID]
	if acc.CurrentBalance != 900 {
		t.Fatalf("expected account balance 900, got %.2f", acc.CurrentBalance)
	}
}

func TestRecalculateBalance_IncludesInitialBalance(t *testing.T) {
	profileID := "profile-1"
	accountID := "account-1"
	now := time.Now()

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:             accountID,
			ProfileID:      profileID,
			Name:           "Nubank Juridica",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 1000, // Started with 1000
			CurrentBalance: 999,  // Drifted
			Currency:       "BRL",
			IsActive:       true,
		},
	}}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		{
			ID:            "tx-1",
			ProfileID:     profileID,
			BankAccountID: accountID,
			Type:          transaction.TypeExpense,
			Status:        transaction.StatusConfirmed,
			Amount:        300,
			OccurredOn:    now,
		},
	}}

	useCase := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	result, err := useCase.Execute(accountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 1000 (initial) - 300 (expense) = 700
	if result.NewBalance != 700 {
		t.Fatalf("expected new balance 700, got %.2f", result.NewBalance)
	}
}

func TestCreateTransaction_Installments_ShouldCreateMultipleTransactions(t *testing.T) {
	// Given: A credit card with closingDay=27, dueDay=3
	// When: Creating an expense of R$31.46 with 2 installments on 2026-02-28
	// Then: Should create 2 transactions:
	//   - Parcela 1/2: R$15.73, on 2026-02-28, in the March invoice (closes 02/27, due 03/03)
	//   - Parcela 2/2: R$15.73, on 2026-03-28, in the April invoice (closes 03/27, due 04/03)

	profileID := "profile-1"
	creditCardID := "cc-nubank"

	now := time.Now()
	closingDay := 27
	dueDay := 3

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

	limit := 5000.0
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		creditCardID: {
			ID:             creditCardID,
			ProfileID:      profileID,
			Name:           "Cartão Pessoal Nubank",
			Type:           bankaccount.AccountTypeCreditCard,
			InitialBalance: 0,
			CurrentBalance: 0,
			Currency:       "BRL",
			IsActive:       true,
			CreditLimit:    &limit,
			ClosingDay:     &closingDay,
			DueDay:         &dueDay,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	installmentTotal := 2

	input := CreateTransactionInput{
		ProfileID:        profileID,
		BankAccountID:    creditCardID,
		Type:             "EXPENSE",
		Status:           &confirmedStatus,
		Amount:           31.46,
		Currency:         "BRL",
		Description:      "Kvn Imersao 2026 Dolar",
		OccurredOn:       "2026-02-28",
		InstallmentTotal: &installmentTotal,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created 2 transactions
	if len(txRepo.created) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txRepo.created))
	}

	// Check installment 1
	tx1 := txRepo.created[0]
	if tx1.InstallmentNumber == nil || *tx1.InstallmentNumber != 1 {
		t.Fatalf("expected installment 1, got %v", tx1.InstallmentNumber)
	}
	if tx1.InstallmentTotal == nil || *tx1.InstallmentTotal != 2 {
		t.Fatalf("expected installment total 2, got %v", tx1.InstallmentTotal)
	}
	expectedAmount1 := 15.73
	if tx1.Amount != expectedAmount1 {
		t.Fatalf("expected amount %.2f, got %.2f", expectedAmount1, tx1.Amount)
	}
	if !strings.Contains(tx1.Description, "Parcela 1/2") {
		t.Fatalf("expected description to contain 'Parcela 1/2', got '%s'", tx1.Description)
	}
	// Should be on 2026-02-28
	if tx1.OccurredOn.Day() != 28 || tx1.OccurredOn.Month() != 2 {
		t.Fatalf("expected installment 1 on 2026-02-28, got %s", tx1.OccurredOn.Format("2006-01-02"))
	}

	// Check installment 2
	tx2 := txRepo.created[1]
	if tx2.InstallmentNumber == nil || *tx2.InstallmentNumber != 2 {
		t.Fatalf("expected installment 2, got %v", tx2.InstallmentNumber)
	}
	expectedAmount2 := 15.73
	if tx2.Amount != expectedAmount2 {
		t.Fatalf("expected amount %.2f, got %.2f", expectedAmount2, tx2.Amount)
	}
	if !strings.Contains(tx2.Description, "Parcela 2/2") {
		t.Fatalf("expected description to contain 'Parcela 2/2', got '%s'", tx2.Description)
	}
	// Should be on 2026-03-28
	if tx2.OccurredOn.Day() != 28 || tx2.OccurredOn.Month() != 3 {
		t.Fatalf("expected installment 2 on 2026-03-28, got %s", tx2.OccurredOn.Format("2006-01-02"))
	}

	// Both should have invoiceIDs (different invoices)
	if tx1.InvoiceID == nil {
		t.Fatal("expected installment 1 to have invoiceID")
	}
	if tx2.InvoiceID == nil {
		t.Fatal("expected installment 2 to have invoiceID")
	}
	if *tx1.InvoiceID == *tx2.InvoiceID {
		t.Fatal("expected installments to be in different invoices")
	}
}

func TestCreateTransaction_Installments_RegularAccount_ShouldCreateMultiple(t *testing.T) {
	// Given: A regular checking account (not credit card)
	// When: Creating an expense of R$300 with 3 installments on 2026-03-01
	// Then: Should create 3 transactions, each R$100, one month apart, no invoiceID

	profileID := "profile-1"
	accountID := "checking-1"

	now := time.Now()

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Bruno",
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
			InitialBalance: 5000,
			CurrentBalance: 5000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	plannedStatus := "PLANNED"
	installmentTotal := 3

	input := CreateTransactionInput{
		ProfileID:        profileID,
		BankAccountID:    accountID,
		Type:             "EXPENSE",
		Status:           &plannedStatus,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra parcelada",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &installmentTotal,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.created) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(txRepo.created))
	}

	// Each should be R$100
	for i, tx := range txRepo.created {
		if tx.Amount != 100 {
			t.Fatalf("installment %d: expected amount 100, got %.2f", i+1, tx.Amount)
		}
		if tx.InstallmentNumber == nil || *tx.InstallmentNumber != i+1 {
			t.Fatalf("installment %d: expected installmentNumber %d", i+1, i+1)
		}
		if tx.InstallmentTotal == nil || *tx.InstallmentTotal != 3 {
			t.Fatalf("installment %d: expected installmentTotal 3", i+1)
		}
	}

	// Check dates: March, April, May
	expectedMonths := []time.Month{3, 4, 5}
	for i, tx := range txRepo.created {
		if tx.OccurredOn.Month() != expectedMonths[i] {
			t.Fatalf("installment %d: expected month %d, got %d", i+1, expectedMonths[i], tx.OccurredOn.Month())
		}
	}
}

func TestCreateTransaction_SingleInstallment_ShouldCreateNormally(t *testing.T) {
	// Given: installmentTotal=1 (or not set)
	// Then: Should create a single transaction normally

	profileID := "profile-1"
	accountID := "checking-1"

	now := time.Now()

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {
			ID:         profileID,
			CalendarID: "cal-1",
			Name:       "Bruno",
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
			InitialBalance: 5000,
			CurrentBalance: 5000,
			Currency:       "BRL",
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	plannedStatus := "PLANNED"
	installmentTotal := 1

	input := CreateTransactionInput{
		ProfileID:        profileID,
		BankAccountID:    accountID,
		Type:             "EXPENSE",
		Status:           &plannedStatus,
		Amount:           100,
		Currency:         "BRL",
		Description:      "Compra avulsa",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &installmentTotal,
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txRepo.created))
	}

	if txRepo.created[0].Amount != 100 {
		t.Fatalf("expected amount 100, got %.2f", txRepo.created[0].Amount)
	}
}

func TestCreateTransaction_ManualInstallment_ShouldNotAutoCreate(t *testing.T) {
	now := time.Now()
	profileID := "profile-manual-inst"
	accountID := "account-manual-inst"

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {ID: profileID, Name: "Test", CreatedAt: now, UpdatedAt: now},
	}}

	closingDay := 27
	dueDay := 3
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {
			ID:         accountID,
			ProfileID:  profileID,
			Name:       "Nubank",
			Type:       bankaccount.AccountTypeCreditCard,
			Currency:   "BRL",
			IsActive:   true,
			ClosingDay: &closingDay,
			DueDay:     &dueDay,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}

	useCase := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	plannedStatus := "PLANNED"
	installmentNumber := 2
	installmentTotal := 2

	input := CreateTransactionInput{
		ProfileID:         profileID,
		BankAccountID:     accountID,
		Type:              "EXPENSE",
		Status:            &plannedStatus,
		Amount:            15.73,
		Currency:          "BRL",
		Description:       "Kvn Imersao 2026 Dolar - Parcela 2/2",
		OccurredOn:        "2026-03-28",
		InstallmentNumber: &installmentNumber,
		InstallmentTotal:  &installmentTotal,
	}

	txn, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create exactly 1 transaction (not auto-split into 2)
	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txRepo.created))
	}

	// Amount should be unchanged (not divided)
	if txn.Amount != 15.73 {
		t.Fatalf("expected amount 15.73, got %.2f", txn.Amount)
	}

	// Installment metadata preserved
	if txn.InstallmentNumber == nil || *txn.InstallmentNumber != 2 {
		t.Fatalf("expected installmentNumber 2")
	}
	if txn.InstallmentTotal == nil || *txn.InstallmentTotal != 2 {
		t.Fatalf("expected installmentTotal 2")
	}

	// Should be assigned to an invoice (credit card)
	if txn.InvoiceID == nil {
		t.Fatalf("expected invoiceID to be set for credit card transaction")
	}
}

func (f *fakeTransactionRepo) DeleteMany(ids []string) error {
	for _, id := range ids {
		if err := f.Delete(id); err != nil {
			return err
		}
	}
	return nil
}
