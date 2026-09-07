package persistence

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

type StatementRepository struct {
	db *sql.DB
}

func NewStatementRepository(db *sql.DB) *StatementRepository {
	return &StatementRepository{db: db}
}

// UpsertMany imports lines idempotently on (provider, external_id).
//
// Re-importing must refresh what the bank NOW says — Pluggy rewrites transactions,
// categories change, PENDING becomes POSTED — without duplicating rows and without
// discarding the reconciliation status already recorded. So the bank's own fields are
// overwritten and status/ignored_reason are left alone.
//
// The whole batch is one unit of work: a half-imported statement is worse than none,
// because the gap looks like the bank simply not having those lines.
func (r *StatementRepository) UpsertMany(lines []*statement.Line) (int, int, error) {
	if len(lines) == 0 {
		return 0, 0, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	// The bank's own fields are refreshed; the reconciliation status is not touched
	// here EXCEPT when the money itself changed. A PENDING authorisation settling as
	// POSTED routinely changes the amount — especially on foreign purchases, which is
	// most of this card — and a match whose bank side moved is by definition
	// unverified. Leaving it MATCHED asserts a check nobody performed.
	stmt, err := tx.Prepare(`
		INSERT INTO finance.bank_statement_lines
			(id, account_id, provider, external_id, booked_date, value_date,
			 amount_minor, currency, amount_account_minor, fx_rate,
			 description, end_to_end_id, raw, status, imported_at, last_seen_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW(),NOW(),NOW())
		ON CONFLICT (account_id, provider, external_id) DO UPDATE SET
			booked_date = EXCLUDED.booked_date,
			value_date = EXCLUDED.value_date,
			amount_minor = EXCLUDED.amount_minor,
			currency = EXCLUDED.currency,
			amount_account_minor = EXCLUDED.amount_account_minor,
			fx_rate = EXCLUDED.fx_rate,
			description = EXCLUDED.description,
			end_to_end_id = EXCLUDED.end_to_end_id,
			raw = EXCLUDED.raw,
			status = CASE
				WHEN finance.bank_statement_lines.status = 'MATCHED'
				 AND (finance.bank_statement_lines.amount_minor,
				      finance.bank_statement_lines.amount_account_minor,
				      finance.bank_statement_lines.booked_date)
				     IS DISTINCT FROM
				     (EXCLUDED.amount_minor, EXCLUDED.amount_account_minor, EXCLUDED.booked_date)
				THEN 'UNMATCHED'
				ELSE finance.bank_statement_lines.status
			END,
			last_seen_at = NOW(),
			updated_at = NOW()
		RETURNING id, (xmax = 0) AS inserted
	`)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	// Every version the provider reported is appended, never replaced. Overwriting
	// raw would destroy the evidence that a change came from the bank rather than
	// from us — and the previous version is what explains a dropped match.
	revision, err := tx.Prepare(`
		INSERT INTO finance.bank_statement_line_revisions
			(line_id, booked_date, amount_minor, currency, amount_account_minor, description, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer revision.Close()

	var inserted, updated int
	for _, l := range lines {
		var storedID string
		var isNew bool
		err := stmt.QueryRow(
			l.ID, l.AccountID, string(l.Provider), l.ExternalID, l.BookedDate, l.ValueDate,
			l.AmountMinor, l.Currency, l.AmountAccountMinor, l.FXRate,
			l.Description, l.EndToEndID, []byte(l.Raw), string(l.Status),
		).Scan(&storedID, &isNew)
		if err != nil {
			return 0, 0, err
		}
		if _, err := revision.Exec(storedID, l.BookedDate, l.AmountMinor, l.Currency,
			l.AmountAccountMinor, l.Description, []byte(l.Raw)); err != nil {
			return 0, 0, err
		}
		if isNew {
			inserted++
		} else {
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return inserted, updated, nil
}

// Revisions returns every version the provider reported for a line, newest first.
func (r *StatementRepository) Revisions(lineID string) ([]statement.Revision, error) {
	rows, err := r.db.Query(`
		SELECT seen_at, booked_date, amount_minor, currency, amount_account_minor, description, raw
		FROM finance.bank_statement_line_revisions
		WHERE line_id = $1 ORDER BY seen_at DESC`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []statement.Revision
	for rows.Next() {
		var rev statement.Revision
		var raw []byte
		if err := rows.Scan(&rev.SeenAt, &rev.BookedDate, &rev.AmountMinor, &rev.Currency,
			&rev.AmountAccountMinor, &rev.Description, &raw); err != nil {
			return nil, err
		}
		rev.Raw = raw
		out = append(out, rev)
	}
	return out, rows.Err()
}

func (r *StatementRepository) FindByExternalID(provider statement.Provider, externalID string) (*statement.Line, error) {
	rows, err := r.query(`WHERE provider = $1 AND external_id = $2`, string(provider), externalID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, statement.ErrNotFound
	}
	return rows[0], nil
}

func (r *StatementRepository) List(filter statement.ListFilter) ([]*statement.Line, error) {
	conditions := []string{}
	args := []any{}
	add := func(cond string, value any) {
		args = append(args, value)
		conditions = append(conditions, strings.Replace(cond, "?", "$"+strconv.Itoa(len(args)), 1))
	}

	if filter.AccountID != "" {
		add("account_id = ?", filter.AccountID)
	}
	if filter.From != nil {
		add("booked_date >= ?", *filter.From)
	}
	if filter.To != nil {
		add("booked_date <= ?", *filter.To)
	}
	if filter.Status != nil {
		add("status = ?", string(*filter.Status))
	}
	if filter.Provider != nil {
		add("provider = ?", string(*filter.Provider))
	}
	if filter.ExternalID != nil {
		add("external_id = ?", *filter.ExternalID)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	return r.query(where+" ORDER BY booked_date, external_id", args...)
}

func (r *StatementRepository) Update(line *statement.Line) error {
	result, err := r.db.Exec(`
		UPDATE finance.bank_statement_lines
		SET status = $2, ignored_reason = $3, matched_transaction_id = $4, updated_at = NOW()
		WHERE id = $1
	`, line.ID, string(line.Status), line.IgnoredReason, line.MatchedTransactionID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return statement.ErrNotFound
	}
	return nil
}

func (r *StatementRepository) query(clause string, args ...any) ([]*statement.Line, error) {
	rows, err := r.db.Query(`
		SELECT id, account_id, provider, external_id, booked_date, value_date,
		       amount_minor, currency, amount_account_minor, fx_rate,
		       description, end_to_end_id, raw, status, ignored_reason,
		       matched_transaction_id, imported_at, last_seen_at, updated_at
		FROM finance.bank_statement_lines `+clause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []*statement.Line
	for rows.Next() {
		l := &statement.Line{}
		var provider, status string
		var raw []byte
		if err := rows.Scan(&l.ID, &l.AccountID, &provider, &l.ExternalID, &l.BookedDate, &l.ValueDate,
			&l.AmountMinor, &l.Currency, &l.AmountAccountMinor, &l.FXRate,
			&l.Description, &l.EndToEndID, &raw, &status, &l.IgnoredReason,
			&l.MatchedTransactionID, &l.ImportedAt, &l.LastSeenAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.Provider = statement.Provider(provider)
		l.Status = statement.Status(status)
		l.Raw = raw
		lines = append(lines, l)
	}
	return lines, rows.Err()
}
