package recurringtransaction

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// satisfactionWindowDays tolerates paying a bill a few days late, or a few days early
// in the previous month, without losing track of which occurrence it settled.
const satisfactionWindowDays = 7

// IsDue reports whether the obligation has already come around and is still active.
func (r *RecurringTransaction) IsDue(reference time.Time) bool {
	if r.Status != StatusActive {
		return false
	}
	return !dateOf(r.NextOccurrence).After(dateOf(reference))
}

// IsSatisfiedBy reports whether a transaction settles this occurrence of the
// obligation — the question behind "what do I still owe this month".
//
// The amount is deliberately NOT compared. A recurring bill is an estimate: a phone
// plan quoted at 99 arrives at 111,57, and comparing amounts kept obligations flagged
// as pending long after they had been paid.
//
// Two shapes match. A transaction generated from this recurrence carries its rule, so
// it is recognised even if someone renamed it afterwards; anything paid by hand is
// matched by description, which is what a person repeats when they pay the same bill.
func (r *RecurringTransaction) IsSatisfiedBy(tx *transaction.Transaction) bool {
	if tx == nil {
		return false
	}
	if tx.Status == transaction.StatusCancelled {
		return false
	}
	if string(tx.Type) != r.Type {
		return false
	}
	if r.BankAccountID != nil && tx.BankAccountID != *r.BankAccountID {
		return false
	}
	if !r.sameOccurrence(tx.OccurredOn) {
		return false
	}

	if tx.RecurrenceRule != nil && *tx.RecurrenceRule != "" && *tx.RecurrenceRule == r.RecurrenceRule {
		return true
	}
	return normalize(tx.Description) == normalize(r.Description)
}

// sameOccurrence accepts the same calendar month, or a payment within the tolerance
// window on either side — which is how a bill due on the 30th gets paid on the 3rd.
func (r *RecurringTransaction) sameOccurrence(paidOn time.Time) bool {
	due := dateOf(r.NextOccurrence)
	paid := dateOf(paidOn)

	if due.Year() == paid.Year() && due.Month() == paid.Month() {
		return true
	}

	diff := paid.Sub(due).Hours() / 24
	if diff < 0 {
		diff = -diff
	}
	return diff <= satisfactionWindowDays
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
