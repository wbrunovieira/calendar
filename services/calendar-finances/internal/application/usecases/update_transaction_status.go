package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type UpdateTransactionStatusInput struct {
	Status     string  `json:"status"`
	OccurredOn *string `json:"occurredOn,omitempty"`
	Reason     *string `json:"reason,omitempty"`
}

type UpdateTransactionStatusUseCase struct {
	repo                transaction.Repository
	accountRepo         bankaccount.Repository
	balanceRecalculator BalanceRecalculator
}

func NewUpdateTransactionStatusUseCase(repo transaction.Repository, accountRepo bankaccount.Repository, recalculator BalanceRecalculator) *UpdateTransactionStatusUseCase {
	return &UpdateTransactionStatusUseCase{repo: repo, accountRepo: accountRepo, balanceRecalculator: recalculator}
}

func (uc *UpdateTransactionStatusUseCase) Execute(id string, input UpdateTransactionStatusInput) (*transaction.Transaction, error) {
	tx, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, ErrTransactionNotFound
	}

	targetStatus, err := parseTransactionStatus(input.Status)
	if err != nil {
		return nil, err
	}

	// Capture old status BEFORE any mutations
	oldStatus := tx.Status

	var occurredAt time.Time
	switch targetStatus {
	case transaction.StatusConfirmed:
		if input.OccurredOn != nil {
			occurredAt, err = parseDate(*input.OccurredOn)
			if err != nil {
				return nil, err
			}
		} else {
			occurredAt = time.Now()
		}
		if err := tx.Confirm(occurredAt); err != nil {
			return nil, err
		}
	case transaction.StatusCancelled:
		var reason string
		if input.Reason != nil {
			reason = *input.Reason
		}
		tx.Cancel(reason)
		occurredAt = tx.OccurredOn
	case transaction.StatusPlanned:
		tx.Status = transaction.StatusPlanned
		if input.OccurredOn != nil {
			if occurredAt, err = parseDate(*input.OccurredOn); err != nil {
				return nil, err
			}
			tx.OccurredOn = occurredAt
		}
		tx.UpdatedAt = time.Now()
		occurredAt = tx.OccurredOn
	default:
		return nil, ErrInvalidInput
	}

	if tx.OccurredOn.IsZero() {
		occurredAt = time.Now()
		tx.OccurredOn = occurredAt
	}

	if err := uc.repo.UpdateStatus(tx.ID, targetStatus, occurredAt, tx.Notes); err != nil {
		return nil, err
	}

	// Update bank account balance based on status change
	if err := uc.updateBalanceOnStatusChange(tx, oldStatus, targetStatus); err != nil {
		// Log error but don't fail the operation
		// The transaction status is already updated
	}
	if acc, err := uc.accountRepo.FindByID(tx.BankAccountID); err == nil && acc.Type != bankaccount.AccountTypeCreditCard {
		_ = recalculateAccounts(uc.balanceRecalculator, tx.BankAccountID)
	}

	// Handle linked transaction (cross-profile paired transactions)
	if tx.LinkedTransactionID != nil {
		linkedTx, err := uc.repo.GetByID(*tx.LinkedTransactionID)
		if err == nil {
			linkedOldStatus := linkedTx.Status
			// Update linked transaction status
			_ = uc.repo.UpdateStatus(linkedTx.ID, targetStatus, occurredAt, linkedTx.Notes)
			// Update linked transaction balance
			_ = uc.updateBalanceOnStatusChange(linkedTx, linkedOldStatus, targetStatus)
			if linkedAcc, err := uc.accountRepo.FindByID(linkedTx.BankAccountID); err == nil && linkedAcc.Type != bankaccount.AccountTypeCreditCard {
				_ = recalculateAccounts(uc.balanceRecalculator, linkedTx.BankAccountID)
			}
			linkedTx.Status = targetStatus
		}
	}

	tx.Status = targetStatus
	return tx, nil
}

// updateBalanceOnStatusChange updates the bank account balance when transaction status changes
// NOTE: Credit card transactions do NOT update balance - the balance is only
// affected when the invoice is paid (via PayInvoiceUseCaseV2)
func (uc *UpdateTransactionStatusUseCase) updateBalanceOnStatusChange(tx *transaction.Transaction, oldStatus, newStatus transaction.Status) error {
	if uc.accountRepo == nil {
		return nil
	}

	account, err := uc.accountRepo.FindByID(tx.BankAccountID)
	if err != nil {
		return err
	}

	// Credit card transactions don't affect balance - only invoice payment does
	if account.Type == bankaccount.AccountTypeCreditCard {
		return nil
	}

	// Calculate balance change based on status transition
	var balanceChange float64

	// If moving TO CONFIRMED: apply the transaction
	// If moving FROM CONFIRMED: reverse the transaction
	if newStatus == transaction.StatusConfirmed && oldStatus != transaction.StatusConfirmed {
		// Apply transaction (PLANNED/CANCELLED -> CONFIRMED)
		switch tx.Type {
		case transaction.TypeExpense:
			balanceChange = -tx.Amount
		case transaction.TypeIncome:
			balanceChange = tx.Amount
		case transaction.TypeTransfer:
			balanceChange = -tx.Amount
		}
	} else if oldStatus == transaction.StatusConfirmed && newStatus != transaction.StatusConfirmed {
		// Reverse transaction (CONFIRMED -> PLANNED/CANCELLED)
		switch tx.Type {
		case transaction.TypeExpense:
			balanceChange = tx.Amount
		case transaction.TypeIncome:
			balanceChange = -tx.Amount
		case transaction.TypeTransfer:
			balanceChange = tx.Amount
		}
	}

	if balanceChange != 0 {
		account.CurrentBalance += balanceChange
		account.UpdatedAt = time.Now()
		if err := uc.accountRepo.Update(account); err != nil {
			return err
		}

		// For TRANSFER: also update the destination account
		if tx.Type == transaction.TypeTransfer && tx.DestinationAccountID != nil {
			destAccount, err := uc.accountRepo.FindByID(*tx.DestinationAccountID)
			if err != nil {
				return err
			}
			// Destination gets the opposite of source: credited on confirm, debited on cancel
			destAccount.CurrentBalance -= balanceChange
			destAccount.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(destAccount); err != nil {
				return err
			}
		}
	}

	return nil
}
