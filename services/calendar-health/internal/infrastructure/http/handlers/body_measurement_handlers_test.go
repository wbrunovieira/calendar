package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"

	"github.com/brunovieira/calendar-health/internal/database"
)

func setupTestDB(t *testing.T) *HealthHandlers {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://calendar:calendar123@localhost:5433/calendar_db?sslmode=disable"
	}
	db, err := database.Connect(dbURL)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM health.body_measurements")
		db.Exec("DELETE FROM health.profiles WHERE name LIKE 'test-%'")
		db.Close()
	})
	return &HealthHandlers{db: db}
}

func createTestProfile(t *testing.T, h *HealthHandlers) string {
	t.Helper()
	body := `{"calendarId":"test-cal-` + t.Name() + `","name":"test-profile"}`
	req := httptest.NewRequest("POST", "/profiles", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateProfile(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create test profile: %d %s", w.Code, w.Body.String())
	}
	var p Profile
	json.NewDecoder(w.Body).Decode(&p)
	return p.ID
}

func TestCreateBodyMeasurement(t *testing.T) {
	h := setupTestDB(t)
	profileID := createTestProfile(t, h)

	t.Run("creates measurement with weight", func(t *testing.T) {
		body := `{"profileId":"` + profileID + `","measuredAt":"2026-02-17","weight":82.5}`
		req := httptest.NewRequest("POST", "/body-measurements", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateBodyMeasurement(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var result BodyMeasurement
		json.NewDecoder(w.Body).Decode(&result)

		if result.ID == "" {
			t.Error("expected ID to be set")
		}
		if result.ProfileID != profileID {
			t.Errorf("expected profileId %s, got %s", profileID, result.ProfileID)
		}
		if result.Weight == nil || *result.Weight != 82.5 {
			t.Errorf("expected weight 82.5, got %v", result.Weight)
		}
		if result.MeasuredAt != "2026-02-17" {
			t.Errorf("expected measuredAt 2026-02-17, got %s", result.MeasuredAt)
		}
	})

	t.Run("requires profileId and measuredAt", func(t *testing.T) {
		body := `{"weight":80}`
		req := httptest.NewRequest("POST", "/body-measurements", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateBodyMeasurement(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("creates measurement with body fat", func(t *testing.T) {
		body := `{"profileId":"` + profileID + `","measuredAt":"2026-02-16","weight":83.0,"bodyFatPct":15.5,"notes":"morning"}`
		req := httptest.NewRequest("POST", "/body-measurements", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateBodyMeasurement(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var result BodyMeasurement
		json.NewDecoder(w.Body).Decode(&result)

		if result.BodyFatPct == nil || *result.BodyFatPct != 15.5 {
			t.Errorf("expected bodyFatPct 15.5, got %v", result.BodyFatPct)
		}
		if result.Notes == nil || *result.Notes != "morning" {
			t.Errorf("expected notes 'morning', got %v", result.Notes)
		}
	})
}

func TestListBodyMeasurements(t *testing.T) {
	h := setupTestDB(t)
	profileID := createTestProfile(t, h)

	// Create 3 measurements
	dates := []string{"2026-02-15", "2026-02-16", "2026-02-17"}
	weights := []float64{83.0, 82.5, 82.0}
	for i, d := range dates {
		body := `{"profileId":"` + profileID + `","measuredAt":"` + d + `","weight":` + jsonFloat(weights[i]) + `}`
		req := httptest.NewRequest("POST", "/body-measurements", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateBodyMeasurement(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: failed to create measurement: %d", w.Code)
		}
	}

	t.Run("lists measurements ordered by date desc", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/body-measurements?profileId="+profileID, nil)
		w := httptest.NewRecorder()
		h.ListBodyMeasurements(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []BodyMeasurement
		json.NewDecoder(w.Body).Decode(&results)

		if len(results) != 3 {
			t.Fatalf("expected 3 measurements, got %d", len(results))
		}

		// Should be ordered by date DESC (most recent first)
		if results[0].MeasuredAt != "2026-02-17" {
			t.Errorf("expected first result date 2026-02-17, got %s", results[0].MeasuredAt)
		}
		if results[2].MeasuredAt != "2026-02-15" {
			t.Errorf("expected last result date 2026-02-15, got %s", results[2].MeasuredAt)
		}
	})

	t.Run("returns empty array when no measurements", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/body-measurements?profileId=nonexistent-id", nil)
		w := httptest.NewRecorder()
		h.ListBodyMeasurements(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var results []BodyMeasurement
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 0 {
			t.Errorf("expected 0 measurements, got %d", len(results))
		}
	})
}

func TestDeleteBodyMeasurement(t *testing.T) {
	h := setupTestDB(t)
	profileID := createTestProfile(t, h)

	// Create a measurement
	body := `{"profileId":"` + profileID + `","measuredAt":"2026-02-17","weight":82.0}`
	req := httptest.NewRequest("POST", "/body-measurements", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateBodyMeasurement(w, req)
	var created BodyMeasurement
	json.NewDecoder(w.Body).Decode(&created)

	t.Run("deletes measurement", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/body-measurements/"+created.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": created.ID})
		w := httptest.NewRecorder()
		h.DeleteBodyMeasurement(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/body-measurements/nonexistent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.DeleteBodyMeasurement(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func jsonFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}
