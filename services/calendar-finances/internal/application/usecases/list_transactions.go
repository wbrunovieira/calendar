package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

const DefaultPageSize = 50

type ListTransactionsInput struct {
	ProfileID            string
	BankAccountID        *string
	InvoiceID            *string
	Status               *string
	Type                 *string
	OccurredFrom         *string
	OccurredTo           *string
	IncludeAsDestination bool
	Page                 int // 1-based; 0 or negative defaults to 1
	PageSize             int // defaults to DefaultPageSize
}

type ListTransactionsResult struct {
	Items    []*transaction.Transaction
	Total    int
	Page     int
	PageSize int
}

type ListTransactionsUseCase struct {
	repo transaction.Repository
}

func NewListTransactionsUseCase(repo transaction.Repository) *ListTransactionsUseCase {
	return &ListTransactionsUseCase{repo: repo}
}

func (uc *ListTransactionsUseCase) Execute(input ListTransactionsInput) (*ListTransactionsResult, error) {
	if strings.TrimSpace(input.ProfileID) == "" {
		return nil, ErrInvalidInput
	}

	filter := transaction.ListFilter{ProfileID: input.ProfileID}

	if input.BankAccountID != nil {
		trimmed := strings.TrimSpace(*input.BankAccountID)
		if trimmed != "" {
			filter.BankAccountID = &trimmed
			filter.IncludeAsDestination = input.IncludeAsDestination
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
			return nil, ErrInvalidInput
		}
		filter.OccurredFrom = &t
	}

	if input.OccurredTo != nil {
		t, err := parseDate(*input.OccurredTo)
		if err != nil {
			return nil, ErrInvalidInput
		}
		filter.OccurredTo = &t
	}

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	total, err := uc.repo.Count(filter)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	filter.Limit = &pageSize
	filter.Offset = &offset

	items, err := uc.repo.List(filter)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []*transaction.Transaction{}
	}

	return &ListTransactionsResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
