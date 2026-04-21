package persistence

import (
	"database/sql"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/balancecheckpoint"
)

type CheckpointRepository struct {
	db *sql.DB
}

func NewCheckpointRepository(db *sql.DB) *CheckpointRepository {
	return &CheckpointRepository{db: db}
}

func (r *CheckpointRepository) Save(cp *balancecheckpoint.BalanceCheckpoint) error {
	_, err := r.db.Exec(`
		INSERT INTO finance.balance_checkpoints
			(id, account_id, reference_month, closing_balance, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (account_id, reference_month)
		DO UPDATE SET closing_balance = EXCLUDED.closing_balance
	`, cp.ID, cp.AccountID, cp.ReferenceMonth, cp.ClosingBalance, cp.CreatedAt)
	return err
}

func (r *CheckpointRepository) FindLatestBefore(accountID string, before time.Time) (*balancecheckpoint.BalanceCheckpoint, error) {
	row := r.db.QueryRow(`
		SELECT id, account_id, reference_month, closing_balance, created_at
		FROM finance.balance_checkpoints
		WHERE account_id = $1 AND reference_month < $2
		ORDER BY reference_month DESC
		LIMIT 1
	`, accountID, before)

	cp := &balancecheckpoint.BalanceCheckpoint{}
	err := row.Scan(&cp.ID, &cp.AccountID, &cp.ReferenceMonth, &cp.ClosingBalance, &cp.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cp, nil
}

func (r *CheckpointRepository) FindByAccountID(accountID string) ([]*balancecheckpoint.BalanceCheckpoint, error) {
	rows, err := r.db.Query(`
		SELECT id, account_id, reference_month, closing_balance, created_at
		FROM finance.balance_checkpoints
		WHERE account_id = $1
		ORDER BY reference_month DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*balancecheckpoint.BalanceCheckpoint
	for rows.Next() {
		cp := &balancecheckpoint.BalanceCheckpoint{}
		if err := rows.Scan(&cp.ID, &cp.AccountID, &cp.ReferenceMonth, &cp.ClosingBalance, &cp.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, cp)
	}
	return result, rows.Err()
}
