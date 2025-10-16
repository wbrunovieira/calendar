package transaction

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Split allows distributing a transaction amount across multiple categories.
type Split struct {
	ID         string    `json:"id"`
	CategoryID *string   `json:"categoryId,omitempty"`
	Amount     float64   `json:"amount"`
	Memo       *string   `json:"memo,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// NewSplit validates data and builds a split entry.
func NewSplit(categoryID *string, amount float64, memo *string) (*Split, error) {
	if amount <= 0 {
		return nil, errors.New("split amount must be greater than zero")
	}

	split := &Split{
		ID:         uuid.New().String(),
		CategoryID: cloneString(categoryID),
		Amount:     round2(amount),
		Memo:       normalizeMemo(memo),
		CreatedAt:  time.Now(),
	}

	return split, nil
}

func normalizeMemo(memo *string) *string {
	if memo == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*memo)
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}
