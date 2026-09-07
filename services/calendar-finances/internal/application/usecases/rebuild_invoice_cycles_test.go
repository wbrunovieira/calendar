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
// purchases it was paid for — but only when the reshape actually moves one. What
// needs a human is money changing bills, not a boundary being tidied.
func TestRebuildPlan_ReshapingASettledBillThatMovesMoneyNeedsAHumanToSayYes(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	paid := 500.0
	// Stored window runs a month late: it closes on 01/03 where the card closes 27/02.
	invoices.list = []*invoice.Invoice{
		{ID: "paid", BankAccountID: "card", Status: invoice.StatusPaid, Amount: 500, PaidAmount: &paid,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 2), ClosingDate: day(2026, time.March, 1),
			DueDate: day(2026, time.March, 8)},
	}
	// This purchase sits inside the stored window and outside the canonical one, so
	// replacing the window takes it off the bill that was paid for it.
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.February, 28), 500)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	reshaped := plan.OfKind(RebuildReshapeWindow)
	if len(reshaped) != 1 {
		t.Fatalf("expected one reshape, got %+v", plan.Actions)
	}
	if reshaped[0].PurchasesAtRisk != 1 {
		t.Fatalf("the purchase that changes bills must be counted, got %d", reshaped[0].PurchasesAtRisk)
	}
	if !reshaped[0].RequiresApproval {
		t.Error("taking a purchase off a settled bill is not a routine repair")
	}
	if plan.SafeToApplyUnattended {
		t.Error("a plan that moves money out of a paid bill must not read as safe")
	}
}

// Reshaping a settled bill needs a human even when nothing moves.
//
// The window differs, so applying it would rewrite the date a paid bill closed or fell
// due — replacing a historical fact with today's configuration. Against the real
// database this was the difference between "safe to apply unattended" and not: two of
// three cards flipped to safe while still carrying a fused invoice.
func TestRebuildPlan_ReshapingASettledBillNeedsAHumanEvenIfNothingMoves(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	paid := 500.0
	invoices.list = []*invoice.Invoice{
		{ID: "paid", BankAccountID: "card", Status: invoice.StatusPaid, Amount: 500, PaidAmount: &paid,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 2), ClosingDate: day(2026, time.March, 1),
			DueDate: day(2026, time.March, 8)},
	}
	// Mid-February: inside both the stored window and the canonical one.
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.February, 15), 500)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	reshaped := plan.OfKind(RebuildReshapeWindow)
	if len(reshaped) != 1 || reshaped[0].PurchasesAtRisk != 0 {
		t.Fatalf("nothing changes bills here: %+v", plan.Actions)
	}
	if !reshaped[0].RequiresApproval {
		t.Error("rewriting the dates a paid bill recorded is a decision for a human")
	}
	if plan.SafeToApplyUnattended {
		t.Error("a plan that rewrites a settled bill must not read as safe")
	}
}

// A fused invoice covering two months happens to close on the right day. Calling that
// an "alignment" — with a reason describing a one-day nudge — is how it hid on a card
// the plan declared safe.
func TestRebuildPlan_AFusedInvoiceIsNotAnAlignment(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "fundida", BankAccountID: "card", Status: invoice.StatusClosed,
			ReferenceDate: day(2026, time.August, 1),
			// Closes and falls due correctly, but opens two months early.
			OpeningDate: day(2026, time.June, 28), ClosingDate: day(2026, time.August, 27),
			DueDate: day(2026, time.September, 3)},
	}
	txns.created = []*transaction.Transaction{
		purchase("t1", day(2026, time.July, 10), 100),
		purchase("t2", day(2026, time.August, 10), 100),
	}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	if len(plan.OfKind(RebuildAlignOpening)) != 0 {
		t.Errorf("two months out is not a boundary to tidy: %+v", plan.Actions)
	}
	if len(plan.OfKind(RebuildReshapeWindow)) != 1 {
		t.Fatalf("it is a cycle in the wrong place: %+v", plan.Actions)
	}
	if plan.SafeToApplyUnattended {
		t.Error("a fused invoice must never read as safe to apply unattended")
	}
}

// A due date that moved says the card's terms changed, not that the bill is wrong.
//
// Canonical cycles come from the closing and due day the card has TODAY. Correcting
// the Nubank Juridica card's due day from the 6th to the 3rd instantly made every
// older invoice differ — and every one of them was right for its time.
func TestRebuildPlan_ADueDateFromOldTermsIsNotDamage(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t) // card closes 27, due 3
	invoices.list = []*invoice.Invoice{
		{ID: "old-terms", BankAccountID: "card", Status: invoice.StatusClosed,
			ReferenceDate: day(2026, time.August, 1),
			OpeningDate:   day(2026, time.June, 27), ClosingDate: day(2026, time.July, 27),
			DueDate: day(2026, time.August, 6)}, // due day was 6 back then
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.July, 10), 100)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")

	changed := plan.OfKind(RebuildTermsChanged)
	if len(changed) != 1 {
		t.Fatalf("expected the difference to be named a change of terms, got %+v", plan.Actions)
	}
	if changed[0].PurchasesAtRisk != 0 {
		t.Errorf("a due date decides nothing about which purchases belong, got %d", changed[0].PurchasesAtRisk)
	}
	if len(plan.OfKind(RebuildReshapeWindow)) != 0 {
		t.Error("the cycle covers exactly the right days; it is not in the wrong place")
	}
	if !plan.SafeToApplyUnattended {
		t.Error("nothing here moves money")
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

// The shape that dominates the real database: a cycle that closes and falls due on
// exactly the right days, opened one day late by an older convention.
//
// Reported as a reshape it drowned the genuine findings — on Bruno's three cards, 8 of
// 10 actions were this and the other 2 were real. It is not cosmetic either: with
// [opening, closing) half-open, opening a day late leaves one day covered by no
// invoice at all. So it gets its own name and its own weight.
func TestRebuildPlan_ACycleOpenedADayLateIsNamedForWhatItIs(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "off-by-one", BankAccountID: "card", Status: invoice.StatusPaid,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 28), // canonical is 27/02
			ClosingDate:   day(2026, time.March, 27), DueDate: day(2026, time.April, 3)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.March, 15), 100)}

	plan, err := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	aligned := plan.OfKind(RebuildAlignOpening)
	if len(aligned) != 1 {
		t.Fatalf("expected one alignment, got %+v", plan.Actions)
	}
	if len(plan.OfKind(RebuildReshapeWindow)) != 0 {
		t.Error("a boundary a day out is not a cycle in the wrong place")
	}
	if aligned[0].PurchasesAtRisk != 0 {
		t.Errorf("no purchase sits on the disputed day, got %d", aligned[0].PurchasesAtRisk)
	}
	// A settled bill whose boundary moves over nobody reallocates nothing. Demanding a
	// human for it is how the flag stopped meaning anything: every card came back unsafe.
	if aligned[0].RequiresApproval {
		t.Error("moving a boundary no purchase stands on is not a decision for a human")
	}
	if !plan.SafeToApplyUnattended {
		t.Error("a plan that reallocates no money is safe")
	}
}

// The same boundary, with a purchase standing on the disputed day, is a real
// reallocation and does need a human.
func TestRebuildPlan_ADayOutWithAPurchaseOnItNeedsApproval(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "off-by-one", BankAccountID: "card", Status: invoice.StatusPaid,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 28),
			ClosingDate:   day(2026, time.March, 27), DueDate: day(2026, time.April, 3)},
	}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.February, 27), 100)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")
	aligned := plan.OfKind(RebuildAlignOpening)
	if len(aligned) != 1 || aligned[0].PurchasesAtRisk != 1 {
		t.Fatalf("the purchase on the disputed day must be counted: %+v", plan.Actions)
	}
	if !aligned[0].RequiresApproval {
		t.Error("moving a purchase between settled bills is a decision for a human")
	}
}

// Two invoices claiming one cycle used to vanish from the report, leaving a bare
// boolean and no way to find the rows.
func TestRebuildPlan_TwoInvoicesOnOneCycleAreNamed(t *testing.T) {
	accounts, txns, invoices := rebuildFixture(t)
	window := func(id string) *invoice.Invoice {
		return &invoice.Invoice{ID: id, BankAccountID: "card", Status: invoice.StatusClosed,
			ReferenceDate: day(2026, time.March, 1),
			OpeningDate:   day(2026, time.February, 27), ClosingDate: day(2026, time.March, 27),
			DueDate: day(2026, time.April, 3)}
	}
	invoices.list = []*invoice.Invoice{window("first"), window("second")}
	txns.created = []*transaction.Transaction{purchase("t1", day(2026, time.March, 15), 100)}

	plan, _ := NewRebuildInvoiceCyclesUseCase(accounts, txns, invoices).Plan("card")
	clashes := plan.OfKind(RebuildOverlappingCycles)
	if len(clashes) != 1 {
		t.Fatalf("the clash must be reported, got %+v", plan.Actions)
	}
	if clashes[0].InvoiceID == "" || clashes[0].OtherInvoiceID == "" {
		t.Error("both invoices must be named, or nobody can find them")
	}
	if plan.SafeToApplyUnattended {
		t.Error("a clash is not safe to resolve unattended")
	}
}
