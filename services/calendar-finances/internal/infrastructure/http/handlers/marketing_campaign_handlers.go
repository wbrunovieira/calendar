package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type MarketingCampaignHandlers struct {
	createUC     *usecases.CreateCampaignUseCase
	listUC       *usecases.ListCampaignsUseCase
	getUC        *usecases.GetCampaignUseCase
	updateUC     *usecases.UpdateCampaignUseCase
	deleteUC     *usecases.DeleteCampaignUseCase
	getMetricsUC *usecases.GetCampaignWithMetricsUseCase
}

func NewMarketingCampaignHandlers(
	createUC *usecases.CreateCampaignUseCase,
	listUC *usecases.ListCampaignsUseCase,
	getUC *usecases.GetCampaignUseCase,
	updateUC *usecases.UpdateCampaignUseCase,
	deleteUC *usecases.DeleteCampaignUseCase,
	getMetricsUC *usecases.GetCampaignWithMetricsUseCase,
) *MarketingCampaignHandlers {
	return &MarketingCampaignHandlers{
		createUC: createUC, listUC: listUC, getUC: getUC,
		updateUC: updateUC, deleteUC: deleteUC, getMetricsUC: getMetricsUC,
	}
}

// List handles GET /api/v1/marketing-campaigns?profileId=...
func (h *MarketingCampaignHandlers) List(w http.ResponseWriter, r *http.Request) {
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

// Get handles GET /api/v1/marketing-campaigns/{id}
func (h *MarketingCampaignHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := h.getUC.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Create handles POST /api/v1/marketing-campaigns
func (h *MarketingCampaignHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateCampaignInput
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

// Update handles PUT /api/v1/marketing-campaigns/{id}
func (h *MarketingCampaignHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.UpdateCampaignInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.updateUC.Execute(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrMarketingCampaignNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	respondJSON(w, map[string]any{"data": item})
}

// Delete handles DELETE /api/v1/marketing-campaigns/{id}
func (h *MarketingCampaignHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.deleteUC.Execute(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetWithMetrics handles GET /api/v1/marketing-campaigns/{id}/metrics
func (h *MarketingCampaignHandlers) GetWithMetrics(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	metrics, err := h.getMetricsUC.Execute(id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrMarketingCampaignNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	respondJSON(w, map[string]any{"data": metrics})
}
