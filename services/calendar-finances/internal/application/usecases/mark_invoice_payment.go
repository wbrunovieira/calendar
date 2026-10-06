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
	// ErrPaymentAlreadyRecorded rejects the second row of ONE payment. A cross-profile
	// payment creates two linked rows for a single movement of money, and linking both
	// records the bill as paid twice over.
	ErrPaymentAlreadyRecorded = errors.New("the other half of this same payment already settles that bill")
	// ErrWouldLowerRecordedPayment rejects a restatement that would reduce what a bill
	// records as paid, because the amount being dropped has no transaction behind it
	// and dropping it silently turns a settled bill into an owed one.
	ErrWouldLowerRecordedPayment = errors.New("that would reduce what the bill records as paid")
	// ErrCardCannotFundAPayment rejects a transfer whose source is itself a card.
	ErrCardCannotFundAPayment = errors.New("a credit card cannot fund the payment of a bill")
)

// settlesABill reports whether this transaction can settle a bill at all: a credit on
// the card, or the funding leg of a payment into it. load() has already established
// that the card is the right one.
func settlesABill(txn *transactionPkg.Transaction) bool {
	switch txn.Type {
	case transactionPkg.TypeIncome:
		// On the card, since load() resolved the card from bank_account_id here.
		return true
	case transactionPkg.TypeTransfer:
		// Only a leg that names a destination reaches a card at all.
		return txn.DestinationAccountID != nil
	default:
		return false
	}
}

// invoiceLineSign is how a row contributes to its bill's total, mirroring
// SumByInvoiceID's `CASE WHEN type = 'INCOME' THEN -amount ELSE amount END`. A credit
// lowers the bill; anything else raises it. Getting this backwards for a TRANSFER let
// a bill of 1.000,00 be recorded as paid 2.000,00.
func invoiceLineContribution(txn *transactionPkg.Transaction) float64 {
	if txn.Type == transactionPkg.TypeIncome {
		return -txn.Amount
	}
	return txn.Amount
}

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
	atomically  UnitOfWork
}

// SetUnitOfWork makes the marking all-or-nothing.
//
// Execute writes the row and then up to three bills. Without this, a failure after
// the row moved left the payment pointing at one bill while another still recorded it
// -- and the promise that refusals precede the first write says nothing about a
// failure BETWEEN writes. The repair walks a list, so it has to be resumable.
func (uc *MarkInvoicePaymentUseCase) SetUnitOfWork(u UnitOfWork) { uc.atomically = u }

// boundTo returns a copy whose repositories write through the open transaction.
// Holding repositories built on the *sql.DB while inside someone else's transaction
// is how writes that must land together land one by one.
func (uc *MarkInvoicePaymentUseCase) boundTo(r TxRepos) *MarkInvoicePaymentUseCase {
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
	if uc.atomically != nil {
		return uc.atomically.Do(func(r TxRepos) error {
			return uc.boundTo(r).mark(transactionID, invoiceID)
		})
	}
	return uc.mark(transactionID, invoiceID)
}

func (uc *MarkInvoicePaymentUseCase) mark(transactionID, invoiceID string) error {
	txn, account, err := uc.load(transactionID)
	if err != nil {
		return err
	}
	// A payment either credits the card (INCOME on the card) or reaches it as the
	// funding leg (TRANSFER into the card). An EXPENSE is a purchase, and accepting
	// one would take a real charge off the bill and call it settled.
	if !settlesABill(txn) {
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

	// The bill the row currently SETTLES, if a different one. Re-pointing has to take
	// the payment off it: restating only the new bill left the old one reading PAID
	// with no payment behind it, and GetCreditUsage handing back that much phantom
	// limit.
	var previous *invoice.Invoice
	if txn.PaidInvoiceID != nil && *txn.PaidInvoiceID != inv.ID {
		previous, err = uc.invoiceRepo.FindByID(*txn.PaidInvoiceID)
		if err != nil {
			return fmt.Errorf("reading the bill it currently settles: %w", err)
		}
	}

	// The bill the row is currently a LINE of -- a different thing, and the normal
	// case: the date rule files a payment made on 03/08 into the cycle that opens
	// 27/07, so the bill it leaves is almost never the bill it settles. Leaving that
	// bill unrestated left it stating a total its own lines no longer support, which
	// GetCreditUsage then reads as credit consumed.
	var leaving *invoice.Invoice
	if txn.InvoiceID != nil && *txn.InvoiceID != inv.ID {
		leaving, err = uc.invoiceRepo.FindByID(*txn.InvoiceID)
		if err != nil {
			return fmt.Errorf("reading the bill it is filed in: %w", err)
		}
	}

	// A cross-profile payment is ONE movement of money written as two linked rows.
	// Linking both records the bill as paid twice, and the money comparison alone
	// cannot see it whenever the bill is at least twice the payment.
	if err := uc.checkNotTheOtherHalf(txn, inv); err != nil {
		return err
	}
	for _, bill := range []*invoice.Invoice{inv, previous, leaving} {
		if err := uc.checkRestatable(bill); err != nil {
			return err
		}
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

	// inv must only gain, and leaving is losing a LINE, not a payment -- a drop in
	// either is an accident. previous is losing the payment itself, which is the
	// point of re-pointing, so there a drop is expected.
	if err := uc.restate(inv, restateKeepingWhatWasPaid); err != nil {
		return err
	}
	if err := uc.restate(leaving, restateKeepingWhatWasPaid); err != nil {
		return err
	}
	return uc.restate(previous, restateAllowingADrop)
}

// checkNotTheOtherHalf refuses the second row of one payment.
//
// A cross-profile transfer and PayInvoiceV2 both write two mutually linked rows for a
// single movement of money. checkNotOverpaid compares money only, so it misses the
// pair whenever the bill is at least twice the payment -- and two such pairs already
// exist on the WB card.
func (uc *MarkInvoicePaymentUseCase) checkNotTheOtherHalf(txn *transactionPkg.Transaction, inv *invoice.Invoice) error {
	if txn.LinkedTransactionID == nil {
		return nil
	}
	other, err := uc.txRepo.GetByID(*txn.LinkedTransactionID)
	if err != nil {
		if errors.Is(err, transactionPkg.ErrNotFound) {
			// The partner is gone; nothing can be double counted through it.
			return nil
		}
		return fmt.Errorf("reading the other half of this payment: %w", err)
	}
	if other == nil || other.PaidInvoiceID == nil {
		return nil
	}
	if *other.PaidInvoiceID == inv.ID {
		return ErrPaymentAlreadyRecorded
	}
	return nil
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
	// Detaching the row from the bill's lines changes the total by its contribution,
	// so the comparison is against the total the bill WILL have. A credit contributes
	// -amount and a transfer +amount: adding the amount in both cases let a bill of
	// 1.000,00 be recorded as paid 2.000,00.
	willOwe := inv.Amount
	if txn.InvoiceID != nil && *txn.InvoiceID == inv.ID {
		willOwe -= invoiceLineContribution(txn)
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
	if uc.atomically != nil {
		return uc.atomically.Do(func(r TxRepos) error {
			return uc.boundTo(r).release(transactionID)
		})
	}
	return uc.release(transactionID)
}

func (uc *MarkInvoicePaymentUseCase) release(transactionID string) error {
	txn, account, err := uc.load(transactionID)
	if err != nil {
		return err
	}
	if txn.PaidInvoiceID == nil {
		return nil
	}
	// Release writes invoice_id, and SumByInvoiceID counts anything that is not
	// INCOME as a charge. Execute refuses anything that cannot settle a bill; so
	// must this.
	if !settlesABill(txn) {
		return ErrNotAnInvoicePayment
	}
	settled, err := uc.invoiceRepo.FindByID(*txn.PaidInvoiceID)
	if err != nil {
		return fmt.Errorf("reading the bill: %w", err)
	}

	// Only a card-side credit goes back to the date rule: released, it is an estorno,
	// which really is a line of the bill covering its date. A funding leg is never a
	// line of anything -- SumByInvoiceID counts a TRANSFER as a charge, so filing one
	// in would add its amount to the bill as a purchase.
	var target *invoice.Invoice
	if txn.Type == transactionPkg.TypeIncome {
		target, err = uc.invoiceRepo.FindByBankAccountAndDate(account.ID, txn.OccurredOn)
		if err != nil {
			return fmt.Errorf("finding the bill for the date: %w", err)
		}
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
	if err := uc.restate(settled, restateAllowingADrop); err != nil {
		return err
	}
	if target != nil && (settled == nil || target.ID != settled.ID) {
		return uc.restate(target, restateAllowingADrop)
	}
	return nil
}

// restate recomputes a bill's total and what it has been paid from the transactions
// as they now stand.
//
// RecalculateInvoiceAmountUseCase refuses a PAID bill, which is exactly the state
// these bills are in — a settled bill is the normal case for a payment, not an
// exception — so the two sums are taken here instead of through it.
// restateIntent says whether a drop in what the bill records as paid is the point of
// the call or an accident of it. Releasing a payment must lower it; linking one must
// never do so by surprise.
type restateIntent int

const (
	restateKeepingWhatWasPaid restateIntent = iota
	restateAllowingADrop
)

func (uc *MarkInvoicePaymentUseCase) restate(inv *invoice.Invoice, intent restateIntent) error {
	return invoiceRestater{txRepo: uc.txRepo, invoiceRepo: uc.invoiceRepo}.restate(inv, intent)
}

// invoiceRestater recomputes a bill's total and what it has been paid from the
// transactions as they now stand. Shared, because UpdateTransaction has to do exactly
// this after editing a linked payment.
type invoiceRestater struct {
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
}

func (uc invoiceRestater) restate(inv *invoice.Invoice, intent restateIntent) error {
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
	// A restatement that LOWERS what the bill records as paid is refused. The amount
	// being dropped was written by inv.Pay() with no row carrying paid_invoice_id --
	// so it cannot be re-derived, and dropping it silently turns a settled bill into
	// an owed one. Raising it is the repair and goes through.
	if intent == restateKeepingWhatWasPaid && inv.PaidAmount != nil && *inv.PaidAmount-paid > amountTolerance {
		return fmt.Errorf("%w: it records %.2f paid and the payments behind it sum to %.2f",
			ErrWouldLowerRecordedPayment, *inv.PaidAmount, paid)
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

// load resolves the transaction and the CARD its payment reaches, whichever side of
// the payment the row sits on.
//
// There are two legitimate shapes, and 24 of the 31 real payments use the second:
//
//   - a credit ON the card: INCOME, bank_account_id = the card. The shape for a
//     payment made from an account the system does not know, and the one the
//     statement import produces.
//   - the FUNDING LEG: TRANSFER, bank_account_id = the paying account,
//     destination_account_id = the card. One row debits the account and credits the
//     card, and stays out of income and expense -- which is why PayInvoiceV2 creates
//     this and not an EXPENSE/INCOME pair.
//
// Only one of the two may carry paid_invoice_id for a given payment. Both would
// record the bill as paid twice over, which checkNotOverpaid refuses.
func (uc *MarkInvoicePaymentUseCase) load(transactionID string) (*transactionPkg.Transaction, *bankaccount.BankAccount, error) {
	txn, err := uc.txRepo.GetByID(transactionID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the transaction: %w", err)
	}
	if txn == nil {
		return nil, nil, transactionPkg.ErrNotFound
	}
	account, err := resolveSettledCard(txn, uc.accountRepo)
	if err != nil {
		return nil, nil, err
	}
	return txn, account, nil
}

// resolveSettledCard returns the CARD whose bill this row's payment reaches,
// whichever side of the payment the row sits on.
//
// There are two legitimate shapes, and 24 of the 31 real payments use the second:
//
//   - a credit ON the card: INCOME, bank_account_id = the card. The shape for a
//     payment made from an account the system does not know, and the one the
//     statement import produces.
//   - the FUNDING LEG: TRANSFER, bank_account_id = the paying account,
//     destination_account_id = the card. One row debits the account and credits the
//     card, and stays out of income and expense -- which is why PayInvoiceV2 creates
//     this and not an EXPENSE/INCOME pair.
//
// Only one of the two may carry paid_invoice_id for a given payment. Both would
// record the bill as paid twice over, which checkNotOverpaid refuses.
//
// It lives here, shared, because UpdateTransaction has to ask the same question to
// know whether an edit left the row able to settle anything -- and a rule written in
// two places is enforced in one.
func resolveSettledCard(
	txn *transactionPkg.Transaction,
	accountRepo bankaccount.Repository,
) (*bankaccount.BankAccount, error) {
	cardID := txn.BankAccountID
	if txn.Type == transactionPkg.TypeTransfer && txn.DestinationAccountID != nil {
		cardID = *txn.DestinationAccountID
		// A card's own balance does not move as cash -- the credit-card guard in
		// CreateTransaction skips it -- so a transfer out of one cannot fund a bill.
		// Accepted silently, it let card B's bill read PAID off card A's credit line.
		source, serr := accountRepo.FindByID(txn.BankAccountID)
		if serr != nil {
			return nil, fmt.Errorf("reading the paying account: %w", serr)
		}
		if source != nil && source.Type == bankaccount.AccountTypeCreditCard {
			return nil, ErrCardCannotFundAPayment
		}
	}

	account, err := accountRepo.FindByID(cardID)
	if err != nil {
		return nil, fmt.Errorf("reading the account: %w", err)
	}
	if account == nil {
		return nil, bankaccount.ErrNotFound
	}
	// Only a card has a bill to settle. A transfer between two ordinary accounts, or
	// one with no destination at all, reaches no bill.
	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, ErrNotACreditCard
	}
	return account, nil
}

// canStillSettleABill reports whether a row, as it now stands, is able to settle a
// card bill at all. A row that cannot must not keep a paid_invoice_id: the payment
// sum has no type filter, so the bill would go on counting it with no route left to
// unlink it.
func canStillSettleABill(txn *transactionPkg.Transaction, accountRepo bankaccount.Repository) bool {
	if !settlesABill(txn) {
		return false
	}
	account, err := resolveSettledCard(txn, accountRepo)
	return err == nil && account != nil
}
