package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/app"
)

// The route inventory test lists POST /api/v1/contracts/sync and says that listing
// it makes removing the guard "show up as a deliberate change". On its own it does
// not: mux reports a path regardless of which handler sits behind it, so deleting
// RequireWebhookSecret from the wiring left the entire suite green.
//
// This drives the REAL router — the one main.go serves — and asserts on behaviour
// rather than on the path being present. It needs no database: the guard answers
// before anything reads.
func postSyncThroughRealRouter(t *testing.T, secret string) *httptest.ResponseRecorder {
	t.Helper()

	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("wiring the app: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contracts/sync",
		strings.NewReader(`{"source":"wb-crm"}`))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("x-webhook-secret", secret)
	}
	rec := httptest.NewRecorder()
	application.Router.ServeHTTP(rec, req)
	return rec
}

func TestContractSyncIsGuardedInTheRealWiring(t *testing.T) {
	t.Setenv("CRM_SYNC_WEBHOOK_SECRET", "the-configured-secret")

	for name, secret := range map[string]string{
		"no secret presented": "",
		"wrong secret":        "not-it",
	} {
		t.Run(name, func(t *testing.T) {
			rec := postSyncThroughRealRouter(t, secret)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("code = %d, want 401 — the route is reachable without the secret", rec.Code)
			}
		})
	}
}

// With no secret configured the route must refuse everything rather than pass it
// through. A guard that opens when unconfigured is worse than none: it looks
// protected in the router and is not.
func TestContractSyncFailsClosedInTheRealWiring(t *testing.T) {
	t.Setenv("CRM_SYNC_WEBHOOK_SECRET", "")

	rec := postSyncThroughRealRouter(t, "anything")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", rec.Code)
	}
}

// The idempotency middleware on the /api/v1 subrouter reads the body and WRITES a
// claim row before the handler runs. Sitting behind it, this route let an
// unauthenticated caller reach the database — answering 409 instead of 401 proves
// the store ran, and a failure there is reported as a raw database error, in front
// of a handler that deliberately refuses to leak exactly that.
//
// The idle database would make any such touch fail loudly, so a 401 here is also
// the assertion that nothing was read or written before the guard.
func TestContractSyncIsAuthenticatedBeforeIdempotency(t *testing.T) {
	t.Setenv("CRM_SYNC_WEBHOOK_SECRET", "the-configured-secret")

	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("wiring the app: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contracts/sync",
		strings.NewReader(`{"source":"wb-crm"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "a-key-an-outsider-picked")

	rec := httptest.NewRecorder()
	application.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401; an unauthenticated caller reached the idempotency store", rec.Code)
	}
}
