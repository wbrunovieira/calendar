package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/cryptopurchase"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/binance"
)

type CryptoPurchaseHandlers struct {
	repo          cryptopurchase.Repository
	binanceClient *binance.Client
}

func NewCryptoPurchaseHandlers(repo cryptopurchase.Repository, client *binance.Client) *CryptoPurchaseHandlers {
	return &CryptoPurchaseHandlers{repo: repo, binanceClient: client}
}

type CreateCryptoPurchaseInput struct {
	TransactionID string  `json:"transactionId"`
	BankAccountID string  `json:"bankAccountId"`
	ProfileID     string  `json:"profileId"`
	Asset         string  `json:"asset"`
	Quantity      float64 `json:"quantity"`
	PriceUSD      float64 `json:"priceUsd"`
	ExchangeRate  float64 `json:"exchangeRate"`
	OccurredOn    string  `json:"occurredOn"`
	Notes         string  `json:"notes"`
}

func (h *CryptoPurchaseHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateCryptoPurchaseInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	occurredOn, err := time.Parse("2006-01-02", input.OccurredOn)
	if err != nil {
		http.Error(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	cp, err := cryptopurchase.NewCryptoPurchase(
		input.TransactionID, input.BankAccountID, input.ProfileID, input.Asset,
		input.Quantity, input.PriceUSD, input.ExchangeRate, occurredOn, input.Notes,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(cp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": cp})
}

func (h *CryptoPurchaseHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	asset := r.URL.Query().Get("asset")

	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	var purchases []*cryptopurchase.CryptoPurchase
	var err error
	if asset != "" {
		purchases, err = h.repo.FindByAsset(profileID, asset)
	} else {
		purchases, err = h.repo.FindByProfileID(profileID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch current prices to calculate gains
	var results []cryptopurchase.PurchaseWithGains
	priceCache := map[string]float64{}
	var usdBrl float64

	if h.binanceClient != nil {
		usdBrl, _ = h.binanceClient.GetTickerPrice("USDTBRL")
		for _, cp := range purchases {
			priceUSD, ok := priceCache[cp.Asset]
			if !ok {
				priceUSD, _ = h.binanceClient.GetTickerPrice(cp.Asset + "USDT")
				priceCache[cp.Asset] = priceUSD
			}
			if priceUSD > 0 && usdBrl > 0 {
				results = append(results, cp.CalculateGains(priceUSD, usdBrl))
			} else {
				results = append(results, cryptopurchase.PurchaseWithGains{CryptoPurchase: *cp})
			}
		}
	} else {
		for _, cp := range purchases {
			results = append(results, cryptopurchase.PurchaseWithGains{CryptoPurchase: *cp})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   results,
		"usdBrl": usdBrl,
		"total":  len(results),
	})
}
