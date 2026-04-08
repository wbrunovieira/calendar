package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type CapitalContributionHandlers struct {
	createUC  *usecases.CreateCapitalContributionUseCase
	listUC    *usecases.ListCapitalContributionsUseCase
	getUC     *usecases.GetCapitalContributionUseCase
	updateUC  *usecases.UpdateCapitalContributionUseCase
	deleteUC  *usecases.DeleteCapitalContributionUseCase
	summaryUC *usecases.GetCapitalContributionSummaryUseCase
}

func NewCapitalContributionHandlers(
	createUC *usecases.CreateCapitalContributionUseCase,
	listUC *usecases.ListCapitalContributionsUseCase,
	getUC *usecases.GetCapitalContributionUseCase,
	updateUC *usecases.UpdateCapitalContributionUseCase,
	deleteUC *usecases.DeleteCapitalContributionUseCase,
	summaryUC *usecases.GetCapitalContributionSummaryUseCase,
) *CapitalContributionHandlers {
	return &CapitalContributionHandlers{
		createUC: createUC, listUC: listUC, getUC: getUC,
		updateUC: updateUC, deleteUC: deleteUC, summaryUC: summaryUC,
	}
}

// List handles GET /api/v1/capital-contributions?profileId=...
func (h *CapitalContributionHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}
	items, err := h.listUC.Execute(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"data": items, "total": len(items)})
}

// Get handles GET /api/v1/capital-contributions/{id}
func (h *CapitalContributionHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := h.getUC.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Create handles POST /api/v1/capital-contributions
func (h *CapitalContributionHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateCapitalContributionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.createUC.Execute(input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrProfileNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusCreated)
	respondJSON(w, map[string]any{"data": item})
}

// Update handles PUT /api/v1/capital-contributions/{id}
func (h *CapitalContributionHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.UpdateCapitalContributionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.updateUC.Execute(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrCapitalContributionNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Delete handles DELETE /api/v1/capital-contributions/{id}
func (h *CapitalContributionHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.deleteUC.Execute(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Summary handles GET /api/v1/capital-contributions/summary?profileId=...
func (h *CapitalContributionHandlers) Summary(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}
	summary, err := h.summaryUC.Execute(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"data": summary})
}
