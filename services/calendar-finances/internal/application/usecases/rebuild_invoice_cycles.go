package usecases

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Kinds of repair a rebuild plan can propose.
const (
	// RebuildCreateMissing: a cycle that should exist has no invoice, so the purchases
	// inside it reach no bill.
	RebuildCreateMissing = "CREATE_MISSING"
	// RebuildReshapeWindow: an invoice exists for the cycle but covers the wrong dates.
	// This is what a fused cycle looks like — one invoice standing where two belong.
	RebuildReshapeWindow = "RESHAPE_WINDOW"
	// RebuildAlignOpening: the cycle closes and falls due on the right days, and only
	// its opening is off — in practice by one day, from an older convention that
	// opened a cycle the day AFTER the previous one closed.
	//
	// It is separated from a reshape because it is a different size of problem, and
	// because on real data it is most of the volume: reporting the two under one name
	// buried the genuine findings under the boundary ones. It is not cosmetic, though
	// — [opening, closing) is half-open, so opening a day late leaves a one-day gap
	// that belongs to no invoice at all.
	RebuildAlignOpening = "ALIGN_OPENING"
	// RebuildTermsChanged: the cycle covers exactly the right days and only its due
	// date differs from what the card's CURRENT configuration would produce.
	//
	// This is usually not damage at all. A canonical cycle is derived from the closing
	// and due day the card has today, and those change: the Nubank Juridica card's due
	// day was corrected from the 6th to the 3rd, which instantly made every older
	// invoice look wrong. The bill was right for its time. Cycle boundaries are facts
	// about when the bill actually closed, not a function of today's settings.
	RebuildTermsChanged = "TERMS_CHANGED"
	// RebuildOverlappingCycles: two invoices claim the same cycle. Naming it beats the
	// bare flag it replaced, which dropped the second invoice out of the report
	// entirely and left a reader seeing "nothing to repair".
	RebuildOverlappingCycles = "OVERLAPPING_CYCLES"
)

// RebuildAction is one proposed repair. It carries both windows so the difference is
// readable without re-deriving anything.
type RebuildAction struct {
	Kind          string    `json:"kind"`
	ReferenceDate time.Time `json:"referenceDate"`
	InvoiceID     string    `json:"invoiceId,omitempty"`
	InvoiceStatus string    `json:"invoiceStatus,omitempty"`
	// InvoiceLabel is the stored reference_date, reported but never matched on: it is
	// a name, and two conventions for it exist in this database.
	InvoiceLabel string `json:"invoiceLabel,omitempty"`

	CurrentOpening *time.Time `json:"currentOpening,omitempty"`
	CurrentClosing *time.Time `json:"currentClosing,omitempty"`
	CurrentDue     *time.Time `json:"currentDue,omitempty"`

	CanonicalOpening time.Time `json:"canonicalOpening"`
	CanonicalClosing time.Time `json:"canonicalClosing"`
	CanonicalDue     time.Time `json:"canonicalDue"`

	TransactionsAffected int `json:"transactionsAffected"`
	// Informational marks an action that reports and must never be applied. A cycle
	// whose only difference is a due date reflects the card's terms as they were, and
	// rewriting it would replace a historical fact with today's configuration.
	Informational bool `json:"informational"`
	// PurchasesAtRisk counts the live purchases that would change bills if this action
	// were applied. Zero means the repair moves a boundary nobody is standing on.
	PurchasesAtRisk  int    `json:"purchasesAtRisk"`
	RequiresApproval bool   `json:"requiresApproval"`
	Reason           string `json:"reason"`
	OtherInvoiceID   string `json:"otherInvoiceId,omitempty"`
}

// RebuildPlan is the whole diagnosis for one card. It changes nothing.
type RebuildPlan struct {
	BankAccountID string          `json:"bankAccountId"`
	AccountName   string          `json:"accountName"`
	ClosingDay    int             `json:"closingDay"`
	DueDay        int             `json:"dueDay"`
	Actions       []RebuildAction `json:"actions"`

	// SafeToApplyUnattended is false as soon as one action touches a bill that was
	// settled. Moving the window of a paid invoice can take purchases out of the bill
	// that was paid for them, and no automated repair should decide that alone.
	SafeToApplyUnattended bool `json:"safeToApplyUnattended"`
}

// OfKind returns the actions of one kind, in reference-date order.
func (p RebuildPlan) OfKind(kind string) []RebuildAction {
	out := []RebuildAction{}
	for _, a := range p.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// RebuildInvoiceCyclesUseCase reports the difference between the invoice cycles a
// credit card should have and the ones it does.
//
// The cycles are fully determined by the card's closing and due day, so "should" is
// not a judgement call — it is arithmetic. Everything else is damage.
//
// It is deliberately read-only. The bug that produced fused cycles is fixed in code,
// but the fix did not backfill, and repairing money data is not something to do as a
// side effect of a diagnosis.
type RebuildInvoiceCyclesUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
}

func NewRebuildInvoiceCyclesUseCase(
	accountRepo bankaccount.Repository,
	txRepo transactionPkg.Repository,
	invoiceRepo invoice.Repository,
) *RebuildInvoiceCyclesUseCase {
	return &RebuildInvoiceCyclesUseCase{accountRepo: accountRepo, txRepo: txRepo, invoiceRepo: invoiceRepo}
}

// Plan produces the diagnosis for one card.
func (uc *RebuildInvoiceCyclesUseCase) Plan(bankAccountID string) (*RebuildPlan, error) {
	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("bank account not found")
	}
	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, fmt.Errorf("account %s is a %s: only a credit card has invoice cycles", bankAccountID, account.Type)
	}
	if account.ClosingDay == nil || account.DueDay == nil {
		return nil, fmt.Errorf("card %s has no closing/due day set, so its cycles are undefined", bankAccountID)
	}

	plan := &RebuildPlan{
		BankAccountID:         account.ID,
		AccountName:           account.Name,
		ClosingDay:            *account.ClosingDay,
		DueDay:                *account.DueDay,
		Actions:               []RebuildAction{},
		SafeToApplyUnattended: true,
	}

	existing, err := uc.invoiceRepo.FindByBankAccountID(bankAccountID)
	if err != nil {
		return nil, err
	}
	purchases, err := uc.livePurchases(account)
	if err != nil {
		return nil, err
	}

	from, to, any := coveredPeriod(purchases, existing)
	if !any {
		return plan, nil
	}

	cycles, err := canonicalCycles(account, from, to)
	if err != nil {
		return nil, err
	}
	if len(cycles) == 0 {
		return plan, nil
	}

	counts := make([]int, len(cycles))
	for _, txn := range purchases {
		if idx := cycleContaining(cycles, txn.OccurredOn); idx >= 0 {
			counts[idx]++
		}
	}

	// Match by WINDOW, never by the reference label.
	//
	// reference_date is only a name: the repository relabels it on a collision, and
	// two conventions live in this database — older rows are labelled by due month,
	// invoice.New by closing month. Trusting the label made a perfectly correct
	// invoice read as damage and proposed shoving it a month forward, orphaning every
	// purchase inside it.
	matched := make([]*invoice.Invoice, len(cycles))
	overlaps := [][2]*invoice.Invoice{}
	for _, inv := range existing {
		idx := bestCycleFor(cycles, inv)
		if idx < 0 {
			continue
		}
		if matched[idx] != nil {
			// Two invoices claiming one cycle. Repairing it is not this planner's
			// call, but hiding it certainly is not either: the flag alone left a
			// reader seeing an empty plan with a bare boolean and no way to find the
			// rows.
			overlaps = append(overlaps, [2]*invoice.Invoice{matched[idx], inv})
			continue
		}
		matched[idx] = inv
	}

	for i, cycle := range cycles {
		action := RebuildAction{
			ReferenceDate:        cycle.ReferenceDate,
			CanonicalOpening:     cycle.OpeningDate,
			CanonicalClosing:     cycle.ClosingDate,
			CanonicalDue:         cycle.DueDate,
			TransactionsAffected: counts[i],
		}

		current := matched[i]
		if current == nil {
			// A cycle with no purchases needs no bill. Proposing one would fill the
			// card's history with empty invoices for every month it existed.
			if counts[i] == 0 {
				continue
			}
			action.Kind = RebuildCreateMissing
			action.Reason = "no invoice covers this cycle, so its purchases reach no bill"
			// Creating the missing bill is harmless when its purchases belong to no
			// invoice at all, and is a reallocation when they currently sit inside a
			// neighbour that swallowed this cycle. Only the second needs a human, and
			// telling them apart is the difference between a repair and a surprise.
			action.PurchasesAtRisk = purchasesHeldByAnotherInvoice(purchases, cycle, existing)
			action.RequiresApproval = action.PurchasesAtRisk > 0
			if action.RequiresApproval {
				plan.SafeToApplyUnattended = false
			}
			plan.Actions = append(plan.Actions, action)
			continue
		}

		if sameDay(current.OpeningDate, cycle.OpeningDate) &&
			sameDay(current.ClosingDate, cycle.ClosingDate) &&
			sameDay(current.DueDate, cycle.DueDate) {
			continue
		}

		action.InvoiceID = current.ID
		action.InvoiceStatus = string(current.Status)
		action.InvoiceLabel = monthKey(current.ReferenceDate)
		action.CurrentOpening = &current.OpeningDate
		action.CurrentClosing = &current.ClosingDate
		action.CurrentDue = &current.DueDate

		// What is at risk is what would change bills, which is the symmetric difference
		// of the two windows — not everything the cycle contains. Counting the whole
		// cycle made a due date one day out read as "22 purchases at risk" and put
		// every card in the database beyond automatic repair.
		action.PurchasesAtRisk = purchasesMoving(purchases, current, cycle)

		switch {
		case sameDay(current.ClosingDate, cycle.ClosingDate) && sameDay(current.OpeningDate, cycle.OpeningDate):
			action.Kind = RebuildTermsChanged
			action.Informational = true
			action.Reason = "the cycle covers the right days; only the due date differs from the card's current setting, which is what a change of terms looks like"
		case sameDay(current.ClosingDate, cycle.ClosingDate) && sameDay(current.DueDate, cycle.DueDate) &&
			withinDays(current.OpeningDate, cycle.OpeningDate, alignmentTolerance):
			action.Kind = RebuildAlignOpening
			action.Reason = "the cycle closes and falls due correctly; only its opening is off, leaving a gap no invoice covers"
		default:
			action.Kind = RebuildReshapeWindow
			action.Reason = windowReason(current, cycle)
		}

		// Moving a purchase to a different bill always needs a human, whether or not
		// this system believes the bill was paid — because that belief is unreliable.
		// Invoice payments were recorded as plain transfers for months, so bills that
		// were genuinely settled still read as unpaid here. Gating on settlement alone
		// would have let a fused invoice reallocate seven charges unattended.
		//
		// Rewriting the CLOSING or DUE date of a settled bill needs one too, even when
		// nothing moves: those two say what was billed and when it was owed, and
		// replacing them rewrites what the bill recorded. A reshape is exactly the case
		// where one of them differs. An opening nudged by a day changes neither, so it
		// does not — and saying otherwise here while the code did something narrower
		// is the kind of comment that sends the next reader looking for a bug.
		//
		// An informational action needs neither, because it is never applied.
		action.RequiresApproval = !action.Informational &&
			(action.PurchasesAtRisk > 0 ||
				(wasEverSettled(current) && action.Kind == RebuildReshapeWindow))
		plan.Actions = append(plan.Actions, action)
	}

	for _, pair := range overlaps {
		plan.Actions = append(plan.Actions, RebuildAction{
			Kind:             RebuildOverlappingCycles,
			InvoiceID:        pair[0].ID,
			OtherInvoiceID:   pair[1].ID,
			InvoiceStatus:    string(pair[0].Status),
			InvoiceLabel:     monthKey(pair[0].ReferenceDate),
			CanonicalOpening: pair[0].OpeningDate,
			CanonicalClosing: pair[0].ClosingDate,
			CanonicalDue:     pair[0].DueDate,
			RequiresApproval: true,
			Reason:           "two invoices cover the same cycle; a charge in it lands on whichever is found first",
		})
	}

	for _, a := range plan.Actions {
		if a.RequiresApproval {
			plan.SafeToApplyUnattended = false
			break
		}
	}
	return plan, nil
}

// livePurchases returns the charges on the card that still count. A cancelled or
// reversed purchase justifies no bill.
func (uc *RebuildInvoiceCyclesUseCase) livePurchases(account *bankaccount.BankAccount) ([]*transactionPkg.Transaction, error) {
	accountID := account.ID
	txns, err := uc.txRepo.List(transactionPkg.ListFilter{ProfileID: account.ProfileID, BankAccountID: &accountID})
	if err != nil {
		return nil, err
	}
	live := make([]*transactionPkg.Transaction, 0, len(txns))
	for _, txn := range txns {
		if txn.BankAccountID != account.ID {
			continue
		}
		if txn.Status == transactionPkg.StatusCancelled || txn.Status == transactionPkg.StatusReversed {
			continue
		}
		live = append(live, txn)
	}
	return live, nil
}

// coveredPeriod is the span the card's cycles have to cover: everything its live
// purchases touch, plus everything its stored invoices claim. The second half matters
// because an invoice built on the wrong closing day may hold no purchase of its own —
// precisely because its window is wrong.
func coveredPeriod(purchases []*transactionPkg.Transaction, invoices []*invoice.Invoice) (from, to time.Time, any bool) {
	extend := func(t time.Time) {
		if t.IsZero() {
			return
		}
		if !any || t.Before(from) {
			from = t
		}
		if !any || t.After(to) {
			to = t
		}
		any = true
	}
	for _, txn := range purchases {
		extend(txn.OccurredOn)
	}
	for _, inv := range invoices {
		extend(inv.OpeningDate)
		extend(inv.ClosingDate)
	}
	return from, to, any
}

// canonicalCycles builds the cycles the card should have across the period. They come
// from invoice.New, so they are exactly the windows the running system produces for a
// new charge — the plan and the live code cannot disagree about where a purchase goes.
func canonicalCycles(account *bankaccount.BankAccount, from, to time.Time) ([]*invoice.Invoice, error) {
	// One month either side, so a date sitting in the tail of an earlier cycle or the
	// head of a later one still has its cycle in the set.
	first := firstOfMonth(from.AddDate(0, -1, 0))
	last := firstOfMonth(to.AddDate(0, 1, 0))

	cycles := []*invoice.Invoice{}
	for ref := first; !ref.After(last); ref = ref.AddDate(0, 1, 0) {
		cycle, err := invoice.New(invoice.CreateParams{
			BankAccountID: account.ID,
			ClosingDay:    *account.ClosingDay,
			DueDay:        *account.DueDay,
			ReferenceDate: ref,
		})
		if err != nil {
			return nil, err
		}
		cycles = append(cycles, cycle)
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].ClosingDate.Before(cycles[j].ClosingDate) })
	return cycles, nil
}

// cycleContaining finds the cycle a date belongs to, using [opening, closing) — the
// same half-open interval the rest of the system uses, so a purchase made on a closing
// day lands in the cycle that is opening, not the one that just closed.
func cycleContaining(cycles []*invoice.Invoice, at time.Time) int {
	for i, c := range cycles {
		if !at.Before(c.OpeningDate) && at.Before(c.ClosingDate) {
			return i
		}
	}
	return -1
}

// bestCycleFor decides which cycle a stored invoice is trying to be. An exact closing
// date settles it; otherwise the cycle it overlaps most does, and an exact tie goes to
// the earlier one, since the comparison is strict and the cycles are in closing order.
// A fused invoice covering two cycles therefore claims whichever it covers more of,
// and the rest are reported missing — which is what they are.
func bestCycleFor(cycles []*invoice.Invoice, inv *invoice.Invoice) int {
	for i, c := range cycles {
		if sameDay(c.ClosingDate, inv.ClosingDate) {
			return i
		}
	}
	best, bestOverlap := -1, time.Duration(0)
	for i, c := range cycles {
		overlap := overlapBetween(inv.OpeningDate, inv.ClosingDate, c.OpeningDate, c.ClosingDate)
		if overlap > bestOverlap {
			best, bestOverlap = i, overlap
		}
	}
	return best
}

// alignmentTolerance is how far an opening may be out before the difference stops
// being a boundary to tidy and becomes a cycle in the wrong place. A fused invoice
// covering two months happens to close on the right day, and without this it was
// reported as an "alignment" — with a reason describing a one-day nudge.
const alignmentTolerance = 7 * 24 * time.Hour

func withinDays(a, b time.Time, tolerance time.Duration) bool {
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

// purchasesHeldByAnotherInvoice counts the charges that would belong to this cycle and
// currently sit inside some other invoice's window. They are the ones that change bills
// when the missing cycle is created.
func purchasesHeldByAnotherInvoice(
	purchases []*transactionPkg.Transaction,
	cycle *invoice.Invoice,
	existing []*invoice.Invoice,
) int {
	n := 0
	for _, txn := range purchases {
		if txn.OccurredOn.Before(cycle.OpeningDate) || !txn.OccurredOn.Before(cycle.ClosingDate) {
			continue
		}
		for _, inv := range existing {
			if !txn.OccurredOn.Before(inv.OpeningDate) && txn.OccurredOn.Before(inv.ClosingDate) {
				n++
				break
			}
		}
	}
	return n
}

// purchasesMoving counts the live charges that would land on a different bill if the
// window were replaced: those inside one window and outside the other, either way.
//
// A due date that moved changes nothing about which purchases belong, so it counts
// zero — which is the difference between a report worth acting on and a wall of red.
func purchasesMoving(purchases []*transactionPkg.Transaction, current, canonical *invoice.Invoice) int {
	inWindow := func(at time.Time, from, to time.Time) bool {
		return !at.Before(from) && at.Before(to)
	}
	n := 0
	for _, txn := range purchases {
		a := inWindow(txn.OccurredOn, current.OpeningDate, current.ClosingDate)
		b := inWindow(txn.OccurredOn, canonical.OpeningDate, canonical.ClosingDate)
		if a != b {
			n++
		}
	}
	return n
}

func overlapBetween(aFrom, aTo, bFrom, bTo time.Time) time.Duration {
	start := aFrom
	if bFrom.After(start) {
		start = bFrom
	}
	end := aTo
	if bTo.Before(end) {
		end = bTo
	}
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}

// wasEverSettled reports whether money was already recorded against this bill. A
// settled bill is not reshaped as routine maintenance.
func wasEverSettled(inv *invoice.Invoice) bool {
	if inv.Status == invoice.StatusPaid || inv.Status == invoice.StatusPartiallyPaid {
		return true
	}
	return inv.PaidAmount != nil && *inv.PaidAmount > 0
}

func windowReason(current *invoice.Invoice, canonical *invoice.Invoice) string {
	currentSpan := current.ClosingDate.Sub(current.OpeningDate)
	canonicalSpan := canonical.ClosingDate.Sub(canonical.OpeningDate)
	if currentSpan > canonicalSpan+(48*time.Hour) {
		return "this invoice spans more than one cycle: it hides the bills before it"
	}
	return "the window does not match the card's closing and due day"
}

func monthKey(t time.Time) string {
	return t.UTC().Format("2006-01")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}
