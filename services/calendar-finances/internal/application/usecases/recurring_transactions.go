package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
)

type CreateRecurringTransactionInput struct {
	ProfileID      string  `json:"profileId"`
	BankAccountID  *string `json:"bankAccountId,omitempty"`
	CategoryID     *string `json:"categoryId,omitempty"`
	Type           string  `json:"type"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	Description    string  `json:"description"`
	RecurrenceRule string  `json:"recurrenceRule"`
	StartOn        string  `json:"startOn"`
	EndOn          *string `json:"endOn,omitempty"`
	NextOccurrence string  `json:"nextOccurrence"`
	Status         string  `json:"status"`
	Notes          *string `json:"notes,omitempty"`
}

type RecurringTransactionPresenter struct {
	ID             string     `json:"id"`
	ProfileID      string     `json:"profileId"`
	BankAccountID  *string    `json:"bankAccountId,omitempty"`
	CategoryID     *string    `json:"categoryId,omitempty"`
	Type           string     `json:"type"`
	Amount         float64    `json:"amount"`
	Currency       string     `json:"currency"`
	Description    string     `json:"description"`
	RecurrenceRule string     `json:"recurrenceRule"`
	StartOn        time.Time  `json:"startOn"`
	EndOn          *time.Time `json:"endOn,omitempty"`
	NextOccurrence time.Time  `json:"nextOccurrence"`
	Status         string     `json:"status"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type RecurringTransactionsService struct {
	repo recurringtransaction.Repository
}

func NewRecurringTransactionsService(repo recurringtransaction.Repository) *RecurringTransactionsService {
	return &RecurringTransactionsService{repo: repo}
}

func (s *RecurringTransactionsService) Create(input CreateRecurringTransactionInput) (*RecurringTransactionPresenter, error) {
	entity, err := s.buildEntity(input)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(entity); err != nil {
		return nil, err
	}

	return presentRecurring(entity), nil
}

func (s *RecurringTransactionsService) Update(id string, input CreateRecurringTransactionInput) (*RecurringTransactionPresenter, error) {
	entity, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrRecurringNotFound
	}

	params, err := s.parseParams(input)
	if err != nil {
		return nil, err
	}

	if err := entity.Update(params); err != nil {
		return nil, err
	}

	if err := s.repo.Update(entity); err != nil {
		return nil, err
	}

	return presentRecurring(entity), nil
}

func (s *RecurringTransactionsService) UpdateStatus(id string, status string) (*RecurringTransactionPresenter, error) {
	entity, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrRecurringNotFound
	}

	if err := entity.SetStatus(recurringtransaction.Status(strings.ToUpper(strings.TrimSpace(status)))); err != nil {
		return nil, err
	}

	if err := s.repo.Update(entity); err != nil {
		return nil, err
	}

	return presentRecurring(entity), nil
}

func (s *RecurringTransactionsService) List(profileID string) ([]*RecurringTransactionPresenter, error) {
	items, err := s.repo.ListByProfile(profileID)
	if err != nil {
		return nil, err
	}

	var result []*RecurringTransactionPresenter
	for _, item := range items {
		result = append(result, presentRecurring(item))
	}
	return result, nil
}

func (s *RecurringTransactionsService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return ErrRecurringNotFound
	}
	return nil
}

func (s *RecurringTransactionsService) buildEntity(input CreateRecurringTransactionInput) (*recurringtransaction.RecurringTransaction, error) {
	params, err := s.parseParams(input)
	if err != nil {
		return nil, err
	}
	return recurringtransaction.New(params)
}

func (s *RecurringTransactionsService) parseParams(input CreateRecurringTransactionInput) (recurringtransaction.CreateParams, error) {
	startOn, err := time.Parse(time.DateOnly, input.StartOn)
	if err != nil {
		return recurringtransaction.CreateParams{}, ErrInvalidInput
	}

	nextOccurrence, err := time.Parse(time.DateOnly, input.NextOccurrence)
	if err != nil {
		return recurringtransaction.CreateParams{}, ErrInvalidInput
	}

	var endOn *time.Time
	if input.EndOn != nil && strings.TrimSpace(*input.EndOn) != "" {
		parsed, err := time.Parse(time.DateOnly, strings.TrimSpace(*input.EndOn))
		if err != nil {
			return recurringtransaction.CreateParams{}, ErrInvalidInput
		}
		endOn = &parsed
	}

	status := recurringtransaction.Status(strings.ToUpper(strings.TrimSpace(input.Status)))
	if status == "" {
		status = recurringtransaction.StatusActive
	}

	return recurringtransaction.CreateParams{
		ProfileID:      strings.TrimSpace(input.ProfileID),
		BankAccountID:  normalizePtr(input.BankAccountID),
		CategoryID:     normalizePtr(input.CategoryID),
		Type:           strings.ToUpper(strings.TrimSpace(input.Type)),
		Amount:         input.Amount,
		Currency:       strings.TrimSpace(input.Currency),
		Description:    input.Description,
		RecurrenceRule: strings.TrimSpace(input.RecurrenceRule),
		StartOn:        startOn,
		EndOn:          endOn,
		NextOccurrence: nextOccurrence,
		Status:         status,
		Notes:          normalizePtr(input.Notes),
	}, nil
}

func presentRecurring(entity *recurringtransaction.RecurringTransaction) *RecurringTransactionPresenter {
	return &RecurringTransactionPresenter{
		ID:             entity.ID,
		ProfileID:      entity.ProfileID,
		BankAccountID:  entity.BankAccountID,
		CategoryID:     entity.CategoryID,
		Type:           entity.Type,
		Amount:         entity.Amount,
		Currency:       entity.Currency,
		Description:    entity.Description,
		RecurrenceRule: entity.RecurrenceRule,
		StartOn:        entity.StartOn,
		EndOn:          entity.EndOn,
		NextOccurrence: entity.NextOccurrence,
		Status:         string(entity.Status),
		Notes:          entity.Notes,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}

func normalizePtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}
