package statement

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("statement line not found")

// ListFilter narrows a statement listing. Reconciling a period means asking for one
// account, one date range, and usually only what is still unmatched.
type ListFilter struct {
	AccountID  string
	From       *time.Time
	To         *time.Time
	Status     *Status
	Provider   *Provider
	ExternalID *string
}

type Repository interface {
	// UpsertMany imports lines idempotently, keyed on (provider, external_id). The
	// same statement window may be pulled any number of times: re-importing must
	// refresh what the bank now says without duplicating rows and without losing the
	// reconciliation status already recorded.
	//
	// It reports how many were inserted and how many were already known, because
	// "the import ran" and "the import brought anything" are different facts.
	UpsertMany(lines []*Line) (inserted int, updated int, err error)
	FindByExternalID(provider Provider, externalID string) (*Line, error)
	List(filter ListFilter) ([]*Line, error)
	Update(line *Line) error
}
