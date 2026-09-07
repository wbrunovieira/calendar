package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// The cycles a card should have are fully determined by its closing and due day.
// Anything else on the card is damage, and the planner's job is to name it without
// touching a thing — the repair is a separate, deliberate act.
//
// The damage this was written against is real: the Nubank Juridica card carries one
// invoice covering 2026-07-28 to 2026-09-27 (two cycles fused), no invoice at all for
// December 2025, and four invoices from 2026-01 to 2026-04 built as if the card closed
// on the 1st when it closes on the 27th.

func rebuildFixture(t *testing.T) (*fakeAccountRepo, *fakeTransactionRepo, *cycleInvoiceRepo) {
	t.Helper()
	return cycleFixture(t)
}

func purchase(id string, on time.Time, amount float64) *transaction.Transaction {
	return &transaction.Transaction{
		ID: id, ProfileID: "p1", BankAccountID: "card",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: amount, Currency: "BRL", Description: "compra", OccurredOn: on,
	}
}

func TestRebuildPlan_AFusedCycleIsSplitIntoTheTwoItHides(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	// One invoice where there should be two.
	invoices.list = []*invoice.Invoice{
		{ID: "fused", BankAccountID: "card", Status: invoice.StatusOpen, ReferenceDate: day(2026, time.September, 1),
			OpeningDate: day(2026, time.July, 28), ClosingDate: day(2026, time.September, 27), DueDate: day(2026, time.October, 6)},
	}
	txns.created = []*transaction.Transaction{
		purchase("t1", day(2026, time.August, 11), 576.03),
		purchase("t2", day(2026, time.September, 2), 40),
	}

	plan, err := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	created := plan.OfKind(RebuildCreateMissing)
	if len(created) != 1 {
		t.Fatalf("the hidden August cycle must be planned, got %d actions: %+v", len(created), plan.Actions)
	}
	if got := created[0].ReferenceDate; !got.Equal(day(2026, time.August, 1)) {
		t.Errorf("the missing cycle is August, got %s", got.Format("2006-01"))
	}
	if got := created[0].CanonicalOpening; !got.Equal(day(2026, time.July, 27)) {
		t.Errorf("August opens the day the July cycle closed, got %s", got.Format("2006-01-02"))
	}
	if got := created[0].CanonicalClosing; !got.Equal(day(2026, time.August, 27)) {
		t.Errorf("August closes on the 27th, got %s", got.Format("2006-01-02"))
	}

	reshaped := plan.OfKind(RebuildReshapeWindow)
	if len(reshaped) != 1 || reshaped[0].InvoiceID != "fused" {
		t.Fatalf("the fused invoice must be narrowed to its own cycle: %+v", plan.Actions)
	}
	if got := reshaped[0].CanonicalOpening; !got.Equal(day(2026, time.August, 27)) {
		t.Errorf("September opens on 27/08, got %s", got.Format("2006-01-02"))
	}
}

// A card that closes on the 27th cannot have invoices closing on the 1st. Four of
// them exist in production, and every purchase in those months sits in the wrong bill.
func TestRebuildPlan_AWindowBuiltOnTheWrongClosingDayIsReshaped(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "wrong", BankAccountID: "card", Status: invoice.StatusClosed, ReferenceDate: day(2026, time.March, 1),
			OpeningDate: day(2026, time.February, 2), ClosingDate: day(2026, time.March, 1), DueDate: day(2026, time.March, 8)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.February, 15), 100)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	reshaped := plan.OfKind(RebuildReshapeWindow)
	if len(reshaped) != 1 {
		t.Fatalf("a window off the card's closing day must be reported: %+v", plan.Actions)
	}
	// The invoice is LABELLED 2026-03, but it covers February purchases: the window is
	// the fact and the label is a name. It belongs to the cycle closing 27/02.
	if !reshaped[0].CanonicalClosing.Equal(day(2026, time.February, 27)) {
		t.Errorf("canonical closing is 27/02, got %s", reshaped[0].CanonicalClosing.Format("2006-01-02"))
	}
}

// A plan that silently reshapes a settled bill is how a paid invoice loses the
// purchases it was paid for. The planner still reports it — it just refuses to call
// it routine.
func TestRebuildPlan_ReshapingASettledBillNeedsAHumanToSayYes(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	paid := 500.0
	invoices.list = []*invoice.Invoice{
		{ID: "paid", BankAccountID: "card", Status: invoice.StatusPaid, Amount: 500, PaidAmount: &paid,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 2), ClosingDate: day(2026, time.March, 1), DueDate: day(2026, time.March, 8)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.February, 15), 500)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	reshaped := plan.OfKind(RebuildReshapeWindow)
	if len(reshaped) != 1 {
		t.Fatalf("expected one reshape, got %+v", plan.Actions)
	}
	if !reshaped[0].RequiresApproval {
		t.Error("moving the window of a settled bill is not a routine repair")
	}
	if plan.SafeToApplyUnattended {
		t.Error("a plan containing a settled bill must not read as safe")
	}
}

func TestRebuildPlan_ACardAlreadyCorrectPlansNothing(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "ok", BankAccountID: "card", Status: invoice.StatusOpen, ReferenceDate: day(2026, time.March, 1),
			OpeningDate: day(2026, time.February, 27), ClosingDate: day(2026, time.March, 27), DueDate: day(2026, time.April, 3)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.March, 15), 100)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	if len(plan.Actions) != 0 {
		t.Errorf("a healthy card needs no repair, got %+v", plan.Actions)
	}
	if !plan.SafeToApplyUnattended {
		t.Error("an empty plan is trivially safe")
	}
}

// A gap is as damaging as an overlap and less visible: purchases in an uncovered
// month reach no bill at all.
func TestRebuildPlan_AMonthWithPurchasesAndNoInvoiceIsCreated(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = nil
	txns.created = []*transaction.Transaction{
		purchase("t1", day(2025, time.December, 10), 200),
		purchase("t2", day(2026, time.January, 10), 300),
	}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	created := plan.OfKind(RebuildCreateMissing)
	if len(created) != 2 {
		t.Fatalf("both uncovered months must be planned, got %d: %+v", len(created), plan.Actions)
	}
	if created[0].TransactionsAffected != 1 || created[1].TransactionsAffected != 1 {
		t.Errorf("each cycle carries one purchase: %+v", created)
	}
}

// Only a credit card has cycles. Asking for a rebuild on a checking account is a
// caller mistake, and answering with an empty plan would hide it.
func TestRebuildPlan_OnlyACreditCardHasCyclesToRebuild(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	accounts.accounts["checking"] = checkingAccountForRebuild()

	if _, err := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("checking"); err == nil {
		t.Fatal("a checking account has no invoice cycles")
	}
}

// A cancelled purchase does not justify a bill.
func TestRebuildPlan_ACancelledPurchaseDoesNotCreateACycle(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	cancelled := purchase("t1", day(2026, time.May, 10), 100)
	cancelled.Status = transaction.StatusCancelled
	txns.created = []*transaction.Transaction{cancelled}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	if len(plan.Actions) != 0 {
		t.Errorf("a cancelled purchase needs no bill: %+v", plan.Actions)
	}
}

func checkingAccountForRebuild() *bankaccount.BankAccount {
	return &bankaccount.BankAccount{
		ID: "checking", ProfileID: "p1", Name: "Conta",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL",
	}
}

// A correct invoice must not be reported as damage.
//
// reference_date carries two conventions in this database: older rows are labelled by
// due month, invoice.New by closing month. Matching on it made a card with nothing
// wrong produce two actions — create a duplicate covering a window that already
// existed, and push the correct invoice a month forward, orphaning everything inside
// it — and call the whole thing safe to apply unattended.
func TestRebuildPlan_AnInvoiceLabelledByDueMonthIsNotDamage(t *testing.T) {
	closing, due := 27, 6
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": {ID: "card", ProfileID: "p1", Name: "Cartao", Type: bankaccount.AccountTypeCreditCard,
			ClosingDay: &closing, DueDay: &due, Currency: "BRL"},
	}}
	txns := &fakeTransactionRepo{}
	invoices := &cycleInvoiceRepo{}

	// The cycle 27/08 -> 27/09 is due on 06/10, so the old convention labels it 2026-10.
	invoices.list = []*invoice.Invoice{
		{ID: "old-convention", BankAccountID: "card", Status: invoice.StatusOpen,
			ReferenceDate: day(2026, time.October, 1),
			OpeningDate:   day(2026, time.August, 27), ClosingDate: day(2026, time.September, 27),
			DueDate: day(2026, time.October, 6)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.September, 2), 40)}

	plan, err := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Actions) != 0 {
		t.Errorf("a correct cycle labelled by due month is not damage, got %+v", plan.Actions)
	}
}

// The label is reported, so a human reading the plan can find the row — it is just
// never what the match is decided on.
func TestRebuildPlan_TheStoredLabelIsReportedButNotMatchedOn(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "wrong", BankAccountID: "card", Status: invoice.StatusClosed,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 2), ClosingDate: day(2026, time.March, 1),
			DueDate: day(2026, time.March, 8)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.February, 15), 100)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")
	reshaped := plan.OfKind(RebuildReshapeWindow)
	if len(reshaped) != 1 || reshaped[0].InvoiceLabel != "2026-03" {
		t.Errorf("the stored label should be reported: %+v", plan.Actions)
	}
}
