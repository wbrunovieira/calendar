package usecases

import (
	"fmt"
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

func (f *FakeTransactionRepository) matchesFilter(tx *transaction.Transaction, filter transaction.ListFilter) bool {
	if tx.ProfileID != filter.ProfileID {
		return false
	}
	if filter.OccurredFrom != nil && tx.OccurredOn.Before(*filter.OccurredFrom) {
		return false
	}
	if filter.OccurredTo != nil && tx.OccurredOn.After(*filter.OccurredTo) {
		return false
	}
	if filter.Status != nil && tx.Status != *filter.Status {
		return false
	}
	if filter.Type != nil && tx.Type != *filter.Type {
		return false
	}
	if filter.InvoiceID != nil && (tx.InvoiceID == nil || *tx.InvoiceID != *filter.InvoiceID) {
		return false
	}
	return true
}

func (f *FakeTransactionRepository) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	var result []*transaction.Transaction
	for _, tx := range f.transactions {
		if f.matchesFilter(tx, filter) {
			result = append(result, tx)
		}
	}

	if filter.Offset != nil && *filter.Offset > 0 {
		if *filter.Offset >= len(result) {
			return nil, nil
		}
		result = result[*filter.Offset:]
	}

	if filter.Limit != nil && *filter.Limit > 0 && len(result) > *filter.Limit {
		result = result[:*filter.Limit]
	}

	return result, nil
}

func (f *FakeTransactionRepository) Count(filter transaction.ListFilter) (int, error) {
	count := 0
	for _, tx := range f.transactions {
		if f.matchesFilter(tx, filter) {
			count++
		}
	}
	return count, nil
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
	var balance float64
	for _, tx := range f.transactions {
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

func (f *FakeTransactionRepository) FindByExternalID(externalID string) (*transaction.Transaction, error) {
	return nil, nil
}

func (f *FakeTransactionRepository) CalculateBalanceSince(_ string, _ time.Time) (float64, error) {
	return 0, nil
}

func (f *FakeTransactionRepository) CalculateBalanceUpTo(_ string, _ time.Time) (float64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeTx(id string, profileID string, month time.Month) *transaction.Transaction {
	return &transaction.Transaction{
		ID:         id,
		ProfileID:  profileID,
		OccurredOn: time.Date(2026, month, 15, 0, 0, 0, 0, time.UTC),
		Type:       transaction.TypeExpense,
		Status:     transaction.StatusConfirmed,
		Amount:     100,
	}
}

func nTransactions(profileID string, n int) []*transaction.Transaction {
	txns := make([]*transaction.Transaction, n)
	for i := 0; i < n; i++ {
		txns[i] = &transaction.Transaction{
			ID:         fmt.Sprintf("tx-%02d", i+1),
			ProfileID:  profileID,
			OccurredOn: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC),
			Type:       transaction.TypeExpense,
			Status:     transaction.StatusConfirmed,
			Amount:     float64(i + 1),
		}
	}
	return txns
}

// ---------------------------------------------------------------------------
// Pagination tests (RED — these will fail until the use case is updated)
// ---------------------------------------------------------------------------

func TestListTransactions_Pagination_DefaultsToPage1Size50(t *testing.T) {
	profileID := "profile-1"
	repo := &FakeTransactionRepository{transactions: nTransactions(profileID, 60)}
	uc := NewListTransactionsUseCase(repo)

	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 50 {
		t.Errorf("expected 50 items (default pageSize), got %d", len(result.Items))
	}
	if result.Total != 60 {
		t.Errorf("expected total=60, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("expected page=1, got %d", result.Page)
	}
	if result.PageSize != 50 {
		t.Errorf("expected pageSize=50, got %d", result.PageSize)
	}
}

func TestListTransactions_Pagination_Page2ReturnsRemaining(t *testing.T) {
	profileID := "profile-1"
	repo := &FakeTransactionRepository{transactions: nTransactions(profileID, 60)}
	uc := NewListTransactionsUseCase(repo)

	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, Page: 2, PageSize: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 10 {
		t.Errorf("expected 10 items on page 2, got %d", len(result.Items))
	}
	if result.Total != 60 {
		t.Errorf("expected total=60, got %d", result.Total)
	}
}

func TestListTransactions_Pagination_CustomPageSize(t *testing.T) {
	profileID := "profile-1"
	repo := &FakeTransactionRepository{transactions: nTransactions(profileID, 10)}
	uc := NewListTransactionsUseCase(repo)

	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, Page: 2, PageSize: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
	if result.Total != 10 {
		t.Errorf("expected total=10, got %d", result.Total)
	}
	if result.Page != 2 {
		t.Errorf("expected page=2, got %d", result.Page)
	}
}

func TestListTransactions_Pagination_LastPageHasFewer(t *testing.T) {
	profileID := "profile-1"
	repo := &FakeTransactionRepository{transactions: nTransactions(profileID, 7)}
	uc := NewListTransactionsUseCase(repo)

	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, Page: 2, PageSize: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items on last page, got %d", len(result.Items))
	}
	if result.Total != 7 {
		t.Errorf("expected total=7, got %d", result.Total)
	}
}

func TestListTransactions_Pagination_BeyondLastPageReturnsEmpty(t *testing.T) {
	profileID := "profile-1"
	repo := &FakeTransactionRepository{transactions: nTransactions(profileID, 5)}
	uc := NewListTransactionsUseCase(repo)

	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, Page: 3, PageSize: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 0 {
		t.Errorf("expected 0 items beyond last page, got %d", len(result.Items))
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
}

func TestListTransactions_Pagination_TotalRespectsFilters(t *testing.T) {
	profileID := "profile-1"
	invoiceID := "inv-abc"

	txns := nTransactions(profileID, 20)
	// Only first 5 have an invoice
	for i := 0; i < 5; i++ {
		id := invoiceID
		txns[i].InvoiceID = &id
	}

	repo := &FakeTransactionRepository{transactions: txns}
	uc := NewListTransactionsUseCase(repo)

	inv := invoiceID
	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, InvoiceID: &inv, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 5 {
		t.Errorf("total should reflect filter (5), got %d", result.Total)
	}
	if len(result.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(result.Items))
	}
}

// ---------------------------------------------------------------------------
// Existing filter tests — updated to use result.Items
// ---------------------------------------------------------------------------

func TestListTransactions_FilterByDateRange_ShouldReturnOnlyTransactionsInRange(t *testing.T) {
	profileID := "profile-1"

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{
			makeTx("tx-jan", profileID, time.January),
			makeTx("tx-feb-1", profileID, time.February),
			makeTx("tx-feb-2", profileID, time.February),
			makeTx("tx-mar", profileID, time.March),
		},
	}
	repo.transactions[2].OccurredOn = time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)

	uc := NewListTransactionsUseCase(repo)

	from := "2026-02-01"
	to := "2026-02-28"
	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, OccurredFrom: &from, OccurredTo: &to})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 transactions for February, got %d", len(result.Items))
	}
	for _, tx := range result.Items {
		if tx.OccurredOn.Month() != time.February {
			t.Errorf("expected transaction in February, got %v", tx.OccurredOn)
		}
	}
}

func TestListTransactions_FilterByDateRange_ShouldExcludeTransactionsBeforeFrom(t *testing.T) {
	profileID := "profile-1"

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{
			{ID: "tx-old", ProfileID: profileID, OccurredOn: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 100},
			{ID: "tx-new", ProfileID: profileID, OccurredOn: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 200},
		},
	}

	uc := NewListTransactionsUseCase(repo)

	from := "2026-02-01"
	to := "2026-02-28"
	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, OccurredFrom: &from, OccurredTo: &to})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].ID != "tx-new" {
		t.Errorf("expected tx-new, got %s", result.Items[0].ID)
	}
}

func TestListTransactions_FilterByDateRange_ShouldExcludeTransactionsAfterTo(t *testing.T) {
	profileID := "profile-1"

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{
			{ID: "tx-feb", ProfileID: profileID, OccurredOn: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 100},
			{ID: "tx-future", ProfileID: profileID, OccurredOn: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 200},
		},
	}

	uc := NewListTransactionsUseCase(repo)

	from := "2026-02-01"
	to := "2026-02-28"
	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, OccurredFrom: &from, OccurredTo: &to})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].ID != "tx-feb" {
		t.Errorf("expected tx-feb, got %s", result.Items[0].ID)
	}
}

func TestListTransactions_FilterByDateRange_InclusiveBoundaries(t *testing.T) {
	profileID := "profile-1"

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{
			{ID: "tx-first", ProfileID: profileID, OccurredOn: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 100},
			{ID: "tx-last", ProfileID: profileID, OccurredOn: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 200},
		},
	}

	uc := NewListTransactionsUseCase(repo)

	from := "2026-02-01"
	to := "2026-02-28"
	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, OccurredFrom: &from, OccurredTo: &to})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 transactions (inclusive boundaries), got %d", len(result.Items))
	}
}

func TestListTransactions_FilterByInvoiceID_ShouldReturnOnlyMatchingInvoice(t *testing.T) {
	profileID := "profile-1"
	invoiceA := "invoice-a"
	invoiceB := "invoice-b"

	repo := &FakeTransactionRepository{
		transactions: []*transaction.Transaction{
			{ID: "tx-a", ProfileID: profileID, InvoiceID: &invoiceA, Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 50},
			{ID: "tx-b", ProfileID: profileID, InvoiceID: &invoiceB, Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 75},
			{ID: "tx-no-inv", ProfileID: profileID, Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 100},
		},
	}

	uc := NewListTransactionsUseCase(repo)

	result, err := uc.Execute(ListTransactionsInput{ProfileID: profileID, InvoiceID: &invoiceA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 transaction for invoice-a, got %d", len(result.Items))
	}
	if result.Items[0].ID != "tx-a" {
		t.Errorf("expected tx-a, got %s", result.Items[0].ID)
	}
}
