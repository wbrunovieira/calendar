package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type DeleteTransactionUseCase struct {
	repo       transaction.Repository
	accountRepo bankaccount.Repository
}

func NewDeleteTransactionUseCase(repo transaction.Repository, accountRepo bankaccount.Repository) *DeleteTransactionUseCase {
	return &DeleteTransactionUseCase{repo: repo, accountRepo: accountRepo}
}

func (uc *DeleteTransactionUseCase) Execute(id string) error {
	txn, err := uc.repo.GetByID(id)
	if err != nil {
		return ErrTransactionNotFound
	}

	// Reverse balance for CONFIRMED non-credit-card transactions
	if txn.Status == transaction.StatusConfirmed {
		account, err := uc.accountRepo.FindByID(txn.BankAccountID)
		if err == nil && account.Type != bankaccount.AccountTypeCreditCard {
			switch txn.Type {
			case transaction.TypeExpense:
				account.CurrentBalance += txn.Amount
			case transaction.TypeIncome:
				account.CurrentBalance -= txn.Amount
			case transaction.TypeTransfer:
				account.CurrentBalance += txn.Amount
			}
			account.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(account); err != nil {
				return err
			}
		}
	}

	if err := uc.repo.Delete(id); err != nil {
		return ErrTransactionNotFound
	}

	return nil
}
