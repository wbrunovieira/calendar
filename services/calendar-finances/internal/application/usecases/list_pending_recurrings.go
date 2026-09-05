package usecases

import (
	"errors"
	"sort"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// lookbackDays bounds how far back to read transactions when deciding whether an
// obligation was already settled — enough to cover the previous occurrence and the
// tolerance window around it, without scanning the whole history.
const lookbackDays = 45

type ListPendingRecurringsInput struct {
	ProfileID string
	Reference string // YYYY-MM-DD; defaults to today
}

// PendingRecurring is an obligation that has come around and has not been settled.
type PendingRecurring struct {
	ID             string    `json:"id"`
	Description    string    `json:"description"`
	Type           string    `json:"type"`
	Amount         float64   `json:"amount"`
	BankAccountID  *string   `json:"bankAccountId,omitempty"`
	CategoryID     *string   `json:"categoryId,omitempty"`
	NextOccurrence time.Time `json:"nextOccurrence"`
	DaysOverdue    int       `json:"daysOverdue"`
}

// ListPendingRecurringsUseCase answers "what do I still owe on a schedule". The rule
// for whether a payment settles an obligation lives on the entity, so the dashboard,
// an agent and a reminder job all agree on what is pending.
type ListPendingRecurringsUseCase struct {
	recurringRepo recurringtransaction.Repository
	txRepo        transaction.Repository
}

func NewListPendingRecurringsUseCase(
	recurringRepo recurringtransaction.Repository,
	txRepo transaction.Repository,
) *ListPendingRecurringsUseCase {
	return &ListPendingRecurringsUseCase{recurringRepo: recurringRepo, txRepo: txRepo}
}

func (uc *ListPendingRecurringsUseCase) Execute(input ListPendingRecurringsInput) ([]PendingRecurring, error) {
	if input.ProfileID == "" {
		return nil, errors.New("profileId is required")
	}

	reference := time.Now()
	if input.Reference != "" {
		parsed, err := parseDate(input.Reference)
		if err != nil {
			return nil, errors.New("reference must be YYYY-MM-DD")
		}
		reference = parsed
	}

	recurrings, err := uc.recurringRepo.ListByProfile(input.ProfileID)
	if err != nil {
		return nil, err
	}

	from := reference.AddDate(0, 0, -lookbackDays)
	to := reference.AddDate(0, 0, lookbackDays)
	txs, err := uc.txRepo.List(transaction.ListFilter{
		ProfileID:    input.ProfileID,
		OccurredFrom: &from,
		OccurredTo:   &to,
	})
	if err != nil {
		return nil, err
	}

	pending := make([]PendingRecurring, 0, len(recurrings))
	for _, r := range recurrings {
		if r.Status != recurringtransaction.StatusActive {
			continue
		}

		occurrence, owed := unpaidOccurrence(r, txs, reference)
		if !owed {
			continue
		}

		pending = append(pending, PendingRecurring{
			ID:             r.ID,
			Description:    r.Description,
			Type:           r.Type,
			Amount:         r.Amount,
			BankAccountID:  r.BankAccountID,
			CategoryID:     r.CategoryID,
			NextOccurrence: occurrence,
			DaysOverdue:    daysBetweenDates(occurrence, reference),
		})
	}

	// Longest overdue first: that is the order someone needs to act in.
	sort.Slice(pending, func(i, j int) bool { return pending[i].DaysOverdue > pending[j].DaysOverdue })

	return pending, nil
}

func daysBetweenDates(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// unpaidOccurrence answers which occurrence of an obligation is still owed, if any.
//
// Looking only at NextOccurrence is not enough: the scheduler generates a PLANNED
// entry for an occurrence and immediately steps NextOccurrence forward, so the
// obligation stops being "due" while nothing has actually been paid. An entry sitting
// there unpaid is precisely what is still owed, so it is what this reports.
func unpaidOccurrence(
	r *recurringtransaction.RecurringTransaction,
	txs []*transaction.Transaction,
	reference time.Time,
) (time.Time, bool) {
	var earliestUnpaid time.Time
	for _, tx := range txs {
		if !r.Matches(tx) || tx.OccurredOn.After(reference) {
			continue
		}
		if tx.Status == transaction.StatusConfirmed {
			// Paid. Anything scheduled for the same occurrence is settled with it.
			if r.IsSatisfiedBy(tx) || earliestUnpaid.IsZero() {
				return time.Time{}, false
			}
			continue
		}
		if earliestUnpaid.IsZero() || tx.OccurredOn.Before(earliestUnpaid) {
			earliestUnpaid = tx.OccurredOn
		}
	}
	if !earliestUnpaid.IsZero() {
		return earliestUnpaid, true
	}

	// Nothing generated yet: the obligation is owed once its date has come around.
	if r.IsDue(reference) {
		return r.NextOccurrence, true
	}
	return time.Time{}, false
}
