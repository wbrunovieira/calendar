package usecases

import (
	"errors"
	"fmt"
	"log"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	sa "github.com/brunovieira/calendar-finances/internal/domain/strategyallocation"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

var (
	ErrAllocationNotFound = errors.New("allocation not found")
)

// DetectPendingAllocationsUseCase finds TRANSFER transactions to the exchange account
// that don't have an allocation record yet, and creates pending allocations.
type DetectPendingAllocationsUseCase struct {
	allocationRepo  sa.Repository
	transactionRepo transaction.Repository
	accountRepo     bankaccount.Repository
	defaultStrategy string
}

func NewDetectPendingAllocationsUseCase(
	allocationRepo sa.Repository,
	transactionRepo transaction.Repository,
	accountRepo bankaccount.Repository,
	defaultStrategy string,
) *DetectPendingAllocationsUseCase {
	if defaultStrategy == "" {
		defaultStrategy = "MACross1"
	}
	return &DetectPendingAllocationsUseCase{
		allocationRepo:  allocationRepo,
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		defaultStrategy: defaultStrategy,
	}
}

type DetectResult struct {
	NewPending int `json:"newPending"`
	Existing   int `json:"existing"`
}

func (uc *DetectPendingAllocationsUseCase) Execute(profileID string) (*DetectResult, error) {
	result := &DetectResult{}

	// Find exchange account
	accounts, err := uc.accountRepo.FindByProfileID(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find accounts: %w", err)
	}

	var exchangeAccountID string
	for _, acc := range accounts {
		if acc.Type == bankaccount.AccountTypeExchange {
			exchangeAccountID = acc.ID
			break
		}
	}
	if exchangeAccountID == "" {
		return result, nil // no exchange account
	}

	// Find all TRANSFER transactions where destination is the exchange account
	confirmed := transaction.StatusConfirmed
	txType := transaction.TypeTransfer
	bankAccID := exchangeAccountID
	transactions, err := uc.transactionRepo.List(transaction.ListFilter{
		ProfileID:            profileID,
		BankAccountID:        &bankAccID,
		Status:               &confirmed,
		Type:                 &txType,
		IncludeAsDestination: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find transactions: %w", err)
	}

	for _, tx := range transactions {
		// Only transfers TO the exchange account
		if tx.DestinationAccountID == nil || *tx.DestinationAccountID != exchangeAccountID {
			continue
		}

		// Check if allocation already exists for this transaction
		existing, _ := uc.allocationRepo.FindByTransactionID(tx.ID)
		if existing != nil {
			result.Existing++
			continue
		}

		// Create pending allocation
		allocation, err := sa.NewStrategyAllocation(profileID, uc.defaultStrategy, tx.ID, tx.Amount)
		if err != nil {
			log.Printf("Failed to create allocation for tx %s: %v", tx.ID, err)
			continue
		}

		if err := uc.allocationRepo.Create(allocation); err != nil {
			log.Printf("Failed to persist allocation for tx %s: %v", tx.ID, err)
			continue
		}

		result.NewPending++
	}

	return result, nil
}

// ApproveAllocationUseCase approves a pending allocation
type ApproveAllocationUseCase struct {
	repo sa.Repository
}

func NewApproveAllocationUseCase(repo sa.Repository) *ApproveAllocationUseCase {
	return &ApproveAllocationUseCase{repo: repo}
}

func (uc *ApproveAllocationUseCase) Execute(allocationID string) (*sa.StrategyAllocation, error) {
	allocation, err := uc.repo.FindByID(allocationID)
	if err != nil {
		return nil, ErrAllocationNotFound
	}

	if err := allocation.Approve(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(allocation); err != nil {
		return nil, fmt.Errorf("failed to update allocation: %w", err)
	}

	return allocation, nil
}

// DeclineAllocationUseCase declines a pending allocation
type DeclineAllocationUseCase struct {
	repo sa.Repository
}

func NewDeclineAllocationUseCase(repo sa.Repository) *DeclineAllocationUseCase {
	return &DeclineAllocationUseCase{repo: repo}
}

func (uc *DeclineAllocationUseCase) Execute(allocationID string) (*sa.StrategyAllocation, error) {
	allocation, err := uc.repo.FindByID(allocationID)
	if err != nil {
		return nil, ErrAllocationNotFound
	}

	if err := allocation.Decline(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(allocation); err != nil {
		return nil, fmt.Errorf("failed to update allocation: %w", err)
	}

	return allocation, nil
}

// GetStrategySummaryUseCase returns budget summary for a strategy
type GetStrategySummaryUseCase struct {
	allocationRepo sa.Repository
}

func NewGetStrategySummaryUseCase(allocationRepo sa.Repository) *GetStrategySummaryUseCase {
	return &GetStrategySummaryUseCase{allocationRepo: allocationRepo}
}

func (uc *GetStrategySummaryUseCase) Execute(profileID, strategy string) (*sa.StrategySummary, error) {
	allocations, err := uc.allocationRepo.FindByStrategy(profileID, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to find allocations: %w", err)
	}

	summary := sa.NewStrategySummary(strategy, allocations)
	return &summary, nil
}
