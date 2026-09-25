package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

type InvariantsHandlers struct {
	reattachUC     *usecases.ReattachOrphanInvoiceLinesUseCase
	checkUseCase   *usecases.CheckInvariantsUseCase
	rebuildUseCase *usecases.RebuildInvoiceCyclesUseCase
	applyCycleUC   *usecases.ApplyInvoiceCyclePlanUseCase
}

func NewInvariantsHandlers(
	checkUC *usecases.CheckInvariantsUseCase,
	rebuildUC *usecases.RebuildInvoiceCyclesUseCase,
	reattachUC *usecases.ReattachOrphanInvoiceLinesUseCase,
	applyCycleUC *usecases.ApplyInvoiceCyclePlanUseCase,
) *InvariantsHandlers {
	return &InvariantsHandlers{checkUseCase: checkUC, rebuildUseCase: rebuildUC,
		reattachUC: reattachUC, applyCycleUC: applyCycleUC}
}

// ApplyInvoiceCyclePlan handles
// POST /api/v1/bank-accounts/{id}/invoice-cycles/apply.
//
// The GET plan is the dry run, and it stays free of side effects on purpose — it can
// be read by anyone at any time. This writes, and only the cycles it was told to:
// each is named by its reference date, repeated as ?referenceDate=YYYY-MM-DD.
func (h *InvariantsHandlers) ApplyInvoiceCyclePlan(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var approve []string
	for _, raw := range r.URL.Query()["referenceDate"] {
		if raw != "" {
			approve = append(approve, raw)
		}
	}

	report, err := h.applyCycleUC.Execute(id, approve)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, map[string]any{"data": report})
}

// ReattachOrphanInvoiceLines handles
// POST /api/v1/bank-accounts/{id}/invoice-cycles/reattach.
//
// DRY RUN BY DEFAULT. It walks real money, so it reports what it would do and
// writes nothing unless ?apply=true is passed. The report is the same either way,
// which is what makes the dry run worth reading.
func (h *InvariantsHandlers) ReattachOrphanInvoiceLines(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	apply := r.URL.Query().Get("apply") == "true"

	// The rows to write are named one by one. Applying without them is refused by
	// the use case, not by this handler — the rule belongs where the damage is.
	var only []string
	for _, raw := range r.URL.Query()["transactionId"] {
		if raw != "" {
			only = append(only, raw)
		}
	}

	report, err := h.reattachUC.Execute(id, apply, only)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, map[string]any{"data": report})
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
