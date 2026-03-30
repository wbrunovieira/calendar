package persistence

import (
	"database/sql"

	sa "github.com/brunovieira/calendar-finances/internal/domain/strategyallocation"
)

type StrategyAllocationRepository struct {
	db *sql.DB
}

func NewStrategyAllocationRepository(db *sql.DB) *StrategyAllocationRepository {
	return &StrategyAllocationRepository{db: db}
}

func (r *StrategyAllocationRepository) Create(a *sa.StrategyAllocation) error {
	query := `INSERT INTO finance.strategy_allocations
		(profile_id, strategy, transaction_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(query,
		a.ProfileID, a.Strategy, a.TransactionID, a.Amount, a.Status,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *StrategyAllocationRepository) Update(a *sa.StrategyAllocation) error {
	query := `UPDATE finance.strategy_allocations SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, a.Status, a.ID)
	return err
}

func (r *StrategyAllocationRepository) FindByID(id string) (*sa.StrategyAllocation, error) {
	results, err := r.query(`SELECT id, profile_id, strategy, transaction_id, amount, status, created_at, updated_at
		FROM finance.strategy_allocations WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

func (r *StrategyAllocationRepository) FindByTransactionID(transactionID string) (*sa.StrategyAllocation, error) {
	results, err := r.query(`SELECT id, profile_id, strategy, transaction_id, amount, status, created_at, updated_at
		FROM finance.strategy_allocations WHERE transaction_id = $1`, transactionID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func (r *StrategyAllocationRepository) FindByProfileID(profileID string) ([]*sa.StrategyAllocation, error) {
	return r.query(`SELECT id, profile_id, strategy, transaction_id, amount, status, created_at, updated_at
		FROM finance.strategy_allocations WHERE profile_id = $1 ORDER BY created_at DESC`, profileID)
}

func (r *StrategyAllocationRepository) FindByStrategy(profileID, strategy string) ([]*sa.StrategyAllocation, error) {
	return r.query(`SELECT id, profile_id, strategy, transaction_id, amount, status, created_at, updated_at
		FROM finance.strategy_allocations WHERE profile_id = $1 AND strategy = $2 ORDER BY created_at DESC`, profileID, strategy)
}

func (r *StrategyAllocationRepository) FindPending(profileID string) ([]*sa.StrategyAllocation, error) {
	return r.query(`SELECT id, profile_id, strategy, transaction_id, amount, status, created_at, updated_at
		FROM finance.strategy_allocations WHERE profile_id = $1 AND status = 'PENDING' ORDER BY created_at DESC`, profileID)
}

func (r *StrategyAllocationRepository) query(q string, args ...interface{}) ([]*sa.StrategyAllocation, error) {
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*sa.StrategyAllocation
	for rows.Next() {
		a := &sa.StrategyAllocation{}
		if err := rows.Scan(
			&a.ID, &a.ProfileID, &a.Strategy, &a.TransactionID,
			&a.Amount, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}
