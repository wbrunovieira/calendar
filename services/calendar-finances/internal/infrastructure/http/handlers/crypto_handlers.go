package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

type CryptoHandlers struct {
	syncUC       *usecases.CryptoSyncUseCase
	syncTradesUC *usecases.SyncTradesUseCase
}

func NewCryptoHandlers(syncUC *usecases.CryptoSyncUseCase) *CryptoHandlers {
	return &CryptoHandlers{syncUC: syncUC}
}

func (h *CryptoHandlers) SetSyncTradesUseCase(uc *usecases.SyncTradesUseCase) {
	h.syncTradesUC = uc
}

// Sync fetches current prices from Binance and updates crypto account balances
func (h *CryptoHandlers) Sync(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	result, err := h.syncUC.Execute(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}

// SyncTrades fetches trade history from Binance and creates transactions/purchases
func (h *CryptoHandlers) SyncTrades(w http.ResponseWriter, r *http.Request) {
	if h.syncTradesUC == nil {
		http.Error(w, "trade sync not configured", http.StatusServiceUnavailable)
		return
	}

	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	result, err := h.syncTradesUC.Execute(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}
