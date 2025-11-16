package transaction

import "time"

// ListFilter encapsulates query options for fetching transactions.
type ListFilter struct {
	ProfileID     string
	BankAccountID *string
	Status        *Status
	Type          *Type
	OccurredFrom  *time.Time
	OccurredTo    *time.Time
}

// Repository represents the persistence contract for transactions.
type Repository interface {
	Create(tx *Transaction) error
	GetByID(id string) (*Transaction, error)
	List(filter ListFilter) ([]*Transaction, error)
	UpdateStatus(id string, status Status, occurredOn time.Time, notes *string) error
	Delete(id string) error
	SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error)
}
