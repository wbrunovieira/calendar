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

// A refusal the caller can act on is a 400; something absent is a 404; anything else
// is a 500 with no driver detail in it. With 400 as the fallback, a Postgres outage
// came back as "400: reading the transaction: pq: ..." -- the error leaked and every
// caller that retries on 5xx dropped the work instead.
func TestMapMarkInvoicePaymentError(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantBody   string
	}{
		{transactionPkg.ErrNotFound, http.StatusNotFound, transactionPkg.ErrNotFound.Error()},
		{bankaccount.ErrNotFound, http.StatusNotFound, bankaccount.ErrNotFound.Error()},
		{usecases.ErrInvoiceNotFound, http.StatusNotFound, usecases.ErrInvoiceNotFound.Error()},
		{usecases.ErrNotACreditCard, http.StatusBadRequest, usecases.ErrNotACreditCard.Error()},
		{usecases.ErrNotAnInvoicePayment, http.StatusBadRequest, usecases.ErrNotAnInvoicePayment.Error()},
		{usecases.ErrPaymentNotConfirmed, http.StatusBadRequest, usecases.ErrPaymentNotConfirmed.Error()},
		{usecases.ErrInvoiceNotThisCard, http.StatusBadRequest, usecases.ErrInvoiceNotThisCard.Error()},
		{usecases.ErrInvoiceStillOpen, http.StatusBadRequest, usecases.ErrInvoiceStillOpen.Error()},
		{usecases.ErrPaymentExceedsInvoice, http.StatusBadRequest, usecases.ErrPaymentExceedsInvoice.Error()},
		// Wrapped, the way the use case returns them.
		{
			fmt.Errorf("%w: it stores 2702.66 and its lines sum to 2647.97", usecases.ErrInvoiceAmountOutOfSync),
			http.StatusBadRequest,
			"the bill's stored total does not match its own lines: it stores 2702.66 and its lines sum to 2647.97",
		},
		{
			fmt.Errorf("reading the transaction: %w", errors.New("pq: connection refused")),
			http.StatusInternalServerError,
			"could not record the invoice payment",
		},
	}
	for _, c := range cases {
		status, body := mapMarkInvoicePaymentError(c.err)
		if status != c.wantStatus {
			t.Errorf("%v: status = %d, want %d", c.err, status, c.wantStatus)
		}
		if body != c.wantBody {
			t.Errorf("%v: body = %q, want %q", c.err, body, c.wantBody)
		}
	}
}
