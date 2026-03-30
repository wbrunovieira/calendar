package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	sa "github.com/brunovieira/calendar-finances/internal/domain/strategyallocation"
	"github.com/gorilla/mux"
)

type AllocationHandlers struct {
	allocationRepo sa.Repository
	detectUC       *usecases.DetectPendingAllocationsUseCase
	approveUC      *usecases.ApproveAllocationUseCase
	declineUC      *usecases.DeclineAllocationUseCase
	summaryUC      *usecases.GetStrategySummaryUseCase
}

func NewAllocationHandlers(
	repo sa.Repository,
	detectUC *usecases.DetectPendingAllocationsUseCase,
	approveUC *usecases.ApproveAllocationUseCase,
	declineUC *usecases.DeclineAllocationUseCase,
	summaryUC *usecases.GetStrategySummaryUseCase,
) *AllocationHandlers {
	return &AllocationHandlers{
		allocationRepo: repo,
		detectUC:       detectUC,
		approveUC:      approveUC,
		declineUC:      declineUC,
		summaryUC:      summaryUC,
	}
}

// Detect handles POST /api/v1/allocations/detect
func (h *AllocationHandlers) Detect(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	result, err := h.detectUC.Execute(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": result})
}

// ListPending handles GET /api/v1/allocations/pending
func (h *AllocationHandlers) ListPending(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	pending, err := h.allocationRepo.FindPending(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  pending,
		"total": len(pending),
	})
}

// List handles GET /api/v1/allocations
func (h *AllocationHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	strategy := r.URL.Query().Get("strategy")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	var allocations []*sa.StrategyAllocation
	var err error
	if strategy != "" {
		allocations, err = h.allocationRepo.FindByStrategy(profileID, strategy)
	} else {
		allocations, err = h.allocationRepo.FindByProfileID(profileID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  allocations,
		"total": len(allocations),
	})
}

// Approve handles POST /api/v1/allocations/{id}/approve
func (h *AllocationHandlers) Approve(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	allocation, err := h.approveUC.Execute(id)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrAllocationNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": allocation})
}

// Decline handles POST /api/v1/allocations/{id}/decline
func (h *AllocationHandlers) Decline(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	allocation, err := h.declineUC.Execute(id)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrAllocationNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": allocation})
}

// Summary handles GET /api/v1/allocations/summary
func (h *AllocationHandlers) Summary(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	strategy := r.URL.Query().Get("strategy")
	if profileID == "" || strategy == "" {
		http.Error(w, "profileId and strategy are required", http.StatusBadRequest)
		return
	}

	summary, err := h.summaryUC.Execute(profileID, strategy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": summary})
}
