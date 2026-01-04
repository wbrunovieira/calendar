package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type InvoiceHandlers struct {
	createUC         *usecases.CreateInvoiceUseCase
	listUC           *usecases.ListInvoicesUseCase
	getUC            *usecases.GetInvoiceUseCase
	getCurrentUC     *usecases.GetCurrentInvoiceUseCase
	closeUC          *usecases.CloseInvoiceUseCase
	payUC            *usecases.PayInvoiceUseCase
	addAmountUC      *usecases.AddAmountToInvoiceUseCase
}

func NewInvoiceHandlers(
	createUC *usecases.CreateInvoiceUseCase,
	listUC *usecases.ListInvoicesUseCase,
	getUC *usecases.GetInvoiceUseCase,
	getCurrentUC *usecases.GetCurrentInvoiceUseCase,
	closeUC *usecases.CloseInvoiceUseCase,
	payUC *usecases.PayInvoiceUseCase,
	addAmountUC *usecases.AddAmountToInvoiceUseCase,
) *InvoiceHandlers {
	return &InvoiceHandlers{
		createUC:         createUC,
		listUC:           listUC,
		getUC:            getUC,
		getCurrentUC:     getCurrentUC,
		closeUC:          closeUC,
		payUC:            payUC,
		addAmountUC:      addAmountUC,
	}
}

// List handles GET /api/v1/invoices?bankAccountId=...
func (h *InvoiceHandlers) List(w http.ResponseWriter, r *http.Request) {
	bankAccountID := r.URL.Query().Get("bankAccountId")
	if bankAccountID == "" {
		http.Error(w, "bankAccountId is required", http.StatusBadRequest)
		return
	}

	invoices, err := h.listUC.Execute(bankAccountID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrBankAccountNotFound {
			status = http.StatusNotFound
		} else if err == usecases.ErrNotCreditCard {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{
		"data":  invoices,
		"total": len(invoices),
	})
}

// Get handles GET /api/v1/invoices/{id}
func (h *InvoiceHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	invoice, err := h.getUC.Execute(id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrInvoiceNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": invoice})
}

// GetCurrent handles GET /api/v1/invoices/current?bankAccountId=...
func (h *InvoiceHandlers) GetCurrent(w http.ResponseWriter, r *http.Request) {
	bankAccountID := r.URL.Query().Get("bankAccountId")
	if bankAccountID == "" {
		http.Error(w, "bankAccountId is required", http.StatusBadRequest)
		return
	}

	invoice, err := h.getCurrentUC.Execute(bankAccountID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrBankAccountNotFound {
			status = http.StatusNotFound
		} else if err == usecases.ErrNotCreditCard {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": invoice})
}

// Create handles POST /api/v1/invoices
func (h *InvoiceHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateInvoiceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	invoice, err := h.createUC.Execute(input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrBankAccountNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, map[string]any{"data": invoice})
}

// Close handles POST /api/v1/invoices/{id}/close
func (h *InvoiceHandlers) Close(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	invoice, err := h.closeUC.Execute(id)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrInvoiceNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": invoice})
}

// Pay handles POST /api/v1/invoices/{id}/pay
func (h *InvoiceHandlers) Pay(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input usecases.PayInvoiceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	input.InvoiceID = id

	invoice, err := h.payUC.Execute(input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrInvoiceNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": invoice})
}

// AddAmount handles POST /api/v1/invoices/{id}/add-amount
func (h *InvoiceHandlers) AddAmount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input usecases.AddAmountToInvoiceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	input.InvoiceID = id

	invoice, err := h.addAmountUC.Execute(input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrInvoiceNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	respondJSON(w, map[string]any{"data": invoice})
}
