package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

type CryptoHandlers struct {
	syncUC *usecases.CryptoSyncUseCase
}

func NewCryptoHandlers(syncUC *usecases.CryptoSyncUseCase) *CryptoHandlers {
	return &CryptoHandlers{syncUC: syncUC}
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
