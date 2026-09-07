package persistence

import (
	"database/sql"
	"errors"
)

// ErrIdempotencyConflict means the key was already used for a DIFFERENT request.
// That is a client bug, not a replay: answering with the first response would hide it,
// and performing the second would defeat the key.
var ErrIdempotencyConflict = errors.New("idempotency key reused for a different request")

// Claimed is the outcome of claiming a key. Fresh says whether the caller should do
// the work; when it is false, Status and Body carry what the first attempt answered.
type Claimed struct {
	Fresh  bool
	Status int
	Body   []byte
}

// IdempotencyStore keeps one row per key so a retried request performs its effect
// once.
//
// Four different clients write to this API — the web frontend, n8n, the WhatsApp
// agent, and direct calls — and retries from n8n are simultaneous by nature, so the
// low volume of this system offers no protection.
type IdempotencyStore struct {
	db Querier
}

func NewIdempotencyStore(db Querier) *IdempotencyStore {
	return &IdempotencyStore{db: db}
}

// Claim reserves a key, or reports that it was already used.
//
// It must run in the SAME transaction as the effect it guards. Recording the key after
// the effect commits leaves a window in which the money moved and the key did not, and
// the retry duplicates — the mistake this design exists to avoid.
func (s *IdempotencyStore) Claim(key, endpoint, requestHash string) (Claimed, error) {
	// The unique key does the arbitration. A read-then-write would let two concurrent
	// callers both find nothing and both proceed — which is exactly the case this
	// guards, since n8n retries arrive simultaneously.
	var claimed string
	err := s.db.QueryRow(`
		INSERT INTO finance.idempotency_keys (key, endpoint, request_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO NOTHING
		RETURNING key
	`, key, endpoint, requestHash).Scan(&claimed)
	if err == nil {
		return Claimed{Fresh: true}, nil
	}
	if err != sql.ErrNoRows {
		return Claimed{}, err
	}

	// The key was already there: this is a replay, or a client reusing a key.
	var (
		storedHash string
		status     sql.NullInt64
		body       sql.NullString
	)
	if err := s.db.QueryRow(`
		SELECT request_hash, response_status, response_body
		FROM finance.idempotency_keys WHERE key = $1
	`, key).Scan(&storedHash, &status, &body); err != nil {
		return Claimed{}, err
	}
	if storedHash != requestHash {
		return Claimed{}, ErrIdempotencyConflict
	}
	return Claimed{Fresh: false, Status: int(status.Int64), Body: []byte(body.String)}, nil
}

// RecordResponse stores what the first attempt answered, so a replay can be told the
// same thing instead of being told nothing.
func (s *IdempotencyStore) RecordResponse(key string, status int, body []byte) error {
	_, err := s.db.Exec(`
		UPDATE finance.idempotency_keys
		SET response_status = $2, response_body = $3
		WHERE key = $1
	`, key, status, body)
	return err
}
