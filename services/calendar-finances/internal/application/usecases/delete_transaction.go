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
func (uc *DeleteTransactionUseCase) Execute(id string) error {
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
	toDelete := []string{id}
	if linked != nil {
		toDelete = append(toDelete, linked.ID)
	}
	if err := uc.repo.DeleteMany(toDelete); err != nil {
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
	return uc.reverseByHand(txn, linked)
}

// affectedAccounts lists every account whose balance depended on the rows being
// removed. Only confirmed transactions ever moved money.
//
// Credit cards are left out on purpose. Creating a card transaction does not move the
// card's balance — every write path skips them, and the balance is a consequence of
// paying the invoice — so recomputing one here would make the card jump by the sum of
// every purchase since the last bill, purely because something unrelated was deleted.
// Making a card's balance track its transactions is a worthwhile change, but it is a
// different one and belongs in its own PR.
func (uc *DeleteTransactionUseCase) affectedAccounts(txns ...*transaction.Transaction) []string {
	ids := make([]string, 0, len(txns)*2)
	for _, t := range txns {
		if t == nil || t.Status != transaction.StatusConfirmed {
			continue
		}
		if !uc.isCreditCard(t.BankAccountID) {
			ids = append(ids, t.BankAccountID)
		}
		if t.DestinationAccountID != nil {
			ids = append(ids, *t.DestinationAccountID)
		}
	}
	return deduplicateIDs(ids...)
}

func (uc *DeleteTransactionUseCase) isCreditCard(accountID string) bool {
	account, err := uc.accountRepo.FindByID(accountID)
	return err == nil && account.IsCreditCard()
}

func (uc *DeleteTransactionUseCase) reverseByHand(txns ...*transaction.Transaction) error {
	for _, txn := range txns {
		if txn == nil || txn.Status != transaction.StatusConfirmed {
			continue
		}

		// Creating a transaction on a credit card does not move the card's balance —
		// a card's position is derived from its invoices — so undoing one must not
		// move it either. The recalculator path above has no such asymmetry: it
		// recomputes every account from the ledger.
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
	seen := make(map[string]bool, len(txns))
	for _, t := range txns {
		if t == nil || t.InvoiceID == nil || seen[*t.InvoiceID] {
			continue
		}
		seen[*t.InvoiceID] = true
		_, _ = uc.invoiceRecalculator.Execute(*t.InvoiceID)
	}
}
