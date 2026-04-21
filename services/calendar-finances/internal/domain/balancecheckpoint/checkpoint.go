package balancecheckpoint

import "time"

// BalanceCheckpoint stores the verified closing balance for an account at the
// end of a calendar month. Future recalculations only need to sum transactions
// that occurred after this checkpoint, keeping the query bounded regardless of
// total transaction history.
type BalanceCheckpoint struct {
	ID             string
	AccountID      string
	ReferenceMonth time.Time // always stored as the 1st day of the month
	ClosingBalance float64
	CreatedAt      time.Time
}

// Repository is the persistence contract for balance checkpoints.
type Repository interface {
	// Save creates or replaces the checkpoint for (account_id, reference_month).
	Save(cp *BalanceCheckpoint) error
	// FindLatestBefore returns the most recent checkpoint whose reference_month
	// is strictly before `before`. Returns nil, nil when none exists.
	FindLatestBefore(accountID string, before time.Time) (*BalanceCheckpoint, error)
	// FindByAccountID returns all checkpoints for an account, newest first.
	FindByAccountID(accountID string) ([]*BalanceCheckpoint, error)
}
