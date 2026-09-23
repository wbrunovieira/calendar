package persistence

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/brunovieira/calendar-finances/internal/domain/contract"
)

// ContractRepository stores the local mirror of deals decided in another system.
type ContractRepository struct {
	db *sql.DB
}

func NewContractRepository(db *sql.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

const contractColumns = `id, profile_id, cost_center_id, source, external_id, title,
	total_minor, currency, status, closed_at, remote_updated_at, created_at, updated_at`

func (r *ContractRepository) Create(c *contract.Contract) error {
	_, err := r.db.Exec(`
		INSERT INTO finance.contracts (`+contractColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		c.ID, c.ProfileID, c.CostCenterID, c.Source, c.ExternalID, c.Title,
		nullableMinor(c.TotalMinor), nullableCurrency(c.Currency), c.Status,
		c.ClosedAt, c.RemoteUpdatedAt, c.CreatedAt, c.UpdatedAt,
	)
	// Losing the race to another delivery for the same deal is recoverable, and the
	// caller can only recover if it can tell that case from any other write failure.
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
		return contract.ErrDuplicate
	}
	return err
}

func (r *ContractRepository) Update(c *contract.Contract) error {
	result, err := r.db.Exec(`
		UPDATE finance.contracts
		SET cost_center_id = $2, title = $3, total_minor = $4, currency = $5,
		    status = $6, closed_at = $7, remote_updated_at = $8, updated_at = $9
		WHERE id = $1`,
		c.ID, c.CostCenterID, c.Title, nullableMinor(c.TotalMinor),
		nullableCurrency(c.Currency), c.Status, c.ClosedAt, c.RemoteUpdatedAt, c.UpdatedAt,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		// Reporting success for a write that changed nothing would tell the sender
		// its state was stored when the row is not there.
		return contract.ErrNotFound
	}
	return nil
}

func (r *ContractRepository) FindByID(id string) (*contract.Contract, error) {
	return scanContract(r.db.QueryRow(
		`SELECT `+contractColumns+` FROM finance.contracts WHERE id = $1`, id))
}

func (r *ContractRepository) FindByExternalRef(source, externalID string) (*contract.Contract, error) {
	return scanContract(r.db.QueryRow(
		`SELECT `+contractColumns+` FROM finance.contracts
		 WHERE source = $1 AND external_id = $2`, source, externalID))
}

func (r *ContractRepository) ListByProfile(profileID string) ([]*contract.Contract, error) {
	rows, err := r.db.Query(
		`SELECT `+contractColumns+` FROM finance.contracts
		 WHERE profile_id = $1 ORDER BY remote_updated_at DESC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contracts := []*contract.Contract{}
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, c)
	}
	return contracts, rows.Err()
}

type contractScanner interface {
	Scan(dest ...interface{}) error
}

func scanContract(s contractScanner) (*contract.Contract, error) {
	c := &contract.Contract{}
	var (
		totalMinor sql.NullInt64
		currency   sql.NullString
		closedAt   sql.NullTime
	)
	err := s.Scan(
		&c.ID, &c.ProfileID, &c.CostCenterID, &c.Source, &c.ExternalID, &c.Title,
		&totalMinor, &currency, &c.Status, &closedAt, &c.RemoteUpdatedAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contract.ErrNotFound
		}
		return nil, err
	}
	if totalMinor.Valid {
		value := totalMinor.Int64
		c.TotalMinor = &value
	}
	c.Currency = currency.String
	if closedAt.Valid {
		when := closedAt.Time
		c.ClosedAt = &when
	}
	return c, nil
}

// nullableMinor keeps a deal with no value yet out of the column as NULL. Storing a
// zero instead would read back as free work.
func nullableMinor(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullableCurrency(c string) interface{} {
	if c == "" {
		return nil
	}
	return c
}
