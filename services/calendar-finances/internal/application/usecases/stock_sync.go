package usecases

import (
	"regexp"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/brapi"
)

// BrapiClient abstracts the brapi.dev API for testability.
type BrapiClient interface {
	GetQuotes(tickers ...string) ([]brapi.QuoteResult, error)
}

type StockSyncUseCase struct {
	brapiClient BrapiClient
	accountRepo bankaccount.Repository
}

type StockSyncResult struct {
	UpdatedAccounts []StockAccountUpdate `json:"updatedAccounts"`
}

type StockAccountUpdate struct {
	AccountID  string  `json:"accountId"`
	Ticker     string  `json:"ticker"`
	Quotas     float64 `json:"quotas"`
	OldPrice   float64 `json:"oldPrice"`
	NewPrice   float64 `json:"newPrice"`
	OldBalance float64 `json:"oldBalance"`
	NewBalance float64 `json:"newBalance"`
}

func NewStockSyncUseCase(client BrapiClient, accountRepo bankaccount.Repository) *StockSyncUseCase {
	return &StockSyncUseCase{
		brapiClient: client,
		accountRepo: accountRepo,
	}
}

func (uc *StockSyncUseCase) Execute(profileID string) (*StockSyncResult, error) {
	accounts, err := uc.accountRepo.FindByProfileID(profileID)
	if err != nil {
		return nil, err
	}

	// Collect tickers from sub-accounts that have quotas
	tickerMap := map[string]*bankaccount.BankAccount{}
	var tickers []string
	for _, acc := range accounts {
		if acc.LinkedAccountID == nil || !acc.HasQuotas() {
			continue
		}
		ticker := detectTicker(acc.Name)
		if ticker == "" {
			continue
		}
		tickerMap[ticker] = acc
		tickers = append(tickers, ticker)
	}

	if len(tickers) == 0 {
		return &StockSyncResult{}, nil
	}

	quotes, err := uc.brapiClient.GetQuotes(tickers...)
	if err != nil {
		return nil, err
	}

	// Build price map
	priceMap := map[string]float64{}
	for _, q := range quotes {
		priceMap[q.Symbol] = q.RegularMarketPrice
	}

	result := &StockSyncResult{}

	for _, ticker := range tickers {
		newPrice, ok := priceMap[ticker]
		if !ok {
			continue
		}

		acc := tickerMap[ticker]
		oldPrice := float64(0)
		if acc.QuotaPrice != nil {
			oldPrice = *acc.QuotaPrice
		}
		oldBalance := acc.CurrentBalance
		quotas := *acc.NumberOfQuotas

		acc.QuotaPrice = &newPrice
		acc.CurrentBalance = quotas * newPrice
		acc.UpdatedAt = time.Now()

		if err := uc.accountRepo.Update(acc); err != nil {
			continue
		}

		result.UpdatedAccounts = append(result.UpdatedAccounts, StockAccountUpdate{
			AccountID:  acc.ID,
			Ticker:     ticker,
			Quotas:     quotas,
			OldPrice:   oldPrice,
			NewPrice:   newPrice,
			OldBalance: oldBalance,
			NewBalance: acc.CurrentBalance,
		})
	}

	return result, nil
}

// tickerRegex matches B3 ticker format: 4 uppercase letters + 1-2 digits (e.g., PETR4, HGLG11)
var tickerRegex = regexp.MustCompile(`^[A-Z]{4}\d{1,2}$`)

// detectTicker returns the ticker if the account name IS a valid B3 ticker, empty otherwise.
func detectTicker(name string) string {
	name = strings.TrimSpace(name)
	if tickerRegex.MatchString(name) {
		return name
	}
	return ""
}
