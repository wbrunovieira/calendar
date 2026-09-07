package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

type InvariantsHandlers struct {
	checkUseCase   *usecases.CheckInvariantsUseCase
	rebuildUseCase *usecases.RebuildInvoiceCyclesUseCase
}

func NewInvariantsHandlers(
	checkUC *usecases.CheckInvariantsUseCase,
	rebuildUC *usecases.RebuildInvoiceCyclesUseCase,
) *InvariantsHandlers {
	return &InvariantsHandlers{checkUseCase: checkUC, rebuildUseCase: rebuildUC}
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

// InvoiceCyclePlan handles GET /api/v1/bank-accounts/{id}/invoice-cycles/plan.
//
// It answers what a card's invoice cycles should look like and how the stored ones
// differ. It writes nothing: the repair is a separate, deliberate act, and a plan
// that repaired as it diagnosed would be impossible to review before it ran.
//
// 200 means the card's cycles already match its closing and due day; 409 means there
// is damage to look at.
func (h *InvariantsHandlers) InvoiceCyclePlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.rebuildUseCase.Plan(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(plan.Actions) > 0 {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": plan})
}
