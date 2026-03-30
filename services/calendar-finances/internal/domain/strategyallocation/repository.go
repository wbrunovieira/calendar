package strategyallocation

type Repository interface {
	Create(allocation *StrategyAllocation) error
	Update(allocation *StrategyAllocation) error
	FindByID(id string) (*StrategyAllocation, error)
	FindByTransactionID(transactionID string) (*StrategyAllocation, error)
	FindByProfileID(profileID string) ([]*StrategyAllocation, error)
	FindByStrategy(profileID, strategy string) ([]*StrategyAllocation, error)
	FindPending(profileID string) ([]*StrategyAllocation, error)
}
