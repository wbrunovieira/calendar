package strategyallocation

import (
	"errors"
	"time"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusApproved Status = "APPROVED"
	StatusDeclined Status = "DECLINED"
)

type StrategyAllocation struct {
	ID            string    `json:"id"`
	ProfileID     string    `json:"profileId"`
	Strategy      string    `json:"strategy"`
	TransactionID string    `json:"transactionId"`
	Amount        float64   `json:"amount"`
	Status        Status    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type StrategySummary struct {
	Strategy         string  `json:"strategy"`
	TotalAllocatedBRL float64 `json:"totalAllocatedBrl"`
	AllocationCount  int     `json:"allocationCount"`
}

var (
	ErrInvalidStrategy = errors.New("strategy is required")
	ErrInvalidAmount   = errors.New("amount must be positive")
	ErrInvalidProfile  = errors.New("profileId is required")
	ErrNotPending      = errors.New("allocation is not pending")
)

func NewStrategyAllocation(profileID, strategy, transactionID string, amount float64) (*StrategyAllocation, error) {
	if profileID == "" {
		return nil, ErrInvalidProfile
	}
	if strategy == "" {
		return nil, ErrInvalidStrategy
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	now := time.Now()
	return &StrategyAllocation{
		ProfileID:     profileID,
		Strategy:      strategy,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        StatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (sa *StrategyAllocation) Approve() error {
	if sa.Status != StatusPending {
		return ErrNotPending
	}
	sa.Status = StatusApproved
	sa.UpdatedAt = time.Now()
	return nil
}

func (sa *StrategyAllocation) Decline() error {
	if sa.Status != StatusPending {
		return ErrNotPending
	}
	sa.Status = StatusDeclined
	sa.UpdatedAt = time.Now()
	return nil
}

// TotalAllocated returns the sum of approved allocations
func TotalAllocated(allocations []*StrategyAllocation) float64 {
	total := 0.0
	for _, a := range allocations {
		if a.Status == StatusApproved {
			total += a.Amount
		}
	}
	return total
}

// NewStrategySummary builds a summary from allocations
func NewStrategySummary(strategy string, allocations []*StrategyAllocation) StrategySummary {
	count := 0
	total := 0.0
	for _, a := range allocations {
		if a.Status == StatusApproved {
			total += a.Amount
			count++
		}
	}
	return StrategySummary{
		Strategy:          strategy,
		TotalAllocatedBRL: total,
		AllocationCount:   count,
	}
}
