package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type DeleteTransactionUseCase struct {
	repo                transaction.Repository
	accountRepo         bankaccount.Repository
	balanceRecalculator BalanceRecalculator
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

	affected := affectedAccounts(txn, linked)

	if linked != nil {
		_ = uc.repo.Delete(linked.ID)
	}
	if err := uc.repo.Delete(id); err != nil {
		return ErrTransactionNotFound
	}

	if uc.balanceRecalculator != nil {
		recalculateAccounts(uc.balanceRecalculator, affected...)
		return nil
	}

	// Without a recalculator wired, undo each leg by hand. Same result, but derived
	// from the transaction instead of from the ledger.
	return uc.reverseByHand(txn, linked)
}

// affectedAccounts lists every account whose balance depended on the rows being
// removed. Only confirmed transactions ever moved money.
func affectedAccounts(txns ...*transaction.Transaction) []string {
	ids := make([]string, 0, len(txns)*2)
	for _, t := range txns {
		if t == nil || t.Status != transaction.StatusConfirmed {
			continue
		}
		ids = append(ids, t.BankAccountID)
		if t.DestinationAccountID != nil {
			ids = append(ids, *t.DestinationAccountID)
		}
	}
	return deduplicateIDs(ids...)
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
