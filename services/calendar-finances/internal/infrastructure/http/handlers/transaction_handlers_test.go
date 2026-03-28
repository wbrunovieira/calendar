package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// FakeTransactionRepository for handler tests
type FakeTransactionRepository struct {
	transactions []*transaction.Transaction
	lastFilter   transaction.ListFilter
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
	f.lastFilter = filter // Capture the filter for assertions

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

func (f *FakeTransactionRepository) SumByInvoiceIDByStatus(invoiceID string, status transaction.Status) (float64, error) {
	return 0, nil
}

func (f *FakeTransactionRepository) CalculateBalanceByBankAccountID(bankAccountID string) (float64, error) {
	return 0, nil
}

func (f *FakeTransactionRepository) FindByExternalID(externalID string) (*transaction.Transaction, error) {
	return nil, nil
}

func TestTransactionHandlers_List_ShouldAcceptOccurredFromAndOccurredToParams(t *testing.T) {
	// Arrange
	profileID := "test-profile"

	janTx := &transaction.Transaction{
		ID:            "tx-jan",
		ProfileID:     profileID,
		BankAccountID: "acc-1",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
	}

	febTx := &transaction.Transaction{
		ID:            "tx-feb",
		ProfileID:     profileID,
		BankAccountID: "acc-1",
		OccurredOn:    time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200,
		Currency:      "BRL",
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{janTx, febTx},
	}

	listUC := usecases.NewListTransactionsUseCase(repo)
	handler := NewTransactionHandlers(nil, listUC, nil, nil, nil, nil)

	// Act - call with occurredFrom and occurredTo params
	req := httptest.NewRequest("GET", "/api/v1/transactions?profileId=test-profile&occurredFrom=2026-02-01&occurredTo=2026-02-28", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Data  []*transaction.Transaction `json:"data"`
		Total int                        `json:"total"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should only return February transaction
	if response.Total != 1 {
		t.Errorf("Expected 1 transaction, got %d", response.Total)
	}

	if len(response.Data) > 0 && response.Data[0].ID != "tx-feb" {
		t.Errorf("Expected tx-feb, got %s", response.Data[0].ID)
	}

	// Verify the filter was passed correctly to the repository
	if repo.lastFilter.OccurredFrom == nil {
		t.Error("Expected OccurredFrom filter to be set")
	}
	if repo.lastFilter.OccurredTo == nil {
		t.Error("Expected OccurredTo filter to be set")
	}
}

func TestTransactionHandlers_List_ShouldReturnAllTransactionsWithoutDateFilter(t *testing.T) {
	// Arrange
	profileID := "test-profile"

	janTx := &transaction.Transaction{
		ID:            "tx-jan",
		ProfileID:     profileID,
		BankAccountID: "acc-1",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
	}

	febTx := &transaction.Transaction{
		ID:            "tx-feb",
		ProfileID:     profileID,
		BankAccountID: "acc-1",
		OccurredOn:    time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200,
		Currency:      "BRL",
	}

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{janTx, febTx},
	}

	listUC := usecases.NewListTransactionsUseCase(repo)
	handler := NewTransactionHandlers(nil, listUC, nil, nil, nil, nil)

	// Act - call without date filters
	req := httptest.NewRequest("GET", "/api/v1/transactions?profileId=test-profile", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Data  []*transaction.Transaction `json:"data"`
		Total int                        `json:"total"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return all transactions
	if response.Total != 2 {
		t.Errorf("Expected 2 transactions, got %d", response.Total)
	}
}
