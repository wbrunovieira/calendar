package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// FakeTransactionRepository for testing
type FakeTransactionRepository struct {
	transactions []*transaction.Transaction
}

func (f *FakeTransactionRepository) Create(tx *transaction.Transaction) error {
	f.transactions = append(f.transactions, tx)
	return nil
}

func (f *FakeTransactionRepository) GetByID(id string) (*transaction.Transaction, error) {
	for _, tx := range f.transactions {
		if tx.ID == id {
			return tx, nil
		}
	}
	return nil, nil
}

func (f *FakeTransactionRepository) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	var result []*transaction.Transaction
	for _, tx := range f.transactions {
		if tx.ProfileID != filter.ProfileID {
			continue
		}

		// Filter by date range
		if filter.OccurredFrom != nil && tx.OccurredOn.Before(*filter.OccurredFrom) {
			continue
		}
		if filter.OccurredTo != nil && tx.OccurredOn.After(*filter.OccurredTo) {
			continue
		}

		if filter.Status != nil && tx.Status != *filter.Status {
			continue
		}

		if filter.Type != nil && tx.Type != *filter.Type {
			continue
		}

		if filter.InvoiceID != nil && (tx.InvoiceID == nil || *tx.InvoiceID != *filter.InvoiceID) {
			continue
		}

		result = append(result, tx)
	}
	return result, nil
}

func (f *FakeTransactionRepository) Update(tx *transaction.Transaction) error {
	return nil
}

func (f *FakeTransactionRepository) UpdateStatus(id string, status transaction.Status, occurredOn time.Time, notes *string) error {
	return nil
}

func (f *FakeTransactionRepository) Delete(id string) error {
	return nil
}

func (f *FakeTransactionRepository) SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error) {
	return nil, nil
}

func (f *FakeTransactionRepository) SumByInvoiceID(invoiceID string) (float64, error) {
	return 0, nil
}

func TestListTransactions_FilterByDateRange_ShouldReturnOnlyTransactionsInRange(t *testing.T) {
	// Arrange
	profileID := "profile-1"

	// Create transactions in different months
	janTx := &transaction.Transaction{
		ID:         "tx-jan",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     100,
	}

	febTx1 := &transaction.Transaction{
		ID:         "tx-feb-1",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     200,
	}

	febTx2 := &transaction.Transaction{
		ID:         "tx-feb-2",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     300,
	}

	marTx := &transaction.Transaction{
		ID:         "tx-mar",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     400,
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{janTx, febTx1, febTx2, marTx},
	}

	useCase := NewListTransactionsUseCase(repo)

	// Act - filter for February only
	fromDate := "2026-02-01"
	toDate := "2026-02-28"

	result, err := useCase.Execute(ListTransactionsInput{
		ProfileID:    profileID,
		OccurredFrom: &fromDate,
		OccurredTo:   &toDate,
	})

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 transactions for February, got %d", len(result))
	}

	// Verify we got the right transactions
	for _, tx := range result {
		if tx.OccurredOn.Month() != time.February {
			t.Errorf("Expected transaction in February, got %v", tx.OccurredOn)
		}
	}
}

func TestListTransactions_FilterByDateRange_ShouldExcludeTransactionsBeforeFrom(t *testing.T) {
	// Arrange
	profileID := "profile-1"

	oldTx := &transaction.Transaction{
		ID:         "tx-old",
		ProfileID:  profileID,
		OccurredOn: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     100,
	}

	newTx := &transaction.Transaction{
		ID:         "tx-new",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     200,
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{oldTx, newTx},
	}

	useCase := NewListTransactionsUseCase(repo)

	// Act - filter from February 2026
	fromDate := "2026-02-01"
	toDate := "2026-02-28"

	result, err := useCase.Execute(ListTransactionsInput{
		ProfileID:    profileID,
		OccurredFrom: &fromDate,
		OccurredTo:   &toDate,
	})

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(result))
	}

	if len(result) > 0 && result[0].ID != "tx-new" {
		t.Errorf("Expected tx-new, got %s", result[0].ID)
	}
}

func TestListTransactions_FilterByDateRange_ShouldExcludeTransactionsAfterTo(t *testing.T) {
	// Arrange
	profileID := "profile-1"

	febTx := &transaction.Transaction{
		ID:         "tx-feb",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     100,
	}

	futureTx := &transaction.Transaction{
		ID:         "tx-future",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     200,
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{febTx, futureTx},
	}

	useCase := NewListTransactionsUseCase(repo)

	// Act - filter until end of February 2026
	fromDate := "2026-02-01"
	toDate := "2026-02-28"

	result, err := useCase.Execute(ListTransactionsInput{
		ProfileID:    profileID,
		OccurredFrom: &fromDate,
		OccurredTo:   &toDate,
	})

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(result))
	}

	if len(result) > 0 && result[0].ID != "tx-feb" {
		t.Errorf("Expected tx-feb, got %s", result[0].ID)
	}
}

func TestListTransactions_FilterByDateRange_InclusiveBoundaries(t *testing.T) {
	// Arrange
	profileID := "profile-1"

	// Transaction exactly on the first day
	firstDayTx := &transaction.Transaction{
		ID:         "tx-first",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     100,
	}

	// Transaction exactly on the last day
	lastDayTx := &transaction.Transaction{
		ID:         "tx-last",
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     200,
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{firstDayTx, lastDayTx},
	}

	useCase := NewListTransactionsUseCase(repo)

	// Act
	fromDate := "2026-02-01"
	toDate := "2026-02-28"

	result, err := useCase.Execute(ListTransactionsInput{
		ProfileID:    profileID,
		OccurredFrom: &fromDate,
		OccurredTo:   &toDate,
	})

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 transactions (inclusive boundaries), got %d", len(result))
	}
}

func TestListTransactions_FilterByInvoiceID_ShouldReturnOnlyMatchingInvoice(t *testing.T) {
	profileID := "profile-1"
	invoiceA := "invoice-a"
	invoiceB := "invoice-b"

	txA := &transaction.Transaction{
		ID:        "tx-a",
		ProfileID: profileID,
		InvoiceID: &invoiceA,
		Type:      transaction.TypeExpense,
		Status:    transaction.StatusConfirmed,
		Amount:    50,
	}

	txB := &transaction.Transaction{
		ID:        "tx-b",
		ProfileID: profileID,
		InvoiceID: &invoiceB,
		Type:      transaction.TypeExpense,
		Status:    transaction.StatusConfirmed,
		Amount:    75,
	}

	txNoInvoice := &transaction.Transaction{
		ID:        "tx-no-inv",
		ProfileID: profileID,
		Type:      transaction.TypeExpense,
		Status:    transaction.StatusConfirmed,
		Amount:    100,
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{txA, txB, txNoInvoice},
	}

	useCase := NewListTransactionsUseCase(repo)

	result, err := useCase.Execute(ListTransactionsInput{
		ProfileID: profileID,
		InvoiceID: &invoiceA,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 transaction for invoice-a, got %d", len(result))
	}

	if result[0].ID != "tx-a" {
		t.Errorf("Expected tx-a, got %s", result[0].ID)
	}
}
