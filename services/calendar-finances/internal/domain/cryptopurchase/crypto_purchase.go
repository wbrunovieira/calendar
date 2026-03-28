package cryptopurchase

import (
	"errors"
	"time"
)

type CryptoPurchase struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transactionId"`
	BankAccountID string    `json:"bankAccountId"`
	ProfileID     string    `json:"profileId"`
	Asset         string    `json:"asset"`
	Quantity      float64   `json:"quantity"`
	PriceUSD      float64   `json:"priceUsd"`
	ExchangeRate  float64   `json:"exchangeRate"` // USD/BRL at purchase
	InvestedBRL   float64   `json:"investedBrl"`
	InvestedUSD   float64   `json:"investedUsd"`
	OccurredOn    time.Time `json:"occurredOn"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type PurchaseWithGains struct {
	CryptoPurchase
	CurrentPriceUSD      float64 `json:"currentPriceUsd"`
	CurrentExchangeRate  float64 `json:"currentExchangeRate"`
	CurrentValueUSD      float64 `json:"currentValueUsd"`
	CurrentValueBRL      float64 `json:"currentValueBrl"`
	GainCryptoUSD        float64 `json:"gainCryptoUsd"`        // gain from crypto price change in USD
	GainCryptoPercent    float64 `json:"gainCryptoPercent"`
	GainExchangeBRL      float64 `json:"gainExchangeBrl"`      // gain from USD/BRL change
	GainExchangePercent  float64 `json:"gainExchangePercent"`
	GainTotalBRL         float64 `json:"gainTotalBrl"`         // total gain in BRL
	GainTotalPercent     float64 `json:"gainTotalPercent"`
}

var (
	ErrInvalidAsset    = errors.New("asset is required")
	ErrInvalidQuantity = errors.New("quantity must be positive")
	ErrInvalidPrice    = errors.New("price must be positive")
	ErrInvalidRate     = errors.New("exchange rate must be positive")
)

func NewCryptoPurchase(
	transactionID, bankAccountID, profileID, asset string,
	quantity, priceUSD, exchangeRate float64,
	occurredOn time.Time,
	notes string,
) (*CryptoPurchase, error) {
	if asset == "" {
		return nil, ErrInvalidAsset
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	if priceUSD <= 0 {
		return nil, ErrInvalidPrice
	}
	if exchangeRate <= 0 {
		return nil, ErrInvalidRate
	}

	investedUSD := quantity * priceUSD
	investedBRL := investedUSD * exchangeRate

	return &CryptoPurchase{
		TransactionID: transactionID,
		BankAccountID: bankAccountID,
		ProfileID:     profileID,
		Asset:         asset,
		Quantity:      quantity,
		PriceUSD:      priceUSD,
		ExchangeRate:  exchangeRate,
		InvestedBRL:   investedBRL,
		InvestedUSD:   investedUSD,
		OccurredOn:    occurredOn,
		Notes:         notes,
		CreatedAt:     time.Now(),
	}, nil
}

// CalculateGains computes current gains given live prices
func (cp *CryptoPurchase) CalculateGains(currentPriceUSD, currentExchangeRate float64) PurchaseWithGains {
	currentValueUSD := cp.Quantity * currentPriceUSD
	currentValueBRL := currentValueUSD * currentExchangeRate

	gainCryptoUSD := cp.Quantity * (currentPriceUSD - cp.PriceUSD)
	gainCryptoPercent := 0.0
	if cp.PriceUSD > 0 {
		gainCryptoPercent = ((currentPriceUSD - cp.PriceUSD) / cp.PriceUSD) * 100
	}

	// Exchange gain: how much the USD/BRL change affected the investment
	gainExchangeBRL := cp.InvestedUSD * (currentExchangeRate - cp.ExchangeRate)
	gainExchangePercent := 0.0
	if cp.ExchangeRate > 0 {
		gainExchangePercent = ((currentExchangeRate - cp.ExchangeRate) / cp.ExchangeRate) * 100
	}

	gainTotalBRL := currentValueBRL - cp.InvestedBRL
	gainTotalPercent := 0.0
	if cp.InvestedBRL > 0 {
		gainTotalPercent = (gainTotalBRL / cp.InvestedBRL) * 100
	}

	return PurchaseWithGains{
		CryptoPurchase:      *cp,
		CurrentPriceUSD:     currentPriceUSD,
		CurrentExchangeRate: currentExchangeRate,
		CurrentValueUSD:     currentValueUSD,
		CurrentValueBRL:     currentValueBRL,
		GainCryptoUSD:       gainCryptoUSD,
		GainCryptoPercent:   gainCryptoPercent,
		GainExchangeBRL:     gainExchangeBRL,
		GainExchangePercent: gainExchangePercent,
		GainTotalBRL:        gainTotalBRL,
		GainTotalPercent:    gainTotalPercent,
	}
}
