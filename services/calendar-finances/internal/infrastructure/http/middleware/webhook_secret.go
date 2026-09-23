package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// WebhookSecretHeader carries the shared secret an internal caller proves itself
// with. It is a header and not a query parameter so it stays out of access logs.
const WebhookSecretHeader = "x-webhook-secret"

// RequireWebhookSecret guards a route meant only for a known server-to-server
// caller.
//
// It FAILS CLOSED: with no secret configured the route answers 503 and runs
// nothing. The alternative — passing traffic through when unconfigured — would put
// a route in the router that looks guarded and is not, which is how this API ended
// up publicly writable once before.
func RequireWebhookSecret(secret string, next http.Handler) http.Handler {
	expected := strings.TrimSpace(secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expected == "" {
			http.Error(w, `{"error":"webhook secret is not configured on this server"}`,
				http.StatusServiceUnavailable)
			return
		}
		// Constant time so the comparison does not leak the secret one byte at a
		// time to a caller measuring how long the rejection took.
		presented := r.Header.Get(WebhookSecretHeader)
		if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
