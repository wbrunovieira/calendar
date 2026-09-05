package recurringtransaction

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// satisfactionWindow bounds how far a payment can sit from the occurrence it settles.
// It is half the interval the rule implies, so a payment can only ever belong to the
// nearest occurrence: anything further away settled the previous one.
func (r *RecurringTransaction) satisfactionWindow() time.Duration {
	rule := strings.ToUpper(r.RecurrenceRule)
	switch {
	case strings.Contains(rule, "FREQ=DAILY"):
		return 0
	case strings.Contains(rule, "FREQ=WEEKLY"):
		return 3 * 24 * time.Hour
	case strings.Contains(rule, "FREQ=YEARLY"):
		return 180 * 24 * time.Hour
	default: // monthly, and anything unrecognised
		return 15 * 24 * time.Hour
	}
}

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
// Matching is by description, within a window around the occurrence date. The
// recurrence rule is deliberately not used as a fallback: it is a plain string copied
// onto every generated transaction, so two unrelated bills falling on the same day of
// the month share it, and one would settle the other. Until a transaction carries the
// id of the recurrence that generated it, a renamed transaction stays pending — noise,
// where the alternative is silently marking the wrong bill as paid.
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
	if !r.withinWindow(tx.OccurredOn) {
		return false
	}
	return normalize(tx.Description) == normalize(r.Description)
}

func (r *RecurringTransaction) withinWindow(paidOn time.Time) bool {
	distance := dateOf(paidOn).Sub(dateOf(r.NextOccurrence))
	if distance < 0 {
		distance = -distance
	}
	return distance <= r.satisfactionWindow()
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
