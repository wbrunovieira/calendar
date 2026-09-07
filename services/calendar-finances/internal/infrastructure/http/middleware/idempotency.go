// Package middleware holds cross-cutting HTTP concerns.
package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Store is what the middleware needs from the idempotency layer.
type Store interface {
	Claim(key, endpoint, requestHash string) (persistence.Claimed, error)
	RecordResponse(key string, status int, body []byte) error
	// Release returns a key claimed by an attempt that achieved nothing. Without it a
	// failed request holds its key forever and every retry is refused with 409.
	Release(key string) error
}

// Idempotency makes a repeated write perform its effect once.
//
// Four different clients write to this API — the web frontend, n8n, the WhatsApp
// agent, and direct calls — and n8n retries arrive simultaneously by nature, so the
// low volume of this system is no protection. Without this, a resent invoice payment
// debits the funding account twice.
//
// The key is OPTIONAL: a caller sending none is served as before. Making it mandatory
// would break every existing client at once, and a guard nobody can adopt gradually
// gets removed rather than adopted.
func Idempotency(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" || (r.Method != http.MethodPost && r.Method != http.MethodPut) {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "could not read the request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			// The hash covers method, path and body: the same key with a different
			// request is a client bug, and answering it with the first response would
			// hide the bug while performing it would defeat the key.
			sum := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), body...))
			hash := hex.EncodeToString(sum[:])

			claimed, err := store.Claim(key, r.Method+" "+r.URL.Path, hash)
			if errors.Is(err, persistence.ErrIdempotencyConflict) {
				http.Error(w, "this Idempotency-Key was already used for a different request", http.StatusConflict)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if !claimed.Fresh {
				if claimed.InProgress {
					// The first attempt claimed the key and has not finished. Replaying
					// the effect is exactly what must not happen, and inventing a
					// success would be worse.
					http.Error(w, "a request with this Idempotency-Key is still in flight", http.StatusConflict)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Idempotent-Replay", "true")
				w.WriteHeader(claimed.Status)
				_, _ = w.Write(claimed.Body)
				return
			}

			rec := &recorder{ResponseWriter: w, status: http.StatusOK}
			finished := false
			defer func() {
				// A panicking handler must not keep the key. The request answered
				// nothing, so the retry that follows a crash has to be allowed through
				// rather than refused as a replay of something that never happened.
				if !finished {
					_ = store.Release(key)
				}
			}()
			next.ServeHTTP(rec, r)
			finished = true

			// Only a success is recorded. Replaying a failure would deny a retry that
			// should be allowed to succeed.
			if rec.status < 400 {
				if err := store.RecordResponse(key, rec.status, rec.body.Bytes()); err != nil {
					// The effect happened and the key cannot say so. Releasing here
					// would invite a duplicate of a write that already landed, so the
					// key stays claimed and this is logged loudly instead: a later
					// retry gets 409 and someone has to look, which is the safer of
					// two bad outcomes.
					log.Printf("idempotency: effect committed but the response was not recorded for key %q: %v", key, err)
				}
				return
			}

			// The attempt failed, so it holds nothing worth replaying. Keeping the
			// claim would turn a transient error into a permanent 409 and the write
			// would never happen — the failure mode this guard exists to prevent,
			// arriving through the other door.
			if err := store.Release(key); err != nil {
				log.Printf("idempotency: could not release key %q after a failed attempt: %v", key, err)
			}
		})
	}
}

type recorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *recorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *recorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
