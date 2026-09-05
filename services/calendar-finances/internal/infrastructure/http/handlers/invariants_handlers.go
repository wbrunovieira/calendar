package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

type InvariantsHandlers struct {
	checkUseCase *usecases.CheckInvariantsUseCase
}

func NewInvariantsHandlers(checkUC *usecases.CheckInvariantsUseCase) *InvariantsHandlers {
	return &InvariantsHandlers{checkUseCase: checkUC}
}

// Check handles GET /api/v1/health/invariants.
//
// It reports, and only reports: a drift is a transaction to hunt down, never a
// number to overwrite. The status code carries the verdict — 200 when every
// ledger agrees, 409 when one does not — so the production health-check cron can
// alert on it without parsing the body.
func (h *InvariantsHandlers) Check(w http.ResponseWriter, r *http.Request) {
	result, err := h.checkUseCase.Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !result.OK {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}
