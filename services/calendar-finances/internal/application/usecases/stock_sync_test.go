package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/brapi"
)

// mockBrapiClient implements the BrapiClient interface for testing
type mockBrapiClient struct {
	quotes   []brapi.QuoteResult
	quoteErr error
}

func (m *mockBrapiClient) GetQuotes(tickers ...string) ([]brapi.QuoteResult, error) {
	if m.quoteErr != nil {
		return nil, m.quoteErr
	}
	return m.quotes, nil
}

// mockAccountRepo implements bankaccount.Repository for testing
type mockAccountRepo struct {
	accounts []*bankaccount.BankAccount
	updated  []*bankaccount.BankAccount
}

func (m *mockAccountRepo) FindByProfileID(profileID string) ([]*bankaccount.BankAccount, error) {
	return m.accounts, nil
}

func (m *mockAccountRepo) Update(account *bankaccount.BankAccount) error {
	m.updated = append(m.updated, account)
	return nil
}

func (m *mockAccountRepo) Create(account *bankaccount.BankAccount) error        { return nil }
func (m *mockAccountRepo) GetByID(id string) (*bankaccount.BankAccount, error)  { return nil, nil }
func (m *mockAccountRepo) Delete(id string) error                               { return nil }
func (m *mockAccountRepo) Reorder(profileID string, ids []string) error         { return nil }
func (m *mockAccountRepo) FindByID(id string) (*bankaccount.BankAccount, error) { return nil, nil }
func (m *mockAccountRepo) FindAll() ([]*bankaccount.BankAccount, error)         { return nil, nil }
func (m *mockAccountRepo) UpdateDisplayOrders(updates []bankaccount.DisplayOrderUpdate) error {
	return nil
}

func TestStockSync_UpdatesPricesAndBalances(t *testing.T) {
	quotas := 10.0
	oldPrice := 150.0
	parentID := "clear-account-id"
	fiiType := bankaccount.InvestmentTypeFII

	repo := &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{
				ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "profile-1", Currency: "BRL", CurrentBalance: 220.25,
			},
			{
				ID: "hglg-id", Name: "HGLG11", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "profile-1", Currency: "BRL",
				LinkedAccountID: &parentID, InvestmentType: &fiiType,
				NumberOfQuotas: &quotas, QuotaPrice: &oldPrice,
				CurrentBalance: 1500.0,
			},
		},
	}

	brapiClient := &mockBrapiClient{
		quotes: []brapi.QuoteResult{
			{Symbol: "HGLG11", RegularMarketPrice: 158.50},
		},
	}

	uc := NewStockSyncUseCase(brapiClient, repo)
	result, err := uc.Execute("profile-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.UpdatedAccounts) != 1 {
		t.Fatalf("expected 1 updated account, got %d", len(result.UpdatedAccounts))
	}

	update := result.UpdatedAccounts[0]
	if update.Ticker != "HGLG11" {
		t.Errorf("expected ticker HGLG11, got %s", update.Ticker)
	}
	if update.OldPrice != 150.0 {
		t.Errorf("expected old price 150.0, got %f", update.OldPrice)
	}
	if update.NewPrice != 158.50 {
		t.Errorf("expected new price 158.50, got %f", update.NewPrice)
	}
	expectedBalance := 10.0 * 158.50
	if update.NewBalance != expectedBalance {
		t.Errorf("expected new balance %f, got %f", expectedBalance, update.NewBalance)
	}

	// Verify repo was called
	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updated))
	}
	if *repo.updated[0].QuotaPrice != 158.50 {
		t.Errorf("expected updated quota price 158.50, got %f", *repo.updated[0].QuotaPrice)
	}
}

func TestStockSync_SkipsAccountsWithoutQuotas(t *testing.T) {
	parentID := "clear-id"
	repo := &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{
				ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "profile-1", Currency: "BRL",
			},
			{
				ID: "no-quotas", Name: "PETR4", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "profile-1", Currency: "BRL",
				LinkedAccountID: &parentID,
				// No NumberOfQuotas set
			},
		},
	}

	brapiClient := &mockBrapiClient{
		quotes: []brapi.QuoteResult{
			{Symbol: "PETR4", RegularMarketPrice: 38.75},
		},
	}

	uc := NewStockSyncUseCase(brapiClient, repo)
	result, err := uc.Execute("profile-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.UpdatedAccounts) != 0 {
		t.Errorf("expected 0 updated accounts, got %d", len(result.UpdatedAccounts))
	}
}

func TestStockSync_MultipleAssets(t *testing.T) {
	parentID := "clear-id"
	fiiType := bankaccount.InvestmentTypeFII
	stockType := bankaccount.InvestmentTypeStocks
	q1 := 10.0
	q2 := 100.0
	p1 := 150.0
	p2 := 35.0

	repo := &mockAccountRepo{
		accounts: []*bankaccount.BankAccount{
			{ID: parentID, Name: "Clear", Type: bankaccount.AccountTypeInvestment, ProfileID: "p1"},
			{
				ID: "hglg", Name: "HGLG11", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "p1", LinkedAccountID: &parentID, InvestmentType: &fiiType,
				NumberOfQuotas: &q1, QuotaPrice: &p1, CurrentBalance: 1500,
			},
			{
				ID: "petr", Name: "PETR4", Type: bankaccount.AccountTypeInvestment,
				ProfileID: "p1", LinkedAccountID: &parentID, InvestmentType: &stockType,
				NumberOfQuotas: &q2, QuotaPrice: &p2, CurrentBalance: 3500,
			},
		},
	}

	brapiClient := &mockBrapiClient{
		quotes: []brapi.QuoteResult{
			{Symbol: "HGLG11", RegularMarketPrice: 160.0},
			{Symbol: "PETR4", RegularMarketPrice: 40.0},
		},
	}

	uc := NewStockSyncUseCase(brapiClient, repo)
	result, err := uc.Execute("p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.UpdatedAccounts) != 2 {
		t.Fatalf("expected 2 updated accounts, got %d", len(result.UpdatedAccounts))
	}

	// Check HGLG11
	if result.UpdatedAccounts[0].NewBalance != 1600.0 {
		t.Errorf("HGLG11: expected balance 1600.0, got %f", result.UpdatedAccounts[0].NewBalance)
	}
	// Check PETR4
	if result.UpdatedAccounts[1].NewBalance != 4000.0 {
		t.Errorf("PETR4: expected balance 4000.0, got %f", result.UpdatedAccounts[1].NewBalance)
	}
}

func TestDetectTicker_FromAccountName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"HGLG11", "HGLG11"},
		{"PETR4", "PETR4"},
		{"KNCR11", "KNCR11"},
		{"CDB Banco Arbi", ""},
		{"Clear", ""},
		{"Caixinha Nubank Pessoal", ""},
		{"Binance - Solana (SOL)", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectTicker(tt.name)
			if result != tt.expected {
				t.Errorf("detectTicker(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
