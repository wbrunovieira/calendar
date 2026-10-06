package usecases

import (
	"errors"
	"fmt"
	"math"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Refusals, each its own sentinel so the HTTP layer can answer 400 for a bad request
// and 500 for a failure. Returning one code for both told a retrying caller its body
// was malformed when the database was down.
var (
	// ErrNotAnInvoicePayment is returned when the transaction cannot settle a bill.
	ErrNotAnInvoicePayment = errors.New("only a credit on the card can be a payment of its bill")
	// ErrPaymentNotConfirmed rejects money that has not moved.
	ErrPaymentNotConfirmed = errors.New("only a confirmed credit can settle a bill")
	// ErrInvoiceNotThisCard rejects another card's bill.
	ErrInvoiceNotThisCard = errors.New("that bill belongs to another card")
	// ErrInvoiceStillOpen rejects a cycle that is still accruing charges.
	ErrInvoiceStillOpen = errors.New("that cycle is still open, so it has nothing settled yet")
	// ErrInvoiceAmountOutOfSync rejects a bill whose stored total already disagrees
	// with the lines linked to it, because restating it would overwrite a statement
	// figure with an incomplete sum.
	ErrInvoiceAmountOutOfSync = errors.New("the bill's stored total does not match its own lines")
	// ErrPaymentExceedsInvoice rejects recording more paid than the bill is worth.
	ErrPaymentExceedsInvoice = errors.New("that would record more paid than the bill is worth")
)

// amountTolerance is one cent: the sums are money, rounded to cents on both sides.
const amountTolerance = 0.01

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
//
// Every refusal happens BEFORE the first write. A repair that fails halfway leaves a
// credit belonging to no bill and settling no bill, which no route can then find.
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
	// A planned credit has not moved money, and a reversed or cancelled one moved it
	// back. SumByInvoiceID counts a PLANNED row while the payment sum does not, so
	// marking one raises the bill's total and records nothing paid: the bill owes the
	// whole amount again, from money that never left the account.
	if txn.Status != transactionPkg.StatusConfirmed {
		return ErrPaymentNotConfirmed
	}

	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return fmt.Errorf("reading the bill: %w", err)
	}
	if inv == nil {
		return ErrInvoiceNotFound
	}
	if inv.BankAccountID != account.ID {
		return ErrInvoiceNotThisCard
	}
	if inv.Status == invoice.StatusOpen {
		// RederiveStatus returns early on OPEN, so writing paid_amount there would
		// leave the status untouched and AmountRemaining reporting 0 for a cycle
		// that is still collecting charges.
		return ErrInvoiceStillOpen
	}

	// The bill the credit currently settles, if any. Re-pointing has to take the
	// payment OFF it: restating only the new bill left the old one reading PAID with
	// no payment behind it, and GetCreditUsage handing back that much phantom limit.
	var previous *invoice.Invoice
	if txn.PaidInvoiceID != nil && *txn.PaidInvoiceID != inv.ID {
		previous, err = uc.invoiceRepo.FindByID(*txn.PaidInvoiceID)
		if err != nil {
			return fmt.Errorf("reading the bill it currently settles: %w", err)
		}
	}

	if err := uc.checkRestatable(inv); err != nil {
		return err
	}
	if err := uc.checkRestatable(previous); err != nil {
		return err
	}
	// The same payment can reach a bill twice: once as the funding account's TRANSFER
	// leg created by /invoices/{id}/pay, once as the card-side credit imported from
	// the statement. Both carry paid_invoice_id, so the bill would read as paid twice
	// over.
	if err := uc.checkNotOverpaid(inv, txn); err != nil {
		return err
	}

	txn.PaidInvoiceID = &inv.ID
	// It stops being a line of the bill in the same write. Leaving invoice_id set
	// would keep the credit inside the total, which is the whole defect.
	txn.InvoiceID = nil
	// A pin says "file this LINE on that bill". A payment is not a line, so the
	// instruction no longer has a subject. UpdateTransaction exempts a payment from
	// re-derivation on paid_invoice_id instead.
	txn.InvoicePinned = false
	if err := uc.txRepo.Update(txn); err != nil {
		return fmt.Errorf("marking the payment: %w", err)
	}

	if err := uc.restate(inv); err != nil {
		return err
	}
	return uc.restate(previous)
}

// checkRestatable refuses a bill whose stored total already disagrees with the lines
// linked to it.
//
// Restating the total is the repair, so it cannot be skipped -- but it is only safe
// while the stored figure and the lines already agree, because then detaching the
// payment moves the total by exactly the payment. Where they disagree, the stored
// figure is the statement's and the sum is incomplete: on the Mercado Pago card a
// bill read 2.647,97 against the 2.702,66 charged, because four late fees were filed
// on the wrong cycle. Overwriting that replaces the statement number with the smaller
// one, on a settled historical bill, with no route back.
func (uc *MarkInvoicePaymentUseCase) checkRestatable(inv *invoice.Invoice) error {
	if inv == nil {
		return nil
	}
	linked, err := uc.txRepo.SumByInvoiceID(inv.ID)
	if err != nil {
		return fmt.Errorf("totalling the bill: %w", err)
	}
	if math.Abs(linked-inv.Amount) > amountTolerance {
		return fmt.Errorf("%w: it stores %.2f and its lines sum to %.2f", ErrInvoiceAmountOutOfSync, inv.Amount, linked)
	}
	return nil
}

// checkNotOverpaid refuses to record more settled than the bill is worth.
func (uc *MarkInvoicePaymentUseCase) checkNotOverpaid(inv *invoice.Invoice, txn *transactionPkg.Transaction) error {
	paid, err := uc.txRepo.SumLivePaymentsByInvoiceID(inv.ID)
	if err != nil {
		return fmt.Errorf("totalling what was paid: %w", err)
	}
	if txn.PaidInvoiceID != nil && *txn.PaidInvoiceID == inv.ID {
		// Already counted: this is a re-run, not a second payment.
		return nil
	}
	// Detaching the credit from the bill's lines raises the total by its amount, so
	// the comparison is against the total the bill will have.
	willOwe := inv.Amount
	if txn.InvoiceID != nil && *txn.InvoiceID == inv.ID {
		willOwe += txn.Amount
	}
	if paid+txn.Amount > willOwe+amountTolerance {
		return fmt.Errorf("%w: %.2f already settles a bill of %.2f", ErrPaymentExceedsInvoice, paid, willOwe)
	}
	return nil
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
	// Release writes invoice_id, and SumByInvoiceID counts anything that is not
	// INCOME as a charge -- so releasing a TRANSFER would add its amount to a bill
	// as a purchase. Execute refuses non-credits; so must this.
	if txn.Type != transactionPkg.TypeIncome {
		return ErrNotAnInvoicePayment
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
