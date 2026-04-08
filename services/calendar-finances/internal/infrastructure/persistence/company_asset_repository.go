package persistence

import (
	"database/sql"
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/companyasset"
)

type CompanyAssetRepository struct {
	db *sql.DB
}

func NewCompanyAssetRepository(db *sql.DB) *CompanyAssetRepository {
	return &CompanyAssetRepository{db: db}
}

func (r *CompanyAssetRepository) Create(a *companyasset.CompanyAsset) error {
	query := `
		INSERT INTO finance.company_assets (
			id, profile_id, name, category, purchase_date, purchase_amount, current_value,
			depreciation_rate, linked_transaction_id, notes, is_active,
			disposal_date, disposal_amount, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`
	_, err := r.db.Exec(query,
		a.ID, a.ProfileID, a.Name, a.Category, a.PurchaseDate, a.PurchaseAmount, a.CurrentValue,
		a.DepreciationRate, a.LinkedTransactionID, a.Notes, a.IsActive,
		a.DisposalDate, a.DisposalAmount, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *CompanyAssetRepository) FindByID(id string) (*companyasset.CompanyAsset, error) {
	query := `
		SELECT id, profile_id, name, category, purchase_date, purchase_amount, current_value,
			depreciation_rate, linked_transaction_id, notes, is_active,
			disposal_date, disposal_amount, created_at, updated_at
		FROM finance.company_assets WHERE id=$1
	`
	row := r.db.QueryRow(query, id)
	return scanAsset(row)
}

func (r *CompanyAssetRepository) ListByProfile(profileID string) ([]*companyasset.CompanyAsset, error) {
	query := `
		SELECT id, profile_id, name, category, purchase_date, purchase_amount, current_value,
			depreciation_rate, linked_transaction_id, notes, is_active,
			disposal_date, disposal_amount, created_at, updated_at
		FROM finance.company_assets
		WHERE profile_id=$1
		ORDER BY purchase_date DESC
	`
	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*companyasset.CompanyAsset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *CompanyAssetRepository) Update(a *companyasset.CompanyAsset) error {
	query := `
		UPDATE finance.company_assets
		SET name=$2, category=$3, purchase_date=$4, purchase_amount=$5, current_value=$6,
			depreciation_rate=$7, linked_transaction_id=$8, notes=$9, is_active=$10,
			disposal_date=$11, disposal_amount=$12, updated_at=$13
		WHERE id=$1
	`
	result, err := r.db.Exec(query,
		a.ID, a.Name, a.Category, a.PurchaseDate, a.PurchaseAmount, a.CurrentValue,
		a.DepreciationRate, a.LinkedTransactionID, a.Notes, a.IsActive,
		a.DisposalDate, a.DisposalAmount, a.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("company asset not found")
	}
	return nil
}

func (r *CompanyAssetRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.company_assets WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("company asset not found")
	}
	return nil
}

type assetScanner interface {
	Scan(dest ...any) error
}

func scanAsset(s assetScanner) (*companyasset.CompanyAsset, error) {
	a := &companyasset.CompanyAsset{}
	var (
		linkedTxID     sql.NullString
		notes          sql.NullString
		disposalDate   sql.NullTime
		disposalAmount sql.NullFloat64
	)
	err := s.Scan(
		&a.ID, &a.ProfileID, &a.Name, &a.Category, &a.PurchaseDate, &a.PurchaseAmount, &a.CurrentValue,
		&a.DepreciationRate, &linkedTxID, &notes, &a.IsActive,
		&disposalDate, &disposalAmount, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("company asset not found")
		}
		return nil, err
	}
	if linkedTxID.Valid {
		a.LinkedTransactionID = &linkedTxID.String
	}
	if notes.Valid {
		a.Notes = &notes.String
	}
	if disposalDate.Valid {
		a.DisposalDate = &disposalDate.Time
	}
	if disposalAmount.Valid {
		a.DisposalAmount = &disposalAmount.Float64
	}
	return a, nil
}
