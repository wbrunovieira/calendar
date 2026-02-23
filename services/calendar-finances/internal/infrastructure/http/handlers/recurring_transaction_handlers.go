package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type RecurringTransactionHandlers struct {
	service   *usecases.RecurringTransactionsService
	processUC *usecases.ProcessRecurringTransactionsUseCase
}

func NewRecurringTransactionHandlers(
	service *usecases.RecurringTransactionsService,
	processUC *usecases.ProcessRecurringTransactionsUseCase,
) *RecurringTransactionHandlers {
	return &RecurringTransactionHandlers{service: service, processUC: processUC}
}

// List handles GET /api/v1/recurring-transactions?profileId=...
func (h *RecurringTransactionHandlers) List(w http.ResponseWriter, r *http.Request) {
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

// Create handles POST /api/v1/recurring-transactions
func (h *RecurringTransactionHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateRecurringTransactionInput
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

// Update handles PUT /api/v1/recurring-transactions/{id}
func (h *RecurringTransactionHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.CreateRecurringTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entity, err := h.service.Update(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrRecurringNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": entity})
}

// UpdateStatus handles PATCH /api/v1/recurring-transactions/{id}/status
func (h *RecurringTransactionHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entity, err := h.service.UpdateStatus(id, payload.Status)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrRecurringNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": entity})
}

// Delete handles DELETE /api/v1/recurring-transactions/{id}
func (h *RecurringTransactionHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.service.Delete(id); err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrRecurringNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Process handles POST /api/v1/recurring-transactions/process
func (h *RecurringTransactionHandlers) Process(w http.ResponseWriter, r *http.Request) {
	result, err := h.processUC.Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{"data": result})
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
