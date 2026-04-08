package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type CostCenterHandlers struct {
	createUC *usecases.CreateCostCenterUseCase
	listUC   *usecases.ListCostCentersUseCase
	getUC    *usecases.GetCostCenterUseCase
	updateUC *usecases.UpdateCostCenterUseCase
	deleteUC *usecases.DeleteCostCenterUseCase
}

func NewCostCenterHandlers(
	createUC *usecases.CreateCostCenterUseCase,
	listUC *usecases.ListCostCentersUseCase,
	getUC *usecases.GetCostCenterUseCase,
	updateUC *usecases.UpdateCostCenterUseCase,
	deleteUC *usecases.DeleteCostCenterUseCase,
) *CostCenterHandlers {
	return &CostCenterHandlers{
		createUC: createUC, listUC: listUC, getUC: getUC,
		updateUC: updateUC, deleteUC: deleteUC,
	}
}

// List handles GET /api/v1/cost-centers?profileId=...
func (h *CostCenterHandlers) List(w http.ResponseWriter, r *http.Request) {
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

// Get handles GET /api/v1/cost-centers/{id}
func (h *CostCenterHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := h.getUC.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Create handles POST /api/v1/cost-centers
func (h *CostCenterHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateCostCenterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.createUC.Execute(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	respondJSON(w, map[string]any{"data": item})
}

// Update handles PUT /api/v1/cost-centers/{id}
func (h *CostCenterHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.UpdateCostCenterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.updateUC.Execute(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrCostCenterNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Delete handles DELETE /api/v1/cost-centers/{id}
func (h *CostCenterHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.deleteUC.Execute(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
