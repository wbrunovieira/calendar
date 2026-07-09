package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/yahoo"
)

// mockDividendProvider implements the DividendProvider interface for testing
type mockDividendProvider struct {
	dividends []yahoo.Dividend
	err       error
	calls     []struct {
		ticker string
		from   time.Time
	}
}

func (m *mockDividendProvider) GetDividends(ticker string, from time.Time) ([]yahoo.Dividend, error) {
	m.calls = append(m.calls, struct {
		ticker string
		from   time.Time
	}{ticker, from})
	if m.err != nil {
		return nil, m.err
	}
	return m.dividends, nil
}

// mockTransactionRepo implements transaction.Repository for dividend sync tests
type mockTransactionRepo struct {
	created       []*transaction.Transaction
	foundExternal *transaction.Transaction
	lookupErr     error
}

func (m *mockTransactionRepo) Create(tx *transaction.Transaction) error {
	m.created = append(m.created, tx)
	return nil
}
func (m *mockTransactionRepo) FindByExternalID(externalID string) (*transaction.Transaction, error) {
	if m.lookupErr != nil {
		return nil, m.lookupErr
	}
	if m.foundExternal == nil {
		return nil, transaction.ErrNotFound
	}
	return m.foundExternal, nil
}
func (m *mockTransactionRepo) GetByID(id string) (*transaction.Transaction, error) { return nil, nil }
func (m *mockTransactionRepo) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionRepo) Update(tx *transaction.Transaction) error             { return nil }
func (m *mockTransactionRepo) UpdateStatus(id string, status transaction.Status, occurredOn time.Time, notes *string) error {
	return nil
}
func (m *mockTransactionRepo) Delete(id string) error { return nil }
func (m *mockTransactionRepo) SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error) {
	return nil, nil
}
func (m *mockTransactionRepo) SumByInvoiceID(invoiceID string) (float64, error)                       { return 0, nil }
func (m *mockTransactionRepo) SumByInvoiceIDByStatus(invoiceID string, status transaction.Status) (float64, error) {
	return 0, nil
}
func (m *mockTransactionRepo) CalculateBalanceByBankAccountID(bankAccountID string) (float64, error) {
	return 0, nil
}
func (m *mockTransactionRepo) CalculateBalanceSince(_ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (m *mockTransactionRepo) CalculateBalanceUpTo(_ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (m *mockTransactionRepo) Count(_ transaction.ListFilter) (int, error) {
	return 0, nil
}

func dividendTestAccounts(parentID string, quotas, price float64) *mockAccountRepo {
	fiiType := bankaccount.InvestmentTypeFII
	return &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment, ProfileID: "p1"},
			{
				ID: "hglg-id", Name: "HGLG11", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "p1", LinkedAccountID: &parentID, InvestmentType: &fiiType,
				NumberOfQuotas: &quotas, QuotaPrice: &price, CurrentBalance: quotas * price,
			},
		},
	}
}

func TestDividendSync_CreatesIncomeTransactions(t *testing.T) {
	parentID := "clear-id"
	accountRepo := dividendTestAccounts(parentID, 10.0, 158.50)
	txRepo := &mockTransactionRepo{foundExternal: nil} // no existing dividend

	provider := &mockDividendProvider{
		dividends: []yahoo.Dividend{
			{Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Amount: 1.10},
		},
	}

	uc := NewDividendSyncUseCase(provider, accountRepo, txRepo)
	result, err := uc.Execute("p1", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewDividends != 1 {
		t.Errorf("expected 1 new dividend, got %d", result.NewDividends)
	}
	if result.TotalAmount != 11.0 { // 10 quotas * 1.10
		t.Errorf("expected total 11.0, got %f", result.TotalAmount)
	}

	if len(txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction created, got %d", len(txRepo.created))
	}

	tx := txRepo.created[0]
	if tx.Type != transaction.TypeIncome {
		t.Errorf("expected INCOME, got %s", tx.Type)
	}
	if tx.Amount != 11.0 {
		t.Errorf("expected amount 11.0, got %f", tx.Amount)
	}
	if tx.BankAccountID != parentID {
		t.Errorf("expected bank account %s, got %s", parentID, tx.BankAccountID)
	}
	if tx.Description != "Dividendo HGLG11" {
		t.Errorf("expected description 'Dividendo HGLG11', got %q", tx.Description)
	}
	if tx.ExternalID == nil || *tx.ExternalID != "dividend-HGLG11-2026-04-01-1.10" {
		t.Errorf("unexpected external ID: %v", tx.ExternalID)
	}
	if !tx.OccurredOn.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected occurredOn 2026-04-01, got %v", tx.OccurredOn)
	}
}

func TestDividendSync_PassesSinceDateToProvider(t *testing.T) {
	accountRepo := dividendTestAccounts("clear-id", 10.0, 158.50)
	txRepo := &mockTransactionRepo{foundExternal: nil}
	provider := &mockDividendProvider{}

	since := time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC)
	uc := NewDividendSyncUseCase(provider, accountRepo, txRepo)
	if _, err := uc.Execute("p1", since); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(provider.calls) != 1 {
		t.Fatalf("expected 1 provider call, got %d", len(provider.calls))
	}
	if provider.calls[0].ticker != "HGLG11" {
		t.Errorf("expected ticker HGLG11, got %s", provider.calls[0].ticker)
	}
	if !provider.calls[0].from.Equal(since) {
		t.Errorf("expected from %v, got %v", since, provider.calls[0].from)
	}
}

func TestDividendSync_SkipsAlreadyRecordedDividends(t *testing.T) {
	accountRepo := dividendTestAccounts("clear-id", 10.0, 158.50)

	// Simulate already recorded dividend
	txRepo := &mockTransactionRepo{
		foundExternal: &transaction.Transaction{ID: "existing"},
	}

	provider := &mockDividendProvider{
		dividends: []yahoo.Dividend{
			{Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Amount: 1.10},
		},
	}

	uc := NewDividendSyncUseCase(provider, accountRepo, txRepo)
	result, err := uc.Execute("p1", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewDividends != 0 {
		t.Errorf("expected 0 new dividends, got %d", result.NewDividends)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
	if len(txRepo.created) != 0 {
		t.Errorf("expected 0 transactions created, got %d", len(txRepo.created))
	}
}

func TestDividendSync_OnlyProcessesDividendsAfterSinceDate(t *testing.T) {
	accountRepo := dividendTestAccounts("clear-id", 10.0, 158.50)
	txRepo := &mockTransactionRepo{foundExternal: nil}

	provider := &mockDividendProvider{
		dividends: []yahoo.Dividend{
			{Date: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), Amount: 1.05}, // old
			{Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Amount: 1.10},
		},
	}

	uc := NewDividendSyncUseCase(provider, accountRepo, txRepo)
	// since = 2026-01-01, so only the April 2026 dividend should be processed
	result, err := uc.Execute("p1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewDividends != 1 {
		t.Errorf("expected 1 new dividend, got %d", result.NewDividends)
	}
}

func TestDividendSync_LookupErrorDoesNotCreateDuplicate(t *testing.T) {
	accountRepo := dividendTestAccounts("clear-id", 10.0, 158.50)
	// Dedup lookup fails (e.g. scan error) — must NOT create, or the
	// dividend would be duplicated on every sync run.
	txRepo := &mockTransactionRepo{lookupErr: errors.New("sql: expected 23 destination arguments in Scan, not 24")}

	provider := &mockDividendProvider{
		dividends: []yahoo.Dividend{
			{Date: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Amount: 1.10},
		},
	}

	uc := NewDividendSyncUseCase(provider, accountRepo, txRepo)
	result, err := uc.Execute("p1", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.created) != 0 {
		t.Fatalf("expected 0 transactions created on lookup failure, got %d", len(txRepo.created))
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if result.NewDividends != 0 {
		t.Errorf("expected 0 new dividends, got %d", result.NewDividends)
	}
}

func TestDividendSync_CountsProviderErrors(t *testing.T) {
	accountRepo := dividendTestAccounts("clear-id", 10.0, 158.50)
	txRepo := &mockTransactionRepo{foundExternal: nil}
	provider := &mockDividendProvider{err: errors.New("yahoo down")}

	uc := NewDividendSyncUseCase(provider, accountRepo, txRepo)
	result, err := uc.Execute("p1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if len(txRepo.created) != 0 {
		t.Errorf("expected 0 transactions created, got %d", len(txRepo.created))
	}
}

func TestDividendExternalID(t *testing.T) {
	id := dividendExternalID("HGLG11", "2026-04-01", 1.10)
	expected := "dividend-HGLG11-2026-04-01-1.10"
	if id != expected {
		t.Errorf("expected %q, got %q", expected, id)
	}
}
