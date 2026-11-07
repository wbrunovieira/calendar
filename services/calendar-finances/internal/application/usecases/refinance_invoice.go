package usecases

import (
	"fmt"
	"log"
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
	// invoiceRecalculator is attached after construction, like the other
	// cross-cutting collaborators.
	invoiceRecalculator InvoiceRecalculator
	atomically          UnitOfWork
}

// SetUnitOfWork makes the conversion all-or-nothing.
//
// Validation precedes the first write, but that says nothing about a failure
// BETWEEN writes: one conversion settles several bills and then detaches the
// credit, and a statement timeout halfway through leaves some bills converted by
// a credit that is still filed as a line of a cycle — a state the bank never
// produced, with no way for the operator to tell how far it got.
func (uc *RefinanceInvoiceUseCase) SetUnitOfWork(u UnitOfWork) { uc.atomically = u }

// boundTo returns a copy whose repositories write through the open transaction.
// Holding repositories built on the *sql.DB while inside someone else's
// transaction is how writes that must land together land one by one.
func (uc *RefinanceInvoiceUseCase) boundTo(r TxRepos) *RefinanceInvoiceUseCase {
	bound := *uc
	bound.atomically = nil
	if r.Transactions != nil {
		bound.txRepo = r.Transactions
	}
	if r.Invoices != nil {
		bound.invoiceRepo = r.Invoices
	}
	if r.Accounts != nil {
		bound.accountRepo = r.Accounts
	}
	return &bound
}

// SetInvoiceRecalculator wires the bill-total refresh.
func (uc *RefinanceInvoiceUseCase) SetInvoiceRecalculator(r InvoiceRecalculator) {
	uc.invoiceRecalculator = r
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
	if uc.atomically != nil {
		// The refresh runs AFTER the transaction commits: it reads the rows this
		// work wrote, and running it inside would have it read a state that may yet
		// be rolled back.
		var touched []string
		err := uc.atomically.Do(func(r TxRepos) error {
			var err error
			touched, err = uc.boundTo(r).convert(transactionID, invoiceIDs)
			return err
		})
		if err != nil {
			return err
		}
		for _, id := range touched {
			uc.restate(id)
		}
		return nil
	}
	touched, err := uc.convert(transactionID, invoiceIDs)
	if err != nil {
		return err
	}
	for _, id := range touched {
		uc.restate(id)
	}
	return nil
}

// convert does the writing and reports every bill whose total needs refreshing.
func (uc *RefinanceInvoiceUseCase) convert(transactionID string, invoiceIDs []string) ([]string, error) {

	txn, err := uc.txRepo.GetByID(transactionID)
	if err != nil {
		return nil, fmt.Errorf("could not read the conversion transaction: %w", err)
	}
	if txn == nil {
		return nil, fmt.Errorf("transaction %s not found", transactionID)
	}
	if txn.Type != transactionPkg.TypeIncome || txn.Status != transactionPkg.StatusConfirmed {
		return nil, ErrNotAConversionCredit
	}

	// What this credit has ALREADY converted, in earlier requests. Without it the
	// cap can only see the bills in front of it, and the same credit settles
	// unlimited debt by being spent a few bills at a time — each batch under the
	// limit, the total far over it.
	already, err := uc.invoiceRepo.FindByRefinancedBy(transactionID)
	if err != nil {
		return nil, fmt.Errorf("could not read what this credit already converted: %w", err)
	}
	spent := 0.0
	for _, bill := range already {
		if bill.RefinancedAmount != nil {
			spent += *bill.RefinancedAmount
		}
	}

	// Gather first, decide second.
	bills := make([]*invoice.Invoice, 0, len(invoiceIDs))
	outstanding := spent
	seen := make(map[string]bool, len(invoiceIDs))
	for _, id := range invoiceIDs {
		if seen[id] {
			continue
		}
		seen[id] = true

		bill, err := uc.invoiceRepo.FindByID(id)
		if err != nil {
			return nil, fmt.Errorf("could not read bill %s: %w", id, err)
		}
		if bill.BankAccountID != txn.BankAccountID {
			return nil, ErrInvoiceNotThisCard
		}
		// Already converted by this very credit: a retry, not a second conversion.
		if bill.RefinancedBy != nil {
			if *bill.RefinancedBy == transactionID {
				continue
			}
			return nil, ErrAlreadyRefinancedByAnother
		}
		if bill.Status == invoice.StatusOpen {
			return nil, ErrInvoiceStillOpen
		}

		outstanding += bill.AmountRemaining()
		bills = append(bills, bill)
	}

	if len(bills) == 0 {
		return nil, nil // every named bill was already converted by this credit
	}
	if outstanding > txn.Amount+amountTolerance {
		return nil, ErrConversionExceeded
	}

	at := txn.OccurredOn
	if at.IsZero() {
		at = uc.now()
	}
	for _, bill := range bills {
		if err := bill.Refinance(transactionID, at); err != nil {
			return nil, refuse(err.Error())
		}
	}
	for _, bill := range bills {
		if err := uc.invoiceRepo.Update(bill); err != nil {
			return nil, fmt.Errorf("could not record the conversion of bill %s: %w", bill.ID, err)
		}
	}

	// A credit that settles bills is not a LINE of one. Being INCOME on the card,
	// the date rule files it into whichever cycle was open, where a credit scores
	// as -amount: the November bill came out R$ 2.205,86 lighter. Detaching it is
	// the same repair MarkInvoicePayment makes for a payment, and for the same
	// reason.
	touched := make([]string, 0, len(bills)+1)
	if txn.InvoiceID != nil {
		deflated := *txn.InvoiceID
		txn.InvoiceID = nil
		if err := uc.txRepo.Update(txn); err != nil {
			return nil, fmt.Errorf("could not detach the conversion credit from its cycle: %w", err)
		}
		touched = append(touched, deflated)
	}
	for _, bill := range bills {
		touched = append(touched, bill.ID)
	}
	return touched, nil
}

// Release undoes the conversion of one bill, putting its debt back.
//
// The counterpart to Execute, for the same reason MarkInvoicePayment has one: the
// marking is a human judgement, and a credit can be reversed after the fact. With
// no way back a wrong conversion is permanent, since re-running does nothing and
// another credit is refused.
func (uc *RefinanceInvoiceUseCase) Release(invoiceID string) error {
	bill, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return fmt.Errorf("could not read bill %s: %w", invoiceID, err)
	}
	if bill == nil {
		return fmt.Errorf("invoice %s not found", invoiceID)
	}
	if err := bill.ReleaseRefinancing(); err != nil {
		return refuse(err.Error())
	}
	if err := uc.invoiceRepo.Update(bill); err != nil {
		return fmt.Errorf("could not release the conversion of bill %s: %w", bill.ID, err)
	}
	uc.restate(bill.ID)
	return nil
}

// restate refreshes a bill's stored total after its lines changed. Quiet by
// design: a refresh that cannot run must not undo a conversion that already did.
func (uc *RefinanceInvoiceUseCase) restate(invoiceID string) {
	if uc.invoiceRecalculator == nil {
		return
	}
	if _, err := uc.invoiceRecalculator.RefreshTotal(invoiceID); err != nil {
		log.Printf("invoice %s: refreshing the total after a debt conversion failed: %v", invoiceID, err)
	}
}
