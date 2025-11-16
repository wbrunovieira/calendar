package recurringtransaction

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status represents the current lifecycle for a recurring template.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusPaused   Status = "PAUSED"
	StatusCanceled Status = "CANCELLED"
)

// RecurringTransaction defines a scheduled financial entry template.
type RecurringTransaction struct {
	ID             string  `json:"id"`
	ProfileID      string  `json:"profileId"`
	BankAccountID  *string `json:"bankAccountId,omitempty"`
	CategoryID     *string `json:"categoryId,omitempty"`
	Type           string  `json:"type"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	Description    string  `json:"description"`
	RecurrenceRule string  `json:"recurrenceRule"`
	StartOn        time.Time
	EndOn          *time.Time
	NextOccurrence time.Time
	Status         Status
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateParams aggregates the payload to instantiate a recurring template.
type CreateParams struct {
	ProfileID      string
	BankAccountID  *string
	CategoryID     *string
	Type           string
	Amount         float64
	Currency       string
	Description    string
	RecurrenceRule string
	StartOn        time.Time
	EndOn          *time.Time
	NextOccurrence time.Time
	Status         Status
	Notes          *string
}

// New creates a new RecurringTransaction enforcing validation rules.
func New(params CreateParams) (*RecurringTransaction, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}

	if params.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if params.StartOn.IsZero() {
		return nil, errors.New("startOn is required")
	}

	if params.NextOccurrence.IsZero() {
		return nil, errors.New("nextOccurrence is required")
	}

	if err := validateStatus(params.Status); err != nil {
		return nil, err
	}

	recurrence := strings.TrimSpace(params.RecurrenceRule)
	if recurrence == "" {
		return nil, errors.New("recurrenceRule is required")
	}

	currency := strings.ToUpper(strings.TrimSpace(params.Currency))
	if currency == "" {
		currency = "BRL"
	}

	now := time.Now()

	return &RecurringTransaction{
		ID:             uuid.New().String(),
		ProfileID:      params.ProfileID,
		BankAccountID:  normalizeString(params.BankAccountID),
		CategoryID:     normalizeString(params.CategoryID),
		Type:           strings.ToUpper(strings.TrimSpace(params.Type)),
		Amount:         round2(params.Amount),
		Currency:       currency,
		Description:    strings.TrimSpace(params.Description),
		RecurrenceRule: recurrence,
		StartOn:        params.StartOn,
		EndOn:          normalizeTime(params.EndOn),
		NextOccurrence: params.NextOccurrence,
		Status:         params.Status,
		Notes:          normalizeString(params.Notes),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// Update applies mutable fields to an existing template.
func (r *RecurringTransaction) Update(params CreateParams) error {
	if r == nil {
		return errors.New("recurring transaction is nil")
	}

	updated, err := New(params)
	if err != nil {
		return err
	}

	updated.ID = r.ID
	updated.CreatedAt = r.CreatedAt
	updated.UpdatedAt = time.Now()

	*r = *updated
	return nil
}

// SetStatus changes the current status if valid.
func (r *RecurringTransaction) SetStatus(status Status) error {
	if err := validateStatus(status); err != nil {
		return err
	}
	r.Status = status
	r.UpdatedAt = time.Now()
	return nil
}

func validateStatus(status Status) error {
	switch status {
	case StatusActive, StatusPaused, StatusCanceled:
		return nil
	default:
		return errors.New("invalid status")
	}
}

func normalizeString(value *string) *string {
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

func normalizeTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	copy := *value
	return &copy
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
