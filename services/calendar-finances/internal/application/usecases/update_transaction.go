package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type UpdateTransactionInput struct {
	BankAccountID           string   `json:"bankAccountId"`
	DestinationAccountID    *string  `json:"destinationAccountId,omitempty"`
	CategoryID              *string  `json:"categoryId,omitempty"`
	Type                    string   `json:"type"`
	Status                  *string  `json:"status,omitempty"`
	Amount                  float64  `json:"amount"`
	Currency                string   `json:"currency"`
	Description             string   `json:"description"`
	Notes                   *string  `json:"notes,omitempty"`
	CostCenter              *string  `json:"costCenter,omitempty"`
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
	}

	// Capture old state for balance adjustment and invoice reassignment
	oldAmount := existing.Amount
	oldType := existing.Type
	oldAccountID := existing.BankAccountID
	oldStatus := existing.Status
	oldDestAccountID := existing.DestinationAccountID
	oldOccurredOn := existing.OccurredOn

	// Update the transaction fields
	existing.BankAccountID = input.BankAccountID
	existing.DestinationAccountID = destinationAccountID
	existing.CategoryID = input.CategoryID
	existing.Type = typeValue
	existing.Status = status
	existing.Amount = input.Amount
	existing.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if existing.Currency == "" {
		existing.Currency = "BRL"
	}
	existing.Description = strings.TrimSpace(input.Description)
	existing.Notes = input.Notes
	existing.CostCenter = input.CostCenter
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

	// Reassign invoice when bank account or date changes for credit card transactions
	if existing.BankAccountID != oldAccountID || !existing.OccurredOn.Equal(oldOccurredOn) {
		if account.Type == bankaccount.AccountTypeCreditCard && typeValue == transaction.TypeExpense {
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

	// Persist changes
	if err := uc.transactionRepo.Update(existing); err != nil {
		return nil, err
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
