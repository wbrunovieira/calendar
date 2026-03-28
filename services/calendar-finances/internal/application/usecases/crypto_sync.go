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
	// Fetch prices from Binance
	syncData, err := uc.binanceClient.FetchPrices()
	if err != nil {
		return nil, err
	}

	result := &CryptoSyncResult{
		Prices: syncData.Prices,
		UsdBrl: syncData.UsdBrl,
	}

	// Build price map: asset -> price in USD
	priceMap := map[string]float64{}
	for _, p := range syncData.Prices {
		priceMap[p.Symbol] = p.PriceUSD
	}

	// Find crypto accounts (EXCHANGE/WALLET type or linked to one) and update quotaPrice
	accounts, err := uc.accountRepo.FindByProfileID(profileID)
	if err != nil {
		return result, nil // Return prices even if account update fails
	}

	for _, account := range accounts {
		if account.NumberOfQuotas == nil || *account.NumberOfQuotas <= 0 {
			continue
		}
		if account.InvestmentType == nil {
			continue
		}
		if *account.InvestmentType != bankaccount.InvestmentTypeCrypto {
			continue
		}

		// Detect asset from account name (e.g., "Binance - Solana (SOL)" -> "SOL")
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

		// Update quota price and balance
		account.QuotaPrice = &newPriceUSD
		newBalance := *account.NumberOfQuotas * newPriceUSD
		account.CurrentBalance = newBalance
		account.UpdatedAt = time.Now()

		if err := uc.accountRepo.Update(account); err != nil {
			continue
		}

		result.UpdatedAccounts = append(result.UpdatedAccounts, CryptoAccountUpdate{
			AccountID:   account.ID,
			AccountName: account.Name,
			Asset:       asset,
			Quotas:      *account.NumberOfQuotas,
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
