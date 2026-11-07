package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
