package persistence

import (
	"database/sql"
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/category"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(cat *category.Category) error {
	if cat == nil {
		return errors.New("category is nil")
	}

	query := `
		INSERT INTO finance.categories (
			id, profile_id, name, type, color, icon, parent_id, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(query,
		cat.ID, cat.ProfileID, cat.Name, cat.Type,
		cat.Color, cat.Icon, cat.ParentID, cat.IsActive,
		cat.CreatedAt, cat.UpdatedAt,
	)
	return err
}

func (r *CategoryRepository) Update(cat *category.Category) error {
	if cat == nil {
		return errors.New("category is nil")
	}

	query := `
		UPDATE finance.categories
		SET name = $2, type = $3, color = $4, icon = $5, parent_id = $6,
			is_active = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.Exec(query,
		cat.ID, cat.Name, cat.Type, cat.Color, cat.Icon, cat.ParentID,
		cat.IsActive, cat.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("category not found")
	}

	return nil
}

func (r *CategoryRepository) FindByID(id string) (*category.Category, error) {
	query := `
		SELECT id, profile_id, name, type, color, icon, parent_id, is_active, created_at, updated_at
		FROM finance.categories
		WHERE id = $1
	`

	row := r.db.QueryRow(query, id)
	return scanCategory(row)
}

func (r *CategoryRepository) ListByProfile(profileID string) ([]*category.Category, error) {
	query := `
		SELECT id, profile_id, name, type, color, icon, parent_id, is_active, created_at, updated_at
		FROM finance.categories
		WHERE profile_id = $1
		ORDER BY name
	`

	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*category.Category
	for rows.Next() {
		cat, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

func (r *CategoryRepository) Deactivate(id string) error {
	query := `
		UPDATE finance.categories
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("category not found")
	}

	return nil
}

// GetDescendantIDs returns all descendant category IDs for a given category (including itself)
func (r *CategoryRepository) GetDescendantIDs(id string) ([]string, error) {
	query := `
		WITH RECURSIVE descendants AS (
			SELECT id FROM finance.categories WHERE id = $1
			UNION ALL
			SELECT c.id FROM finance.categories c
			INNER JOIN descendants d ON c.parent_id = d.id
		)
		SELECT id FROM descendants
	`

	rows, err := r.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var catID string
		if err := rows.Scan(&catID); err != nil {
			return nil, err
		}
		ids = append(ids, catID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

type categoryScanner interface {
	Scan(dest ...any) error
}

func scanCategory(scanner categoryScanner) (*category.Category, error) {
	cat := &category.Category{}
	var (
		color  sql.NullString
		icon   sql.NullString
		parent sql.NullString
	)

	err := scanner.Scan(
		&cat.ID,
		&cat.ProfileID,
		&cat.Name,
		&cat.Type,
		&color,
		&icon,
		&parent,
		&cat.IsActive,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	if color.Valid {
		s := color.String
		cat.Color = &s
	}
	if icon.Valid {
		s := icon.String
		cat.Icon = &s
	}
	if parent.Valid {
		s := parent.String
		cat.ParentID = &s
	}

	return cat, nil
}
