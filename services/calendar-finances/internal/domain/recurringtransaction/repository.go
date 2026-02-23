package recurringtransaction

import "time"

// Repository abstracts persistence for recurring templates.
type Repository interface {
	Create(entity *RecurringTransaction) error
	Update(entity *RecurringTransaction) error
	FindByID(id string) (*RecurringTransaction, error)
	ListByProfile(profileID string) ([]*RecurringTransaction, error)
	FindDueRecurring(before time.Time) ([]*RecurringTransaction, error)
	Delete(id string) error
}
