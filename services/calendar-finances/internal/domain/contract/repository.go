package contract

import "errors"

// ErrDuplicate means a contract already mirrors that (source, externalID). It is a
// distinct value because losing the insert race is recoverable — the caller re-reads
// and applies on top — while any other write failure is not.
var ErrDuplicate = errors.New("contract already mirrors that external reference")

// Repository defines persistence operations for Contract.
type Repository interface {
	Create(c *Contract) error
	// FindByExternalRef resolves the mirror of a remote deal. It answers ErrNotFound
	// for absence and nothing else, so a caller can tell an unknown deal from a
	// database that is down.
	FindByExternalRef(source, externalID string) (*Contract, error)
	Update(c *Contract) error
}
