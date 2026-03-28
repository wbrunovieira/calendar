package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/binance"
)

type CryptoSyncUseCase struct {
	binanceClient *binance.Client
	accountRepo   bankaccount.Repository
}

type CryptoSyncResult struct {
	Prices         []binance.CryptoPrice  `json:"prices"`
	UsdBrl         float64                `json:"usdBrl"`
	UpdatedAccounts []CryptoAccountUpdate `json:"updatedAccounts"`
}

type CryptoAccountUpdate struct {
	AccountID   string  `json:"accountId"`
	AccountName string  `json:"accountName"`
	Asset       string  `json:"asset"`
	Quotas      float64 `json:"quotas"`
	OldPrice    float64 `json:"oldPrice"`
	NewPrice    float64 `json:"newPrice"`
	OldBalance  float64 `json:"oldBalance"`
	NewBalance  float64 `json:"newBalance"`
	Currency    string  `json:"currency"`
}

func NewCryptoSyncUseCase(client *binance.Client, accountRepo bankaccount.Repository) *CryptoSyncUseCase {
	return &CryptoSyncUseCase{
		binanceClient: client,
		accountRepo:   accountRepo,
	}
}

func (uc *CryptoSyncUseCase) Execute(profileID string) (*CryptoSyncResult, error) {
	// Find crypto accounts first to know which symbols to fetch
	accounts, err := uc.accountRepo.FindByProfileID(profileID)
	if err != nil {
		return nil, err
	}

	// Collect all crypto asset symbols from sub-accounts
	var symbols []string
	symbolSet := map[string]bool{}
	for _, account := range accounts {
		asset := detectAsset(account.Name)
		if asset != "" && !symbolSet[asset] {
			symbolSet[asset] = true
			symbols = append(symbols, asset)
		}
	}

	// Fetch prices from Binance for all detected symbols
	syncData, err := uc.binanceClient.FetchPrices(symbols...)
	if err != nil {
		return nil, err
	}

	result := &CryptoSyncResult{
		Prices: syncData.Prices,
		UsdBrl: syncData.UsdBrl,
	}

	// Build price map: asset -> price in USD
	priceMap := map[string]float64{}
	priceBrlMap := map[string]float64{}
	for _, p := range syncData.Prices {
		priceMap[p.Symbol] = p.PriceUSD
		priceBrlMap[p.Symbol] = p.PriceBRL
	}

	for _, account := range accounts {
		asset := detectAsset(account.Name)
		if asset == "" {
			continue
		}

		newPriceUSD, ok := priceMap[asset]
		if !ok {
			continue
		}

		oldPrice := float64(0)
		if account.QuotaPrice != nil {
			oldPrice = *account.QuotaPrice
		}
		oldBalance := account.CurrentBalance

		// Update quota price
		account.QuotaPrice = &newPriceUSD

		// Calculate balance in BRL: quotas * priceBRL
		quotas := float64(0)
		if account.NumberOfQuotas != nil {
			quotas = *account.NumberOfQuotas
		}
		newBalance := quotas * priceBrlMap[asset]
		account.CurrentBalance = newBalance
		account.UpdatedAt = time.Now()

		if err := uc.accountRepo.Update(account); err != nil {
			continue
		}

		result.UpdatedAccounts = append(result.UpdatedAccounts, CryptoAccountUpdate{
			AccountID:   account.ID,
			AccountName: account.Name,
			Asset:       asset,
			Quotas:      quotas,
			OldPrice:    oldPrice,
			NewPrice:    newPriceUSD,
			OldBalance:  oldBalance,
			NewBalance:  newBalance,
			Currency:    account.Currency,
		})
	}

	return result, nil
}

// detectAsset extracts the crypto asset symbol from account name
// Examples: "Binance - Solana (SOL)" -> "SOL", "Bitcoin (BTC)" -> "BTC"
func detectAsset(name string) string {
	// Look for pattern "(SYMBOL)"
	start := strings.LastIndex(name, "(")
	end := strings.LastIndex(name, ")")
	if start >= 0 && end > start {
		return strings.TrimSpace(name[start+1 : end])
	}
	return ""
}
