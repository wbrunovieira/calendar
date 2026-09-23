package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func reached() (http.HandlerFunc, *bool) {
	hit := false
	return func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}, &hit
}

func call(t *testing.T, guard http.Handler, header string, set bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/contracts/sync", nil)
	if set {
		req.Header.Set(WebhookSecretHeader, header)
	}
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	return rec
}

func TestWebhookSecretLetsTheKnownCallerThrough(t *testing.T) {
	next, hit := reached()
	rec := call(t, RequireWebhookSecret("s3cr3t", next), "s3cr3t", true)

	if rec.Code != http.StatusOK || !*hit {
		t.Fatalf("code = %d, reached = %v", rec.Code, *hit)
	}
}

func TestWebhookSecretRefusesEveryoneElse(t *testing.T) {
	for name, tc := range map[string]struct {
		header string
		set    bool
	}{
		"wrong secret":  {"not-the-secret", true},
		"no header":     {"", false},
		"empty header":  {"", true},
		"prefix only":   {"s3c", true},
		"longer secret": {"s3cr3tXXXX", true},
	} {
		t.Run(name, func(t *testing.T) {
			next, hit := reached()
			rec := call(t, RequireWebhookSecret("s3cr3t", next), tc.header, tc.set)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("code = %d, want 401", rec.Code)
			}
			if *hit {
				t.Fatal("the handler ran for an unauthenticated caller")
			}
		})
	}
}

// An unset secret must close the door, not open it. A guard that passes everything
// through when it is not configured is worse than no guard: it looks protected in
// the router and is not, which is exactly how this API was left publicly writable
// once before.
func TestWebhookSecretFailsClosedWhenNotConfigured(t *testing.T) {
	for _, secret := range []string{"", "   "} {
		next, hit := reached()
		rec := call(t, RequireWebhookSecret(secret, next), "anything", true)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("code = %d, want 503", rec.Code)
		}
		if *hit {
			t.Fatal("an unconfigured guard let the request through")
		}
	}
}
