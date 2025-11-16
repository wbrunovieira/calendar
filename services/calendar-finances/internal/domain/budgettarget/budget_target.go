package budgettarget

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BudgetTarget represents a planned expense value for a category period.
type BudgetTarget struct {
	ID          string    `json:"id"`
	ProfileID   string    `json:"profileId"`
	CategoryID  string    `json:"categoryId"`
	PeriodStart time.Time `json:"periodStart"`
	Amount      float64   `json:"amount"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateParams wraps the input required to instantiate a budget target.
type CreateParams struct {
	ProfileID   string
	CategoryID  string
	PeriodStart time.Time
	Amount      float64
	Notes       *string
}

// NewBudgetTarget validates and builds a new target.
func NewBudgetTarget(params CreateParams) (*BudgetTarget, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}

	if strings.TrimSpace(params.CategoryID) == "" {
		return nil, errors.New("categoryID is required")
	}

	if params.PeriodStart.IsZero() {
		return nil, errors.New("periodStart is required")
	}

	if params.Amount < 0 {
		return nil, errors.New("amount cannot be negative")
	}

	period := alignToMonth(params.PeriodStart)
	now := time.Now()

	return &BudgetTarget{
		ID:          uuid.New().String(),
		ProfileID:   params.ProfileID,
		CategoryID:  params.CategoryID,
		PeriodStart: period,
		Amount:      round2(params.Amount),
		Notes:       normalizeString(params.Notes),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update applies changes while keeping immutable attributes.
func (b *BudgetTarget) Update(params CreateParams) error {
	if b == nil {
		return errors.New("budget target is nil")
	}

	updated, err := NewBudgetTarget(params)
	if err != nil {
		return err
	}

	updated.ID = b.ID
	updated.CreatedAt = b.CreatedAt
	updated.UpdatedAt = time.Now()

	*b = *updated
	return nil
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

func alignToMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
