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
	repo        transaction.Repository
	accountRepo bankaccount.Repository
}

func NewUpdateTransactionStatusUseCase(repo transaction.Repository, accountRepo bankaccount.Repository) *UpdateTransactionStatusUseCase {
	return &UpdateTransactionStatusUseCase{repo: repo, accountRepo: accountRepo}
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

	// Get the old status before updating
	oldStatus := tx.Status

	if err := uc.repo.UpdateStatus(tx.ID, targetStatus, occurredAt, tx.Notes); err != nil {
		return nil, err
	}

	// Update bank account balance based on status change
	if err := uc.updateBalanceOnStatusChange(tx, oldStatus, targetStatus); err != nil {
		// Log error but don't fail the operation
		// The transaction status is already updated
	}

	tx.Status = targetStatus
	return tx, nil
}

// updateBalanceOnStatusChange updates the bank account balance when transaction status changes
func (uc *UpdateTransactionStatusUseCase) updateBalanceOnStatusChange(tx *transaction.Transaction, oldStatus, newStatus transaction.Status) error {
	if uc.accountRepo == nil {
		return nil
	}

	account, err := uc.accountRepo.FindByID(tx.BankAccountID)
	if err != nil {
		return err
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
		return uc.accountRepo.Update(account)
	}

	return nil
}
