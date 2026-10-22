package usecases

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type UpdateTransactionInput struct {
	BankAccountID        string  `json:"bankAccountId"`
	DestinationAccountID *string `json:"destinationAccountId,omitempty"`
	CategoryID           *string `json:"categoryId,omitempty"`
	Type                 string  `json:"type"`
	Status               *string `json:"status,omitempty"`
	Amount               float64 `json:"amount"`
	Currency             string  `json:"currency"`
	Description          string  `json:"description"`
	Notes                *string `json:"notes,omitempty"`
	CostCenter           *string `json:"costCenter,omitempty"`
	// CostCenterID links the row to a client, project or department. The create
	// path has always accepted it; this one never did, so a transaction posted
	// without one could not be given one afterwards — only reversed and re-posted,
	// which spends a reversal on a metadata correction and makes the ledger's
	// history say something did not happen when it did.
	//
	// OMITTING IT KEEPS WHAT THE ROW HAS, which breaks this route's full-replace
	// rule on purpose. Every other field is cleared by omission because the clients
	// echo it back: the dashboard's edit form loads costCenter off the transaction
	// and sends it again, so an omission there is a deliberate clear. No client
	// sends costCenterId — the frontend has no notion of it at all — so clearing on
	// omission would mean every description fix made in the dashboard silently
	// unfiles the row from its client or project, with nothing on screen to show it
	// happened. Consistency with a field the caller can see is not the same as
	// consistency with one it cannot.
	//
	// To clear it deliberately, send an empty string.
	CostCenterID            *string  `json:"costCenterId,omitempty"`
	IsPersonalReimbursement bool     `json:"isPersonalReimbursement"`
	OccurredOn              string   `json:"occurredOn"`
	DueOn                   *string  `json:"dueOn,omitempty"`
	ReminderOn              *string  `json:"reminderOn,omitempty"` // Optional reminder date for alerts
	RecurrenceRule          *string  `json:"recurrenceRule,omitempty"`
	InstallmentNumber       *int     `json:"installmentNumber,omitempty"`
	InstallmentTotal        *int     `json:"installmentTotal,omitempty"`
	ExternalID              *string  `json:"externalId,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
}

type UpdateTransactionUseCase struct {
	accountRepo         bankaccount.Repository
	categoryRepo        category.Repository
	transactionRepo     transaction.Repository
	invoiceRepo         invoice.Repository
	balanceRecalculator BalanceRecalculator
	invoiceRecalculator InvoiceRecalculator
	costCenterRepo      costcenter.Repository
}

// SetCostCenterRepository wires the cost centre check, following how the other
// collaborators attach after construction.
//
// Attached rather than required because the constructor already has five arguments
// and some thirty call sites; the risk that buys — a validator quietly left unwired
// — is answered by refusing an id it cannot check rather than writing it unverified.
// See the guard in Execute.
func (uc *UpdateTransactionUseCase) SetCostCenterRepository(r costcenter.Repository) {
	uc.costCenterRepo = r
}

// resolveCostCenter accepts an id only once it is known to exist and to belong to
// this transaction's profile.
//
// A cost centre from another profile would file one company's spending under
// another's project, which is why the create path checks the same two things.
func (uc *UpdateTransactionUseCase) resolveCostCenter(id *string, profileID string) error {
	// Absent means keep, and the empty string means clear. Neither names a cost
	// centre, so neither needs one to exist.
	if id == nil || *id == "" {
		return nil
	}
	// Fails CLOSED. With no repository there is no way to check the id, and an
	// unchecked id written to the ledger is worse than a refused edit: it looks
	// correct and nothing downstream can tell. This repository has shipped the
	// opposite shape — a collaborator left unwired, silently doing nothing — more
	// than once.
	if uc.costCenterRepo == nil {
		return errors.New("a cost centre cannot be verified: no cost centre repository is wired")
	}
	center, err := uc.costCenterRepo.FindByID(*id)
	if errors.Is(err, costcenter.ErrNotFound) {
		return ErrCostCenterNotFound
	}
	if err != nil {
		// A failed read is not a missing cost centre. Reporting it as one answers a
		// database outage with 400 and tells the caller their input was wrong.
		return err
	}
	if center.ProfileID != profileID {
		return ErrCostCenterNotFound
	}
	return nil
}

// SetInvoiceRecalculator wires the bill-total refresh, following how the other
// cross-cutting collaborators are attached after construction.
func (uc *UpdateTransactionUseCase) SetInvoiceRecalculator(r InvoiceRecalculator) {
	uc.invoiceRecalculator = r
}

// restateLineBills refreshes each bill the edited charge was or is a line of, once.
//
// Both, because an edit can move a charge between cycles: the bill it left has to
// stop counting it and the bill it joined has to start. Errors are logged rather
// than returned — the row is already written, and reporting a failed refresh as a
// failed edit invites a retry that applies the edit twice.
func (uc *UpdateTransactionUseCase) restateLineBills(invoiceIDs ...*string) {
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
			log.Printf("invoice %s: refreshing the total after an edit failed: %v", *id, err)
		}
	}
}

func NewUpdateTransactionUseCase(
	accountRepo bankaccount.Repository,
	categoryRepo category.Repository,
	transactionRepo transaction.Repository,
	invoiceRepo invoice.Repository,
	recalculator BalanceRecalculator,
) *UpdateTransactionUseCase {
	return &UpdateTransactionUseCase{
		accountRepo:         accountRepo,
		categoryRepo:        categoryRepo,
		transactionRepo:     transactionRepo,
		invoiceRepo:         invoiceRepo,
		balanceRecalculator: recalculator,
	}
}

func (uc *UpdateTransactionUseCase) Execute(id string, input UpdateTransactionInput) (*transaction.Transaction, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInput
	}

	// Fetch existing transaction
	existing, err := uc.transactionRepo.GetByID(id)
	if err != nil {
		return nil, ErrTransactionNotFound
	}

	// Validate bank account
	account, err := uc.accountRepo.FindByID(input.BankAccountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	if account.ProfileID != existing.ProfileID {
		return nil, ErrBankAccountMismatch
	}

	// Parse and validate type
	typeValue, err := parseTransactionType(input.Type)
	if err != nil {
		return nil, err
	}

	// Validate category if provided
	if input.CategoryID != nil {
		cat, err := uc.categoryRepo.FindByID(*input.CategoryID)
		if err != nil {
			return nil, ErrCategoryNotFound
		}
		if cat.ProfileID != existing.ProfileID {
			return nil, ErrCategoryNotFound
		}
		if !isCategoryCompatible(typeValue, cat.Type) {
			return nil, ErrInvalidInput
		}
	}

	// Checked here, beside the category, and before anything is written: a refusal
	// that lands after the row has already changed is worse than the refusal it
	// reports.
	if err := uc.resolveCostCenter(input.CostCenterID, existing.ProfileID); err != nil {
		return nil, err
	}

	// Validate destination account for transfers
	var destinationAccountID *string
	if typeValue == transaction.TypeTransfer {
		if input.DestinationAccountID == nil {
			return nil, ErrDestinationRequired
		}
		destination, err := uc.accountRepo.FindByID(*input.DestinationAccountID)
		if err != nil {
			return nil, ErrBankAccountNotFound
		}
		if destination.ID == account.ID {
			return nil, ErrInvalidInput
		}
		if destination.ProfileID != existing.ProfileID {
			return nil, ErrBankAccountMismatch
		}
		// One row, one Amount: editing a transfer to point at an account in another
		// currency would credit it with a number that means nothing there.
		if !strings.EqualFold(strings.TrimSpace(account.Currency), strings.TrimSpace(destination.Currency)) {
			return nil, ErrCurrencyMismatch
		}
		destinationAccountID = &destination.ID
	}

	// Parse dates
	occurredOn, err := parseDate(input.OccurredOn)
	if err != nil {
		return nil, ErrInvalidInput
	}

	var dueOn *time.Time
	if input.DueOn != nil {
		d, err := parseDate(*input.DueOn)
		if err != nil {
			return nil, ErrInvalidInput
		}
		dueOn = &d
	}

	var reminderOn *time.Time
	if input.ReminderOn != nil {
		r, err := parseDate(*input.ReminderOn)
		if err != nil {
			return nil, ErrInvalidInput
		}
		reminderOn = &r
	}

	// Parse status if provided, otherwise keep existing
	status := existing.Status
	if input.Status != nil {
		status, err = parseTransactionStatus(*input.Status)
		if err != nil {
			return nil, ErrInvalidInput
		}
		// Ending a transaction's life has to say why and by whom, and this route
		// carries neither. It is a full replace of the editable fields; the status
		// route is where a lifecycle change belongs, because it demands the motive and
		// the actor and persists them.
		//
		// Without this the requirement was enforceable at one door and open at the
		// other: pausing a recurrence cancelled its planned entry through here, with
		// reversal_reason and reversed_by left NULL, and the database CHECK never saw
		// it because it only covers REVERSED.
		if status != existing.Status && (status == transaction.StatusCancelled || status == transaction.StatusReversed) {
			return nil, fmt.Errorf(
				"%w: use PUT /transactions/{id}/status to cancel (it requires reasonCode and actor) or POST /transactions/{id}/reversal to reverse",
				ErrInvalidInput)
		}
	}

	// Capture old state for balance adjustment and invoice reassignment
	oldAmount := existing.Amount
	oldType := existing.Type
	oldAccountID := existing.BankAccountID
	oldStatus := existing.Status
	oldDestAccountID := existing.DestinationAccountID
	oldOccurredOn := existing.OccurredOn
	// The bill the charge was a line of BEFORE the edit. An edit can change the
	// amount, or move the charge to another cycle or another card, and then two
	// bills are wrong: the one it left and the one it joined.
	oldInvoiceID := existing.InvoiceID

	// Update the transaction fields
	existing.BankAccountID = input.BankAccountID
	existing.DestinationAccountID = destinationAccountID
	existing.CategoryID = input.CategoryID
	existing.Type = typeValue
	// Through the domain, not by assignment: this endpoint was the path that brought
	// a reversed row back to CONFIRMED and applied its balance a second time.
	if err := existing.SetStatus(status); err != nil {
		return nil, err
	}
	existing.Amount = input.Amount
	// The account decides, here too. This is a full replacement, so a body that omits
	// the currency used to rewrite a EUR row to BRL — and now that reconciliation
	// compares currencies, that edit turns a fully accounted charge into reported
	// missing money on the next run.
	currency, err := resolveCurrency(input.Currency, account)
	if err != nil {
		return nil, err
	}
	existing.Currency = currency
	existing.Description = strings.TrimSpace(input.Description)
	existing.Notes = input.Notes
	existing.CostCenter = input.CostCenter
	// Absent keeps, empty string clears, anything else was validated above. See the
	// field's comment for why this one does not follow the full-replace rule.
	if input.CostCenterID != nil {
		if *input.CostCenterID == "" {
			existing.CostCenterID = nil
		} else {
			existing.CostCenterID = input.CostCenterID
		}
	}
	existing.IsPersonalReimbursement = input.IsPersonalReimbursement
	existing.OccurredOn = occurredOn
	existing.DueOn = dueOn
	existing.ReminderOn = reminderOn
	existing.RecurrenceRule = input.RecurrenceRule
	existing.InstallmentNumber = input.InstallmentNumber
	existing.InstallmentTotal = input.InstallmentTotal
	existing.ExternalID = input.ExternalID
	existing.Tags = sanitizeTags(input.Tags)
	existing.UpdatedAt = time.Now()

	// Reassign invoice when bank account or date changes for credit card transactions.
	//
	// A PINNED charge is exempt: its bill was chosen explicitly because the issuer
	// billed it on a cycle the date does not imply. A pin that any later edit
	// silently undoes is not a pin.
	//
	// A PAYMENT is exempt for a stronger reason: it is not a line of any bill, so
	// there is no bill for its date to re-derive. Re-filing it as a line while
	// paid_invoice_id still points at the bill it settled double-counts it -- the
	// bill's total drops by the payment and its paid_amount keeps the payment -- and
	// a one-day nudge to the date was enough to trigger it.
	if existing.InvoicePinned || existing.PaidInvoiceID != nil {
		// keep the chosen bill
	} else if existing.BankAccountID != oldAccountID || !existing.OccurredOn.Equal(oldOccurredOn) {
		if account.Type == bankaccount.AccountTypeCreditCard && isInvoiceLine(typeValue) {
			if account.ClosingDay != nil && account.DueDay != nil {
				inv, invErr := getOrCreateInvoiceForDate(uc.invoiceRepo, account, occurredOn)
				if invErr != nil {
					// Detaching the charge would return 200 while the purchase
					// stays on the card and disappears from its bill. Surface it.
					return nil, invErr
				}
				if inv != nil {
					existing.InvoiceID = &inv.ID
				} else {
					existing.InvoiceID = nil
				}
			} else {
				existing.InvoiceID = nil
			}
		} else {
			// Moving to non-credit-card account: clear invoice
			existing.InvoiceID = nil
		}
	}

	// An edit can leave the row unable to settle anything -- a linked funding leg
	// turned into an EXPENSE keeps type=EXPENSE, no destination and paid_invoice_id
	// still set. SumLivePaymentsByInvoiceID has no type filter, so the bill goes on
	// counting it, and Release then answers ErrNotACreditCard: no route left to
	// unlink it, bill PAID forever. So the link is released here rather than
	// stranded.
	settledBillID := existing.PaidInvoiceID
	if existing.PaidInvoiceID != nil && !canStillSettleABill(existing, uc.accountRepo) {
		existing.PaidInvoiceID = nil
	}

	// Persist changes
	if err := uc.transactionRepo.Update(existing); err != nil {
		return nil, err
	}
	// From here the bills are out of date, so the refresh is deferred rather than
	// placed at the return: every step below can still fail, and the charge is
	// already written.
	//
	// This use case restated only the bill a row PAYS (paid_invoice_id, below). The
	// bill a row is a LINE OF was never touched, so editing a card charge from
	// 135,90 to 200,00 — or nudging its date into the next cycle — left both bills
	// stating totals their own transactions no longer supported. Same defect as the
	// create path, one use case over.
	defer uc.restateLineBills(oldInvoiceID, existing.InvoiceID)

	// The bill it settles is restated from the transactions afterwards. Nothing did
	// this, so correcting a leg from 1.018,18 to 900,00 left the bill still recording
	// 1.018,18 paid: 118,18 of debt invisible, and that much phantom limit.
	//
	// A drop is allowed because the edit is the deliberate act that caused it.
	if settledBillID != nil && uc.invoiceRepo != nil {
		settled, ferr := uc.invoiceRepo.FindByID(*settledBillID)
		if ferr == nil && settled != nil {
			restater := invoiceRestater{txRepo: uc.transactionRepo, invoiceRepo: uc.invoiceRepo}
			if rerr := restater.restate(settled, restateAllowingADrop); rerr != nil {
				// Surfaced, not swallowed: a bill left stating a number its own
				// transactions no longer support is invisible until a reconciliation
				// months later.
				return nil, rerr
			}
		}
	}

	// Adjust bank account balances
	if err := uc.adjustBalances(oldAccountID, oldType, oldAmount, oldStatus, oldDestAccountID,
		input.BankAccountID, typeValue, input.Amount, status, destinationAccountID); err != nil {
		return nil, err
	}

	// Recalculate all affected accounts
	ids := deduplicateIDs(oldAccountID, input.BankAccountID)
	if oldDestAccountID != nil {
		ids = append(ids, *oldDestAccountID)
	}
	if destinationAccountID != nil {
		ids = append(ids, *destinationAccountID)
	}
	_ = recalculateAccounts(uc.balanceRecalculator, ids...)

	return existing, nil
}

func (uc *UpdateTransactionUseCase) adjustBalances(
	oldAccountID string, oldType transaction.Type, oldAmount float64, oldStatus transaction.Status, oldDestAccountID *string,
	newAccountID string, newType transaction.Type, newAmount float64, newStatus transaction.Status, newDestAccountID *string,
) error {
	// Reverse old impact if it was CONFIRMED
	if oldStatus == transaction.StatusConfirmed {
		oldAccount, err := uc.accountRepo.FindByID(oldAccountID)
		if err == nil && oldAccount.Type != bankaccount.AccountTypeCreditCard {
			uc.reverseBalance(oldAccount, oldType, oldAmount)
			oldAccount.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(oldAccount); err != nil {
				return err
			}

			// Reverse destination for transfers
			if oldType == transaction.TypeTransfer && oldDestAccountID != nil {
				oldDest, err := uc.accountRepo.FindByID(*oldDestAccountID)
				if err == nil {
					oldDest.CurrentBalance -= oldAmount
					oldDest.UpdatedAt = time.Now()
					if err := uc.accountRepo.Update(oldDest); err != nil {
						return err
					}
				}
			}
		}
	}

	// Apply new impact if it is CONFIRMED
	if newStatus == transaction.StatusConfirmed {
		newAccount, err := uc.accountRepo.FindByID(newAccountID)
		if err == nil && newAccount.Type != bankaccount.AccountTypeCreditCard {
			uc.applyBalance(newAccount, newType, newAmount)
			newAccount.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(newAccount); err != nil {
				return err
			}

			// Apply destination for transfers
			if newType == transaction.TypeTransfer && newDestAccountID != nil {
				newDest, err := uc.accountRepo.FindByID(*newDestAccountID)
				if err == nil {
					newDest.CurrentBalance += newAmount
					newDest.UpdatedAt = time.Now()
					if err := uc.accountRepo.Update(newDest); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (uc *UpdateTransactionUseCase) reverseBalance(account *bankaccount.BankAccount, txType transaction.Type, amount float64) {
	switch txType {
	case transaction.TypeExpense:
		account.CurrentBalance += amount
	case transaction.TypeIncome:
		account.CurrentBalance -= amount
	case transaction.TypeTransfer:
		account.CurrentBalance += amount
	}
}

func (uc *UpdateTransactionUseCase) applyBalance(account *bankaccount.BankAccount, txType transaction.Type, amount float64) {
	switch txType {
	case transaction.TypeExpense:
		account.CurrentBalance -= amount
	case transaction.TypeIncome:
		account.CurrentBalance += amount
	case transaction.TypeTransfer:
		account.CurrentBalance -= amount
	}
}

func sanitizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	var cleaned []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}
