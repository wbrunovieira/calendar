package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/brapi"
)

// mockTransactionRepo implements transaction.Repository for dividend sync tests
type mockTransactionRepo struct {
	created       []*transaction.Transaction
	foundExternal *transaction.Transaction
}

func (m *mockTransactionRepo) Create(tx *transaction.Transaction) error {
	m.created = append(m.created, tx)
	return nil
}
func (m *mockTransactionRepo) FindByExternalID(externalID string) (*transaction.Transaction, error) {
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

func TestDividendSync_CreatesIncomeTransactions(t *testing.T) {
	parentID := "clear-id"
	fiiType := bankaccount.InvestmentTypeFII
	quotas := 10.0
	price := 158.50

	accountRepo := &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment, ProfileID: "p1"},
			{
				ID: "hglg-id", Name: "HGLG11", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "p1", LinkedAccountID: &parentID, InvestmentType: &fiiType,
				NumberOfQuotas: &quotas, QuotaPrice: &price, CurrentBalance: 1585.0,
			},
		},
	}

	txRepo := &mockTransactionRepo{foundExternal: nil} // no existing dividend

	brapiClient := &mockBrapiClient{
		dividends: []brapi.CashDividend{
			{
				AssetIssued:   "HGLG11",
				PaymentDate:   "2026-04-01",
				Rate:          1.10,
				Label:         "DIVIDENDO",
				LastDatePrior: "2026-03-15",
			},
		},
	}

	uc := NewDividendSyncUseCase(brapiClient, accountRepo, txRepo)
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
}

func TestDividendSync_SkipsAlreadyRecordedDividends(t *testing.T) {
	parentID := "clear-id"
	fiiType := bankaccount.InvestmentTypeFII
	quotas := 10.0
	price := 158.50

	accountRepo := &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment, ProfileID: "p1"},
			{
				ID: "hglg-id", Name: "HGLG11", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "p1", LinkedAccountID: &parentID, InvestmentType: &fiiType,
				NumberOfQuotas: &quotas, QuotaPrice: &price,
			},
		},
	}

	// Simulate already recorded dividend
	txRepo := &mockTransactionRepo{
		foundExternal: &transaction.Transaction{ID: "existing"},
	}

	brapiClient := &mockBrapiClient{
		dividends: []brapi.CashDividend{
			{
				AssetIssued:   "HGLG11",
				PaymentDate:   "2026-04-01",
				Rate:          1.10,
				Label:         "DIVIDENDO",
				LastDatePrior: "2026-03-15",
			},
		},
	}

	uc := NewDividendSyncUseCase(brapiClient, accountRepo, txRepo)
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
	parentID := "clear-id"
	fiiType := bankaccount.InvestmentTypeFII
	quotas := 10.0
	price := 158.50

	accountRepo := &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment, ProfileID: "p1"},
			{
				ID: "hglg-id", Name: "HGLG11", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "p1", LinkedAccountID: &parentID, InvestmentType: &fiiType,
				NumberOfQuotas: &quotas, QuotaPrice: &price,
			},
		},
	}

	txRepo := &mockTransactionRepo{foundExternal: nil}

	brapiClient := &mockBrapiClient{
		dividends: []brapi.CashDividend{
			{AssetIssued: "HGLG11", PaymentDate: "2026-04-01", Rate: 1.10, Label: "DIVIDENDO"},
			{AssetIssued: "HGLG11", PaymentDate: "2025-12-01", Rate: 1.05, Label: "DIVIDENDO"}, // old
		},
	}

	uc := NewDividendSyncUseCase(brapiClient, accountRepo, txRepo)
	// since = 2026-01-01, so only the April 2026 dividend should be processed
	result, err := uc.Execute("p1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewDividends != 1 {
		t.Errorf("expected 1 new dividend, got %d", result.NewDividends)
	}
}

func TestDividendExternalID(t *testing.T) {
	id := dividendExternalID("HGLG11", "2026-04-01", 1.10)
	expected := "dividend-HGLG11-2026-04-01-1.10"
	if id != expected {
		t.Errorf("expected %q, got %q", expected, id)
	}
}
