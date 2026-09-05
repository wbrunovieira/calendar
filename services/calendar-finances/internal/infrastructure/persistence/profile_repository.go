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
		INSERT INTO finance.profiles (
			id, calendar_id, name, type, is_active,
			legal_entity_type, company_name, cnpj, simples_nacional, tax_regime, das_aliquota, opening_date,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Exec(query,
		p.ID, p.CalendarID, p.Name, p.Type, p.IsActive,
		p.LegalEntityType, p.CompanyName, p.CNPJ, p.SimplesNacional, p.TaxRegime, p.DasAliquota, p.OpeningDate,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *ProfileRepository) FindByID(id string) (*profile.Profile, error) {
	query := `
		SELECT id, calendar_id, name, type, is_active,
			legal_entity_type, company_name, cnpj, simples_nacional, tax_regime, das_aliquota, opening_date,
			created_at, updated_at
		FROM finance.profiles
		WHERE id = $1
	`
	row := r.db.QueryRow(query, id)
	return scanProfile(row)
}

func (r *ProfileRepository) FindByCalendarID(calendarID string) (*profile.Profile, error) {
	query := `
		SELECT id, calendar_id, name, type, is_active,
			legal_entity_type, company_name, cnpj, simples_nacional, tax_regime, das_aliquota, opening_date,
			created_at, updated_at
		FROM finance.profiles
		WHERE calendar_id = $1
	`
	row := r.db.QueryRow(query, calendarID)
	p, err := scanProfile(row)
	if err != nil {
		if errors.Is(err, errProfileNotFound) {
			return nil, errors.New("profile not found for this calendar")
		}
		return nil, err
	}
	return p, nil
}

func (r *ProfileRepository) FindAll() ([]*profile.Profile, error) {
	query := `
		SELECT id, calendar_id, name, type, is_active,
			legal_entity_type, company_name, cnpj, simples_nacional, tax_regime, das_aliquota, opening_date,
			created_at, updated_at
		FROM finance.profiles
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*profile.Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *ProfileRepository) Update(p *profile.Profile) error {
	query := `
		UPDATE finance.profiles
		SET name = $1, type = $2, is_active = $3,
			legal_entity_type = $4, company_name = $5, cnpj = $6,
			simples_nacional = $7, tax_regime = $8, das_aliquota = $9, opening_date = $10,
			updated_at = $11
		WHERE id = $12
	`
	result, err := r.db.Exec(query,
		p.Name, p.Type, p.IsActive,
		p.LegalEntityType, p.CompanyName, p.CNPJ,
		p.SimplesNacional, p.TaxRegime, p.DasAliquota, p.OpeningDate,
		p.UpdatedAt, p.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("profile not found")
	}
	return nil
}

func (r *ProfileRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("profile not found")
	}
	return nil
}

// --- scanner helpers ---

var errProfileNotFound = errors.New("profile not found")

type profileScanner interface {
	Scan(dest ...any) error
}

func scanProfile(s profileScanner) (*profile.Profile, error) {
	p := &profile.Profile{}
	var (
		legalEntityType sql.NullString
		companyName     sql.NullString
		cnpj            sql.NullString
		simplesNacional sql.NullBool
		taxRegime       sql.NullString
		dasAliquota     sql.NullFloat64
		openingDate     sql.NullTime
	)

	err := s.Scan(
		&p.ID, &p.CalendarID, &p.Name, &p.Type, &p.IsActive,
		&legalEntityType, &companyName, &cnpj, &simplesNacional, &taxRegime, &dasAliquota, &openingDate,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errProfileNotFound
		}
		return nil, err
	}

	if legalEntityType.Valid {
		v := profile.LegalEntityType(legalEntityType.String)
		p.LegalEntityType = &v
	}
	if companyName.Valid {
		p.CompanyName = &companyName.String
	}
	if cnpj.Valid {
		p.CNPJ = &cnpj.String
	}
	if simplesNacional.Valid {
		p.SimplesNacional = &simplesNacional.Bool
	}
	if taxRegime.Valid {
		v := profile.TaxRegime(taxRegime.String)
		p.TaxRegime = &v
	}
	if dasAliquota.Valid {
		p.DasAliquota = &dasAliquota.Float64
	}
	if openingDate.Valid {
		t := openingDate.Time
		p.OpeningDate = &t
	}

	return p, nil
}
