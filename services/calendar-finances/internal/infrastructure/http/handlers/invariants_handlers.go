package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

// BankReconciler compares the ledger against the bank. Attached after
// construction, like the other cross-cutting collaborators; a nil one answers
// 503 rather than an empty report, because "no divergences" and "nothing was
// checked" must never look alike.
type BankReconciler interface {
	Execute() (usecases.ReconciliationReport, error)
}

type InvariantsHandlers struct {
	reconciler     BankReconciler
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

// SetBankReconciler wires the comparison against the bank.
func (h *InvariantsHandlers) SetBankReconciler(r BankReconciler) { h.reconciler = r }

// BankReconciliation compares every connected account against the bank.
//
// Separate from /health/invariants on purpose: that one checks the ledger
// against ITSELF, and a charge filed on the wrong billing cycle is perfectly
// consistent there. This is the only thing in the system that looks outward.
func (h *InvariantsHandlers) BankReconciliation(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		// 501, NOT 503. They are different problems with different owners: this
		// one means MCP_AI_API_KEY was never set, and reporting it as "the bank
		// is down" sends the investigation to the bank every single day while
		// the feature sits switched off.
		http.Error(w, "bank reconciliation is not configured on this server: MCP_AI_API_KEY is not set", http.StatusNotImplemented)
		return
	}

	report, err := h.reconciler.Execute()
	if err != nil {
		// 503, not 500: the ledger is fine, the bank could not be reached. And
		// never an empty report — a silent clean answer during an outage is the
		// failure this whole check exists to prevent.
		http.Error(w, "could not reach the bank: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	if !report.OK {
		// 409, matching /health/invariants, because the production health-check
		// cron has no jq and reads the STATUS CODE. A 200 carrying a divergence
		// in its body is a finding nobody is told about — which would make this
		// endpoint decorative on the one consumer it was built for.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": report})
		return
	}

	respondJSON(w, map[string]any{"data": report})
}
