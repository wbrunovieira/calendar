package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type CompanyAssetHandlers struct {
	createUC *usecases.CreateCompanyAssetUseCase
	listUC   *usecases.ListCompanyAssetsUseCase
	getUC    *usecases.GetCompanyAssetUseCase
	updateUC *usecases.UpdateCompanyAssetUseCase
	deleteUC *usecases.DeleteCompanyAssetUseCase
}

func NewCompanyAssetHandlers(
	createUC *usecases.CreateCompanyAssetUseCase,
	listUC *usecases.ListCompanyAssetsUseCase,
	getUC *usecases.GetCompanyAssetUseCase,
	updateUC *usecases.UpdateCompanyAssetUseCase,
	deleteUC *usecases.DeleteCompanyAssetUseCase,
) *CompanyAssetHandlers {
	return &CompanyAssetHandlers{
		createUC: createUC, listUC: listUC, getUC: getUC,
		updateUC: updateUC, deleteUC: deleteUC,
	}
}

// List handles GET /api/v1/company-assets?profileId=...
func (h *CompanyAssetHandlers) List(w http.ResponseWriter, r *http.Request) {
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

// Get handles GET /api/v1/company-assets/{id}
func (h *CompanyAssetHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := h.getUC.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Create handles POST /api/v1/company-assets
func (h *CompanyAssetHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateCompanyAssetInput
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

// Update handles PUT /api/v1/company-assets/{id}
func (h *CompanyAssetHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.UpdateCompanyAssetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.updateUC.Execute(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrCompanyAssetNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Delete handles DELETE /api/v1/company-assets/{id}
func (h *CompanyAssetHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.deleteUC.Execute(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
