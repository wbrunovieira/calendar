package recurringtransaction

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func onDay(day int) time.Time { return time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC) }

func obligation() *RecurringTransaction {
	acc := "checking"
	return &RecurringTransaction{
		ProfileID:      "p1",
		BankAccountID:  &acc,
		Type:           "EXPENSE",
		Amount:         99,
		Description:    "Tim Celular",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=20",
		NextOccurrence: onDay(20),
		Status:         StatusActive,
	}
}

func payment(desc string, amount float64, day int) *transaction.Transaction {
	acc := "checking"
	return &transaction.Transaction{
		BankAccountID: acc,
		Type:          transaction.TypeExpense,
		Amount:        amount,
		Description:   desc,
		OccurredOn:    onDay(day),
		Status:        transaction.StatusConfirmed,
	}
}

// The amount is deliberately not compared: a phone bill quoted at 99 arrives at
// 111,57. Comparing it kept the obligation "pending" after it had been paid.
func TestIsSatisfiedBy_IgnoresTheAmount(t *testing.T) {
	if !obligation().IsSatisfiedBy(payment("Tim Celular", 111.57, 20)) {
		t.Fatal("a recurring bill paid at its real amount is satisfied")
	}
}

func TestIsSatisfiedBy_AcceptsAPaymentNearTheDueDate(t *testing.T) {
	if !obligation().IsSatisfiedBy(payment("Tim Celular", 99, 12)) {
		t.Fatal("paid 8 days early is still this month's payment")
	}
}

// A bill due at the end of a month is often paid in the first days of the next one.
func TestIsSatisfiedBy_AcceptsTheWindowAcrossMonths(t *testing.T) {
	obl := obligation()
	obl.NextOccurrence = time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)

	if !obl.IsSatisfiedBy(payment("Tim Celular", 99, 3)) {
		t.Fatal("paid 4 days after it was due, in the next month, still satisfies it")
	}
}

func TestIsSatisfiedBy_RejectsADistantPayment(t *testing.T) {
	obl := obligation()
	obl.NextOccurrence = time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	if obl.IsSatisfiedBy(payment("Tim Celular", 99, 20)) {
		t.Fatal("last month's obligation is not settled by this month's payment")
	}
}

// The same calendar month is NOT enough. Rent due on the 30th and a payment on the
// 2nd are 28 days apart: that payment settled the PREVIOUS occurrence, and treating
// it as this one makes September's rent disappear from the pending list.
func TestIsSatisfiedBy_RejectsThePreviousOccurrencePaidLateInTheSameMonth(t *testing.T) {
	obl := obligation()
	obl.Description = "Aluguel"
	obl.NextOccurrence = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	if obl.IsSatisfiedBy(payment("Aluguel", 2000, 2)) {
		t.Fatal("a payment 28 days before the due date belongs to the previous occurrence")
	}
}

// Two different bills can share a recurrence rule — same day of month, same account.
// The rule string is not an identifier, so it cannot stand in for one.
func TestIsSatisfiedBy_RejectsAnotherBillWithTheSameRule(t *testing.T) {
	internet := obligation()
	internet.Description = "Internet Vivo"

	gym := payment("Academia", 99, 20)
	rule := "FREQ=MONTHLY;BYMONTHDAY=20"
	gym.RecurrenceRule = &rule

	if internet.IsSatisfiedBy(gym) {
		t.Fatal("the gym bill must not settle the internet bill just because both fall on the 20th")
	}
}

func TestIsSatisfiedBy_RejectsAnotherBill(t *testing.T) {
	if obligation().IsSatisfiedBy(payment("Aluguel", 99, 20)) {
		t.Fatal("a different description is a different obligation")
	}
}

func TestIsSatisfiedBy_RejectsTheWrongAccount(t *testing.T) {
	tx := payment("Tim Celular", 99, 20)
	tx.BankAccountID = "other-account"

	if obligation().IsSatisfiedBy(tx) {
		t.Fatal("an obligation bound to an account is not settled from another one")
	}
}

func TestIsSatisfiedBy_RejectsTheOppositeDirection(t *testing.T) {
	tx := payment("Tim Celular", 99, 20)
	tx.Type = transaction.TypeIncome

	if obligation().IsSatisfiedBy(tx) {
		t.Fatal("income does not settle an expense obligation")
	}
}

func TestIsSatisfiedBy_RejectsACancelledTransaction(t *testing.T) {
	tx := payment("Tim Celular", 99, 20)
	tx.Status = transaction.StatusCancelled

	if obligation().IsSatisfiedBy(tx) {
		t.Fatal("a cancelled transaction settles nothing")
	}
}

// A scheduled-but-unpaid entry still counts: the obligation is already accounted for,
// so nagging about it again would be noise.
func TestIsSatisfiedBy_AcceptsAPlannedTransaction(t *testing.T) {
	tx := payment("Tim Celular", 99, 20)
	tx.Status = transaction.StatusPlanned

	if !obligation().IsSatisfiedBy(tx) {
		t.Fatal("an already-scheduled entry satisfies the obligation")
	}
}

// Matching is by description. The recurrence rule is deliberately NOT used as a
// fallback: it is a plain string copied onto every generated transaction, so two
// unrelated bills that fall on the same day of the month share it. Until a
// transaction carries the id of the recurrence that generated it, a renamed
// transaction stays pending — which is noise, where the alternative is silently
// marking the wrong bill as paid.
func TestIsSatisfiedBy_DoesNotMatchARenamedTransaction(t *testing.T) {
	tx := payment("TIM - conta de setembro", 111.57, 20)
	rule := "FREQ=MONTHLY;BYMONTHDAY=20"
	tx.RecurrenceRule = &rule

	if obligation().IsSatisfiedBy(tx) {
		t.Fatal("without an id linking them, a rename must not be assumed to be the same bill")
	}
}

func TestIsDue_OnlyActiveObligationsThatAlreadyCameAround(t *testing.T) {
	obl := obligation()

	if !obl.IsDue(onDay(20)) {
		t.Error("an obligation due today is due")
	}
	if !obl.IsDue(onDay(25)) {
		t.Error("an overdue obligation is due")
	}
	if obl.IsDue(onDay(19)) {
		t.Error("an obligation that has not come around yet is not due")
	}

	obl.Status = StatusPaused
	if obl.IsDue(onDay(25)) {
		t.Error("a paused obligation is never due")
	}
}
