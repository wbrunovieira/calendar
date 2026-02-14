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

	// Handle linked transaction (cross-profile paired transactions)
	if txn.LinkedTransactionID != nil {
		linkedTxn, err := uc.repo.GetByID(*txn.LinkedTransactionID)
		if err == nil {
			// Reverse linked transaction balance
			if linkedTxn.Status == transaction.StatusConfirmed {
				linkedAccount, err := uc.accountRepo.FindByID(linkedTxn.BankAccountID)
				if err == nil && linkedAccount.Type != bankaccount.AccountTypeCreditCard {
					uc.reverseBalance(linkedAccount, linkedTxn.Type, linkedTxn.Amount)
					linkedAccount.UpdatedAt = time.Now()
					if err := uc.accountRepo.Update(linkedAccount); err != nil {
						return err
					}
				}
			}
			// Delete linked transaction
			_ = uc.repo.Delete(linkedTxn.ID)
		}
	}

	// Reverse balance for CONFIRMED non-credit-card transactions
	if txn.Status == transaction.StatusConfirmed {
		account, err := uc.accountRepo.FindByID(txn.BankAccountID)
		if err == nil && account.Type != bankaccount.AccountTypeCreditCard {
			uc.reverseBalance(account, txn.Type, txn.Amount)
			account.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(account); err != nil {
				return err
			}

			// For same-profile TRANSFER: also reverse the destination account credit
			if txn.Type == transaction.TypeTransfer && txn.DestinationAccountID != nil {
				destAccount, err := uc.accountRepo.FindByID(*txn.DestinationAccountID)
				if err == nil {
					destAccount.CurrentBalance -= txn.Amount
					destAccount.UpdatedAt = time.Now()
					if err := uc.accountRepo.Update(destAccount); err != nil {
						return err
					}
				}
			}
		}
	}

	if err := uc.repo.Delete(id); err != nil {
		return ErrTransactionNotFound
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
