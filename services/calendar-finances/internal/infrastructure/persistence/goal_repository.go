package persistence

import (
	"database/sql"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/goal"
)

type GoalRepository struct {
	db *sql.DB
}

func NewGoalRepository(db *sql.DB) *GoalRepository {
	return &GoalRepository{db: db}
}

func (r *GoalRepository) Create(g *goal.Goal) error {
	query := `
		INSERT INTO finance.goals (id, profile_id, category_id, name, description, target_amount, current_amount, priority, target_date, status, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(query,
		g.ID, g.ProfileID, toNullString(g.CategoryID), g.Name, g.Description,
		g.TargetAmount, g.CurrentAmount, string(g.Priority),
		toNullTime(g.TargetDate), string(g.Status), g.Link,
		g.CreatedAt, g.UpdatedAt,
	)
	return err
}

func (r *GoalRepository) Update(g *goal.Goal) error {
	query := `
		UPDATE finance.goals
		SET name = $2, description = $3, target_amount = $4, current_amount = $5,
		    priority = $6, target_date = $7, status = $8, link = $9,
		    category_id = $10, updated_at = $11
		WHERE id = $1
	`
	result, err := r.db.Exec(query,
		g.ID, g.Name, g.Description, g.TargetAmount, g.CurrentAmount,
		string(g.Priority), toNullTime(g.TargetDate), string(g.Status),
		g.Link, toNullString(g.CategoryID), g.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *GoalRepository) FindByID(id string) (*goal.Goal, error) {
	query := `
		SELECT id, profile_id, category_id, name, description, target_amount, current_amount,
		       priority, target_date, status, link, created_at, updated_at
		FROM finance.goals WHERE id = $1
	`
	return r.scanGoal(r.db.QueryRow(query, id))
}

func (r *GoalRepository) ListByProfile(profileID string) ([]*goal.Goal, error) {
	query := `
		SELECT id, profile_id, category_id, name, description, target_amount, current_amount,
		       priority, target_date, status, link, created_at, updated_at
		FROM finance.goals WHERE profile_id = $1
		ORDER BY
			CASE status
				WHEN 'ACTIVE' THEN 0
				WHEN 'COMPLETED' THEN 1
				WHEN 'CANCELLED' THEN 2
			END,
			CASE priority
				WHEN 'HIGH' THEN 0
				WHEN 'MEDIUM' THEN 1
				WHEN 'LOW' THEN 2
			END,
			target_date ASC NULLS LAST
	`
	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []*goal.Goal
	for rows.Next() {
		g, err := r.scanGoalFromRows(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

func (r *GoalRepository) Delete(id string) error {
	result, err := r.db.Exec("DELETE FROM finance.goals WHERE id = $1", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *GoalRepository) scanGoal(row *sql.Row) (*goal.Goal, error) {
	var g goal.Goal
	var categoryID sql.NullString
	var targetDate sql.NullTime
	var priority, status string

	err := row.Scan(
		&g.ID, &g.ProfileID, &categoryID, &g.Name, &g.Description,
		&g.TargetAmount, &g.CurrentAmount, &priority, &targetDate,
		&status, &g.Link, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	g.Priority = goal.Priority(priority)
	g.Status = goal.Status(status)
	if categoryID.Valid {
		g.CategoryID = &categoryID.String
	}
	if targetDate.Valid {
		t := targetDate.Time
		g.TargetDate = &t
	}

	return &g, nil
}

func (r *GoalRepository) scanGoalFromRows(rows *sql.Rows) (*goal.Goal, error) {
	var g goal.Goal
	var categoryID sql.NullString
	var targetDate sql.NullTime
	var priority, status string

	err := rows.Scan(
		&g.ID, &g.ProfileID, &categoryID, &g.Name, &g.Description,
		&g.TargetAmount, &g.CurrentAmount, &priority, &targetDate,
		&status, &g.Link, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	g.Priority = goal.Priority(priority)
	g.Status = goal.Status(status)
	if categoryID.Valid {
		g.CategoryID = &categoryID.String
	}
	if targetDate.Valid {
		t := targetDate.Time
		g.TargetDate = &t
	}

	return &g, nil
}

func toNullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
