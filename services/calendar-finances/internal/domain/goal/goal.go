package goal

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

type Priority string

const (
	PriorityHigh   Priority = "HIGH"
	PriorityMedium Priority = "MEDIUM"
	PriorityLow    Priority = "LOW"
)

type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
)

type GoalType string

const (
	GoalTypePersonalSavings    GoalType = "PERSONAL_SAVINGS"
	GoalTypeOperationalReserve GoalType = "OPERATIONAL_RESERVE"
	GoalTypeTaxFund            GoalType = "TAX_FUND"
	GoalTypeInvestmentFund     GoalType = "INVESTMENT_FUND"
	GoalTypeRevenueTarget      GoalType = "REVENUE_TARGET"
)

type Goal struct {
	ID            string     `json:"id"`
	ProfileID     string     `json:"profileId"`
	CategoryID    *string    `json:"categoryId,omitempty"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	TargetAmount  float64    `json:"targetAmount"`
	CurrentAmount float64    `json:"currentAmount"`
	Priority      Priority   `json:"priority"`
	TargetDate    *time.Time `json:"targetDate,omitempty"`
	Status        Status     `json:"status"`
	Link          string     `json:"link,omitempty"`
	GoalType      GoalType   `json:"goalType"`
	DisplayOrder  int        `json:"displayOrder"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type CreateParams struct {
	ProfileID    string
	CategoryID   *string
	Name         string
	Description  string
	TargetAmount float64
	Priority     Priority
	TargetDate   *time.Time
	Link         string
	GoalType     GoalType
}

func New(params CreateParams) (*Goal, error) {
	if params.ProfileID == "" {
		return nil, errors.New("profileID is required")
	}
	if params.Name == "" {
		return nil, errors.New("name is required")
	}
	if params.TargetAmount <= 0 {
		return nil, errors.New("targetAmount must be greater than zero")
	}

	priority := params.Priority
	if priority == "" {
		priority = PriorityMedium
	}

	goalType := params.GoalType
	if goalType == "" {
		goalType = GoalTypePersonalSavings
	}

	now := time.Now()
	return &Goal{
		ID:            uuid.New().String(),
		ProfileID:     params.ProfileID,
		CategoryID:    params.CategoryID,
		Name:          params.Name,
		Description:   params.Description,
		TargetAmount:  round2(params.TargetAmount),
		CurrentAmount: 0,
		Priority:      priority,
		TargetDate:    params.TargetDate,
		Status:        StatusActive,
		Link:          params.Link,
		GoalType:      goalType,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (g *Goal) AddAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	g.CurrentAmount = round2(g.CurrentAmount + amount)
	g.touch()
	return nil
}

func (g *Goal) Complete() error {
	if g.Status != StatusActive {
		return errors.New("can only complete active goals")
	}
	g.Status = StatusCompleted
	g.touch()
	return nil
}

func (g *Goal) Cancel() error {
	if g.Status != StatusActive {
		return errors.New("can only cancel active goals")
	}
	g.Status = StatusCancelled
	g.touch()
	return nil
}

func (g *Goal) Progress() float64 {
	if g.TargetAmount == 0 {
		return 0
	}
	return round2(g.CurrentAmount / g.TargetAmount * 100)
}

func (g *Goal) touch() {
	g.UpdatedAt = time.Now()
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
