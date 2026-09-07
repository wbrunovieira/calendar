//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Four different clients write to this API — the web frontend, n8n, the WhatsApp
// agent, and direct calls. A duplicate is a matter of time, and today nothing would
// even detect one.
//
// The detail almost everyone gets wrong: the key and the effect must be committed in
// the SAME database transaction. Recording the key after the effect commits leaves a
// window where money moved and the key did not, and the retry duplicates.

func TestE2E_IdempotencyKeyIsClaimedOnce(t *testing.T) {
	db := testDB(t)
	store := persistence.NewIdempotencyStore(db)
	key := "e2e-key-claimed-once"
	t.Cleanup(func() { db.Exec(`DELETE FROM finance.idempotency_keys WHERE key = $1`, key) })

	first, err := store.Claim(key, "POST /transactions", "hash-a")
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if !first.Fresh {
		t.Error("the first claim must be fresh")
	}

	second, err := store.Claim(key, "POST /transactions", "hash-a")
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if second.Fresh {
		t.Error("a replay must not be fresh, or the effect happens twice")
	}
}

func TestE2E_SameKeyWithADifferentBodyIsRefused(t *testing.T) {
	db := testDB(t)
	store := persistence.NewIdempotencyStore(db)
	key := "e2e-key-conflict"
	t.Cleanup(func() { db.Exec(`DELETE FROM finance.idempotency_keys WHERE key = $1`, key) })

	if _, err := store.Claim(key, "POST /transactions", "hash-a"); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	_, err := store.Claim(key, "POST /transactions", "hash-b")
	if err != persistence.ErrIdempotencyConflict {
		t.Errorf("err = %v, want ErrIdempotencyConflict: reusing a key for a different request is a client bug, not a replay", err)
	}
}

func TestE2E_KeyAndEffectCommitTogether(t *testing.T) {
	// If the work fails, the key must not survive — otherwise a legitimate retry is
	// mistaken for a replay and the operation never happens.
	db := testDB(t)
	seedUnitOfWork(t, db)
	key := "e2e-key-rollback"
	t.Cleanup(func() { db.Exec(`DELETE FROM finance.idempotency_keys WHERE key = $1`, key) })

	uow := persistence.NewUnitOfWork(db)
	err := uow.Do(func(r persistence.Repositories) error {
		if _, cerr := r.Idempotency.Claim(key, "POST /transactions", "hash-x"); cerr != nil {
			return cerr
		}
		return sql.ErrConnDone // the work fails after the key was claimed
	})
	if err == nil {
		t.Fatal("the failure must surface")
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM finance.idempotency_keys WHERE key = $1`, key).Scan(&count)
	if count != 0 {
		t.Error("the key survived a rolled-back effect: a legitimate retry would be refused as a replay and the operation would never happen")
	}
}

func TestE2E_ReplayReturnsTheRecordedResponse(t *testing.T) {
	db := testDB(t)
	store := persistence.NewIdempotencyStore(db)
	key := "e2e-key-response"
	t.Cleanup(func() { db.Exec(`DELETE FROM finance.idempotency_keys WHERE key = $1`, key) })

	if _, err := store.Claim(key, "POST /transactions", "hash-a"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := store.RecordResponse(key, 201, []byte(`{"data":{"id":"tx-1"}}`)); err != nil {
		t.Fatalf("recording: %v", err)
	}

	replay, err := store.Claim(key, "POST /transactions", "hash-a")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replay.Fresh {
		t.Fatal("this is a replay")
	}
	if replay.Status != 201 || string(replay.Body) != `{"data":{"id":"tx-1"}}` {
		t.Errorf("the recorded response must come back: got %d %s", replay.Status, replay.Body)
	}
}
