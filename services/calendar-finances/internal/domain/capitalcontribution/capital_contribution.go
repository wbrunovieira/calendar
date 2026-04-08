package capitalcontribution

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ContributionType represents the direction of capital movement
type ContributionType string

const (
	TypeContribution ContributionType = "CONTRIBUTION" // Sócio coloca dinheiro na empresa
	TypeWithdrawal   ContributionType = "WITHDRAWAL"   // Empresa devolve dinheiro ao sócio
	TypeLoan         ContributionType = "LOAN"         // Empréstimo do sócio (deve ser devolvido)
)

// CapitalContribution records a capital movement between the owner and the company
type CapitalContribution struct {
	ID              string           `json:"id"`
	ProfileID       string           `json:"profileId"`
	Type            ContributionType `json:"type"`
	Amount          float64          `json:"amount"`
	Date            time.Time        `json:"date"`
	Description     string           `json:"description"`
	SourceAccountID *string          `json:"sourceAccountId,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	IsReturned      bool             `json:"isReturned"`
	ReturnedAmount  float64          `json:"returnedAmount"`
	ReturnedAt      *time.Time       `json:"returnedAt,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

// CreateParams encapsulates parameters for creating a contribution
type CreateParams struct {
	ProfileID       string
	Type            ContributionType
	Amount          float64
	Date            time.Time
	Description     string
	SourceAccountID *string
	Notes           *string
}

// NewCapitalContribution validates and creates a new record
func NewCapitalContribution(params CreateParams) (*CapitalContribution, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}
	if !isValidType(params.Type) {
		return nil, errors.New("invalid contribution type")
	}
	if params.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if strings.TrimSpace(params.Description) == "" {
		return nil, errors.New("description is required")
	}
	if params.Date.IsZero() {
		return nil, errors.New("date is required")
	}

	now := time.Now()
	return &CapitalContribution{
		ID:              uuid.New().String(),
		ProfileID:       params.ProfileID,
		Type:            params.Type,
		Amount:          params.Amount,
		Date:            params.Date,
		Description:     strings.TrimSpace(params.Description),
		SourceAccountID: params.SourceAccountID,
		Notes:           params.Notes,
		IsReturned:      false,
		ReturnedAmount:  0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// MarkAsReturned marks the contribution as fully returned
func (c *CapitalContribution) MarkAsReturned(returnedAmount float64, returnedAt time.Time) error {
	if returnedAmount <= 0 {
		return errors.New("returnedAmount must be greater than zero")
	}
	c.IsReturned = true
	c.ReturnedAmount = returnedAmount
	c.ReturnedAt = &returnedAt
	c.UpdatedAt = time.Now()
	return nil
}

// OutstandingBalance returns the amount still owed to the owner (for CONTRIBUTION and LOAN types)
func (c *CapitalContribution) OutstandingBalance() float64 {
	if c.Type == TypeWithdrawal {
		return 0
	}
	return c.Amount - c.ReturnedAmount
}

func isValidType(t ContributionType) bool {
	switch t {
	case TypeContribution, TypeWithdrawal, TypeLoan:
		return true
	default:
		return false
	}
}
