package persistence

import (
	"database/sql"
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/capitalcontribution"
)

type CapitalContributionRepository struct {
	db *sql.DB
}

func NewCapitalContributionRepository(db *sql.DB) *CapitalContributionRepository {
	return &CapitalContributionRepository{db: db}
}

func (r *CapitalContributionRepository) Create(c *capitalcontribution.CapitalContribution) error {
	query := `
		INSERT INTO finance.capital_contributions (
			id, profile_id, type, amount, date, description,
			source_account_id, notes, is_returned, returned_amount, returned_at,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`
	_, err := r.db.Exec(query,
		c.ID, c.ProfileID, c.Type, c.Amount, c.Date, c.Description,
		c.SourceAccountID, c.Notes, c.IsReturned, c.ReturnedAmount, c.ReturnedAt,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *CapitalContributionRepository) FindByID(id string) (*capitalcontribution.CapitalContribution, error) {
	query := `
		SELECT id, profile_id, type, amount, date, description,
			source_account_id, notes, is_returned, returned_amount, returned_at,
			created_at, updated_at
		FROM finance.capital_contributions WHERE id = $1
	`
	row := r.db.QueryRow(query, id)
	return scanContribution(row)
}

func (r *CapitalContributionRepository) ListByProfile(profileID string) ([]*capitalcontribution.CapitalContribution, error) {
	query := `
		SELECT id, profile_id, type, amount, date, description,
			source_account_id, notes, is_returned, returned_amount, returned_at,
			created_at, updated_at
		FROM finance.capital_contributions
		WHERE profile_id = $1
		ORDER BY date DESC
	`
	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*capitalcontribution.CapitalContribution
	for rows.Next() {
		c, err := scanContribution(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *CapitalContributionRepository) Update(c *capitalcontribution.CapitalContribution) error {
	query := `
		UPDATE finance.capital_contributions
		SET type=$2, amount=$3, date=$4, description=$5,
			source_account_id=$6, notes=$7, is_returned=$8, returned_amount=$9, returned_at=$10,
			updated_at=$11
		WHERE id=$1
	`
	result, err := r.db.Exec(query,
		c.ID, c.Type, c.Amount, c.Date, c.Description,
		c.SourceAccountID, c.Notes, c.IsReturned, c.ReturnedAmount, c.ReturnedAt,
		c.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("capital contribution not found")
	}
	return nil
}

func (r *CapitalContributionRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.capital_contributions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("capital contribution not found")
	}
	return nil
}

type contributionScanner interface {
	Scan(dest ...any) error
}

func scanContribution(s contributionScanner) (*capitalcontribution.CapitalContribution, error) {
	c := &capitalcontribution.CapitalContribution{}
	var (
		sourceAccountID sql.NullString
		notes           sql.NullString
		returnedAt      sql.NullTime
	)
	err := s.Scan(
		&c.ID, &c.ProfileID, &c.Type, &c.Amount, &c.Date, &c.Description,
		&sourceAccountID, &notes, &c.IsReturned, &c.ReturnedAmount, &returnedAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("capital contribution not found")
		}
		return nil, err
	}
	if sourceAccountID.Valid {
		c.SourceAccountID = &sourceAccountID.String
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if returnedAt.Valid {
		c.ReturnedAt = &returnedAt.Time
	}
	return c, nil
}
