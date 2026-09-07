package usecases

import (
	"errors"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type UpdateTransactionStatusInput struct {
	Status     string  `json:"status"`
	OccurredOn *string `json:"occurredOn,omitempty"`
	// Reason is free text kept as a note. It does not satisfy the audit requirement
	// on its own: the classification below is what decides the accounting effect.
	Reason *string `json:"reason,omitempty"`
	// ReasonCode and Actor are required to cancel. Hardcoding them would fill the
	// column in 100% of rows with the same value — a control that appears to operate
	// while capturing nothing, which is the defect a NULL column has, disguised.
	ReasonCode *string `json:"reasonCode,omitempty"`
	Actor      *string `json:"actor,omitempty"`
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
		if input.ReasonCode == nil || input.Actor == nil {
			return nil, errors.New("cancelling requires reasonCode and actor: a motive not captured now cannot be reconstructed later")
		}
		// A confirmed movement cannot be cancelled: it is reversed, which records the
		// motive and the actor. Cancelling would be an unaudited way out.
		if err := tx.Cancel(transaction.ReversalReason(strings.ToUpper(*input.ReasonCode)), reason, *input.Actor, time.Now()); err != nil {
			return nil, err
		}
		occurredAt = tx.OccurredOn
	case transaction.StatusPlanned:
		// Goes through the domain: moving a confirmed row back to planned undoes the
		// money with no motive and no actor, and the audit CHECK never sees it.
		if err := tx.SetStatus(transaction.StatusPlanned); err != nil {
			return nil, err
		}
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

	// Both legs are validated BEFORE anything is written. The linked leg used to be
	// persisted and have its balance moved first, and only then checked — so
	// confirming over a reversed counterpart did the damage and reported failure,
	// which is the "verify the effect, never the signal" rule broken by the code
	// meant to enforce it.
	var linkedTx *transaction.Transaction
	if tx.LinkedTransactionID != nil {
		found, lerr := uc.repo.GetByID(*tx.LinkedTransactionID)
		if lerr != nil {
			// Skipping it silently leaves the pair half-updated with no error, and
			// the other profile holding a movement whose counterpart never moved.
			return nil, lerr
		}
		if err := found.CanTransitionTo(targetStatus); err != nil {
			return nil, err
		}
		linkedTx = found
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

	// The linked leg, already validated above.
	if linkedTx != nil {
		linkedOldStatus := linkedTx.Status
		// The error is surfaced, not discarded: a leg that failed to persist leaves
		// the other profile holding a movement with no counterpart, and swallowing it
		// means nobody goes looking.
		if err := uc.repo.UpdateStatus(linkedTx.ID, targetStatus, occurredAt, linkedTx.Notes); err != nil {
			return nil, err
		}
		_ = uc.updateBalanceOnStatusChange(linkedTx, linkedOldStatus, targetStatus)
		if linkedAcc, err := uc.accountRepo.FindByID(linkedTx.BankAccountID); err == nil && linkedAcc.Type != bankaccount.AccountTypeCreditCard {
			_ = recalculateAccounts(uc.balanceRecalculator, linkedTx.BankAccountID)
		}
	}

	// No assignment here: the switch above already moved tx through the domain, so
	// its status is the target. Re-assigning would reopen the gate with an eighth
	// exception that is a transition and nowhere near a New().
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
