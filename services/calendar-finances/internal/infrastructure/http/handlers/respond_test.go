package handlers

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A body that cannot be encoded must NOT come back as 200.
//
// Streaming the encoder straight to the ResponseWriter commits the 200 before
// knowing the payload is encodable, and a failure then leaves a 200 with an
// EMPTY body. The production health-check cron reads status codes, so that is a
// green light over nothing — the one failure mode these endpoints exist to
// prevent.
func TestRespondJSONRefusesToAnswer200WithABodyItCouldNotBuild(t *testing.T) {
	rec := httptest.NewRecorder()

	respondJSON(rec, map[string]any{"data": math.NaN()})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 — the body was never built", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) == "" {
		t.Error("a failed encode answered with an empty body and no reason")
	}
}

func TestRespondJSONWritesTheBodyItPromised(t *testing.T) {
	rec := httptest.NewRecorder()

	respondJSON(rec, map[string]any{"data": map[string]float64{"saldo": 718.36}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"saldo":718.36`) {
		t.Errorf("body = %q", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
}
