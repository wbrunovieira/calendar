package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type BudgetHandlers struct {
	service *usecases.BudgetTargetsService
}

func NewBudgetHandlers(service *usecases.BudgetTargetsService) *BudgetHandlers {
	return &BudgetHandlers{service: service}
}

// List handles GET /api/v1/budgets?profileId=...
func (h *BudgetHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	items, err := h.service.List(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"data":  items,
		"total": len(items),
	})
}

// Summary handles GET /api/v1/budgets/summary?profileId=...&period=YYYY-MM
func (h *BudgetHandlers) Summary(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}
	period := r.URL.Query().Get("period")

	items, err := h.service.Summary(profileID, period)
	if err != nil {
		status := http.StatusBadRequest
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": items})
}

// Create handles POST /api/v1/budgets
func (h *BudgetHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.BudgetTargetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entity, err := h.service.Create(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, map[string]any{"data": entity})
}

// Update handles PUT /api/v1/budgets/{id}
func (h *BudgetHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.BudgetTargetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entity, err := h.service.Update(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrBudgetTargetNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": entity})
}

// Delete handles DELETE /api/v1/budgets/{id}
func (h *BudgetHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.service.Delete(id); err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrBudgetTargetNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
