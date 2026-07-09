package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

type StockHandlers struct {
	syncUC     *usecases.StockSyncUseCase
	dividendUC *usecases.DividendSyncUseCase
}

func NewStockHandlers(syncUC *usecases.StockSyncUseCase) *StockHandlers {
	return &StockHandlers{syncUC: syncUC}
}

func (h *StockHandlers) SetDividendUseCase(uc *usecases.DividendSyncUseCase) {
	h.dividendUC = uc
}

// SyncPrices fetches current prices from brapi.dev and updates stock/FII account balances
func (h *StockHandlers) SyncPrices(w http.ResponseWriter, r *http.Request) {
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

// SyncDividends checks for new dividends and creates income transactions
func (h *StockHandlers) SyncDividends(w http.ResponseWriter, r *http.Request) {
	if h.dividendUC == nil {
		http.Error(w, "dividend sync not configured", http.StatusServiceUnavailable)
		return
	}

	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	// Default: check dividends from the last 90 days; override with ?since=YYYY-MM-DD (backfill)
	since := time.Now().AddDate(0, -3, 0)
	if s := r.URL.Query().Get("since"); s != "" {
		parsed, err := time.Parse("2006-01-02", s)
		if err != nil {
			http.Error(w, "invalid since date, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		since = parsed
	}

	result, err := h.dividendUC.Execute(profileID, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}
