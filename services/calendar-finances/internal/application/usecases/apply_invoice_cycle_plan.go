package usecases

import (
	"errors"
	"fmt"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ApplyCyclesReport says what the repair did. Every count is a decision, so a run
// can be argued with after the fact as well as before.
type ApplyCyclesReport struct {
	BankAccountID string `json:"bankAccountId"`
	// Reshaped counts invoices whose window was corrected.
	Reshaped int `json:"reshaped"`
	// Created counts cycles that had no invoice and now do.
	Created int `json:"created"`
	// Moved counts purchases that changed bills because a window moved under them.
	// Reshaping without this is half a repair: the line stays on a bill that no
	// longer covers its date, which is the defect the reshape exists to fix.
	Moved int `json:"moved"`
	// RefusedInformational counts named actions that must never be applied. A cycle
	// whose only difference is a due date reflects the card's terms AS THEY WERE;
	// rewriting it replaces a historical fact with today's configuration.
	RefusedInformational int `json:"refusedInformational"`
	// RefusedNeedsPerson counts named actions of a kind this repair will not decide.
	// Two invoices claiming one cycle is a question about which one wins, and the
	// data cannot answer it. Counted apart from the informational refusals because
	// they are different reasons — and because two branches that report the same
	// number are two branches no test can tell apart.
	RefusedNeedsPerson int `json:"refusedNeedsPerson"`
	// NotSelected counts actions the plan proposed and the caller did not name.
	NotSelected  int                 `json:"notSelected"`
	Recalculated []string            `json:"recalculated,omitempty"`
	StaleTotals  []StaleInvoiceTotal `json:"staleTotals,omitempty"`
	Details      []string            `json:"details"`
	// Failed records actions that could not be applied, WITHOUT stopping the rest.
	// An abort mid-run leaves the card half-repaired and destroys the report of
	// what was already done — state changed, and no record of which change. Each
	// action stands alone, so one failing is a fact to report, not a reason to
	// abandon the others.
	Failed []FailedAction `json:"failed,omitempty"`
}

// ApplyInvoiceCyclePlanUseCase applies the repairs that RebuildInvoiceCyclesUseCase
// only reports.
//
// It is deliberately a second use case rather than a flag on the first: the plan
// changes nothing and can be run by anyone at any time, and keeping that property
// is worth more than the convenience of one entry point.
// FailedAction is one repair that could not be applied, and why.
type FailedAction struct {
	ReferenceDate string `json:"referenceDate"`
	Kind          string `json:"kind"`
	Error         string `json:"error"`
}

type ApplyInvoiceCyclePlanUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
	planner     *RebuildInvoiceCyclesUseCase
	recalc      *RecalculateInvoiceAmountUseCase
}

func NewApplyInvoiceCyclePlanUseCase(
	accountRepo bankaccount.Repository,
	txRepo transactionPkg.Repository,
	invoiceRepo invoice.Repository,
) *ApplyInvoiceCyclePlanUseCase {
	return &ApplyInvoiceCyclePlanUseCase{
		accountRepo: accountRepo, txRepo: txRepo, invoiceRepo: invoiceRepo,
		planner: NewRebuildInvoiceCyclesUseCase(accountRepo, txRepo, invoiceRepo),
		recalc:  NewRecalculateInvoiceAmountUseCase(invoiceRepo, txRepo),
	}
}

// Execute applies the named actions, identified by their reference date (YYYY-MM-DD).
//
// The reference date is the key because CREATE_MISSING has no invoice id yet — the
// whole point of it is that no invoice exists.
func (uc *ApplyInvoiceCyclePlanUseCase) Execute(bankAccountID string, approve []string) (*ApplyCyclesReport, error) {
	named := map[string]bool{}
	for _, d := range approve {
		named[d] = true
	}
	if len(named) == 0 {
		return nil, errors.New("this repair needs an explicit list of cycle reference dates: run the plan, read it, then name the cycles")
	}

	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("reading the account: %w", err)
	}
	if account == nil {
		return nil, errors.New("bank account not found")
	}
	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, ErrNotACreditCard
	}

	plan, err := uc.planner.Plan(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("building the plan: %w", err)
	}

	report := &ApplyCyclesReport{BankAccountID: bankAccountID, Details: []string{}}
	touched := map[string]bool{}

	for _, action := range plan.Actions {
		key := action.ReferenceDate.Format("2006-01-02")
		if !named[key] {
			report.NotSelected++
			continue
		}
		if action.Informational {
			report.RefusedInformational++
			report.Details = append(report.Details, fmt.Sprintf(
				"%s %s refused: %s", key, action.Kind, action.Reason))
			continue
		}

		switch action.Kind {
		case RebuildReshapeWindow, RebuildAlignOpening:
			inv, err := uc.invoiceRepo.FindByID(action.InvoiceID)
			if err != nil {
				report.Failed = append(report.Failed, FailedAction{key, action.Kind, err.Error()})
				continue
			}
			inv.OpeningDate = action.CanonicalOpening
			inv.ClosingDate = action.CanonicalClosing
			if err := uc.invoiceRepo.Update(inv); err != nil {
				report.Failed = append(report.Failed, FailedAction{key, action.Kind, err.Error()})
				continue
			}
			report.Reshaped++
			touched[inv.ID] = true
			report.Details = append(report.Details, fmt.Sprintf(
				"%s reshaped %s..%s -> %s..%s", key,
				action.CurrentOpening.Format("2006-01-02"), action.CurrentClosing.Format("2006-01-02"),
				action.CanonicalOpening.Format("2006-01-02"), action.CanonicalClosing.Format("2006-01-02")))

		case RebuildCreateMissing:
			created, err := invoice.New(invoice.CreateParams{
				BankAccountID: bankAccountID,
				ClosingDay:    *account.ClosingDay,
				DueDay:        *account.DueDay,
				ReferenceDate: action.ReferenceDate,
			})
			if err != nil {
				report.Failed = append(report.Failed, FailedAction{key, action.Kind, err.Error()})
				continue
			}
			// The reference date is a LABEL, and a card with legacy cycles already
			// has rows holding the obvious one — creating this directly collided with
			// uq_invoice_account_reference and took the whole run down with it. The
			// same helper every other creation path uses walks the free labels and
			// recovers from a concurrent winner.
			stored, err := createInvoiceWithFreeLabel(
				uc.invoiceRepo, created, action.CanonicalOpening, action.ReferenceDate)
			if err != nil {
				report.Failed = append(report.Failed, FailedAction{key, action.Kind, err.Error()})
				continue
			}
			report.Created++
			touched[stored.ID] = true
			report.Details = append(report.Details, fmt.Sprintf("%s created the missing cycle", key))

		default:
			// OVERLAPPING_CYCLES and anything new: two invoices claiming one cycle
			// is a question about which one wins, and that is not a decision a
			// repair can take from the data alone.
			report.RefusedNeedsPerson++
			report.Details = append(report.Details, fmt.Sprintf(
				"%s %s refused: needs a person to decide", key, action.Kind))
		}
	}

	moved, err := uc.refileAgainstNewWindows(account, touched)
	if err != nil {
		return nil, err
	}
	report.Moved = moved

	if err := uc.settleTotals(touched, report); err != nil {
		return nil, err
	}
	return report, nil
}

// refileAgainstNewWindows moves every live line to the invoice whose window now
// contains its date. Moving a boundary without this leaves lines on bills that no
// longer cover them — the same defect, one step to the side.
func (uc *ApplyInvoiceCyclePlanUseCase) refileAgainstNewWindows(
	account *bankaccount.BankAccount, touched map[string]bool,
) (int, error) {
	if len(touched) == 0 {
		return 0, nil
	}
	accountID := account.ID
	txns, err := uc.txRepo.List(transactionPkg.ListFilter{
		ProfileID: account.ProfileID, BankAccountID: &accountID,
	})
	if err != nil {
		return 0, fmt.Errorf("listing the card's transactions: %w", err)
	}

	moved := 0
	for _, txn := range txns {
		if txn.BankAccountID != account.ID || txn.PaidInvoiceID != nil {
			continue
		}
		if txn.Status == transactionPkg.StatusCancelled || txn.Status == transactionPkg.StatusReversed {
			continue
		}
		if !isInvoiceLine(txn.Type) {
			continue
		}
		target, err := uc.invoiceRepo.FindByBankAccountAndDate(account.ID, txn.OccurredOn)
		if err != nil {
			return 0, fmt.Errorf("finding the bill for %s: %w", txn.ID, err)
		}
		var want *string
		if target != nil {
			want = &target.ID
		}
		if sameInvoice(txn.InvoiceID, want) {
			continue
		}
		// Only lines whose OLD or NEW bill was touched are in scope. A line
		// elsewhere on the card is none of this repair's business.
		if !inScope(txn.InvoiceID, want, touched) {
			continue
		}
		if txn.InvoiceID != nil {
			touched[*txn.InvoiceID] = true
		}
		txn.InvoiceID = want
		if want != nil {
			touched[*want] = true
		}
		if err := uc.txRepo.Update(txn); err != nil {
			return 0, fmt.Errorf("refiling %s: %w", txn.ID, err)
		}
		moved++
	}
	return moved, nil
}

// settleTotals refreshes the stored total of every bill this run touched, and
// reports the settled ones instead of rewriting them.
func (uc *ApplyInvoiceCyclePlanUseCase) settleTotals(touched map[string]bool, report *ApplyCyclesReport) error {
	for invoiceID := range touched {
		_, err := uc.recalc.Execute(invoiceID)
		if err == nil {
			report.Recalculated = append(report.Recalculated, invoiceID)
			continue
		}
		if !errors.Is(err, ErrInvoiceAlreadyPaid) {
			return fmt.Errorf("refreshing bill %s: %w", invoiceID, err)
		}
		inv, findErr := uc.invoiceRepo.FindByID(invoiceID)
		if findErr != nil {
			return fmt.Errorf("reading settled bill %s: %w", invoiceID, findErr)
		}
		computed, sumErr := uc.txRepo.SumByInvoiceID(invoiceID)
		if sumErr != nil {
			return fmt.Errorf("summing settled bill %s: %w", invoiceID, sumErr)
		}
		if diff := inv.Amount - computed; diff > 0.005 || diff < -0.005 {
			report.StaleTotals = append(report.StaleTotals, StaleInvoiceTotal{
				InvoiceID: invoiceID, Status: string(inv.Status),
				Stored: inv.Amount, Computed: computed,
			})
		}
	}
	return nil
}

func sameInvoice(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func inScope(old, want *string, touched map[string]bool) bool {
	if old != nil && touched[*old] {
		return true
	}
	return want != nil && touched[*want]
}
