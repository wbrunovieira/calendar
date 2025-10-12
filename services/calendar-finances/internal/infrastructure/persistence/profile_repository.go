package persistence

import (
	"database/sql"
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) Create(p *profile.Profile) error {
	query := `
		INSERT INTO finance.profiles (id, calendar_id, name, type, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(query, p.ID, p.CalendarID, p.Name, p.Type, p.IsActive, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *ProfileRepository) FindByID(id string) (*profile.Profile, error) {
	query := `
		SELECT id, calendar_id, name, type, is_active, created_at, updated_at
		FROM finance.profiles
		WHERE id = $1
	`

	p := &profile.Profile{}
	err := r.db.QueryRow(query, id).Scan(
		&p.ID,
		&p.CalendarID,
		&p.Name,
		&p.Type,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("profile not found")
	}

	if err != nil {
		return nil, err
	}

	return p, nil
}

func (r *ProfileRepository) FindByCalendarID(calendarID string) (*profile.Profile, error) {
	query := `
		SELECT id, calendar_id, name, type, is_active, created_at, updated_at
		FROM finance.profiles
		WHERE calendar_id = $1
	`

	p := &profile.Profile{}
	err := r.db.QueryRow(query, calendarID).Scan(
		&p.ID,
		&p.CalendarID,
		&p.Name,
		&p.Type,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("profile not found for this calendar")
	}

	if err != nil {
		return nil, err
	}

	return p, nil
}

func (r *ProfileRepository) FindAll() ([]*profile.Profile, error) {
	query := `
		SELECT id, calendar_id, name, type, is_active, created_at, updated_at
		FROM finance.profiles
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*profile.Profile

	for rows.Next() {
		p := &profile.Profile{}
		err := rows.Scan(
			&p.ID,
			&p.CalendarID,
			&p.Name,
			&p.Type,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (r *ProfileRepository) Update(p *profile.Profile) error {
	query := `
		UPDATE finance.profiles
		SET name = $1, type = $2, is_active = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := r.db.Exec(query, p.Name, p.Type, p.IsActive, p.UpdatedAt, p.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("profile not found")
	}

	return nil
}

func (r *ProfileRepository) Delete(id string) error {
	query := `DELETE FROM finance.profiles WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("profile not found")
	}

	return nil
}
