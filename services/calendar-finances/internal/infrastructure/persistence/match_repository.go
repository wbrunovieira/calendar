package persistence

import (
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

// MatchRepository stores reconciliation matches. Append-only: a match is undone with a
// reason, never removed, so the history of a reconciliation survives its corrections.
type MatchRepository struct {
	db Querier
}

func NewMatchRepository(db Querier) *MatchRepository {
	return &MatchRepository{db: db}
}

func (r *MatchRepository) Create(m *statement.Match) error {
	_, err := r.db.Exec(`
		INSERT INTO finance.reconciliation_matches
			(id, line_id, transaction_id, amount_minor, method, score, matched_by, matched_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, m.ID, m.LineID, m.TransactionID, m.AmountMinor, string(m.Method), m.Score, m.MatchedBy, m.MatchedAt)
	return err
}

// Unmatch records the undo on the row itself. The WHERE clause makes a second undo a
// no-op rather than an overwrite: the first reason is the one that explains it.
func (r *MatchRepository) Unmatch(matchID, reason string) error {
	result, err := r.db.Exec(`
		UPDATE finance.reconciliation_matches
		SET unmatched_at = NOW(), unmatched_reason = $2
		WHERE id = $1 AND unmatched_at IS NULL
	`, matchID, reason)
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

// ByLine returns every match ever made for a line, undone ones included: the undone
// ones are the explanation for why it is pending again.
func (r *MatchRepository) ByLine(lineID string) ([]*statement.Match, error) {
	return r.query(`WHERE line_id = $1 ORDER BY matched_at`, lineID)
}

func (r *MatchRepository) ByTransaction(transactionID string) ([]*statement.Match, error) {
	return r.query(`WHERE transaction_id = $1 ORDER BY matched_at`, transactionID)
}

// ClaimedOnAccount reports whether a LIVE match already links this entry to a line of
// THIS account.
//
// Scoped on purpose. One transfer is a single row that appears on two statements — it
// leaves the checking account and arrives on the card — so it must be claimable once
// on each. Asking globally made whichever account reconciled first win, and the other
// report money missing that was not: a phantom, and an order-dependent one.
func (r *MatchRepository) ClaimedOnAccount(transactionID, accountID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM finance.reconciliation_matches m
			JOIN finance.bank_statement_lines l ON l.id = m.line_id
			WHERE m.transaction_id = $1
			  AND l.account_id = $2
			  AND m.unmatched_at IS NULL
		)`, transactionID, accountID).Scan(&exists)
	return exists, err
}

func (r *MatchRepository) query(clause string, args ...any) ([]*statement.Match, error) {
	rows, err := r.db.Query(`
		SELECT id, line_id, transaction_id, amount_minor, method, score,
		       matched_by, matched_at, unmatched_at, unmatched_reason
		FROM finance.reconciliation_matches `+clause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*statement.Match
	for rows.Next() {
		m := &statement.Match{}
		var method string
		if err := rows.Scan(&m.ID, &m.LineID, &m.TransactionID, &m.AmountMinor, &method, &m.Score,
			&m.MatchedBy, &m.MatchedAt, &m.UnmatchedAt, &m.UnmatchedReason); err != nil {
			return nil, err
		}
		m.Method = statement.Method(method)
		out = append(out, m)
	}
	return out, rows.Err()
}
