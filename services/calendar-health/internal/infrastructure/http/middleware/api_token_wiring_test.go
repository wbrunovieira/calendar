package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Mirrors the wiring in cmd/api/main.go: CORS outside, token check inside, router innermost.
func wiredServer() http.Handler {
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	api.HandleFunc("/health", ok).Methods("GET")
	api.HandleFunc("/profiles", ok).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://health.wbdigitalsolutions.com"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
	})
	return c.Handler(APIToken(token, "/api/v1/health")(router))
}

const origin = "https://health.wbdigitalsolutions.com"

func TestWiringPreflightIsAnsweredWithoutToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "http://x/api/v1/profiles", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "authorization")
	rec := httptest.NewRecorder()
	wiredServer().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatal("preflight is missing Access-Control-Allow-Origin")
	}
}

func TestWiringRejectionCarriesCORSHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://x/api/v1/profiles", nil)
	req.Header.Set("Origin", origin)
	rec := httptest.NewRecorder()
	wiredServer().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatal("401 is missing Access-Control-Allow-Origin, so the browser can't read it")
	}
}

func TestWiringValidTokenReachesTheRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://x/api/v1/profiles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	wiredServer().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
