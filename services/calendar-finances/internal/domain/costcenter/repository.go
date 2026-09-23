package costcenter

import "errors"

// ErrNotFound means the cost center is absent, and ONLY that. It exists so a
// caller can tell absence from a failed read: without the distinction, a database
// outage looks exactly like "no such client", and a sync would answer it by
// creating a duplicate.
var ErrNotFound = errors.New("cost center not found")

// ErrDuplicate means a cost center already mirrors that external reference. Like
// ErrNotFound it is a distinct value so the caller can recover — losing the insert
// race to a concurrent delivery is not a failure, it just means someone else won.
var ErrDuplicate = errors.New("cost center already mirrors that external reference")

// Repository defines persistence operations for CostCenter
type Repository interface {
	Create(c *CostCenter) error
	FindByID(id string) (*CostCenter, error)
	FindByProfile(profileID string) ([]*CostCenter, error)
	// FindByExternalRef resolves a cost center from the id it mirrors elsewhere,
	// which is what lets a repeated sync find the same client instead of a twin.
	FindByExternalRef(profileID, source, externalID string) (*CostCenter, error)
	Update(c *CostCenter) error
	Delete(id string) error
}
