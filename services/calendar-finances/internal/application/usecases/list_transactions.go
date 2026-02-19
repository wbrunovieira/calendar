package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type ListTransactionsInput struct {
	ProfileID     string
	BankAccountID *string
	InvoiceID     *string
	Status        *string
	Type          *string
	OccurredFrom  *string
	OccurredTo    *string
}

type ListTransactionsUseCase struct {
	repo transaction.Repository
}

func NewListTransactionsUseCase(repo transaction.Repository) *ListTransactionsUseCase {
	return &ListTransactionsUseCase{repo: repo}
}

func (uc *ListTransactionsUseCase) Execute(input ListTransactionsInput) ([]*transaction.Transaction, error) {
	if strings.TrimSpace(input.ProfileID) == "" {
		return nil, ErrInvalidInput
	}

	filter := transaction.ListFilter{ProfileID: input.ProfileID}

	if input.BankAccountID != nil {
		trimmed := strings.TrimSpace(*input.BankAccountID)
		if trimmed != "" {
			filter.BankAccountID = &trimmed
		}
	}

	if input.InvoiceID != nil {
		trimmed := strings.TrimSpace(*input.InvoiceID)
		if trimmed != "" {
			filter.InvoiceID = &trimmed
		}
	}

	if input.Status != nil {
		status, err := parseTransactionStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		filter.Status = &status
	}

	if input.Type != nil {
		typeValue, err := parseTransactionType(*input.Type)
		if err != nil {
			return nil, err
		}
		filter.Type = &typeValue
	}

	if input.OccurredFrom != nil {
		t, err := parseDate(*input.OccurredFrom)
		if err != nil {
			return nil, err
		}
		filter.OccurredFrom = &t
	}

	if input.OccurredTo != nil {
		t, err := parseDate(*input.OccurredTo)
		if err != nil {
			return nil, err
		}
		filter.OccurredTo = &t
	}

	return uc.repo.List(filter)
}
