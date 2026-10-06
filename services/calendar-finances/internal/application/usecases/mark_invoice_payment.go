package usecases

import (
	"errors"
	"fmt"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ErrNotAnInvoicePayment is returned when the transaction cannot settle a bill.
var ErrNotAnInvoicePayment = errors.New("only a credit on the card can be a payment of its bill")

// MarkInvoicePaymentUseCase records that a card credit SETTLES a bill rather than
// being a line of one.
//
// A payment and a refund are the same shape — both are INCOME on the card — so the
// date rule files both as contents of the open cycle. For a refund that is right. For
// a payment it inverts the bill: the Nubank PF cycle of 27/11..27/12 held two credits
// of 2.693,73 and 127,34 and therefore reported a total of -2.566,39, a bill that owes
// the cardholder money. Seven payments across four cards were filed this way, none of
// them carrying paid_invoice_id, because nothing in the system could say what they
// were.
//
// Nothing here guesses. The caller names the payment and names the bill it settles,
// which is the one piece of information no rule can derive from a statement.
type MarkInvoicePaymentUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
}

func NewMarkInvoicePaymentUseCase(
	accountRepo bankaccount.Repository,
	txRepo transactionPkg.Repository,
	invoiceRepo invoice.Repository,
) *MarkInvoicePaymentUseCase {
	return &MarkInvoicePaymentUseCase{accountRepo: accountRepo, txRepo: txRepo, invoiceRepo: invoiceRepo}
}

// Execute marks the transaction as the payment of the given invoice.
//
// Re-running it is safe: both the bill's total and what it has been paid are summed
// from the transactions afterwards, never accumulated, so a second run restates the
// same numbers instead of counting the payment twice. That matters because the repair
// runs over a list and a failure halfway through has to be resumable.
func (uc *MarkInvoicePaymentUseCase) Execute(transactionID, invoiceID string) error {
	txn, account, err := uc.load(transactionID)
	if err != nil {
		return err
	}
	// A payment credits the card. An EXPENSE is a purchase, and accepting one here
	// would take a real charge off the bill and call it settled.
	if txn.Type != transactionPkg.TypeIncome {
		return ErrNotAnInvoicePayment
	}

	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return fmt.Errorf("reading the bill: %w", err)
	}
	if inv == nil {
		return ErrInvoiceNotFound
	}
	if inv.BankAccountID != account.ID {
		return errors.New("that bill belongs to another card")
	}

	txn.PaidInvoiceID = &inv.ID
	// It stops being a line of the bill in the same write. Leaving invoice_id set
	// would keep the credit inside the total, which is the whole defect.
	txn.InvoiceID = nil
	// A pin says "file this line on that bill". A payment is not a line, so the
	// instruction no longer has a subject.
	txn.InvoicePinned = false
	if err := uc.txRepo.Update(txn); err != nil {
		return fmt.Errorf("marking the payment: %w", err)
	}

	return uc.restate(inv)
}

// Release undoes the marking, handing the credit back to the date rule.
//
// It exists because the marking is a human judgement and judgements are sometimes
// wrong: a refund marked as a payment must be able to go back to being a line of the
// bill, without an operator editing a column by hand.
func (uc *MarkInvoicePaymentUseCase) Release(transactionID string) error {
	txn, account, err := uc.load(transactionID)
	if err != nil {
		return err
	}
	if txn.PaidInvoiceID == nil {
		return nil
	}
	settled, err := uc.invoiceRepo.FindByID(*txn.PaidInvoiceID)
	if err != nil {
		return fmt.Errorf("reading the bill: %w", err)
	}

	target, err := uc.invoiceRepo.FindByBankAccountAndDate(account.ID, txn.OccurredOn)
	if err != nil {
		return fmt.Errorf("finding the bill for the date: %w", err)
	}

	txn.PaidInvoiceID = nil
	if target != nil {
		txn.InvoiceID = &target.ID
	} else {
		txn.InvoiceID = nil
	}
	if err := uc.txRepo.Update(txn); err != nil {
		return fmt.Errorf("releasing the payment: %w", err)
	}

	// Both bills move: the one that loses the payment, and the one that gains the
	// line. Restating only one of them leaves the other stating a number the
	// transactions no longer support.
	if err := uc.restate(settled); err != nil {
		return err
	}
	if target != nil && (settled == nil || target.ID != settled.ID) {
		return uc.restate(target)
	}
	return nil
}

// restate recomputes a bill's total and what it has been paid from the transactions
// as they now stand.
//
// RecalculateInvoiceAmountUseCase refuses a PAID bill, which is exactly the state
// these bills are in — a settled bill is the normal case for a payment, not an
// exception — so the two sums are taken here instead of through it.
func (uc *MarkInvoicePaymentUseCase) restate(inv *invoice.Invoice) error {
	if inv == nil {
		return nil
	}
	total, err := uc.txRepo.SumByInvoiceID(inv.ID)
	if err != nil {
		return fmt.Errorf("totalling the bill: %w", err)
	}
	paid, err := uc.txRepo.SumLivePaymentsByInvoiceID(inv.ID)
	if err != nil {
		return fmt.Errorf("totalling what was paid: %w", err)
	}
	inv.Amount = total
	// RestatePayments re-derives the status from the two numbers, so a bill whose
	// total turns out to exceed what was paid correctly stops claiming to be paid.
	inv.RestatePayments(paid)
	if err := uc.invoiceRepo.Update(inv); err != nil {
		return fmt.Errorf("saving the bill: %w", err)
	}
	return nil
}

func (uc *MarkInvoicePaymentUseCase) load(transactionID string) (*transactionPkg.Transaction, *bankaccount.BankAccount, error) {
	txn, err := uc.txRepo.GetByID(transactionID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the transaction: %w", err)
	}
	if txn == nil {
		return nil, nil, transactionPkg.ErrNotFound
	}
	account, err := uc.accountRepo.FindByID(txn.BankAccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the account: %w", err)
	}
	if account == nil {
		return nil, nil, bankaccount.ErrNotFound
	}
	// Only a card has a bill to settle.
	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, nil, ErrNotACreditCard
	}
	return txn, account, nil
}
