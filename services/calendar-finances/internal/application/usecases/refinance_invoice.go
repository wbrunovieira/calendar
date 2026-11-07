package usecases

import (
	"fmt"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// The refusals specific to debt conversion. Each carries the ErrPaymentRefused
// marker, so the HTTP layer answers 400 without a list to maintain.
var (
	// ErrNoInvoicesToRefinance rejects a conversion that names no bill.
	ErrNoInvoicesToRefinance = refuse("name at least one bill the conversion settled")
	// ErrNotAConversionCredit rejects anything that is not a credit on the card.
	ErrNotAConversionCredit = refuse("only a confirmed credit on the card can convert its debt")
	// ErrConversionExceeded rejects converting more debt than the bank credited.
	// Without it, a bill nobody cleared could be settled by pointing at a credit
	// too small to have covered it — a forced settlement wearing a receipt.
	ErrConversionExceeded = refuse("those bills owe more than this credit covers")
	// ErrAlreadyRefinancedByAnother rejects taking a bill from the conversion that
	// already settled it.
	ErrAlreadyRefinancedByAnother = refuse("that bill was already converted by a different credit")
)

// RefinanceInvoiceUseCase records that a card credit SETTLED one or more bills by
// converting their debt into instalments.
//
// This is not a payment and must not be recorded as one. When the bank parcels a
// bill it posts a credit that clears the balance and charges the same money again
// on later cycles; no money leaves any account. MarkInvoicePayment rightly refuses
// the job — it caps what it records at the bill's own value, and one conversion
// routinely clears several cycles at once, including the remainder a partial cash
// payment left behind.
//
// The cap here is the credit itself rather than any single bill: the bank cleared
// exactly what it credited, so that is the only honest limit.
type RefinanceInvoiceUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
	now         func() time.Time
}

func NewRefinanceInvoiceUseCase(
	accountRepo bankaccount.Repository,
	txRepo transactionPkg.Repository,
	invoiceRepo invoice.Repository,
) *RefinanceInvoiceUseCase {
	return &RefinanceInvoiceUseCase{
		accountRepo: accountRepo,
		txRepo:      txRepo,
		invoiceRepo: invoiceRepo,
		now:         time.Now,
	}
}

// Execute converts the debt of every named bill, or none of them.
//
// Validation runs over the whole batch before anything is written. A conversion
// that settles the first bill and then refuses the second leaves the ledger in a
// state the bank never produced, and the operator with no way to tell how far it
// got.
func (uc *RefinanceInvoiceUseCase) Execute(transactionID string, invoiceIDs []string) error {
	if len(invoiceIDs) == 0 {
		return ErrNoInvoicesToRefinance
	}

	txn, err := uc.txRepo.GetByID(transactionID)
	if err != nil {
		return fmt.Errorf("could not read the conversion transaction: %w", err)
	}
	if txn == nil {
		return fmt.Errorf("transaction %s not found", transactionID)
	}
	if txn.Type != transactionPkg.TypeIncome || txn.Status != transactionPkg.StatusConfirmed {
		return ErrNotAConversionCredit
	}

	// Gather first, decide second.
	bills := make([]*invoice.Invoice, 0, len(invoiceIDs))
	outstanding := 0.0
	seen := make(map[string]bool, len(invoiceIDs))
	for _, id := range invoiceIDs {
		if seen[id] {
			continue
		}
		seen[id] = true

		bill, err := uc.invoiceRepo.FindByID(id)
		if err != nil {
			return fmt.Errorf("could not read bill %s: %w", id, err)
		}
		if bill.BankAccountID != txn.BankAccountID {
			return ErrInvoiceNotThisCard
		}
		// Already converted by this very credit: a retry, not a second conversion.
		if bill.RefinancedBy != nil {
			if *bill.RefinancedBy == transactionID {
				continue
			}
			return ErrAlreadyRefinancedByAnother
		}
		if bill.Status == invoice.StatusOpen {
			return ErrInvoiceStillOpen
		}

		outstanding += bill.AmountRemaining()
		bills = append(bills, bill)
	}

	if len(bills) == 0 {
		return nil // every named bill was already converted by this credit
	}
	if outstanding > txn.Amount+amountTolerance {
		return ErrConversionExceeded
	}

	at := txn.OccurredOn
	if at.IsZero() {
		at = uc.now()
	}
	for _, bill := range bills {
		if err := bill.Refinance(transactionID, at); err != nil {
			return refuse(err.Error())
		}
	}
	for _, bill := range bills {
		if err := uc.invoiceRepo.Update(bill); err != nil {
			return fmt.Errorf("could not record the conversion of bill %s: %w", bill.ID, err)
		}
	}
	return nil
}
