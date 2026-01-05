package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type UpdateTransactionInput struct {
	BankAccountID        string   `json:"bankAccountId"`
	DestinationAccountID *string  `json:"destinationAccountId,omitempty"`
	CategoryID           *string  `json:"categoryId,omitempty"`
	Type                 string   `json:"type"`
	Status               *string  `json:"status,omitempty"`
	Amount               float64  `json:"amount"`
	Currency             string   `json:"currency"`
	Description          string   `json:"description"`
	Notes                *string  `json:"notes,omitempty"`
	CostCenter           *string  `json:"costCenter,omitempty"`
	OccurredOn           string   `json:"occurredOn"`
	DueOn                *string  `json:"dueOn,omitempty"`
	RecurrenceRule       *string  `json:"recurrenceRule,omitempty"`
	InstallmentNumber    *int     `json:"installmentNumber,omitempty"`
	InstallmentTotal     *int     `json:"installmentTotal,omitempty"`
	ExternalID           *string  `json:"externalId,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
}

type UpdateTransactionUseCase struct {
	accountRepo     bankaccount.Repository
	categoryRepo    category.Repository
	transactionRepo transaction.Repository
}

func NewUpdateTransactionUseCase(
	accountRepo bankaccount.Repository,
	categoryRepo category.Repository,
	transactionRepo transaction.Repository,
) *UpdateTransactionUseCase {
	return &UpdateTransactionUseCase{
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
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

	// Parse status if provided, otherwise keep existing
	status := existing.Status
	if input.Status != nil {
		status, err = parseTransactionStatus(*input.Status)
		if err != nil {
			return nil, ErrInvalidInput
		}
	}

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
	existing.OccurredOn = occurredOn
	existing.DueOn = dueOn
	existing.RecurrenceRule = input.RecurrenceRule
	existing.InstallmentNumber = input.InstallmentNumber
	existing.InstallmentTotal = input.InstallmentTotal
	existing.ExternalID = input.ExternalID
	existing.Tags = sanitizeTags(input.Tags)
	existing.UpdatedAt = time.Now()

	// Persist changes
	if err := uc.transactionRepo.Update(existing); err != nil {
		return nil, err
	}

	return existing, nil
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
