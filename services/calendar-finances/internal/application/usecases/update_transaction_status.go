package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type UpdateTransactionStatusInput struct {
	Status     string  `json:"status"`
	OccurredOn *string `json:"occurredOn,omitempty"`
	Reason     *string `json:"reason,omitempty"`
}

type UpdateTransactionStatusUseCase struct {
	repo transaction.Repository
}

func NewUpdateTransactionStatusUseCase(repo transaction.Repository) *UpdateTransactionStatusUseCase {
	return &UpdateTransactionStatusUseCase{repo: repo}
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

	if err := uc.repo.UpdateStatus(tx.ID, tx.Status, occurredAt, tx.Notes); err != nil {
		return nil, err
	}

	return tx, nil
}
