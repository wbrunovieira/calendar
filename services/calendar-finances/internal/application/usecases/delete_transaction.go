package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// InvoiceRecalculator recomputes a card invoice's total from the transactions still
// attached to it.
type InvoiceRecalculator interface {
	// RestatePayments brings a bill's paid amount back to what its live payment legs
	// add up to. Reversing a payment used to leave the bill PAID forever, because the
	// leg carries paid_invoice_id and not invoice_id, so nothing here ever looked at it.
	RestatePayments(invoiceID string) (*invoice.Invoice, error)
	Execute(invoiceID string) (*invoice.Invoice, error)
}

type DeleteTransactionUseCase struct {
	repo                transaction.Repository
	accountRepo         bankaccount.Repository
	balanceRecalculator BalanceRecalculator
	invoiceRecalculator InvoiceRecalculator
}

// SetInvoiceRecalculator wires the invoice total recomputation after construction,
// following how the other cross-cutting collaborators are attached in main.go.
func (uc *DeleteTransactionUseCase) SetInvoiceRecalculator(r InvoiceRecalculator) {
	uc.invoiceRecalculator = r
}

func NewDeleteTransactionUseCase(repo transaction.Repository, accountRepo bankaccount.Repository, recalculator BalanceRecalculator) *DeleteTransactionUseCase {
	return &DeleteTransactionUseCase{repo: repo, accountRepo: accountRepo, balanceRecalculator: recalculator}
}

// Execute removes a transaction and leaves every account it touched reading as if it
// had never existed.
//
// The rows are deleted BEFORE the balances are recomputed. Recomputing first — which
// is what this used to do — reads the transaction that is about to disappear and puts
// the old numbers straight back, so the deletion silently left a phantom balance
// behind: on the destination of a transfer, and on the source of any confirmed entry.
// ReverseTransactionInput carries what a reversal must record. Reason and Actor are
// required by the domain: a reversal whose motive was not captured at the time cannot
// be reconstructed later, and the incident that motivated all this was an automated
// agent removing a legitimate entry.
type ReverseTransactionInput struct {
	ID     string
	Reason transaction.ReversalReason
	Note   string
	By     string
}

// Execute reverses a transaction. Kept for callers that already carry their own
// context; prefer ExecuteWithReason.
func (uc *DeleteTransactionUseCase) Execute(id string) error {
	return uc.ExecuteWithReason(ReverseTransactionInput{
		ID:     id,
		Reason: transaction.ReasonNeverHappened,
		By:     "unspecified",
	})
}

func (uc *DeleteTransactionUseCase) ExecuteWithReason(input ReverseTransactionInput) error {
	id := input.ID
	txn, err := uc.repo.GetByID(id)
	if err != nil {
		return ErrTransactionNotFound
	}

	// A cross-profile transfer is stored as two linked rows; deleting one without the
	// other would leave money that arrived from nowhere.
	var linked *transaction.Transaction
	if txn.LinkedTransactionID != nil {
		if found, err := uc.repo.GetByID(*txn.LinkedTransactionID); err == nil {
			linked = found
		}
	}

	affected := uc.affectedAccounts(txn, linked)

	// Both legs go in one unit of work. Removing them one at a time can leave the
	// pair half-deleted — the other profile holding a credit with no row behind it —
	// while the caller is told the whole thing failed, so nobody goes looking.
	// A ledger does not delete. Reversing keeps the row, stops it counting towards
	// balances (they are derived from CONFIRMED), and records when it was undone.
	//
	// Deleting destroyed evidence: during the reconciliation of 06/09/2026 a real
	// R$ 55,58 charge was removed as a supposed phantom, and nothing in the system
	// can now say what was removed or why.
	now := time.Now()
	toReverse := []*transaction.Transaction{txn}
	if linked != nil {
		toReverse = append(toReverse, linked)
	}

	// Whether a leg moved a balance is decided by the status it had BEFORE the
	// reversal. Reading it afterwards finds REVERSED on every leg and undoes
	// nothing — the balance stays as if the transaction were still there.
	wasConfirmed := make([]*transaction.Transaction, 0, len(toReverse))
	for _, t := range toReverse {
		if t.Status == transaction.StatusConfirmed {
			wasConfirmed = append(wasConfirmed, t)
		}
	}

	// The verb follows the state, not the caller. A planned row never moved money,
	// so undoing it is a cancellation; a confirmed one is a reversal. Letting both
	// produce the same status would leave two meanings of "does not count" with no
	// written rule for which — and an ambiguous status in a ledger becomes a balance
	// that differs depending on who wrote the query.
	for _, t := range toReverse {
		var err error
		if t.Status == transaction.StatusPlanned {
			err = t.Cancel(input.Reason, input.Note, input.By, now)
		} else {
			err = t.Reverse(input.Reason, input.Note, input.By, now)
		}
		if err != nil {
			return err
		}
	}
	if err := uc.repo.ReverseMany(toReverse); err != nil {
		return err
	}

	uc.recomputeInvoices(txn, linked)

	if uc.balanceRecalculator != nil {
		// The rows are already gone. Swallowing a failure here would leave the
		// balances permanently stale behind a 204, and nobody would know to run the
		// recalculation by hand.
		return recalculateAccounts(uc.balanceRecalculator, affected...)
	}

	// Without a recalculator wired, undo each leg by hand. Same result, but derived
	// from the transaction instead of from the ledger.
	return uc.reverseByHand(wasConfirmed...)
}

// affectedAccounts lists every account whose balance depended on the rows being
// removed. Only confirmed transactions ever moved money.
//
// Credit cards are left out on purpose, on BOTH sides of a transfer. Creating a card
// transaction does not move the card's balance — every write path skips them, and the
// balance is a consequence of paying the invoice — so recomputing one here would make
// the card jump by the sum of every purchase since the last bill, purely because
// something unrelated was deleted.
//
// Excluding it on one side only made the result depend on the order the deletions
// happened in: removing a purchase and then a payment left the card at -100, and doing
// the same two in the other order left it at -200. Whether a card's balance should
// track its transactions at all is a separate decision (#1277 on the board); until
// then, deleting does not move it.
func (uc *DeleteTransactionUseCase) affectedAccounts(txns ...*transaction.Transaction) []string {
	ids := make([]string, 0, len(txns)*2)
	for _, t := range txns {
		if t == nil || t.Status != transaction.StatusConfirmed {
			continue
		}
		if !uc.isCreditCard(t.BankAccountID) {
			ids = append(ids, t.BankAccountID)
		}
		if t.DestinationAccountID != nil && !uc.isCreditCard(*t.DestinationAccountID) {
			ids = append(ids, *t.DestinationAccountID)
		}
	}
	return deduplicateIDs(ids...)
}

func (uc *DeleteTransactionUseCase) isCreditCard(accountID string) bool {
	account, err := uc.accountRepo.FindByID(accountID)
	return err == nil && account.IsCreditCard()
}

// reverseByHand undoes the balance effect of legs that WERE confirmed. The caller
// decides which ones those are, because by the time this runs their status already
// says REVERSED.
func (uc *DeleteTransactionUseCase) reverseByHand(txns ...*transaction.Transaction) error {
	for _, txn := range txns {
		if txn == nil {
			continue
		}

		// Creating a transaction on a credit card does not move the card's balance —
		// a card's position is derived from its invoices — so undoing one must not
		// move it either. Same rule as the recalculator path above.
		if account, err := uc.accountRepo.FindByID(txn.BankAccountID); err == nil && !account.IsCreditCard() {
			uc.reverseBalance(account, txn.Type, txn.Amount)
			account.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(account); err != nil {
				return err
			}
		}

		if txn.DestinationAccountID == nil {
			continue
		}
		if destination, err := uc.accountRepo.FindByID(*txn.DestinationAccountID); err == nil {
			destination.CurrentBalance -= txn.Amount
			destination.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(destination); err != nil {
				return err
			}
		}
	}
	return nil
}

func (uc *DeleteTransactionUseCase) reverseBalance(account *bankaccount.BankAccount, txType transaction.Type, amount float64) {
	switch txType {
	case transaction.TypeExpense:
		account.CurrentBalance += amount
	case transaction.TypeIncome:
		account.CurrentBalance -= amount
	case transaction.TypeTransfer:
		account.CurrentBalance += amount
	}
}

// recomputeInvoices brings the bills of the removed charges back in line. Leaving
// them stale makes an invoice claim money that is no longer owed, and the card's
// balance and its invoice then disagree — the drift a reconciliation has to chase.
//
// A refusal is not fatal: a PAID invoice is closed history and the recalculation
// declines it, which must not undo a deletion that already happened.
func (uc *DeleteTransactionUseCase) recomputeInvoices(txns ...*transaction.Transaction) {
	if uc.invoiceRecalculator == nil {
		return
	}
	// Two different links, two different meanings. invoice_id says "this charge is ON
	// that bill"; paid_invoice_id says "this entry PAYS that bill". Undoing a charge
	// changes what the bill is worth; undoing a payment changes what it still owes.
	seenCharges := make(map[string]bool, len(txns))
	seenPayments := make(map[string]bool, len(txns))
	for _, t := range txns {
		if t == nil {
			continue
		}
		if t.InvoiceID != nil && !seenCharges[*t.InvoiceID] {
			seenCharges[*t.InvoiceID] = true
			_, _ = uc.invoiceRecalculator.Execute(*t.InvoiceID)
		}
		if t.PaidInvoiceID != nil && !seenPayments[*t.PaidInvoiceID] {
			seenPayments[*t.PaidInvoiceID] = true
			_, _ = uc.invoiceRecalculator.RestatePayments(*t.PaidInvoiceID)
		}
	}
}
