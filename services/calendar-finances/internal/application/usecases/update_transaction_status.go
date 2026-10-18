package usecases

import (
	"errors"
	"log"
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

// cancelIntent carries the motive of a cancellation from the addressed row to its
// linked leg. A cross-profile pair is one act in two rows, so both have to record the
// same motive, the same actor and the same instant — a pair audited on one side only
// is a pair whose two halves cannot be matched up later.
type cancelIntent struct {
	reason transaction.ReversalReason
	note   string
	actor  string
	at     time.Time
}

type UpdateTransactionStatusUseCase struct {
	repo                transaction.Repository
	accountRepo         bankaccount.Repository
	balanceRecalculator BalanceRecalculator
	invoiceRecalculator InvoiceRecalculator
}

func NewUpdateTransactionStatusUseCase(repo transaction.Repository, accountRepo bankaccount.Repository, recalculator BalanceRecalculator) *UpdateTransactionStatusUseCase {
	return &UpdateTransactionStatusUseCase{repo: repo, accountRepo: accountRepo, balanceRecalculator: recalculator}
}

// SetInvoiceRecalculator wires the bill-total refresh, following how the other
// cross-cutting collaborators are attached after construction.
//
// What makes this necessary is CANCELLED, not CONFIRMED. SumByInvoiceID excludes
// CANCELLED and REVERSED and counts everything else, so a planned charge is already
// in its bill's total and confirming it moves nothing — cancelling one takes it out.
// This path had no notion of an invoice at all, so cancelling a planned card charge
// left the bill still claiming it.
//
// Confirming is refreshed anyway, and deliberately: the two transitions are one act
// here, the recomputation is idempotent, and pairing the refresh with the status
// change rather than with one particular status is what stops this from breaking
// silently the day that filter changes.
func (uc *UpdateTransactionStatusUseCase) SetInvoiceRecalculator(r InvoiceRecalculator) {
	uc.invoiceRecalculator = r
}

// restateBills refreshes each named bill once. A failure is logged, never returned,
// for the same reason the balance update above does not fail the operation: the
// status is already persisted, so reporting an error here would describe a change
// that did happen as one that did not.
func (uc *UpdateTransactionStatusUseCase) restateBills(invoiceIDs ...*string) {
	if uc.invoiceRecalculator == nil {
		return
	}
	seen := make(map[string]bool, len(invoiceIDs))
	for _, id := range invoiceIDs {
		if id == nil || seen[*id] {
			continue
		}
		seen[*id] = true
		if _, err := uc.invoiceRecalculator.RefreshTotal(*id); err != nil {
			log.Printf("invoice %s: refreshing the total after a status change failed: %v", *id, err)
		}
	}
}

func (uc *UpdateTransactionStatusUseCase) Execute(id string, input UpdateTransactionStatusInput) (*transaction.Transaction, error) {
	tx, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, ErrTransactionNotFound
	}

	// Set only on the cancel path, and read again when the linked leg is validated.
	var cancellation *cancelIntent

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
		// Captured so the linked leg goes through the SAME domain call with the same
		// motive, actor and instant.
		cancellation = &cancelIntent{
			reason: transaction.ReversalReason(strings.ToUpper(*input.ReasonCode)),
			note:   reason,
			actor:  *input.Actor,
			at:     time.Now(),
		}
		if err := tx.Cancel(cancellation.reason, cancellation.note, cancellation.actor, cancellation.at); err != nil {
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

		// The linked leg has to go through Cancel() because CancelStatus writes
		// `SET status = $2` from txn.Status, the struct's own field. A leg that never
		// changed status therefore had its OLD status written back, and the write
		// reported success: a silent no-op, audit columns left NULL. The comment in
		// the write block used to claim the domain had already moved this leg. It had
		// not — only the addressed one.
		//
		// What that costs, measured: the surviving leg goes on counting. A linked leg
		// that is a card charge stays on its bill, so the other profile's invoice still
		// claims a debt whose counterpart was cancelled.
		//
		// It does NOT move that profile's balance. updateBalanceOnStatusChange only
		// acts on a transition to or from CONFIRMED, and a CONFIRMED leg is refused by
		// CanTransitionTo just above — "a confirmed transaction must be reversed, not
		// cancelled". An earlier version of this comment asserted the balance damage
		// anyway, without measuring it; every combination of the two legs' statuses was
		// then run and no balance moved in any of them.
		//
		// Done HERE, in the validation block, and the position is LOAD-BEARING: the
		// found.CanTransitionTo above has to run before persistStatus writes the
		// addressed row, or a mixed pair — this leg CONFIRMED, the addressed one
		// PLANNED — is refused only after the addressed row is already cancelled and
		// audited. (Cancel() itself cannot refuse what CanTransitionTo accepted, same
		// reason and actor already checked, so that call alone would be hygiene. The
		// BLOCK is not. An earlier comment said "hygiene" of the whole thing, which
		// invites moving it down.)
		//
		// A leg that already carries a cancellation record keeps it. CanTransitionTo
		// permits CANCELLED -> CANCELLED, and Cancel() overwrites reversed_at, reason
		// and actor unconditionally while keeping the older note — forging a record of
		// an event that never happened: this cancellation's motive and actor welded to
		// the first one's note.
		//
		// Tested on the RECORD, not on the status. A row can be CANCELLED with every
		// audit column NULL — POST /transactions accepts status=CANCELLED and gives the
		// cross-profile destination leg the same status without writing any reversal
		// column, and the CHECK constraint only enforces the trail for REVERSED. Keying
		// off the status alone skipped those too, leaving the pair audited on one side
		// only: precisely the state this cancelIntent exists to prevent.
		//
		// Skipping Cancel() does not mean the row is untouched — persistStatus below
		// still writes it, which is how issue 2625 (the addressed row's occurred_on
		// landing on this leg) reaches even an exempted leg.
		if cancellation != nil && (found.Status != transaction.StatusCancelled || found.ReversedBy == nil) {
			if err := found.Cancel(cancellation.reason, cancellation.note, cancellation.actor, cancellation.at); err != nil {
				return nil, err
			}
		}

		linkedTx = found
	}

	if err := uc.persistStatus(tx, targetStatus, occurredAt); err != nil {
		return nil, err
	}
	// Deferred from here, because from here the bill is out of date. Placing it at
	// the end let the linked-leg write below return early and skip it: measured on a
	// cancelled card charge, the bill stayed stored at 500.00 while its own lines
	// added up to 0.00 — this PR's own defect, through the path it did not cover.
	//
	// Both legs: a cross-profile pair can have one side on a card. Reading
	// linkedTx inside the closure rather than as an argument keeps it correct if the
	// leg is only resolved later.
	defer func() {
		if linkedTx != nil {
			uc.restateBills(tx.InvoiceID, linkedTx.InvoiceID)
			return
		}
		uc.restateBills(tx.InvoiceID)
	}()

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
		if err := uc.persistStatus(linkedTx, targetStatus, occurredAt); err != nil {
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

// persistStatus writes a status change through the path that keeps its audit. A
// cancellation carries why and who; everything else has neither and takes the plain
// route.
func (uc *UpdateTransactionStatusUseCase) persistStatus(
	txn *transaction.Transaction,
	target transaction.Status,
	occurredAt time.Time,
) error {
	if target == transaction.StatusCancelled {
		return uc.repo.CancelStatus(txn, occurredAt)
	}
	return uc.repo.UpdateStatus(txn.ID, target, occurredAt, txn.Notes)
}
