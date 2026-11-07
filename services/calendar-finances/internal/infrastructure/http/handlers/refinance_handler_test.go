package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/gorilla/mux"
)

// The Refinance handler had no test at all: mutating its success status to 418 and
// deleting the nil-use-case guard both left the whole suite green. A route nobody
// exercises is a route whose contract is whatever the code happens to do.

func refinanceRouter(h *InvoiceHandlers) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/invoices/refinance", h.Refinance).Methods("POST")
	r.HandleFunc("/invoices/{id}/release-refinancing", h.ReleaseRefinancing).Methods("POST")
	return r
}

func postJSON(t *testing.T, r *mux.Router, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// Fails CLOSED. Answering 2xx to a request that recorded nothing is worse than an
// error: the caller is told a bill was settled when it was not.
func TestRefinanceAnswers500WhenTheUseCaseIsNotWired(t *testing.T) {
	rec := postJSON(t, refinanceRouter(&InvoiceHandlers{}),
		"/invoices/refinance", `{"transactionId":"t","invoiceIds":["a"]}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 for an unwired use case", rec.Code)
	}
	if rec.Code < 400 {
		t.Error("an unwired server reported success")
	}
}

func TestReleaseRefinancingAnswers500WhenTheUseCaseIsNotWired(t *testing.T) {
	rec := postJSON(t, refinanceRouter(&InvoiceHandlers{}),
		"/invoices/inv-1/release-refinancing", ``)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 for an unwired use case", rec.Code)
	}
}

func TestRefinanceRejectsAMalformedBody(t *testing.T) {
	rec := postJSON(t, refinanceRouter(&InvoiceHandlers{}), "/invoices/refinance", `{not json`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a malformed body", rec.Code)
	}
}

// The body is read before the wiring is checked, so a malformed one must not be
// masked by the 500 — and a valid one must reach the use case rather than being
// rejected for its shape.
func TestRefinanceAcceptsTheDocumentedBodyShape(t *testing.T) {
	rec := postJSON(t, refinanceRouter(&InvoiceHandlers{}),
		"/invoices/refinance", `{"transactionId":"credito","invoiceIds":["set","out"]}`)

	if rec.Code == http.StatusBadRequest {
		t.Errorf("the documented body was rejected as malformed: %s", rec.Body.String())
	}
}

// Every test above runs against an UNWIRED handler, so they never reach the
// success path or the refusal mapping: changing 204 to 418, or mapping a refusal
// to 500 instead of 400, both survived. These wire a real use case.
func wiredRefinanceHandler(t *testing.T) *InvoiceHandlers {
	t.Helper()
	const cardID, profileID = "card", "personal"
	closing, due := 9, 14
	card := &bankaccount.BankAccount{
		ID: cardID, ProfileID: profileID, Name: "Cartão",
		Type: bankaccount.AccountTypeCreditCard, ClosingDay: &closing, DueDay: &due,
		Currency: "BRL", IsActive: true,
	}
	bill := &invoice.Invoice{
		ID: "out", BankAccountID: cardID, Amount: 100, Status: invoice.StatusClosed,
		ReferenceDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		OpeningDate:   time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
	}
	billID := bill.ID
	credit := &transactionPkg.Transaction{
		ID: "credito", ProfileID: profileID, BankAccountID: cardID,
		Type: transactionPkg.TypeIncome, Status: transactionPkg.StatusConfirmed,
		Amount: 500, Currency: "BRL", OccurredOn: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC),
	}
	line := &transactionPkg.Transaction{
		ID: "linha", ProfileID: profileID, BankAccountID: cardID,
		Type: transactionPkg.TypeExpense, Status: transactionPkg.StatusConfirmed,
		Amount: 100, Currency: "BRL", InvoiceID: &billID,
		OccurredOn: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
	}

	h := &InvoiceHandlers{}
	h.SetRefinanceUseCase(usecases.NewRefinanceInvoiceUseCase(
		&handlerAccountRepo{card: card},
		&handlerTxRepo{txs: []*transactionPkg.Transaction{credit, line}},
		&handlerInvoiceRepo{bills: map[string]*invoice.Invoice{bill.ID: bill}},
	))
	return h
}

func TestRefinanceAnswers204OnSuccess(t *testing.T) {
	rec := postJSON(t, refinanceRouter(wiredRefinanceHandler(t)),
		"/invoices/refinance", `{"transactionId":"credito","invoiceIds":["out"]}`)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204; body: %s", rec.Code, rec.Body.String())
	}
}

// A refusal the caller can act on is a 400, not a 500 that retry logic treats as
// a transient fault and replays.
func TestRefinanceAnswers400OnARefusal(t *testing.T) {
	rec := postJSON(t, refinanceRouter(wiredRefinanceHandler(t)),
		"/invoices/refinance", `{"transactionId":"credito","invoiceIds":[]}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a refusal; body: %s", rec.Code, rec.Body.String())
	}
}

func TestReleaseRefinancingAnswers204OnSuccess(t *testing.T) {
	h := wiredRefinanceHandler(t)
	r := refinanceRouter(h)
	if rec := postJSON(t, r, "/invoices/refinance", `{"transactionId":"credito","invoiceIds":["out"]}`); rec.Code != http.StatusNoContent {
		t.Fatalf("setup: convert returned %d", rec.Code)
	}

	rec := postJSON(t, r, "/invoices/out/release-refinancing", ``)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204; body: %s", rec.Code, rec.Body.String())
	}
}

func TestReleaseRefinancingAnswers400WhenTheBillWasNotConverted(t *testing.T) {
	rec := postJSON(t, refinanceRouter(wiredRefinanceHandler(t)), "/invoices/out/release-refinancing", ``)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

// Minimal repositories, enough for the handler path. They hand back copies, like
// rows out of Postgres, so a use case that mutates without writing does not look
// like it persisted.
type handlerAccountRepo struct {
	bankaccount.Repository
	card *bankaccount.BankAccount
}

func (r *handlerAccountRepo) FindByID(id string) (*bankaccount.BankAccount, error) {
	if r.card != nil && r.card.ID == id {
		loaded := *r.card
		return &loaded, nil
	}
	return nil, bankaccount.ErrNotFound
}

type handlerTxRepo struct {
	transactionPkg.Repository
	txs []*transactionPkg.Transaction
}

func (r *handlerTxRepo) GetByID(id string) (*transactionPkg.Transaction, error) {
	for _, tx := range r.txs {
		if tx.ID == id {
			loaded := *tx
			return &loaded, nil
		}
	}
	return nil, transactionPkg.ErrNotFound
}

func (r *handlerTxRepo) Update(tx *transactionPkg.Transaction) error {
	for i, existing := range r.txs {
		if existing.ID == tx.ID {
			stored := *tx
			r.txs[i] = &stored
			return nil
		}
	}
	return transactionPkg.ErrNotFound
}

func (r *handlerTxRepo) SumByInvoiceID(invoiceID string) (float64, error) {
	var total float64
	for _, tx := range r.txs {
		if tx.InvoiceID == nil || *tx.InvoiceID != invoiceID {
			continue
		}
		if tx.Type == transactionPkg.TypeIncome {
			total -= tx.Amount
		} else {
			total += tx.Amount
		}
	}
	return total, nil
}

type handlerInvoiceRepo struct {
	invoice.Repository
	bills map[string]*invoice.Invoice
}

func (r *handlerInvoiceRepo) FindByID(id string) (*invoice.Invoice, error) {
	if inv, ok := r.bills[id]; ok {
		loaded := *inv
		return &loaded, nil
	}
	return nil, usecases.ErrInvoiceNotFound
}

func (r *handlerInvoiceRepo) Update(inv *invoice.Invoice) error {
	if _, ok := r.bills[inv.ID]; !ok {
		return usecases.ErrInvoiceNotFound
	}
	stored := *inv
	r.bills[inv.ID] = &stored
	return nil
}

func (r *handlerInvoiceRepo) FindByRefinancedBy(transactionID string) ([]*invoice.Invoice, error) {
	var list []*invoice.Invoice
	for _, inv := range r.bills {
		if inv.RefinancedBy != nil && *inv.RefinancedBy == transactionID {
			loaded := *inv
			list = append(list, &loaded)
		}
	}
	return list, nil
}
