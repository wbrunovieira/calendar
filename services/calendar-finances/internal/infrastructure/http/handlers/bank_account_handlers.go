package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type BankAccountHandlers struct {
	createUseCase             *usecases.CreateBankAccountUseCase
	listUseCase               *usecases.ListBankAccountsUseCase
	getUseCase                *usecases.GetBankAccountUseCase
	updateUseCase             *usecases.UpdateBankAccountUseCase
	deleteUseCase             *usecases.DeleteBankAccountUseCase
	reorderUseCase            *usecases.ReorderBankAccountsUseCase
	recalculateBalanceUseCase *usecases.RecalculateBalanceUseCase
	closeMonthUseCase         *usecases.CloseMonthUseCase
	creditUsageUC             *usecases.GetCreditUsageUseCase
	upcomingMaturitiesUseCase *usecases.ListUpcomingMaturitiesUseCase
	sellPositionUseCase       *usecases.SellPositionUseCase
}

func NewBankAccountHandlers(
	createUC *usecases.CreateBankAccountUseCase,
	listUC *usecases.ListBankAccountsUseCase,
	getUC *usecases.GetBankAccountUseCase,
	updateUC *usecases.UpdateBankAccountUseCase,
	deleteUC *usecases.DeleteBankAccountUseCase,
	reorderUC *usecases.ReorderBankAccountsUseCase,
	recalculateBalanceUC *usecases.RecalculateBalanceUseCase,
	closeMonthUC *usecases.CloseMonthUseCase,
	upcomingMaturitiesUC *usecases.ListUpcomingMaturitiesUseCase,
	sellPositionUC *usecases.SellPositionUseCase,
) *BankAccountHandlers {
	return &BankAccountHandlers{
		createUseCase:             createUC,
		listUseCase:               listUC,
		getUseCase:                getUC,
		updateUseCase:             updateUC,
		deleteUseCase:             deleteUC,
		reorderUseCase:            reorderUC,
		recalculateBalanceUseCase: recalculateBalanceUC,
		closeMonthUseCase:         closeMonthUC,
		upcomingMaturitiesUseCase: upcomingMaturitiesUC,
		sellPositionUseCase:       sellPositionUC,
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

// SetCreditUsageUseCase wires the credit-usage query after construction, following
// how the other cross-cutting queries are attached in main.go.
func (h *BankAccountHandlers) SetCreditUsageUseCase(uc *usecases.GetCreditUsageUseCase) {
	h.creditUsageUC = uc
}

// CreditUsage handles GET /api/v1/bank-accounts/{id}/credit-usage
func (h *BankAccountHandlers) CreditUsage(w http.ResponseWriter, r *http.Request) {
	if h.creditUsageUC == nil {
		http.Error(w, "credit usage not available", http.StatusNotImplemented)
		return
	}

	usage, err := h.creditUsageUC.Execute(mux.Vars(r)["id"])
	if err != nil {
		switch err {
		case usecases.ErrBankAccountNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		case usecases.ErrNotACreditCard:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": usage})
}

// UpcomingMaturities handles GET /api/v1/bank-accounts/maturities?profileId=xxx&withinDays=30
// Returns active investments that have matured or will mature within the window,
// for home-screen alerts (mirrors planned-transaction alerts).
func (h *BankAccountHandlers) UpcomingMaturities(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	withinDays := 0
	if v := r.URL.Query().Get("withinDays"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			withinDays = n
		}
	}

	alerts, err := h.upcomingMaturitiesUseCase.Execute(usecases.ListUpcomingMaturitiesInput{
		ProfileID:  profileID,
		WithinDays: withinDays,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  alerts,
		"total": len(alerts),
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

// CloseMonth handles POST /api/v1/bank-accounts/close-month
// Body: { "referenceMonth": "2026-03-01" }  (any date in the target month)
func (h *BankAccountHandlers) CloseMonth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ReferenceMonth string `json:"referenceMonth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var t usecases.CloseMonthInput
	if body.ReferenceMonth != "" {
		parsed, err := time.Parse("2006-01-02", body.ReferenceMonth)
		if err != nil {
			http.Error(w, "referenceMonth must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		t.ReferenceMonth = parsed
	} else {
		// Default: close previous calendar month
		t.ReferenceMonth = time.Now().AddDate(0, -1, 0)
	}

	result, err := h.closeMonthUseCase.Execute(t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": result})
}

// Sell handles POST /api/v1/bank-accounts/{id}/sell
// Records the sale of shares/quotas from an investment position: the position's
// shares go down and the linked cash account is credited with the proceeds.
func (h *BankAccountHandlers) Sell(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input usecases.SellPositionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.sellPositionUseCase.Execute(id, input)
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
