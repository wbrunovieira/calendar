package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// fakeStore behaves like the real one: a claim with no recorded response reads as
// in flight, and releasing removes it.
type fakeStore struct {
	rows     map[string]*fakeRow
	released int
}

type fakeRow struct {
	hash   string
	status int
	body   []byte
}

func newFakeStore() *fakeStore { return &fakeStore{rows: map[string]*fakeRow{}} }

func (f *fakeStore) Claim(key, _, requestHash string) (persistence.Claimed, error) {
	row, ok := f.rows[key]
	if !ok {
		f.rows[key] = &fakeRow{hash: requestHash}
		return persistence.Claimed{Fresh: true}, nil
	}
	if row.hash != requestHash {
		return persistence.Claimed{}, persistence.ErrIdempotencyConflict
	}
	if row.status == 0 {
		return persistence.Claimed{InProgress: true}, nil
	}
	return persistence.Claimed{Status: row.status, Body: row.body}, nil
}

func (f *fakeStore) RecordResponse(key string, status int, body []byte) error {
	if row, ok := f.rows[key]; ok {
		row.status, row.body = status, body
	}
	return nil
}

func (f *fakeStore) Release(key string) error {
	if row, ok := f.rows[key]; ok && row.status == 0 {
		delete(f.rows, key)
		f.released++
	}
	return nil
}

func send(t *testing.T, h http.Handler, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(body))
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// A failed attempt achieved nothing, so it must not hold the key.
//
// Before this, the middleware recorded a response only on success and had no way to
// give a key back. A transient 500 therefore left the row claimed with a NULL status
// forever: every later retry read it as still in flight and got 409. n8n retries by
// default, so the real shape was a database blip followed by a write that never
// happened and never could — worse than having no idempotency at all, because it
// turned a retryable error into silent loss.
func TestIdempotency_AFailedAttemptDoesNotHoldTheKeyForever(t *testing.T) {
	store := newFakeStore()
	fail := true
	handler := Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "the database blinked", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"tx-1"}}`))
	}))

	if got := send(t, handler, "n8n-42", `{"amount":100}`).Code; got != http.StatusInternalServerError {
		t.Fatalf("first attempt: got %d, want 500", got)
	}

	fail = false
	retry := send(t, handler, "n8n-42", `{"amount":100}`)
	if retry.Code == http.StatusConflict {
		t.Fatal("the retry was refused as a replay of something that never happened")
	}
	if retry.Code != http.StatusCreated {
		t.Fatalf("the retry should have been allowed through: got %d", retry.Code)
	}
}

// A handler that panics answered nothing either.
func TestIdempotency_APanickingHandlerReleasesItsKey(t *testing.T) {
	store := newFakeStore()
	handler := Idempotency(store)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	func() {
		defer func() { _ = recover() }()
		send(t, handler, "n8n-43", `{"amount":100}`)
	}()

	if store.released != 1 {
		t.Fatalf("the key was not released after the panic (released=%d)", store.released)
	}
}

// The guarantee that must survive all of the above: a successful write is performed
// once and the repeat is answered from the record.
func TestIdempotency_ASuccessIsPerformedOnceAndReplayed(t *testing.T) {
	store := newFakeStore()
	calls := 0
	handler := Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"tx-1"}}`))
	}))

	first := send(t, handler, "n8n-44", `{"amount":100}`)
	second := send(t, handler, "n8n-44", `{"amount":100}`)

	if calls != 1 {
		t.Errorf("the effect ran %d times", calls)
	}
	if second.Code != first.Code || second.Body.String() != first.Body.String() {
		t.Errorf("the replay answered differently: %d %q", second.Code, second.Body.String())
	}
	if second.Header().Get("Idempotent-Replay") != "true" {
		t.Error("a replay must say so")
	}
}

// The same key with a different body is a client bug, and neither answering the first
// response nor performing the second is acceptable.
func TestIdempotency_TheSameKeyWithADifferentRequestIsRefused(t *testing.T) {
	store := newFakeStore()
	handler := Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	send(t, handler, "n8n-45", `{"amount":100}`)
	if got := send(t, handler, "n8n-45", `{"amount":999}`).Code; got != http.StatusConflict {
		t.Fatalf("got %d, want 409", got)
	}
}
