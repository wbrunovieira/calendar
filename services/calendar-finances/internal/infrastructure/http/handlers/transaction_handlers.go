package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type TransactionHandlers struct {
	createUseCase       *usecases.CreateTransactionUseCase
	listUseCase         *usecases.ListTransactionsUseCase
	getUseCase          *usecases.GetTransactionUseCase
	updateUseCase       *usecases.UpdateTransactionUseCase
	updateStatusUseCase *usecases.UpdateTransactionStatusUseCase
	deleteUseCase       *usecases.DeleteTransactionUseCase
}

func NewTransactionHandlers(
	createUC *usecases.CreateTransactionUseCase,
	listUC *usecases.ListTransactionsUseCase,
	getUC *usecases.GetTransactionUseCase,
	updateUC *usecases.UpdateTransactionUseCase,
	updateStatusUC *usecases.UpdateTransactionStatusUseCase,
	deleteUC *usecases.DeleteTransactionUseCase,
) *TransactionHandlers {
	return &TransactionHandlers{
		createUseCase:       createUC,
		listUseCase:         listUC,
		getUseCase:          getUC,
		updateUseCase:       updateUC,
		updateStatusUseCase: updateStatusUC,
		deleteUseCase:       deleteUC,
	}
}

// List handles GET /api/v1/transactions
func (h *TransactionHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	bankAccountID := r.URL.Query().Get("bankAccountId")
	status := r.URL.Query().Get("status")
	typeValue := r.URL.Query().Get("type")
	occurredFrom := r.URL.Query().Get("occurredFrom")
	occurredTo := r.URL.Query().Get("occurredTo")

	var bankAccountPtr, statusPtr, typePtr, fromPtr, toPtr *string
	if bankAccountID != "" {
		bankAccountPtr = &bankAccountID
	}
	if status != "" {
		statusPtr = &status
	}
	if typeValue != "" {
		typePtr = &typeValue
	}
	if occurredFrom != "" {
		fromPtr = &occurredFrom
	}
	if occurredTo != "" {
		toPtr = &occurredTo
	}

	transactions, err := h.listUseCase.Execute(usecases.ListTransactionsInput{
		ProfileID:     profileID,
		BankAccountID: bankAccountPtr,
		Status:        statusPtr,
		Type:          typePtr,
		OccurredFrom:  fromPtr,
		OccurredTo:    toPtr,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  transactions,
		"total": len(transactions),
	})
}

// Create handles POST /api/v1/transactions
func (h *TransactionHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.createUseCase.Execute(input)
	if err != nil {
		status := mapTransactionError(err)
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": tx,
	})
}

// Get handles GET /api/v1/transactions/{id}
func (h *TransactionHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	tx, err := h.getUseCase.Execute(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": tx,
	})
}

// Update handles PUT /api/v1/transactions/{id}
func (h *TransactionHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input usecases.UpdateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.updateUseCase.Execute(id, input)
	if err != nil {
		status := mapTransactionError(err)
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": tx,
	})
}

// UpdateStatus handles PUT /api/v1/transactions/{id}/status
func (h *TransactionHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input usecases.UpdateTransactionStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.updateStatusUseCase.Execute(id, input)
	if err != nil {
		status := mapTransactionError(err)
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": tx,
	})
}

// Delete handles DELETE /api/v1/transactions/{id}
func (h *TransactionHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.deleteUseCase.Execute(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mapTransactionError(err error) int {
	switch err {
	case usecases.ErrProfileNotFound, usecases.ErrCategoryNotFound, usecases.ErrTransactionNotFound:
		return http.StatusNotFound
	case usecases.ErrBankAccountNotFound, usecases.ErrBankAccountMismatch, usecases.ErrDestinationRequired,
		usecases.ErrInvalidInput, usecases.ErrInvalidTransactionType,
		usecases.ErrInsufficientBalance, usecases.ErrCreditLimitExceeded:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
