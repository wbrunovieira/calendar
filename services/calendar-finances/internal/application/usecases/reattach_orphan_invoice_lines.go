package usecases

import (
	"errors"
	"fmt"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ReattachReport says what the repair did, or would do. Every number is a decision
// the tool made, so a dry run can be read and argued with before anything is written.
type ReattachReport struct {
	BankAccountID string `json:"bankAccountId"`
	Applied       bool   `json:"applied"`
	// Attached is the whole point: lines that belong to a bill and pointed at none.
	Attached int `json:"attached"`
	// NoInvoiceForDate counts lines whose date no existing bill covers. They are
	// REPORTED, never fixed by inventing a bill — creating one for an old period is
	// a separate decision with its own consequences.
	NoInvoiceForDate int `json:"noInvoiceForDate"`
	// SkippedPayments counts invoice payments, which are not lines of a bill. Filing
	// one into the bill would turn a settled bill into a smaller one.
	SkippedPayments int `json:"skippedPayments"`
	// NotSelected counts lines the dry run found that the caller did not name. On a
	// dry run everything lands here, which is the point: the report is the menu.
	NotSelected int             `json:"notSelected"`
	Lines       []ReattachedRow `json:"lines"`
	// Recalculated names the bills whose stored total was refreshed after the
	// attach. Only OPEN bills are refreshed here.
	Recalculated []string `json:"recalculated,omitempty"`
	// StaleTotals names SETTLED bills whose stored total no longer matches the sum
	// of their lines. They are reported and NOT rewritten: recomputing the total of
	// a paid bill rewrites history, and the existing recalculation refuses it on
	// purpose. Correcting one is a decision, made per bill, with these numbers in
	// front of whoever makes it.
	StaleTotals []StaleInvoiceTotal `json:"staleTotals,omitempty"`
}

type StaleInvoiceTotal struct {
	InvoiceID string  `json:"invoiceId"`
	Status    string  `json:"status"`
	Stored    float64 `json:"stored"`
	Computed  float64 `json:"computed"`
}

type ReattachedRow struct {
	TransactionID string  `json:"transactionId"`
	OccurredOn    string  `json:"occurredOn"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"`
	Description   string  `json:"description"`
	InvoiceID     string  `json:"invoiceId,omitempty"`
	Outcome       string  `json:"outcome"`
}

// ReattachOrphanInvoiceLinesUseCase files card lines that belong to a bill and point
// at none.
//
// It exists because the invoice link used to be written only for expenses, so every
// credit on a card — estorno, refund, credit granted by the issuer — stayed outside
// its bill and never reduced it. The code no longer creates those, but the rows
// already in the database still point at nothing.
//
// It never moves a date. The date came from the bank statement and was never the
// thing that was wrong; only the link was.
type ReattachOrphanInvoiceLinesUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
	recalc      *RecalculateInvoiceAmountUseCase
}

func NewReattachOrphanInvoiceLinesUseCase(
	accountRepo bankaccount.Repository,
	txRepo transactionPkg.Repository,
	invoiceRepo invoice.Repository,
) *ReattachOrphanInvoiceLinesUseCase {
	return &ReattachOrphanInvoiceLinesUseCase{
		accountRepo: accountRepo, txRepo: txRepo, invoiceRepo: invoiceRepo,
		recalc: NewRecalculateInvoiceAmountUseCase(invoiceRepo, txRepo),
	}
}

// Execute reports what belongs where, and writes only the rows it was explicitly
// told to write.
//
// APPLY REQUIRES AN EXPLICIT LIST, and that is the whole safety model. The tool
// cannot tell a legacy invoice payment from a credit: a payment is supposed to carry
// PaidInvoiceID, and the six payments already in this database carry nothing, because
// they predate that field being written. Left to decide on its own, the tool would
// file them into their bills and turn settled bills into smaller ones — the exact
// damage its own guard exists to prevent, waved through by data that cannot answer.
//
// So the dry run enumerates, a person reads it, and apply names the rows. A repair
// that walks real money should not be allowed to guess.
func (uc *ReattachOrphanInvoiceLinesUseCase) Execute(bankAccountID string, apply bool, only []string) (*ReattachReport, error) {
	selected := map[string]bool{}
	for _, id := range only {
		selected[id] = true
	}
	if apply && len(selected) == 0 {
		return nil, errors.New("apply needs an explicit list of transaction ids: run the dry run, read it, then name the rows")
	}
	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("reading the account: %w", err)
	}
	if account == nil {
		return nil, errors.New("bank account not found")
	}
	if account.Type != bankaccount.AccountTypeCreditCard {
		// A checking account has no bills. Answering "nothing to do" would read as a
		// clean result for a run that was pointed at the wrong account.
		return nil, errors.New("only a credit card has invoices to reattach")
	}

	accountID := account.ID
	txns, err := uc.txRepo.List(transactionPkg.ListFilter{
		ProfileID: account.ProfileID, BankAccountID: &accountID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing the card's transactions: %w", err)
	}

	report := &ReattachReport{BankAccountID: account.ID, Applied: apply, Lines: []ReattachedRow{}}
	touched := map[string]bool{}
	for _, txn := range txns {
		if txn.BankAccountID != account.ID || txn.InvoiceID != nil {
			continue
		}
		if txn.Status == transactionPkg.StatusCancelled || txn.Status == transactionPkg.StatusReversed {
			continue
		}
		if !isInvoiceLine(txn.Type) {
			continue
		}

		row := ReattachedRow{
			TransactionID: txn.ID,
			OccurredOn:    txn.OccurredOn.Format("2006-01-02"),
			Amount:        txn.Amount,
			Type:          string(txn.Type),
			Description:   txn.Description,
		}

		if txn.PaidInvoiceID != nil {
			report.SkippedPayments++
			row.Outcome = "skipped: invoice payment, not a line of the bill"
			report.Lines = append(report.Lines, row)
			continue
		}

		inv, err := uc.invoiceRepo.FindByBankAccountAndDate(account.ID, txn.OccurredOn)
		if err != nil {
			return nil, fmt.Errorf("finding the bill for %s: %w", txn.ID, err)
		}
		if inv == nil {
			report.NoInvoiceForDate++
			row.Outcome = "no bill covers this date"
			report.Lines = append(report.Lines, row)
			continue
		}

		row.InvoiceID = inv.ID
		if apply && !selected[txn.ID] {
			report.NotSelected++
			row.Outcome = "not selected by the caller"
			report.Lines = append(report.Lines, row)
			continue
		}

		report.Attached++
		row.Outcome = "attached"
		if !apply {
			row.Outcome = "would attach"
		}
		report.Lines = append(report.Lines, row)

		if apply {
			txn.InvoiceID = &inv.ID
			if err := uc.txRepo.Update(txn); err != nil {
				return nil, fmt.Errorf("attaching %s to bill %s: %w", txn.ID, inv.ID, err)
			}
			touched[inv.ID] = true
		}
	}

	if apply {
		if err := uc.settleTotals(touched, report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

// settleTotals brings each touched bill's STORED total back in line with its lines,
// or reports that it cannot. Attaching a line changes what the bill is worth, and
// the stored total is the number anyone actually reads.
func (uc *ReattachOrphanInvoiceLinesUseCase) settleTotals(touched map[string]bool, report *ReattachReport) error {
	for invoiceID := range touched {
		_, err := uc.recalc.Execute(invoiceID)
		if err == nil {
			report.Recalculated = append(report.Recalculated, invoiceID)
			continue
		}
		if !errors.Is(err, ErrInvoiceAlreadyPaid) {
			return fmt.Errorf("refreshing bill %s: %w", invoiceID, err)
		}
		// A settled bill: report the gap instead of rewriting it.
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
