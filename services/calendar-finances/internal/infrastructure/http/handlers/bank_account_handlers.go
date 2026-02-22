package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type BankAccountHandlers struct {
	createUseCase              *usecases.CreateBankAccountUseCase
	listUseCase                *usecases.ListBankAccountsUseCase
	getUseCase                 *usecases.GetBankAccountUseCase
	updateUseCase              *usecases.UpdateBankAccountUseCase
	deleteUseCase              *usecases.DeleteBankAccountUseCase
	reorderUseCase             *usecases.ReorderBankAccountsUseCase
	recalculateBalanceUseCase  *usecases.RecalculateBalanceUseCase
}

func NewBankAccountHandlers(
	createUC *usecases.CreateBankAccountUseCase,
	listUC *usecases.ListBankAccountsUseCase,
	getUC *usecases.GetBankAccountUseCase,
	updateUC *usecases.UpdateBankAccountUseCase,
	deleteUC *usecases.DeleteBankAccountUseCase,
	reorderUC *usecases.ReorderBankAccountsUseCase,
	recalculateBalanceUC *usecases.RecalculateBalanceUseCase,
) *BankAccountHandlers {
	return &BankAccountHandlers{
		createUseCase:              createUC,
		listUseCase:                listUC,
		getUseCase:                 getUC,
		updateUseCase:              updateUC,
		deleteUseCase:              deleteUC,
		reorderUseCase:             reorderUC,
		recalculateBalanceUseCase:  recalculateBalanceUC,
	}
}

// Create handles POST /api/v1/bank-accounts
func (h *BankAccountHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateBankAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	account, err := h.createUseCase.Execute(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": account,
	})
}

// List handles GET /api/v1/bank-accounts?profileId=xxx
func (h *BankAccountHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")

	accounts, err := h.listUseCase.Execute(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  accounts,
		"total": len(accounts),
	})
}

// Get handles GET /api/v1/bank-accounts/{id}
func (h *BankAccountHandlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	account, err := h.getUseCase.Execute(id)
	if err != nil {
		if err == usecases.ErrBankAccountNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": account,
	})
}

// Update handles PUT /api/v1/bank-accounts/{id}
func (h *BankAccountHandlers) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input usecases.UpdateBankAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	account, err := h.updateUseCase.Execute(id, input)
	if err != nil {
		if err == usecases.ErrBankAccountNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": account,
	})
}

// Delete handles DELETE /api/v1/bank-accounts/{id}
func (h *BankAccountHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.deleteUseCase.Execute(id); err != nil {
		if err == usecases.ErrBankAccountNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RecalculateBalance handles POST /api/v1/bank-accounts/{id}/recalculate-balance
func (h *BankAccountHandlers) RecalculateBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := h.recalculateBalanceUseCase.Execute(id)
	if err != nil {
		if err == usecases.ErrBankAccountNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}

// Reorder handles PUT /api/v1/bank-accounts/reorder
func (h *BankAccountHandlers) Reorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []usecases.ReorderItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.reorderUseCase.Execute(body.Items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
