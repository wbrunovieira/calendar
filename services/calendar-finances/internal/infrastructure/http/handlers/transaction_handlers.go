package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/gorilla/mux"
)

type TransactionHandlers struct {
	createUseCase           *usecases.CreateTransactionUseCase
	listUseCase             *usecases.ListTransactionsUseCase
	getUseCase              *usecases.GetTransactionUseCase
	updateUseCase           *usecases.UpdateTransactionUseCase
	updateStatusUseCase     *usecases.UpdateTransactionStatusUseCase
	deleteUseCase           *usecases.DeleteTransactionUseCase
	dailyBalancesUseCase    *usecases.GetDailyBalancesUseCase
	cashflowSummaryUseCase  *usecases.GetCashflowSummaryUseCase
	expenseAnalysisUseCase  *usecases.GetExpenseAnalysisUseCase
	financialSummaryUseCase *usecases.GetFinancialSummaryUseCase
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

func (h *TransactionHandlers) SetDailyBalancesUseCase(uc *usecases.GetDailyBalancesUseCase) {
	h.dailyBalancesUseCase = uc
}

func (h *TransactionHandlers) SetCashflowSummaryUseCase(uc *usecases.GetCashflowSummaryUseCase) {
	h.cashflowSummaryUseCase = uc
}

func (h *TransactionHandlers) SetExpenseAnalysisUseCase(uc *usecases.GetExpenseAnalysisUseCase) {
	h.expenseAnalysisUseCase = uc
}

func (h *TransactionHandlers) SetFinancialSummaryUseCase(uc *usecases.GetFinancialSummaryUseCase) {
	h.financialSummaryUseCase = uc
}

// List handles GET /api/v1/transactions
func (h *TransactionHandlers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	profileID := q.Get("profileId")
	bankAccountID := q.Get("bankAccountId")
	invoiceID := q.Get("invoiceId")
	costCenterID := q.Get("costCenterId")
	status := q.Get("status")
	typeValue := q.Get("type")
	occurredFrom := q.Get("occurredFrom")
	occurredTo := q.Get("occurredTo")
	includeAsDestination := q.Get("includeAsDestination") == "true"

	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))

	var bankAccountPtr, invoicePtr, statusPtr, typePtr, fromPtr, toPtr *string
	if bankAccountID != "" {
		bankAccountPtr = &bankAccountID
	}
	if invoiceID != "" {
		invoicePtr = &invoiceID
	}
	var costCenterPtr *string
	if costCenterID != "" {
		costCenterPtr = &costCenterID
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

	result, err := h.listUseCase.Execute(usecases.ListTransactionsInput{
		ProfileID:            profileID,
		BankAccountID:        bankAccountPtr,
		InvoiceID:            invoicePtr,
		CostCenterID:         costCenterPtr,
		Status:               statusPtr,
		Type:                 typePtr,
		OccurredFrom:         fromPtr,
		OccurredTo:           toPtr,
		IncludeAsDestination: includeAsDestination,
		Page:                 page,
		PageSize:             pageSize,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":     result.Items,
		"total":    result.Total,
		"page":     result.Page,
		"pageSize": result.PageSize,
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

// Delete handles DELETE /api/v1/transactions/{id}.
//
// The verb no longer matches what happens: a ledger reverses instead of deleting, and
// a caller told "204 No Content" that then finds the row in the next GET concludes the
// call failed. Kept only to point at the route that does say what it does.
func (h *TransactionHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "POST")
	http.Error(w, "a transaction is not deleted, it is reversed: POST /api/v1/transactions/{id}/reversal?reason=...&by=...", http.StatusMethodNotAllowed)
}

// Reverse handles POST /api/v1/transactions/{id}/reversal
//
// Reason and actor are required. A reversal whose motive was not captured at the time
// cannot be reconstructed later, and the incident behind this feature was an automated
// agent removing a legitimate entry — so knowing who reversed it is the control, not a
// detail.
func (h *TransactionHandlers) Reverse(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	q := r.URL.Query()

	reason := transactionPkg.ReversalReason(strings.ToUpper(q.Get("reason")))
	if !reason.Valid() {
		http.Error(w, "reason is required and must be one of: NUNCA_OCORREU, DUPLICADO, VALOR_INCORRETO, CONTA_INCORRETA, ESTORNADO_PELO_BANCO, PAGAMENTO_DEVOLVIDO, COMPRA_CANCELADA", http.StatusBadRequest)
		return
	}
	by := q.Get("by")
	if by == "" {
		by = r.Header.Get("X-Actor")
	}
	if by == "" {
		http.Error(w, "by is required: name the caller (frontend, n8n, agent, api) so a reversal can be traced", http.StatusBadRequest)
		return
	}

	err := h.deleteUseCase.ExecuteWithReason(usecases.ReverseTransactionInput{
		ID: id, Reason: reason, Note: q.Get("note"), By: by,
	})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, transactionPkg.ErrAlreadyReversed):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, usecases.ErrTransactionNotFound), errors.Is(err, transactionPkg.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		// Anything else is our side failing. Reporting it as 404 tells the caller
		// there was nothing there, and a balance left crooked by a failed reversal
		// is never investigated.
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DailyBalances handles GET /api/v1/transactions/daily-balances
func (h *TransactionHandlers) DailyBalances(w http.ResponseWriter, r *http.Request) {
	if h.dailyBalancesUseCase == nil {
		http.Error(w, "daily balances not available", http.StatusNotImplemented)
		return
	}

	profileID := r.URL.Query().Get("profileId")
	bankAccountID := r.URL.Query().Get("bankAccountId")
	occurredFrom := r.URL.Query().Get("occurredFrom")
	occurredTo := r.URL.Query().Get("occurredTo")

	var fromPtr, toPtr *string
	if occurredFrom != "" {
		fromPtr = &occurredFrom
	}
	if occurredTo != "" {
		toPtr = &occurredTo
	}

	balances, err := h.dailyBalancesUseCase.Execute(usecases.GetDailyBalancesInput{
		ProfileID:     profileID,
		BankAccountID: bankAccountID,
		OccurredFrom:  fromPtr,
		OccurredTo:    toPtr,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  balances,
		"total": len(balances),
	})
}

// CashflowSummary handles GET /api/v1/transactions/cashflow-summary
func (h *TransactionHandlers) CashflowSummary(w http.ResponseWriter, r *http.Request) {
	if h.cashflowSummaryUseCase == nil {
		http.Error(w, "cashflow summary not available", http.StatusNotImplemented)
		return
	}

	out, err := h.cashflowSummaryUseCase.Execute(usecases.CashflowSummaryInput{
		ProfileID: r.URL.Query().Get("profileId"),
		From:      r.URL.Query().Get("from"),
		To:        r.URL.Query().Get("to"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": out})
}

// ExpenseAnalysis handles GET /api/v1/transactions/expense-analysis
func (h *TransactionHandlers) ExpenseAnalysis(w http.ResponseWriter, r *http.Request) {
	if h.expenseAnalysisUseCase == nil {
		http.Error(w, "expense analysis not available", http.StatusNotImplemented)
		return
	}

	out, err := h.expenseAnalysisUseCase.Execute(usecases.GetExpenseAnalysisInput{
		ProfileID: r.URL.Query().Get("profileId"),
		From:      r.URL.Query().Get("from"),
		To:        r.URL.Query().Get("to"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": out})
}

// FinancialSummary handles GET /api/v1/transactions/financial-summary
func (h *TransactionHandlers) FinancialSummary(w http.ResponseWriter, r *http.Request) {
	if h.financialSummaryUseCase == nil {
		http.Error(w, "financial summary not available", http.StatusNotImplemented)
		return
	}

	out, err := h.financialSummaryUseCase.Execute(usecases.GetFinancialSummaryInput{
		ProfileID: r.URL.Query().Get("profileId"),
		From:      r.URL.Query().Get("from"),
		To:        r.URL.Query().Get("to"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": out})
}

func mapTransactionError(err error) int {
	// A reference-month conflict is an operator-actionable data problem, not a
	// server fault. Reported as 500 it reads like an outage, and callers that
	// retry on 5xx (n8n, the agents) would retry it forever.
	// ErrInvoiceCycleMismatch deliberately falls through to 500: it is our bug,
	// and retrying callers must see a 5xx so it surfaces instead of being dropped.
	if errors.Is(err, usecases.ErrInvoiceReferenceConflict) {
		return http.StatusConflict
	}

	switch err {
	case usecases.ErrProfileNotFound, usecases.ErrCategoryNotFound, usecases.ErrCostCenterNotFound,
		usecases.ErrTransactionNotFound:
		return http.StatusNotFound
	case usecases.ErrBankAccountNotFound, usecases.ErrBankAccountMismatch, usecases.ErrDestinationRequired,
		usecases.ErrCurrencyMismatch,
		usecases.ErrInvalidInput, usecases.ErrInvalidTransactionType,
		usecases.ErrInsufficientBalance, usecases.ErrCreditLimitExceeded:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
