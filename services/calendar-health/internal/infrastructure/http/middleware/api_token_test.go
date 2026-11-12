package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const token = "s3cret-token"

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func serve(t *testing.T, expected string, req *http.Request) int {
	t.Helper()
	rec := httptest.NewRecorder()
	APIToken(expected, "/api/v1/health")(okHandler()).ServeHTTP(rec, req)
	return rec.Code
}

func TestAPIToken(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		path     string
		header   map[string]string
		want     int
	}{
		{"open when no token is configured", "", "/api/v1/profiles", nil, http.StatusOK},
		{"open when the configured token is blank", "   ", "/api/v1/profiles", nil, http.StatusOK},
		{"rejects a request without a token", token, "/api/v1/profiles", nil, http.StatusUnauthorized},
		{"accepts a valid bearer token", token, "/api/v1/profiles", map[string]string{"Authorization": "Bearer " + token}, http.StatusOK},
		{"accepts a valid X-API-Key", token, "/api/v1/profiles", map[string]string{"X-API-Key": token}, http.StatusOK},
		{"rejects a wrong bearer token", token, "/api/v1/profiles", map[string]string{"Authorization": "Bearer nope"}, http.StatusUnauthorized},
		{"rejects a token that is a prefix of the real one", token, "/api/v1/profiles", map[string]string{"Authorization": "Bearer s3cret"}, http.StatusUnauthorized},
		{"rejects a non-bearer scheme", token, "/api/v1/profiles", map[string]string{"Authorization": "Basic " + token}, http.StatusUnauthorized},
		{"rejects an empty bearer value", token, "/api/v1/profiles", map[string]string{"Authorization": "Bearer "}, http.StatusUnauthorized},
		{"leaves the health check public", token, "/api/v1/health", nil, http.StatusOK},
		{"does not treat health-like paths as public", token, "/api/v1/health/../profiles", nil, http.StatusUnauthorized},
		{"does not treat a health prefix as public", token, "/api/v1/healthz", nil, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://x"+tt.path, nil)
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}
			if got := serve(t, tt.expected, req); got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAPITokenAllowsCORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "http://x/api/v1/profiles", nil)
	req.Header.Set("Origin", "https://health.wbdigitalsolutions.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	if got := serve(t, token, req); got != http.StatusOK {
		t.Fatalf("preflight status = %d, want 200", got)
	}
}

func TestAPITokenRejectionBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://x/api/v1/profiles", nil)
	APIToken(token)(okHandler()).ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected a WWW-Authenticate header on 401")
	}
}
