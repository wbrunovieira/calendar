package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Every refusal a caller can act on is a 400. Listing them by hand is what let
// ErrWouldLowerRecordedPayment ship as a 500 with the text "could not record the
// invoice payment" -- a guard that had both numbers to report, reported as an outage,
// on a real repair. The sentinels now carry a shared marker, so a new guard is mapped
// by construction rather than by remembering.
func TestEveryPaymentRefusalIsABadRequest(t *testing.T) {
	refusals := []error{
		usecases.ErrNotACreditCard,
		usecases.ErrNotAnInvoicePayment,
		usecases.ErrPaymentNotConfirmed,
		usecases.ErrInvoiceNotThisCard,
		usecases.ErrInvoiceStillOpen,
		usecases.ErrInvoiceAmountOutOfSync,
		usecases.ErrPaymentExceedsInvoice,
		usecases.ErrPaymentAlreadyRecorded,
		usecases.ErrWouldLowerRecordedPayment,
		usecases.ErrCardCannotFundAPayment,
	}
	for _, r := range refusals {
		// Bare.
		if status, body := mapMarkInvoicePaymentError(r); status != http.StatusBadRequest || body != r.Error() {
			t.Errorf("%v: got %d %q, want 400 with its own words", r, status, body)
		}
		// Wrapped with the numbers, the way the use case returns them.
		wrapped := fmt.Errorf("%w: it stores 814.92 and its lines sum to 940.93", r)
		if status, body := mapMarkInvoicePaymentError(wrapped); status != http.StatusBadRequest || body != wrapped.Error() {
			t.Errorf("%v wrapped: got %d %q, want 400 carrying the numbers", r, status, body)
		}
		// And every one of them is reachable through the shared marker, which is what
		// makes the mapping proof against the next guard being forgotten.
		if !errors.Is(r, usecases.ErrPaymentRefused) {
			t.Errorf("%v does not carry the refusal marker, so it will fall through to 500", r)
		}
	}
}

func TestNotFoundAndFailuresStayDistinct(t *testing.T) {
	for _, e := range []error{transactionPkg.ErrNotFound, bankaccount.ErrNotFound, usecases.ErrInvoiceNotFound} {
		if status, _ := mapMarkInvoicePaymentError(e); status != http.StatusNotFound {
			t.Errorf("%v: got %d, want 404", e, status)
		}
	}
	// A driver error is still a 500, and still says nothing about the database.
	boom := fmt.Errorf("reading the transaction: %w", errors.New("pq: connection refused"))
	status, body := mapMarkInvoicePaymentError(boom)
	if status != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", status)
	}
	if body != "could not record the invoice payment" {
		t.Errorf("body = %q, want the neutral message", body)
	}
}
