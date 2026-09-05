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

// The cron generates a PLANNED transaction and immediately advances NextOccurrence.
// The obligation then stops being "due" although nothing was paid, and it used to
// vanish from this endpoint — which promises what is still owed, not what has yet to
// be scheduled.
func TestListPendingRecurrings_SurvivesTheCronAdvancingTheOccurrence(t *testing.T) {
	obligation := recurringOn("Aluguel", 5, recurringtransaction.StatusActive)
	obligation.NextOccurrence = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC) // already advanced

	scheduled := []*transaction.Transaction{{
		Type:        transaction.TypeExpense,
		Amount:      2000,
		Description: "Aluguel",
		OccurredOn:  time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
		Status:      transaction.StatusPlanned, // generated, not paid
	}}

	uc := NewListPendingRecurringsUseCase(
		&pendingRecurringRepo{list: []*recurringtransaction.RecurringTransaction{obligation}},
		&cashflowTxRepo{txs: scheduled},
	)

	out, err := uc.Execute(ListPendingRecurringsInput{ProfileID: "p1", Reference: "2026-09-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("expected the unpaid obligation to remain pending, got %d", len(out))
	}
	if out[0].DaysOverdue != 5 {
		t.Errorf("expected the overdue count from the scheduled date (5 days), got %d", out[0].DaysOverdue)
	}
}

// Once it is actually paid it drops off, even though the planned entry may still be
// sitting there.
func TestListPendingRecurrings_DropsOffOncePaid(t *testing.T) {
	obligation := recurringOn("Aluguel", 5, recurringtransaction.StatusActive)
	obligation.NextOccurrence = time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	txs := []*transaction.Transaction{{
		Type:        transaction.TypeExpense,
		Amount:      2000,
		Description: "Aluguel",
		OccurredOn:  time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
		Status:      transaction.StatusConfirmed,
	}}

	uc := NewListPendingRecurringsUseCase(
		&pendingRecurringRepo{list: []*recurringtransaction.RecurringTransaction{obligation}},
		&cashflowTxRepo{txs: txs},
	)

	out, err := uc.Execute(ListPendingRecurringsInput{ProfileID: "p1", Reference: "2026-09-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected nothing pending after payment, got %+v", out)
	}
}

// A planned entry for an occurrence that has not arrived yet is not owed yet.
func TestListPendingRecurrings_IgnoresAFutureScheduledEntry(t *testing.T) {
	obligation := recurringOn("IPTV", 25, recurringtransaction.StatusActive)

	txs := []*transaction.Transaction{{
		Type:        transaction.TypeExpense,
		Amount:      25,
		Description: "IPTV",
		OccurredOn:  time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		Status:      transaction.StatusPlanned,
	}}

	uc := NewListPendingRecurringsUseCase(
		&pendingRecurringRepo{list: []*recurringtransaction.RecurringTransaction{obligation}},
		&cashflowTxRepo{txs: txs},
	)

	out, err := uc.Execute(ListPendingRecurringsInput{ProfileID: "p1", Reference: "2026-09-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected nothing pending before the date arrives, got %+v", out)
	}
}

// A yearly bill tolerates a payment up to 180 days from its date. Reading only 45
// days of history could never find one, so it stayed pending forever.
func TestListPendingRecurrings_SeesAPaymentInsideAYearlyWindow(t *testing.T) {
	yearly := recurringOn("Seguro anual", 5, recurringtransaction.StatusActive)
	yearly.RecurrenceRule = "FREQ=YEARLY;BYMONTHDAY=5"
	yearly.NextOccurrence = time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)

	paid := []*transaction.Transaction{{
		Type:        transaction.TypeExpense,
		Amount:      1200,
		Description: "Seguro anual",
		OccurredOn:  time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), // 59 days early
		Status:      transaction.StatusConfirmed,
	}}

	uc := NewListPendingRecurringsUseCase(
		&pendingRecurringRepo{list: []*recurringtransaction.RecurringTransaction{yearly}},
		&cashflowTxRepo{txs: paid},
	)

	out, err := uc.Execute(ListPendingRecurringsInput{ProfileID: "p1", Reference: "2026-09-10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected the early payment to settle the yearly bill, got %+v", out)
	}
}
