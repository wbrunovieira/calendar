// Package middleware holds HTTP middleware shared by the API routes.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIToken requires a shared token on every request, mirroring calendar-core's ApiTokenGuard.
//
// Accepted forms: `Authorization: Bearer <token>` or `X-API-Key: <token>`.
//
// Exceptions:
//   - publicPaths (exact match, e.g. the health check) stay open;
//   - CORS preflights pass, since browsers never attach credentials to them;
//   - a blank expected token leaves the API open. That is deliberate: the middleware can ship
//     before the token is configured, and enforcement starts the moment it is set.
func APIToken(expected string, publicPaths ...string) func(http.Handler) http.Handler {
	expected = strings.TrimSpace(expected)
	public := make(map[string]struct{}, len(publicPaths))
	for _, p := range publicPaths {
		public[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" || isPreflight(r) {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := public[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			provided := extractToken(r)
			// ConstantTimeCompare returns 0 for different lengths without leaking where they differ.
			if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("WWW-Authenticate", `Bearer realm="calendar-health"`)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid or missing API token"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	if scheme, value, ok := strings.Cut(r.Header.Get("Authorization"), " "); ok && scheme == "Bearer" {
		return strings.TrimSpace(value)
	}
	return r.Header.Get("X-API-Key")
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}
