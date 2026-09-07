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

	TransactionsAffected int    `json:"transactionsAffected"`
	RequiresApproval     bool   `json:"requiresApproval"`
	Reason               string `json:"reason"`
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
	for _, inv := range existing {
		idx := bestCycleFor(cycles, inv)
		if idx < 0 {
			continue
		}
		if matched[idx] != nil {
			// Two invoices claiming one cycle is an overlap, which the invariant
			// report names. Repairing it is not this planner's call.
			plan.SafeToApplyUnattended = false
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
			plan.Actions = append(plan.Actions, action)
			continue
		}

		if sameDay(current.OpeningDate, cycle.OpeningDate) &&
			sameDay(current.ClosingDate, cycle.ClosingDate) &&
			sameDay(current.DueDate, cycle.DueDate) {
			continue
		}

		action.Kind = RebuildReshapeWindow
		action.InvoiceID = current.ID
		action.InvoiceStatus = string(current.Status)
		action.InvoiceLabel = monthKey(current.ReferenceDate)
		action.CurrentOpening = &current.OpeningDate
		action.CurrentClosing = &current.ClosingDate
		action.CurrentDue = &current.DueDate
		action.Reason = windowReason(current, cycle)
		action.RequiresApproval = wasEverSettled(current)
		plan.Actions = append(plan.Actions, action)
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
// date settles it; otherwise the cycle it overlaps most does. A fused invoice covering
// two cycles therefore claims the later one, and the earlier one is reported missing —
// which is what it is.
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

func sortedKeys(m map[string]time.Time) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
