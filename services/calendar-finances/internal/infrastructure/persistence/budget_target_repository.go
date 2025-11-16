package persistence

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/budgettarget"
)

type BudgetTargetRepository struct {
	db *sql.DB
}

func NewBudgetTargetRepository(db *sql.DB) *BudgetTargetRepository {
	return &BudgetTargetRepository{db: db}
}

func (r *BudgetTargetRepository) Create(entity *budgettarget.BudgetTarget) error {
	query := `
		INSERT INTO finance.budget_targets (
			id, profile_id, category_id, period_start, amount, notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	_, err := r.db.Exec(query,
		entity.ID,
		entity.ProfileID,
		entity.CategoryID,
		entity.PeriodStart,
		entity.Amount,
		nullableString(entity.Notes),
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	return err
}

func (r *BudgetTargetRepository) Update(entity *budgettarget.BudgetTarget) error {
    query := `
        UPDATE finance.budget_targets
        SET profile_id = $2,
            category_id = $3,
            amount = $4,
            notes = $5,
            period_start = $6,
            updated_at = $7
        WHERE id = $1
    `

    result, err := r.db.Exec(query,
        entity.ID,
        entity.ProfileID,
        entity.CategoryID,
        entity.Amount,
        nullableString(entity.Notes),
        entity.PeriodStart,
        entity.UpdatedAt,
    )
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("budget target not found")
	}
	return nil
}

func (r *BudgetTargetRepository) FindByID(id string) (*budgettarget.BudgetTarget, error) {
	query := `
		SELECT id, profile_id, category_id, period_start, amount, notes, created_at, updated_at
		FROM finance.budget_targets
		WHERE id = $1
	`

	row := r.db.QueryRow(query, id)
	return scanBudgetTarget(row)
}

func (r *BudgetTargetRepository) ListByProfile(profileID string) ([]*budgettarget.BudgetTarget, error) {
	query := `
		SELECT id, profile_id, category_id, period_start, amount, notes, created_at, updated_at
		FROM finance.budget_targets
		WHERE profile_id = $1
		ORDER BY period_start DESC, created_at DESC
	`

	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*budgettarget.BudgetTarget
	for rows.Next() {
		item, err := scanBudgetTarget(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *BudgetTargetRepository) ListByProfileAndPeriod(profileID string, periodStart time.Time) ([]*budgettarget.BudgetTarget, error) {
	query := `
		SELECT id, profile_id, category_id, period_start, amount, notes, created_at, updated_at
		FROM finance.budget_targets
		WHERE profile_id = $1 AND period_start = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, profileID, periodStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*budgettarget.BudgetTarget
	for rows.Next() {
		item, err := scanBudgetTarget(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *BudgetTargetRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.budget_targets WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("budget target not found")
	}
	return nil
}

type budgetScanner interface {
	Scan(dest ...any) error
}

func scanBudgetTarget(scanner budgetScanner) (*budgettarget.BudgetTarget, error) {
	entity := &budgettarget.BudgetTarget{}
	var notes sql.NullString

	err := scanner.Scan(
		&entity.ID,
		&entity.ProfileID,
		&entity.CategoryID,
		&entity.PeriodStart,
		&entity.Amount,
		&notes,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("budget target not found")
		}
		return nil, err
	}

	if notes.Valid {
		value := notes.String
		entity.Notes = &value
	}

	return entity, nil
}
