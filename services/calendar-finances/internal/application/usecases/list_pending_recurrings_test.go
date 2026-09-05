package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type pendingRecurringRepo struct {
	fakeRecurringRepo
	list []*recurringtransaction.RecurringTransaction
}

func (r *pendingRecurringRepo) ListByProfile(string) ([]*recurringtransaction.RecurringTransaction, error) {
	return r.list, nil
}

func recurringOn(desc string, day int, status recurringtransaction.Status) *recurringtransaction.RecurringTransaction {
	return &recurringtransaction.RecurringTransaction{
		ID:             desc,
		ProfileID:      "p1",
		Type:           "EXPENSE",
		Amount:         99,
		Description:    desc,
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=20",
		NextOccurrence: time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC),
		Status:         status,
	}
}

// What the dashboard asks: of everything I owe on a schedule, what has come around
// and has not been paid yet.
func TestListPendingRecurrings_ReturnsOnlyWhatIsDueAndUnpaid(t *testing.T) {
	recurrings := []*recurringtransaction.RecurringTransaction{
		recurringOn("Aluguel", 5, recurringtransaction.StatusActive),     // due and unpaid
		recurringOn("Tim Celular", 8, recurringtransaction.StatusActive), // due, already paid
		recurringOn("IPTV", 25, recurringtransaction.StatusActive),       // not due yet
		recurringOn("Academia", 5, recurringtransaction.StatusPaused),    // paused
	}
	paid := []*transaction.Transaction{{
		Type:        transaction.TypeExpense,
		Amount:      111.57,
		Description: "Tim Celular",
		OccurredOn:  time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC),
		Status:      transaction.StatusConfirmed,
	}}

	uc := NewListPendingRecurringsUseCase(&pendingRecurringRepo{list: recurrings}, &cashflowTxRepo{txs: paid})

	out, err := uc.Execute(ListPendingRecurringsInput{ProfileID: "p1", Reference: "2026-09-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("expected exactly one pending obligation, got %d: %+v", len(out), out)
	}
	if out[0].Description != "Aluguel" {
		t.Fatalf("expected Aluguel pending, got %q", out[0].Description)
	}
	if out[0].DaysOverdue != 5 {
		t.Errorf("expected 5 days overdue, got %d", out[0].DaysOverdue)
	}
}

func TestListPendingRecurrings_NothingDueYet(t *testing.T) {
	uc := NewListPendingRecurringsUseCase(
		&pendingRecurringRepo{list: []*recurringtransaction.RecurringTransaction{
			recurringOn("IPTV", 25, recurringtransaction.StatusActive),
		}},
		&cashflowTxRepo{},
	)

	out, err := uc.Execute(ListPendingRecurringsInput{ProfileID: "p1", Reference: "2026-09-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected nothing pending, got %d", len(out))
	}
}

func TestListPendingRecurrings_RequiresAProfile(t *testing.T) {
	uc := NewListPendingRecurringsUseCase(&pendingRecurringRepo{}, &cashflowTxRepo{})

	if _, err := uc.Execute(ListPendingRecurringsInput{Reference: "2026-09-10"}); err == nil {
		t.Fatal("expected an error when no profile is given")
	}
}
