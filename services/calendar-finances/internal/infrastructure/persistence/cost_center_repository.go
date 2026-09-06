package persistence

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
)

// CostCenterRepository implements costcenter.Repository using PostgreSQL
type CostCenterRepository struct {
	db *sql.DB
}

func NewCostCenterRepository(db *sql.DB) *CostCenterRepository {
	return &CostCenterRepository{db: db}
}

func (r *CostCenterRepository) Create(c *costcenter.CostCenter) error {
	query := `
		INSERT INTO finance.cost_centers (
			id, profile_id, name, type, color, external_id, external_source, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(query,
		c.ID, c.ProfileID, c.Name, c.Type, c.Color,
		c.ExternalID, nullableSource(c.ExternalSrc),
		c.IsActive, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *CostCenterRepository) FindByID(id string) (*costcenter.CostCenter, error) {
	query := `
		SELECT id, profile_id, name, type, color, external_id, external_source, is_active, created_at, updated_at
		FROM finance.cost_centers WHERE id = $1
	`
	row := r.db.QueryRow(query, id)
	return scanCostCenter(row)
}

func (r *CostCenterRepository) FindByProfile(profileID string) ([]*costcenter.CostCenter, error) {
	query := `
		SELECT id, profile_id, name, type, color, external_id, external_source, is_active, created_at, updated_at
		FROM finance.cost_centers
		WHERE profile_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*costcenter.CostCenter
	for rows.Next() {
		c, err := scanCostCenter(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *CostCenterRepository) Update(c *costcenter.CostCenter) error {
	query := `
		UPDATE finance.cost_centers
		SET name=$2, type=$3, color=$4, is_active=$5, updated_at=$6
		WHERE id=$1
	`
	result, err := r.db.Exec(query,
		c.ID, c.Name, c.Type, c.Color, c.IsActive, c.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("cost center not found")
	}
	return nil
}

func (r *CostCenterRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.cost_centers WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("cost center not found")
	}
	return nil
}

type costCenterScanner interface {
	Scan(dest ...any) error
}

func scanCostCenter(s costCenterScanner) (*costcenter.CostCenter, error) {
	c := &costcenter.CostCenter{}
	var color, externalID, externalSource sql.NullString
	err := s.Scan(
		&c.ID, &c.ProfileID, &c.Name, &c.Type, &color,
		&externalID, &externalSource,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("cost center not found")
		}
		return nil, err
	}
	if color.Valid {
		c.Color = &color.String
	}
	if externalID.Valid {
		value := externalID.String
		c.ExternalID = &value
	}
	c.ExternalSrc = externalSource.String
	return c, nil
}

// FindByExternalRef resolves a cost center from the id it mirrors in another
// system. This is what makes a repeated sync idempotent: the CRM can announce
// the same organization any number of times and find the same client here,
// instead of creating a twin on every retry.
func (r *CostCenterRepository) FindByExternalRef(profileID, source, externalID string) (*costcenter.CostCenter, error) {
	query := `
		SELECT id, profile_id, name, type, color, external_id, external_source, is_active, created_at, updated_at
		FROM finance.cost_centers
		WHERE profile_id = $1 AND external_source = $2 AND external_id = $3
	`
	return scanCostCenter(r.db.QueryRow(query, profileID, source, externalID))
}

// nullableSource keeps an empty source out of the column, so the partial unique
// index on (profile_id, external_source, external_id) behaves predictably.
func nullableSource(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
