package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/google/uuid"
)

// Helper to create a recurring transaction entity for tests
func makeRecurring(profileID string, bankAccountID *string, categoryID *string, txType string, amount float64, description string, rule string, nextOccurrence time.Time) *recurringtransaction.RecurringTransaction {
	return &recurringtransaction.RecurringTransaction{
		ID:             uuid.New().String(),
		ProfileID:      profileID,
		BankAccountID:  bankAccountID,
		CategoryID:     categoryID,
		Type:           txType,
		Amount:         amount,
		Currency:       "BRL",
		Description:    description,
		RecurrenceRule: rule,
		StartOn:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NextOccurrence: nextOccurrence,
		Status:         recurringtransaction.StatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// Helper to set up common test fixtures for process recurring tests
func setupProcessRecurringTest() (
	*fakeRecurringRepo,
	*fakeProfileRepo,
	*fakeAccountRepo,
	*fakeCategoryRepo,
	*fakeTransactionRepo,
	*fakeInvoiceRepo,
	*profile.Profile,
	*bankaccount.BankAccount,
	*category.Category,
) {
	prof := &profile.Profile{
		ID:         uuid.New().String(),
		CalendarID: "cal-1",
		Name:       "WB Digital",
	}

	acct := &bankaccount.BankAccount{
		ID:             uuid.New().String(),
		ProfileID:      prof.ID,
		Name:           "Nubank Juridica",
		Type:           bankaccount.AccountTypeChecking,
		Currency:       "BRL",
		CurrentBalance: 5000.00,
		InitialBalance: 5000.00,
	}

	cat := &category.Category{
		ID:        uuid.New().String(),
		ProfileID: prof.ID,
		Name:      "Assinaturas",
		Type:      category.TypeExpense,
	}

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{prof.ID: prof}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{acct.ID: acct}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{cat.ID: cat}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}
	recurringRepo := newFakeRecurringRepo()

	return recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, cat
}

func TestProcessRecurring_NoDueTransactions(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, _, _, _ := setupProcessRecurringTest()

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 0 {
		t.Errorf("expected processed=0, got %d", result.Processed)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
	}
	if len(result.Results) != 0 {
		t.Errorf("expected empty results, got %d", len(result.Results))
	}
}

func TestProcessRecurring_SingleExpense_CreatesTransaction(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, cat := setupProcessRecurringTest()

	today := time.Now().UTC().Truncate(24 * time.Hour)
	rec := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 87.00, "GoTo VoIP", "FREQ=MONTHLY;BYMONTHDAY=15", today)
	recurringRepo.items[rec.ID] = rec

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed=1, got %d", result.Processed)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
	}

	// Verify transaction was created
	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction created, got %d", len(txRepo.created))
	}

	tx := txRepo.created[0]
	if tx.Amount != 87.00 {
		t.Errorf("expected amount 87.00, got %.2f", tx.Amount)
	}
	if tx.Type != transaction.TypeExpense {
		t.Errorf("expected type EXPENSE, got %s", tx.Type)
	}
	if tx.Description != "GoTo VoIP" {
		t.Errorf("expected description 'GoTo VoIP', got '%s'", tx.Description)
	}
	if tx.BankAccountID != acct.ID {
		t.Errorf("expected bankAccountId %s, got %s", acct.ID, tx.BankAccountID)
	}
	if tx.Status != transaction.StatusConfirmed {
		t.Errorf("expected status CONFIRMED, got %s", tx.Status)
	}

	// Verify next_occurrence was advanced
	updated := recurringRepo.items[rec.ID]
	if !updated.NextOccurrence.After(today) {
		t.Errorf("expected next_occurrence to advance beyond today, got %s", updated.NextOccurrence.Format("2006-01-02"))
	}
}

func TestProcessRecurring_Income_CreatesTransaction(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, _ := setupProcessRecurringTest()

	incomeCat := &category.Category{
		ID:        uuid.New().String(),
		ProfileID: prof.ID,
		Name:      "Receita Clientes",
		Type:      category.TypeIncome,
	}
	categoryRepo.categories[incomeCat.ID] = incomeCat

	today := time.Now().UTC().Truncate(24 * time.Hour)
	rec := makeRecurring(prof.ID, &acct.ID, &incomeCat.ID, "INCOME", 1500.00, "Revalida Italia", "FREQ=MONTHLY;BYMONTHDAY=5", today)
	recurringRepo.items[rec.ID] = rec

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed=1, got %d", result.Processed)
	}

	tx := txRepo.created[0]
	if tx.Type != transaction.TypeIncome {
		t.Errorf("expected type INCOME, got %s", tx.Type)
	}
	if tx.Status != transaction.StatusConfirmed {
		t.Errorf("expected status CONFIRMED, got %s", tx.Status)
	}
	if tx.Amount != 1500.00 {
		t.Errorf("expected amount 1500.00, got %.2f", tx.Amount)
	}
}

func TestProcessRecurring_AdvancesNextOccurrence(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, cat := setupProcessRecurringTest()

	// Feb 5 recurring, should advance to Mar 5
	feb5 := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	rec := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 25.00, "VPS", "FREQ=MONTHLY;BYMONTHDAY=5", feb5)
	recurringRepo.items[rec.ID] = rec

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	_, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := recurringRepo.items[rec.ID]
	expected := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	if !updated.NextOccurrence.Equal(expected) {
		t.Errorf("expected next_occurrence %s, got %s", expected.Format("2006-01-02"), updated.NextOccurrence.Format("2006-01-02"))
	}
}

func TestProcessRecurring_SkipsPausedAndCancelled(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, cat := setupProcessRecurringTest()

	today := time.Now().UTC().Truncate(24 * time.Hour)

	active := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 50.00, "Active Sub", "FREQ=MONTHLY;BYMONTHDAY=1", today)
	recurringRepo.items[active.ID] = active

	paused := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 30.00, "Paused Sub", "FREQ=MONTHLY;BYMONTHDAY=1", today)
	paused.Status = recurringtransaction.StatusPaused
	recurringRepo.items[paused.ID] = paused

	cancelled := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 20.00, "Cancelled Sub", "FREQ=MONTHLY;BYMONTHDAY=1", today)
	cancelled.Status = recurringtransaction.StatusCanceled
	recurringRepo.items[cancelled.ID] = cancelled

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the active one should be processed (FindDueRecurring filters by ACTIVE)
	if result.Processed != 1 {
		t.Errorf("expected processed=1, got %d", result.Processed)
	}
	if len(txRepo.created) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(txRepo.created))
	}
	if txRepo.created[0].Description != "Active Sub" {
		t.Errorf("expected 'Active Sub' transaction, got '%s'", txRepo.created[0].Description)
	}
}

func TestProcessRecurring_MultipleDue_ProcessesAll(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, cat := setupProcessRecurringTest()

	today := time.Now().UTC().Truncate(24 * time.Hour)

	rec1 := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 87.00, "GoTo VoIP", "FREQ=MONTHLY;BYMONTHDAY=15", today)
	rec2 := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 59.90, "Freelahub", "FREQ=MONTHLY;BYMONTHDAY=1", today)
	rec3 := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 98.00, "Google Workspace", "FREQ=MONTHLY;BYMONTHDAY=1", today)

	recurringRepo.items[rec1.ID] = rec1
	recurringRepo.items[rec2.ID] = rec2
	recurringRepo.items[rec3.ID] = rec3

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 3 {
		t.Errorf("expected processed=3, got %d", result.Processed)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed=0, got %d", result.Failed)
	}
	if len(txRepo.created) != 3 {
		t.Errorf("expected 3 transactions, got %d", len(txRepo.created))
	}

	// All next_occurrences should be advanced
	for _, rec := range []*recurringtransaction.RecurringTransaction{rec1, rec2, rec3} {
		updated := recurringRepo.items[rec.ID]
		if !updated.NextOccurrence.After(today) {
			t.Errorf("recurring %s next_occurrence not advanced: %s", rec.Description, updated.NextOccurrence.Format("2006-01-02"))
		}
	}
}

func TestProcessRecurring_CreditCard_CreatesTransaction(t *testing.T) {
	recurringRepo, profileRepo, _, categoryRepo, txRepo, invoiceRepo, prof, _, cat := setupProcessRecurringTest()

	closingDay := 24
	dueDay := 3
	ccAcct := &bankaccount.BankAccount{
		ID:             uuid.New().String(),
		ProfileID:      prof.ID,
		Name:           "Cartao Nubank",
		Type:           bankaccount.AccountTypeCreditCard,
		Currency:       "BRL",
		CreditLimit:    floatPtr(5000.00),
		CurrentBalance: 0,
		ClosingDay:     &closingDay,
		DueDay:         &dueDay,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{ccAcct.ID: ccAcct}}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	rec := makeRecurring(prof.ID, &ccAcct.ID, &cat.ID, "EXPENSE", 572.89, "Claude Code", "FREQ=MONTHLY;BYMONTHDAY=16", today)
	recurringRepo.items[rec.ID] = rec

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed=1, got %d", result.Processed)
	}

	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txRepo.created))
	}

	tx := txRepo.created[0]
	if tx.Status != transaction.StatusConfirmed {
		t.Errorf("expected CONFIRMED, got %s", tx.Status)
	}
	// Credit card balance should NOT change (handled by invoice)
	if ccAcct.CurrentBalance != 0 {
		t.Errorf("expected credit card balance unchanged at 0, got %.2f", ccAcct.CurrentBalance)
	}
}

func TestProcessRecurring_PartialFailure_ContinuesProcessing(t *testing.T) {
	recurringRepo, profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, prof, acct, cat := setupProcessRecurringTest()

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First recurring has invalid bank account (will fail)
	badAccountID := uuid.New().String()
	recBad := makeRecurring(prof.ID, &badAccountID, &cat.ID, "EXPENSE", 50.00, "Bad Account", "FREQ=MONTHLY;BYMONTHDAY=1", today)
	recurringRepo.items[recBad.ID] = recBad

	// Second recurring is valid (will succeed)
	recGood := makeRecurring(prof.ID, &acct.ID, &cat.ID, "EXPENSE", 87.00, "GoTo VoIP", "FREQ=MONTHLY;BYMONTHDAY=15", today)
	recurringRepo.items[recGood.ID] = recGood

	createUC := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo)
	uc := NewProcessRecurringTransactionsUseCase(recurringRepo, createUC)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected processed=1, got %d", result.Processed)
	}
	if result.Failed != 1 {
		t.Errorf("expected failed=1, got %d", result.Failed)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}

	// Verify the good one created a transaction
	if len(txRepo.created) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(txRepo.created))
	}
}

func floatPtr(f float64) *float64 { return &f }
