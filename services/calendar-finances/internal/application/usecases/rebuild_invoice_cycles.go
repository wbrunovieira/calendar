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

	existing, err := uc.invoiceRepo.FindByBankAccountID(bankAccountID)
	if err != nil {
		return nil, err
	}
	byLabel := map[string]*invoice.Invoice{}
	for _, inv := range existing {
		byLabel[monthKey(inv.ReferenceDate)] = inv
	}

	perLabel, err := uc.countPurchasesPerCycle(account)
	if err != nil {
		return nil, err
	}

	// A label deserves a look if purchases point at it or an invoice claims it. The
	// second half is what catches an invoice built on the wrong closing day: its own
	// month may hold no purchase at all, precisely because the window is wrong.
	labels := map[string]time.Time{}
	for key, ref := range perLabel.refs {
		labels[key] = ref
	}
	for key, inv := range byLabel {
		labels[key] = firstOfMonth(inv.ReferenceDate)
	}

	plan := &RebuildPlan{
		BankAccountID: account.ID,
		AccountName:   account.Name,
		ClosingDay:    *account.ClosingDay,
		DueDay:        *account.DueDay,
		Actions:       []RebuildAction{},
	}

	for _, key := range sortedKeys(labels) {
		ref := labels[key]
		canonical, err := invoice.New(invoice.CreateParams{
			BankAccountID: account.ID,
			ClosingDay:    *account.ClosingDay,
			DueDay:        *account.DueDay,
			ReferenceDate: ref,
		})
		if err != nil {
			return nil, err
		}

		action := RebuildAction{
			ReferenceDate:        ref,
			CanonicalOpening:     canonical.OpeningDate,
			CanonicalClosing:     canonical.ClosingDate,
			CanonicalDue:         canonical.DueDate,
			TransactionsAffected: perLabel.counts[key],
		}

		current, ok := byLabel[key]
		if !ok {
			action.Kind = RebuildCreateMissing
			action.Reason = "no invoice covers this cycle, so its purchases reach no bill"
			plan.Actions = append(plan.Actions, action)
			continue
		}

		if sameDay(current.OpeningDate, canonical.OpeningDate) &&
			sameDay(current.ClosingDate, canonical.ClosingDate) &&
			sameDay(current.DueDate, canonical.DueDate) {
			continue
		}

		action.Kind = RebuildReshapeWindow
		action.InvoiceID = current.ID
		action.InvoiceStatus = string(current.Status)
		action.CurrentOpening = &current.OpeningDate
		action.CurrentClosing = &current.ClosingDate
		action.CurrentDue = &current.DueDate
		action.Reason = windowReason(current, canonical)
		action.RequiresApproval = wasEverSettled(current)
		plan.Actions = append(plan.Actions, action)
	}

	plan.SafeToApplyUnattended = true
	for _, a := range plan.Actions {
		if a.RequiresApproval {
			plan.SafeToApplyUnattended = false
			break
		}
	}
	return plan, nil
}

type cycleCounts struct {
	counts map[string]int
	refs   map[string]time.Time
}

// countPurchasesPerCycle maps each live purchase to the cycle it belongs in by the
// card's own closing day — which is the same rule new purchases follow, so the plan
// and the running system agree on where a charge goes.
func (uc *RebuildInvoiceCyclesUseCase) countPurchasesPerCycle(account *bankaccount.BankAccount) (cycleCounts, error) {
	out := cycleCounts{counts: map[string]int{}, refs: map[string]time.Time{}}

	accountID := account.ID
	txns, err := uc.txRepo.List(transactionPkg.ListFilter{ProfileID: account.ProfileID, BankAccountID: &accountID})
	if err != nil {
		return out, err
	}

	for _, txn := range txns {
		if txn.BankAccountID != account.ID {
			continue
		}
		if txn.Status == transactionPkg.StatusCancelled || txn.Status == transactionPkg.StatusReversed {
			continue
		}
		ref := calculateReferenceMonth(txn.OccurredOn, *account.ClosingDay)
		key := monthKey(ref)
		out.counts[key]++
		out.refs[key] = ref
	}
	return out, nil
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
