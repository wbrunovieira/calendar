package budgettarget

import "time"

// Repository abstracts persistence of budget targets.
type Repository interface {
	Create(entity *BudgetTarget) error
	Update(entity *BudgetTarget) error
	FindByID(id string) (*BudgetTarget, error)
	ListByProfile(profileID string) ([]*BudgetTarget, error)
	ListByProfileAndPeriod(profileID string, periodStart time.Time) ([]*BudgetTarget, error)
	Delete(id string) error
}
