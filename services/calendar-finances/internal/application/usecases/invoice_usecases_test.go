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

// Test fixtures for invoice tests
type testFixtures struct {
	profileID    string
	cardID       string
	checkingID   string
	categoryID   string
	closingDay   int
	dueDay       int
	profileRepo  *fakeProfileRepo
	accountRepo  *fakeAccountRepo
	categoryRepo *fakeCategoryRepo
	txRepo       *fakeTransactionRepoWithInvoice
	invoiceRepo  *fakeInvoiceRepo
}

func newTestFixtures() *testFixtures {
	profileID := "profile-1"
	cardID := "card-mercado-pago"
	checkingID := "checking-nubank"
	categoryID := "cat-restaurante"
	closingDay := 9
	dueDay := 14

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

	limit := 5000.0
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		cardID: {
			ID:             cardID,
			ProfileID:      profileID,
			Name:           "Cartão Mercado Pago",
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
		checkingID: {
			ID:             checkingID,
			ProfileID:      profileID,
			Name:           "Nubank Conta Corrente",
			Type:           bankaccount.AccountTypeChecking,
			InitialBalance: 10000,
			CurrentBalance: 10000,
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
			Name:      "Restaurante",
			Type:      category.TypeExpense,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}

	txRepo := &fakeTransactionRepoWithInvoice{}
	invoiceRepo := &fakeInvoiceRepo{invoices: make(map[string]*invoice.Invoice)}

	return &testFixtures{
		profileID:    profileID,
		cardID:       cardID,
		checkingID:   checkingID,
		categoryID:   categoryID,
		closingDay:   closingDay,
		dueDay:       dueDay,
		profileRepo:  profileRepo,
		accountRepo:  accountRepo,
		categoryRepo: categoryRepo,
		txRepo:       txRepo,
		invoiceRepo:  invoiceRepo,
	}
}

// Extended fake transaction repository that supports invoice operations
type fakeTransactionRepoWithInvoice struct {
	transactions []*transaction.Transaction
}

func (f *fakeTransactionRepoWithInvoice) Create(tx *transaction.Transaction) error {
	f.transactions = append(f.transactions, tx)
	return nil
}

func (f *fakeTransactionRepoWithInvoice) GetByID(id string) (*transaction.Transaction, error) {
	for _, tx := range f.transactions {
		if tx.ID == id {
			return tx, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeTransactionRepoWithInvoice) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	return f.transactions, nil
}

func (f *fakeTransactionRepoWithInvoice) Update(tx *transaction.Transaction) error {
	for i, existing := range f.transactions {
		if existing.ID == tx.ID {
			f.transactions[i] = tx
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeTransactionRepoWithInvoice) UpdateStatus(string, transaction.Status, time.Time, *string) error {
	return nil
}

func (f *fakeTransactionRepoWithInvoice) Delete(id string) error {
	for i, tx := range f.transactions {
		if tx.ID == id {
			f.transactions = append(f.transactions[:i], f.transactions[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeTransactionRepoWithInvoice) SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error) {
	return nil, nil
}

func (f *fakeTransactionRepoWithInvoice) SumByInvoiceID(invoiceID string) (float64, error) {
	var total float64
	for _, tx := range f.transactions {
		if tx.InvoiceID != nil && *tx.InvoiceID == invoiceID && tx.Status != transaction.StatusCancelled {
			total += invoiceSigned(tx)
		}
	}
	return total, nil
}

func (f *fakeTransactionRepoWithInvoice) SumByInvoiceIDByStatus(invoiceID string, status transaction.Status) (float64, error) {
	var total float64
	for _, tx := range f.transactions {
		if tx.InvoiceID != nil && *tx.InvoiceID == invoiceID && tx.Status == status {
			total += invoiceSigned(tx)
		}
	}
	return total, nil
}

func (f *fakeTransactionRepoWithInvoice) CalculateBalanceByBankAccountID(bankAccountID string) (float64, error) {
	return 0, nil
}

func (f *fakeTransactionRepoWithInvoice) FindByExternalID(externalID string) (*transaction.Transaction, error) {
	return nil, nil
}
func (f *fakeTransactionRepoWithInvoice) CalculateBalanceSince(_ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (f *fakeTransactionRepoWithInvoice) CalculateBalanceUpTo(_ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (f *fakeTransactionRepoWithInvoice) Count(_ transaction.ListFilter) (int, error) {
	return len(f.transactions), nil
}

// Test: GetCurrentInvoice should return confirmed and planned amounts separately
func TestGetCurrentInvoice_ShouldReturnConfirmedAndPlannedAmounts(t *testing.T) {
	f := newTestFixtures()

	invoiceID := "inv-april"
	now := time.Now()

	// Create an open invoice
	f.invoiceRepo.invoices[invoiceID] = &invoice.Invoice{
		ID:            invoiceID,
		BankAccountID: f.cardID,
		ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		OpeningDate:   time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
		Amount:        0,
		Status:        invoice.StatusOpen,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Add confirmed transactions (R$449.99 + R$109.99 = R$559.98)
	invID := invoiceID
	f.txRepo.transactions = append(f.txRepo.transactions,
		&transaction.Transaction{
			ID:            "tx-confirmed-1",
			ProfileID:     f.profileID,
			BankAccountID: f.cardID,
			InvoiceID:     &invID,
			Type:          transaction.TypeExpense,
			Status:        transaction.StatusConfirmed,
			Amount:        449.99,
			OccurredOn:    time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		},
		&transaction.Transaction{
			ID:            "tx-confirmed-2",
			ProfileID:     f.profileID,
			BankAccountID: f.cardID,
			InvoiceID:     &invID,
			Type:          transaction.TypeExpense,
			Status:        transaction.StatusConfirmed,
			Amount:        109.99,
			OccurredOn:    time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),
		},
		// Add planned transaction (R$87.90)
		&transaction.Transaction{
			ID:            "tx-planned-1",
			ProfileID:     f.profileID,
			BankAccountID: f.cardID,
			InvoiceID:     &invID,
			Type:          transaction.TypeExpense,
			Status:        transaction.StatusPlanned,
			Amount:        87.90,
			OccurredOn:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		},
	)

	uc := NewGetCurrentInvoiceUseCase(f.invoiceRepo, f.accountRepo, f.txRepo)
	result, err := uc.Execute(f.cardID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Total should be confirmed + planned = 647.88
	expectedTotal := 559.98 + 87.90
	if result.Amount != expectedTotal {
		t.Errorf("Amount: expected %.2f, got %.2f", expectedTotal, result.Amount)
	}

	// ConfirmedAmount should be 559.98
	if result.ConfirmedAmount != 559.98 {
		t.Errorf("ConfirmedAmount: expected 559.98, got %.2f", result.ConfirmedAmount)
	}

	// PlannedAmount should be 87.90
	if result.PlannedAmount != 87.90 {
		t.Errorf("PlannedAmount: expected 87.90, got %.2f", result.PlannedAmount)
	}
}

// Test: Transaction before closing day should go to current month's invoice
func TestCreditCardTransaction_BeforeClosingDay_GoesToCurrentMonthInvoice(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	// Card closes on day 9, transaction on day 5 should go to January invoice
	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Almoço",
		OccurredOn:    "2026-01-05", // Before closing day 9
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have invoice assigned
	if tx.InvoiceID == nil {
		t.Fatal("expected transaction to have invoice ID assigned")
	}

	// Invoice should be for January
	inv, err := f.invoiceRepo.FindByID(*tx.InvoiceID)
	if err != nil {
		t.Fatalf("failed to find invoice: %v", err)
	}

	if inv.ReferenceDate.Month() != time.January {
		t.Errorf("expected January invoice, got %s", inv.ReferenceDate.Month())
	}

	// Invoice amount should equal transaction amount (recalculated via GetInvoiceUseCase)
	getUC := NewGetInvoiceUseCase(f.invoiceRepo, f.txRepo)
	invWithAmount, err := getUC.Execute(*tx.InvoiceID)
	if err != nil {
		t.Fatalf("failed to get invoice: %v", err)
	}
	if invWithAmount.Amount != 100.00 {
		t.Errorf("expected invoice amount 100.00, got %f", invWithAmount.Amount)
	}
}

// Test: Transaction on closing day should go to next month's invoice
func TestCreditCardTransaction_OnClosingDay_GoesToNextMonthInvoice(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        75.50,
		Currency:      "BRL",
		Description:   "Jantar",
		OccurredOn:    "2026-01-09", // On closing day 9
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.InvoiceID == nil {
		t.Fatal("expected transaction to have invoice ID assigned")
	}

	inv, err := f.invoiceRepo.FindByID(*tx.InvoiceID)
	if err != nil {
		t.Fatalf("failed to find invoice: %v", err)
	}

	if inv.ReferenceDate.Month() != time.February {
		t.Errorf("expected February invoice (on closing day goes to next), got %s", inv.ReferenceDate.Month())
	}
}

// Test: Transaction after closing day should go to next month's invoice
func TestCreditCardTransaction_AfterClosingDay_GoesToNextMonthInvoice(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        200.00,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2026-01-10", // After closing day 9
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.InvoiceID == nil {
		t.Fatal("expected transaction to have invoice ID assigned")
	}

	inv, err := f.invoiceRepo.FindByID(*tx.InvoiceID)
	if err != nil {
		t.Fatalf("failed to find invoice: %v", err)
	}

	if inv.ReferenceDate.Month() != time.February {
		t.Errorf("expected February invoice (after closing), got %s", inv.ReferenceDate.Month())
	}
}

// Test: Multiple transactions should accumulate in same invoice
func TestCreditCardTransaction_MultipleTransactions_AccumulateInSameInvoice(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"

	// First transaction
	input1 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Almoço 1",
		OccurredOn:    "2026-01-02",
	}
	tx1, err := useCase.Execute(input1)
	if err != nil {
		t.Fatalf("unexpected error on tx1: %v", err)
	}

	// Second transaction - same billing cycle
	input2 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        50.00,
		Currency:      "BRL",
		Description:   "Almoço 2",
		OccurredOn:    "2026-01-05",
	}
	tx2, err := useCase.Execute(input2)
	if err != nil {
		t.Fatalf("unexpected error on tx2: %v", err)
	}

	// Third transaction - same billing cycle
	input3 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        75.50,
		Currency:      "BRL",
		Description:   "Jantar",
		OccurredOn:    "2026-01-08",
	}
	tx3, err := useCase.Execute(input3)
	if err != nil {
		t.Fatalf("unexpected error on tx3: %v", err)
	}

	// All transactions should reference same invoice
	if *tx1.InvoiceID != *tx2.InvoiceID || *tx2.InvoiceID != *tx3.InvoiceID {
		t.Error("expected all transactions to have same invoice ID")
	}

	// Invoice amount should be sum of all transactions (recalculated via GetInvoiceUseCase)
	getUC := NewGetInvoiceUseCase(f.invoiceRepo, f.txRepo)
	inv, err := getUC.Execute(*tx1.InvoiceID)
	if err != nil {
		t.Fatalf("failed to get invoice: %v", err)
	}
	expectedTotal := 100.00 + 50.00 + 75.50
	if inv.Amount != expectedTotal {
		t.Errorf("expected invoice amount %f, got %f", expectedTotal, inv.Amount)
	}
}

// Test: Transactions across closing day boundary should go to different invoices
func TestCreditCardTransaction_AcrossClosingBoundary_DifferentInvoices(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"

	// Transaction before closing (January invoice)
	input1 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Antes do fechamento",
		OccurredOn:    "2026-01-08", // Day before closing
	}
	tx1, err := useCase.Execute(input1)
	if err != nil {
		t.Fatalf("unexpected error on tx1: %v", err)
	}

	// Transaction after closing (February invoice)
	input2 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        200.00,
		Currency:      "BRL",
		Description:   "Depois do fechamento",
		OccurredOn:    "2026-01-10", // Day after closing
	}
	tx2, err := useCase.Execute(input2)
	if err != nil {
		t.Fatalf("unexpected error on tx2: %v", err)
	}

	// Should have different invoices
	if *tx1.InvoiceID == *tx2.InvoiceID {
		t.Error("expected transactions to have different invoice IDs")
	}

	// Verify correct months
	inv1, _ := f.invoiceRepo.FindByID(*tx1.InvoiceID)
	inv2, _ := f.invoiceRepo.FindByID(*tx2.InvoiceID)

	if inv1.ReferenceDate.Month() != time.January {
		t.Errorf("expected January invoice for tx1, got %s", inv1.ReferenceDate.Month())
	}
	if inv2.ReferenceDate.Month() != time.February {
		t.Errorf("expected February invoice for tx2, got %s", inv2.ReferenceDate.Month())
	}

	// Verify amounts (recalculated via GetInvoiceUseCase)
	getUC := NewGetInvoiceUseCase(f.invoiceRepo, f.txRepo)
	inv1WithAmt, err := getUC.Execute(*tx1.InvoiceID)
	if err != nil {
		t.Fatalf("failed to get invoice1: %v", err)
	}
	inv2WithAmt, err := getUC.Execute(*tx2.InvoiceID)
	if err != nil {
		t.Fatalf("failed to get invoice2: %v", err)
	}
	if inv1WithAmt.Amount != 100.00 {
		t.Errorf("expected January invoice amount 100.00, got %f", inv1WithAmt.Amount)
	}
	if inv2WithAmt.Amount != 200.00 {
		t.Errorf("expected February invoice amount 200.00, got %f", inv2WithAmt.Amount)
	}
}

// Test: Non-credit card transaction should not have invoice
func TestCheckingAccountTransaction_NoInvoice(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.checkingID, // Checking account, not credit card
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        500.00,
		Currency:      "BRL",
		Description:   "Transferência PIX",
		OccurredOn:    "2026-01-05",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Checking account transactions should NOT have invoice
	if tx.InvoiceID != nil {
		t.Error("expected checking account transaction to have no invoice ID")
	}
}

// Test: Income transaction on credit card should not have invoice
func TestCreditCardIncomeTransaction_NoInvoice(t *testing.T) {
	f := newTestFixtures()

	// Add income category
	incomeCatID := "cat-income"
	f.categoryRepo.categories[incomeCatID] = &category.Category{
		ID:        incomeCatID,
		ProfileID: f.profileID,
		Name:      "Reembolso",
		Type:      category.TypeIncome,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &incomeCatID,
		Type:          "INCOME", // Income, not expense
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Estorno",
		OccurredOn:    "2026-01-05",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Income transactions should NOT affect invoice
	if tx.InvoiceID != nil {
		t.Error("expected income transaction on credit card to have no invoice ID")
	}
}

// Test: December year-end crossing
func TestCreditCardTransaction_YearEndCrossing(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"

	// Transaction in December after closing (should go to January next year)
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        150.00,
		Currency:      "BRL",
		Description:   "Compra de Natal",
		OccurredOn:    "2025-12-15", // After closing day 9
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.InvoiceID == nil {
		t.Fatal("expected transaction to have invoice ID assigned")
	}

	inv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)

	// Should be January 2026 invoice
	if inv.ReferenceDate.Month() != time.January || inv.ReferenceDate.Year() != 2026 {
		t.Errorf("expected January 2026 invoice, got %s %d", inv.ReferenceDate.Month(), inv.ReferenceDate.Year())
	}
}

// Test: Invoice dates calculation
func TestInvoiceDatesCalculation(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Test",
		OccurredOn:    "2026-01-05",
	}

	tx, _ := useCase.Execute(input)
	inv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)

	// Card: closingDay=9, dueDay=14
	// January 2026 invoice:
	// - Opening: December 10, 2025 (day after previous closing)
	// - Closing: January 9, 2026
	// - Due: January 14, 2026

	expectedOpening := time.Date(2025, 12, 9, 0, 0, 0, 0, time.UTC)
	expectedClosing := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	expectedDue := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)

	if !inv.OpeningDate.Equal(expectedOpening) {
		t.Errorf("expected opening date %v, got %v", expectedOpening, inv.OpeningDate)
	}
	if !inv.ClosingDate.Equal(expectedClosing) {
		t.Errorf("expected closing date %v, got %v", expectedClosing, inv.ClosingDate)
	}
	if !inv.DueDate.Equal(expectedDue) {
		t.Errorf("expected due date %v, got %v", expectedDue, inv.DueDate)
	}
}

// Test: RecalculateInvoiceAmount use case
func TestRecalculateInvoiceAmount(t *testing.T) {
	f := newTestFixtures()
	createUC := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)
	recalcUC := NewRecalculateInvoiceAmountUseCase(f.invoiceRepo, f.txRepo)

	confirmedStatus := "CONFIRMED"

	// Create some transactions
	input1 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Tx 1",
		OccurredOn:    "2026-01-02",
	}
	tx1, _ := createUC.Execute(input1)

	input2 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        50.00,
		Currency:      "BRL",
		Description:   "Tx 2",
		OccurredOn:    "2026-01-05",
	}
	createUC.Execute(input2)

	// Manually corrupt the invoice amount to simulate inconsistency
	inv, _ := f.invoiceRepo.FindByID(*tx1.InvoiceID)
	inv.Amount = 999.99 // Wrong value
	f.invoiceRepo.Update(inv)

	// Recalculate should fix it
	recalculatedInv, err := recalcUC.Execute(*tx1.InvoiceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotal := 100.00 + 50.00
	if recalculatedInv.Amount != expectedTotal {
		t.Errorf("expected recalculated amount %f, got %f", expectedTotal, recalculatedInv.Amount)
	}
}

// Test: calculateReferenceMonth function
func TestCalculateReferenceMonth(t *testing.T) {
	testCases := []struct {
		name        string
		txDate      time.Time
		closingDay  int
		expectedRef time.Time
	}{
		{
			name:        "Before closing day - same month",
			txDate:      time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			closingDay:  9,
			expectedRef: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "On closing day - next month",
			txDate:      time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
			closingDay:  9,
			expectedRef: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "After closing day - next month",
			txDate:      time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
			closingDay:  9,
			expectedRef: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "December after closing - January next year",
			txDate:      time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
			closingDay:  9,
			expectedRef: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Last day of month - next month",
			txDate:      time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
			closingDay:  25,
			expectedRef: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateReferenceMonth(tc.txDate, tc.closingDay)
			if !result.Equal(tc.expectedRef) {
				t.Errorf("expected %v, got %v", tc.expectedRef, result)
			}
		})
	}
}

// Test: PLANNED transaction should also be assigned to invoice
func TestCreditCardPlannedTransaction_AssignedToInvoice(t *testing.T) {
	f := newTestFixtures()
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	// PLANNED status (default)
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Compra planejada",
		OccurredOn:    "2026-01-05",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != transaction.StatusPlanned {
		t.Errorf("expected PLANNED status, got %s", tx.Status)
	}

	// PLANNED transactions should also have invoice assigned
	if tx.InvoiceID == nil {
		t.Error("expected PLANNED transaction to have invoice ID assigned")
	}
}

// Test: Invoice ContainsDate method
func TestInvoiceContainsDate(t *testing.T) {
	inv := &invoice.Invoice{
		OpeningDate: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
		ClosingDate: time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
	}

	testCases := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{"Before opening", time.Date(2025, 12, 9, 0, 0, 0, 0, time.UTC), false},
		{"On opening", time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC), true},
		{"Middle of cycle", time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), true},
		{"On closing", time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC), false},
		{"After closing", time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := inv.ContainsDate(tc.date)
			if result != tc.expected {
				t.Errorf("ContainsDate(%v) = %v, expected %v", tc.date, result, tc.expected)
			}
		})
	}
}

// Test: Invoice with different closing/due day configurations
func TestInvoiceWithDifferentConfigurations(t *testing.T) {
	testCases := []struct {
		name             string
		closingDay       int
		dueDay           int
		txDate           string
		expectedRefMonth time.Month
		expectedDueMonth time.Month
	}{
		{
			name:             "Due day after closing day (same month)",
			closingDay:       9,
			dueDay:           14,
			txDate:           "2026-01-05",
			expectedRefMonth: time.January,
			expectedDueMonth: time.January,
		},
		{
			name:             "Due day before closing day (next month)",
			closingDay:       25,
			dueDay:           5,
			txDate:           "2026-01-20",
			expectedRefMonth: time.January,
			expectedDueMonth: time.February,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTestFixtures()

			// Update card configuration
			card := f.accountRepo.accounts[f.cardID]
			card.ClosingDay = &tc.closingDay
			card.DueDay = &tc.dueDay

			useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

			confirmedStatus := "CONFIRMED"
			input := CreateTransactionInput{
				ProfileID:     f.profileID,
				BankAccountID: f.cardID,
				CategoryID:    &f.categoryID,
				Type:          "EXPENSE",
				Status:        &confirmedStatus,
				Amount:        100.00,
				Currency:      "BRL",
				Description:   "Test",
				OccurredOn:    tc.txDate,
			}

			tx, err := useCase.Execute(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			inv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)

			if inv.ReferenceDate.Month() != tc.expectedRefMonth {
				t.Errorf("expected reference month %s, got %s", tc.expectedRefMonth, inv.ReferenceDate.Month())
			}
			if inv.DueDate.Month() != tc.expectedDueMonth {
				t.Errorf("expected due month %s, got %s", tc.expectedDueMonth, inv.DueDate.Month())
			}
		})
	}
}

// =============================================================================
// Pay Invoice Tests (TDD)
// =============================================================================

func TestPayInvoice_ShouldCreateTransactionOnLinkedCheckingAccount(t *testing.T) {
	// Given: A credit card with a linked checking account and an open invoice of R$500
	// When: Paying the invoice
	// Then: A transaction should be created on the linked checking account
	//       And the checking account balance should decrease

	f := newTestFixtures()

	// Link credit card to checking account
	card := f.accountRepo.accounts[f.cardID]
	card.LinkedAccountID = &f.checkingID

	// Create an invoice with some amount
	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Amount = 500.00
	f.invoiceRepo.Create(inv)

	// Create PayInvoice use case with all dependencies
	useCase := NewPayInvoiceUseCaseV2(f.invoiceRepo, f.accountRepo, f.txRepo)

	input := PayInvoiceInput{
		InvoiceID:  inv.ID,
		PaidAmount: 500.00,
		PaidAt:     "2026-01-14",
	}

	paidInv, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invoice should be marked as paid
	if paidInv.Status != invoice.StatusPaid {
		t.Errorf("expected invoice status PAID, got %s", paidInv.Status)
	}

	// The payment is modeled as a SINGLE TRANSFER (funding account -> card), never as
	// INCOME/EXPENSE, so it does not pollute monthly cashflow on any consumer.
	if len(f.txRepo.transactions) != 1 {
		t.Fatalf("expected 1 transfer transaction, got %d", len(f.txRepo.transactions))
	}
	transferTx := f.txRepo.transactions[0]
	if transferTx.Type != transaction.TypeTransfer {
		t.Errorf("expected TRANSFER, got %s", transferTx.Type)
	}
	if transferTx.BankAccountID != f.checkingID {
		t.Errorf("expected source = checking %s, got %s", f.checkingID, transferTx.BankAccountID)
	}
	if transferTx.DestinationAccountID == nil || *transferTx.DestinationAccountID != f.cardID {
		t.Errorf("expected destination = card %s, got %v", f.cardID, transferTx.DestinationAccountID)
	}
	if transferTx.Amount != 500.00 {
		t.Errorf("expected amount 500.00, got %.2f", transferTx.Amount)
	}
	if transferTx.Status != transaction.StatusConfirmed {
		t.Errorf("expected CONFIRMED, got %s", transferTx.Status)
	}
	expectedDesc := "Pagamento fatura Cartão Mercado Pago"
	if transferTx.Description != expectedDesc {
		t.Errorf("expected description '%s', got '%s'", expectedDesc, transferTx.Description)
	}

	// Checking account balance should decrease by the paid amount.
	checking := f.accountRepo.accounts[f.checkingID]
	expectedBalance := 10000.0 - 500.0 // Initial 10000 - paid 500
	if checking.CurrentBalance != expectedBalance {
		t.Errorf("expected checking balance %.2f, got %.2f", expectedBalance, checking.CurrentBalance)
	}
}

func TestPayInvoice_NoLinkedAccount_ShouldStillCreditCard(t *testing.T) {
	// Given: A credit card WITHOUT a linked checking account
	// When: Paying the invoice
	// Then: Invoice should be marked as paid
	//       No checking-side transaction is created (no linked account)
	//       But the card-side credit IS still recorded, so the card balance
	//       remains a true consequence of its transactions.

	f := newTestFixtures()

	// Card has NO linked account
	card := f.accountRepo.accounts[f.cardID]
	card.LinkedAccountID = nil

	// Create an invoice
	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Amount = 300.00
	f.invoiceRepo.Create(inv)

	useCase := NewPayInvoiceUseCaseV2(f.invoiceRepo, f.accountRepo, f.txRepo)

	input := PayInvoiceInput{
		InvoiceID:  inv.ID,
		PaidAmount: 300.00,
		PaidAt:     "2026-01-14",
	}

	paidInv, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invoice should be marked as paid
	if paidInv.Status != invoice.StatusPaid {
		t.Errorf("expected invoice status PAID, got %s", paidInv.Status)
	}

	// Exactly one transaction: the card-side credit (no checking leg without a link)
	if len(f.txRepo.transactions) != 1 {
		t.Fatalf("expected 1 transaction (card credit), got %d", len(f.txRepo.transactions))
	}
	cardCredit := f.txRepo.transactions[0]
	if cardCredit.BankAccountID != f.cardID {
		t.Errorf("expected credit on card %s, got %s", f.cardID, cardCredit.BankAccountID)
	}
	if cardCredit.Type != transaction.TypeIncome {
		t.Errorf("expected INCOME credit on card, got %s", cardCredit.Type)
	}
	if cardCredit.Amount != 300.00 {
		t.Errorf("expected card credit 300.00, got %.2f", cardCredit.Amount)
	}
}

func TestPayInvoice_CreditsCardAndRecalculatesCardBalance(t *testing.T) {
	// Given: A credit card linked to a checking account, an open invoice of R$500,
	//        and a balance recalculator wired in.
	// When: Paying the invoice
	// Then: A credit (INCOME) is recorded on the card for the paid amount,
	//       and the card balance is recomputed from its transactions.

	f := newTestFixtures()
	card := f.accountRepo.accounts[f.cardID]
	card.LinkedAccountID = &f.checkingID

	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Amount = 500.00
	f.invoiceRepo.Create(inv)

	recalc := &trackingRecalculator{}
	useCase := NewPayInvoiceUseCaseV2(f.invoiceRepo, f.accountRepo, f.txRepo, recalc)

	_, err := useCase.Execute(PayInvoiceInput{
		InvoiceID:  inv.ID,
		PaidAmount: 500.00,
		PaidAt:     "2026-01-14",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The payment must be a TRANSFER that credits (pays down) the card.
	var transferTx *transaction.Transaction
	for _, tx := range f.txRepo.transactions {
		if tx.Type == transaction.TypeTransfer &&
			tx.DestinationAccountID != nil && *tx.DestinationAccountID == f.cardID {
			transferTx = tx
		}
	}
	if transferTx == nil {
		t.Fatalf("expected a TRANSFER crediting the card")
	}
	if transferTx.Amount != 500.00 {
		t.Errorf("expected transfer 500.00, got %.2f", transferTx.Amount)
	}

	// The card balance must be recomputed from transactions (not set arbitrarily).
	if !recalc.calledWith(f.cardID) {
		t.Errorf("expected card balance to be recalculated, calls=%v", recalc.calls)
	}
}

func TestPayInvoice_PartialPayment_ShouldCreateTransactionWithPaidAmount(t *testing.T) {
	// Given: An invoice of R$1000
	// When: Paying R$500 (partial payment)
	// Then: Transaction should be created for R$500

	f := newTestFixtures()

	card := f.accountRepo.accounts[f.cardID]
	card.LinkedAccountID = &f.checkingID

	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Amount = 1000.00
	f.invoiceRepo.Create(inv)

	useCase := NewPayInvoiceUseCaseV2(f.invoiceRepo, f.accountRepo, f.txRepo)

	input := PayInvoiceInput{
		InvoiceID:  inv.ID,
		PaidAmount: 500.00, // Partial payment
		PaidAt:     "2026-01-14",
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Transaction amount should be the paid amount, not invoice amount
	paymentTx := f.txRepo.transactions[0]
	if paymentTx.Amount != 500.00 {
		t.Errorf("expected amount 500.00 (paid amount), got %.2f", paymentTx.Amount)
	}

	// Checking account should decrease by paid amount
	checking := f.accountRepo.accounts[f.checkingID]
	expectedBalance := 10000.0 - 500.0
	if checking.CurrentBalance != expectedBalance {
		t.Errorf("expected checking balance %.2f, got %.2f", expectedBalance, checking.CurrentBalance)
	}
}

func TestPayInvoice_DoesNotCreateIncomeOrExpense(t *testing.T) {
	// Regression guard: paying an invoice must NOT create any INCOME or EXPENSE
	// transaction (those inflate the monthly cashflow on every consumer). A linked
	// card payment is a TRANSFER and nothing else.
	f := newTestFixtures()
	card := f.accountRepo.accounts[f.cardID]
	card.LinkedAccountID = &f.checkingID

	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Amount = 907.52
	f.invoiceRepo.Create(inv)

	useCase := NewPayInvoiceUseCaseV2(f.invoiceRepo, f.accountRepo, f.txRepo)
	if _, err := useCase.Execute(PayInvoiceInput{InvoiceID: inv.ID, PaidAmount: 907.52, PaidAt: "2026-01-14"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, tx := range f.txRepo.transactions {
		if tx.Type == transaction.TypeIncome || tx.Type == transaction.TypeExpense {
			t.Errorf("invoice payment created a %s transaction (pollutes cashflow): %+v", tx.Type, tx)
		}
	}
}

// =============================================================================
// Update Invoice Tests (TDD)
// =============================================================================

func TestUpdateInvoice_ShouldUpdateDueDate(t *testing.T) {
	f := newTestFixtures()

	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	f.invoiceRepo.Create(inv)

	useCase := NewUpdateInvoiceUseCase(f.invoiceRepo)

	newDueDate := "2026-03-16"
	input := UpdateInvoiceInput{
		DueDate: &newDueDate,
	}

	updated, err := useCase.Execute(inv.ID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDue := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	if !updated.DueDate.Equal(expectedDue) {
		t.Errorf("expected due date %v, got %v", expectedDue, updated.DueDate)
	}
}

func TestUpdateInvoice_ShouldUpdateMultipleFields(t *testing.T) {
	f := newTestFixtures()

	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	f.invoiceRepo.Create(inv)

	useCase := NewUpdateInvoiceUseCase(f.invoiceRepo)

	newDueDate := "2026-03-16"
	newClosingDate := "2026-03-09"
	input := UpdateInvoiceInput{
		DueDate:     &newDueDate,
		ClosingDate: &newClosingDate,
	}

	updated, err := useCase.Execute(inv.ID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDue := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	expectedClosing := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)

	if !updated.DueDate.Equal(expectedDue) {
		t.Errorf("expected due date %v, got %v", expectedDue, updated.DueDate)
	}
	if !updated.ClosingDate.Equal(expectedClosing) {
		t.Errorf("expected closing date %v, got %v", expectedClosing, updated.ClosingDate)
	}
}

func TestUpdateInvoice_NotFound_ShouldReturnError(t *testing.T) {
	f := newTestFixtures()
	useCase := NewUpdateInvoiceUseCase(f.invoiceRepo)

	newDueDate := "2026-03-16"
	input := UpdateInvoiceInput{
		DueDate: &newDueDate,
	}

	_, err := useCase.Execute("non-existent-id", input)
	if err != ErrInvoiceNotFound {
		t.Errorf("expected ErrInvoiceNotFound, got %v", err)
	}
}

// =============================================================================
// Auto-Close Invoice Tests (TDD)
// =============================================================================

func TestAutoCloseInvoices_ShouldCloseOpenInvoicesPastClosingDate(t *testing.T) {
	f := newTestFixtures()

	// Create an OPEN invoice with closing date 2026-03-09 (already past)
	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Amount = 1883.53
	f.invoiceRepo.Create(inv)

	useCase := NewAutoCloseInvoicesUseCase(f.invoiceRepo)

	// Run auto-close as of 2026-03-10 (one day after closing)
	result, err := useCase.Execute(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Closed != 1 {
		t.Errorf("expected 1 invoice closed, got %d", result.Closed)
	}

	// Verify invoice is now CLOSED
	updated, _ := f.invoiceRepo.FindByID(inv.ID)
	if updated.Status != invoice.StatusClosed {
		t.Errorf("expected CLOSED status, got %s", updated.Status)
	}
}

func TestAutoCloseInvoices_ShouldNotCloseInvoiceBeforeClosingDate(t *testing.T) {
	f := newTestFixtures()

	// Create an OPEN invoice with closing date 2026-03-09
	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	f.invoiceRepo.Create(inv)

	useCase := NewAutoCloseInvoicesUseCase(f.invoiceRepo)

	// Run auto-close as of 2026-03-08 (before closing)
	result, err := useCase.Execute(time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Closed != 0 {
		t.Errorf("expected 0 invoices closed, got %d", result.Closed)
	}

	// Verify invoice is still OPEN
	updated, _ := f.invoiceRepo.FindByID(inv.ID)
	if updated.Status != invoice.StatusOpen {
		t.Errorf("expected OPEN status, got %s", updated.Status)
	}
}

func TestAutoCloseInvoices_ShouldNotCloseAlreadyClosedOrPaid(t *testing.T) {
	f := newTestFixtures()

	// Create a CLOSED invoice
	inv1, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	inv1.Close()
	f.invoiceRepo.Create(inv1)

	// Create a PAID invoice
	inv2, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	inv2.Pay(500, time.Now())
	f.invoiceRepo.Create(inv2)

	useCase := NewAutoCloseInvoicesUseCase(f.invoiceRepo)

	result, err := useCase.Execute(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Neither should be affected
	if result.Closed != 0 {
		t.Errorf("expected 0 invoices closed (already closed/paid), got %d", result.Closed)
	}
}

func TestAutoCloseInvoices_ShouldCloseMultipleInvoices(t *testing.T) {
	f := newTestFixtures()

	// Add a second credit card
	closingDay2 := 25
	dueDay2 := 5
	limit := 3000.0
	f.accountRepo.accounts["card-nubank"] = &bankaccount.BankAccount{
		ID: "card-nubank", ProfileID: f.profileID, Name: "Cartao Nubank",
		Type: bankaccount.AccountTypeCreditCard, Currency: "BRL", IsActive: true,
		CreditLimit: &limit, ClosingDay: &closingDay2, DueDay: &dueDay2,
	}

	// Invoice 1: MP closing 09/03
	inv1, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID, ClosingDay: f.closingDay, DueDay: f.dueDay,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	f.invoiceRepo.Create(inv1)

	// Invoice 2: Nubank closing 25/02
	inv2, _ := invoice.New(invoice.CreateParams{
		BankAccountID: "card-nubank", ClosingDay: closingDay2, DueDay: dueDay2,
		ReferenceDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	f.invoiceRepo.Create(inv2)

	useCase := NewAutoCloseInvoicesUseCase(f.invoiceRepo)

	result, err := useCase.Execute(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Closed != 2 {
		t.Errorf("expected 2 invoices closed, got %d", result.Closed)
	}
}

func TestUpdateInvoice_PaidInvoice_ShouldReturnError(t *testing.T) {
	f := newTestFixtures()

	inv, _ := invoice.New(invoice.CreateParams{
		BankAccountID: f.cardID,
		ClosingDay:    f.closingDay,
		DueDay:        f.dueDay,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	inv.Pay(500, time.Now())
	f.invoiceRepo.Create(inv)

	useCase := NewUpdateInvoiceUseCase(f.invoiceRepo)

	newDueDate := "2026-03-16"
	input := UpdateInvoiceInput{
		DueDate: &newDueDate,
	}

	_, err := useCase.Execute(inv.ID, input)
	if err != ErrInvoiceAlreadyPaid {
		t.Errorf("expected ErrInvoiceAlreadyPaid, got %v", err)
	}
}

// Test: ListInvoices should NOT recalculate amount for CLOSED invoices
func TestListInvoices_ShouldAlwaysRecalculateAmount_EvenWhenClosed(t *testing.T) {
	f := newTestFixtures()
	createTxUC := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"

	// Create an expense transaction (creates invoice automatically)
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        200.00,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2026-01-05",
	}
	tx, err := createTxUC.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %v", err)
	}

	// Close the invoice and manually corrupt its amount
	inv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)
	inv.Close()
	inv.Amount = 150.00 // stale/wrong value
	f.invoiceRepo.Update(inv)

	// List should ALWAYS recalculate from transactions, regardless of status
	listUC := NewListInvoicesUseCase(f.invoiceRepo, f.accountRepo, f.txRepo)
	invoices, err := listUC.Execute(f.cardID)
	if err != nil {
		t.Fatalf("unexpected error listing invoices: %v", err)
	}

	var found *invoice.Invoice
	for _, i := range invoices {
		if i.ID == inv.ID {
			found = i
			break
		}
	}
	if found == nil {
		t.Fatal("expected to find the invoice")
	}
	if found.Amount != 200.00 {
		t.Errorf("expected recalculated amount 200.00, got %.2f (should always recalculate)", found.Amount)
	}
}

// Test: ListInvoices SHOULD recalculate amount for OPEN invoices
func TestListInvoices_ShouldRecalculateOpenInvoiceAmount(t *testing.T) {
	f := newTestFixtures()
	createTxUC := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"

	// Create two expense transactions
	input1 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Compra 1",
		OccurredOn:    "2026-01-05",
	}
	tx, err := createTxUC.Execute(input1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input2 := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        75.00,
		Currency:      "BRL",
		Description:   "Compra 2",
		OccurredOn:    "2026-01-06",
	}
	createTxUC.Execute(input2)

	// Manually corrupt the invoice amount
	inv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)
	inv.Amount = 999.99
	f.invoiceRepo.Update(inv)

	// List should recalculate OPEN invoice amount from transactions
	listUC := NewListInvoicesUseCase(f.invoiceRepo, f.accountRepo, f.txRepo)
	invoices, err := listUC.Execute(f.cardID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found *invoice.Invoice
	for _, i := range invoices {
		if i.ID == inv.ID {
			found = i
			break
		}
	}
	if found == nil {
		t.Fatal("expected to find the invoice")
	}
	if found.Amount != 175.00 {
		t.Errorf("expected open invoice amount 175.00 (recalculated), got %.2f", found.Amount)
	}
}

// Test: GetInvoice should ALWAYS recalculate amount from transactions, even for CLOSED invoices
func TestGetInvoice_ShouldAlwaysRecalculateAmount_EvenWhenClosed(t *testing.T) {
	f := newTestFixtures()
	createTxUC := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        200.00,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2026-01-05",
	}
	tx, err := createTxUC.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close and corrupt the stored amount
	inv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)
	inv.Close()
	inv.Amount = 150.00 // stale/wrong value
	f.invoiceRepo.Update(inv)

	// Get should ALWAYS recalculate from transactions regardless of status
	getUC := NewGetInvoiceUseCase(f.invoiceRepo, f.txRepo)
	result, err := getUC.Execute(inv.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Amount != 200.00 {
		t.Errorf("expected recalculated amount 200.00, got %.2f (should always recalculate)", result.Amount)
	}
}

// Test: Auto-create invoice for high closing day cards with existing paid invoices
// Scenario: closingDay=24, existing March invoice (closing Feb 24, PAID),
// new transaction on Mar 4 should auto-create April invoice
func TestGetOrCreateInvoice_HighClosingDay_ShouldCreateNextInvoiceWhenCurrentExists(t *testing.T) {
	f := newTestFixtures()

	// Configure card with closingDay=24, dueDay=3
	closingDay := 24
	dueDay := 3
	card := f.accountRepo.accounts[f.cardID]
	card.ClosingDay = &closingDay
	card.DueDay = &dueDay

	// Create existing March invoice with real Nubank convention:
	// ref=March, closing=Feb 24, opening=Jan 25 (created externally, not by New())
	marchInv := &invoice.Invoice{
		ID:            "march-inv",
		BankAccountID: f.cardID,
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		OpeningDate:   time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		Amount:        3000,
		Status:        invoice.StatusPaid,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	paidAmount := 3000.0
	paidAt := time.Now()
	marchInv.PaidAmount = &paidAmount
	marchInv.PaidAt = &paidAt
	f.invoiceRepo.Create(marchInv)

	// Transaction on Mar 4 should auto-create April invoice
	useCase := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmedStatus := "CONFIRMED"
	input := CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmedStatus,
		Amount:        10.00,
		Currency:      "BRL",
		Description:   "Lanche SESC",
		OccurredOn:    "2026-03-04",
	}

	tx, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if tx.InvoiceID == nil {
		t.Fatal("expected transaction to have an invoice ID")
	}

	// The new invoice should be for April (not March which already exists)
	newInv, _ := f.invoiceRepo.FindByID(*tx.InvoiceID)
	if newInv.ReferenceDate.Month() != time.April {
		t.Errorf("expected April invoice, got %s", newInv.ReferenceDate.Month())
	}

	// The April invoice opening should be Feb 25 (day after March invoice closing Feb 24)
	expectedOpening := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	if !newInv.OpeningDate.Equal(expectedOpening) {
		t.Errorf("expected April invoice opening on %s, got %s", expectedOpening.Format("2006-01-02"), newInv.OpeningDate.Format("2006-01-02"))
	}

	// The invoice should now contain the transaction date Mar 4
	if !newInv.ContainsDate(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected April invoice to contain transaction date Mar 4")
	}

	// A second transaction on the same date should reuse the same invoice
	input2 := input
	input2.Description = "Cerveja SESC"
	input2.Amount = 15.00
	tx2, err := useCase.Execute(input2)
	if err != nil {
		t.Fatalf("expected no error for second transaction, got: %v", err)
	}
	if *tx2.InvoiceID != *tx.InvoiceID {
		t.Errorf("expected same invoice ID for second transaction, got different: %s vs %s", *tx2.InvoiceID, *tx.InvoiceID)
	}
}

// A refund is credited back onto the card carrying the same invoice_id as the
// purchase it reverses. Reading the invoice must net it out, not add it: the
// correction cannot make the bill bigger. Editing a card purchase and flipping
// only its type from EXPENSE to INCOME keeps the invoice link, so this state is
// reachable through the API and not a hypothetical.
func TestGetInvoice_NetsARefundInsteadOfAddingIt(t *testing.T) {
	f := newTestFixtures()
	createTxUC := NewCreateTransactionUseCase(f.profileRepo, f.accountRepo, f.categoryRepo, f.txRepo, f.invoiceRepo, nil, nil)

	confirmed := "CONFIRMED"
	purchase, err := createTxUC.Execute(CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		Type:          "EXPENSE",
		Status:        &confirmed,
		Amount:        100.00,
		Currency:      "BRL",
		Description:   "Compra",
		OccurredOn:    "2026-01-05",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The refund is written straight to the repository: creating an INCOME on a
	// card through the use case is a different argument, and what matters here
	// is how the invoice reads a credit that is already linked to it. Editing a
	// card purchase and flipping only its type produces exactly this row.
	refund, err := transaction.New(transaction.CreateParams{
		ProfileID:     f.profileID,
		BankAccountID: f.cardID,
		CategoryID:    &f.categoryID,
		InvoiceID:     purchase.InvoiceID,
		Type:          transaction.TypeIncome,
		Amount:        30.00,
		Currency:      "BRL",
		Description:   "Estorno parcial",
		OccurredOn:    time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("building the refund: %v", err)
	}
	refund.Status = transaction.StatusConfirmed
	if err := f.txRepo.Create(refund); err != nil {
		t.Fatalf("storing the refund: %v", err)
	}

	result, err := NewGetInvoiceUseCase(f.invoiceRepo, f.txRepo).Execute(*purchase.InvoiceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Amount != 70.00 {
		t.Errorf("invoice amount = %.2f, want 70.00 (100 charged, 30 refunded)", result.Amount)
	}
}

func (f *fakeTransactionRepoWithInvoice) DeleteMany(ids []string) error {
	for _, id := range ids {
		if err := f.Delete(id); err != nil {
			return err
		}
	}
	return nil
}
