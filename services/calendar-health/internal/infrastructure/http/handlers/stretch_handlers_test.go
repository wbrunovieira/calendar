package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func setupStretchTest(t *testing.T) (*HealthHandlers, string) {
	t.Helper()
	h := setupTestDB(t)
	profileID := createTestProfile(t, h)
	t.Cleanup(func() {
		h.db.Exec("DELETE FROM health.stretch_session_movements")
		h.db.Exec("DELETE FROM health.stretch_sessions")
		h.db.Exec("DELETE FROM health.stretch_movements")
		h.db.Exec("DELETE FROM health.stretch_sequences")
	})
	return h, profileID
}

// --- Sequences ---

func TestCreateStretchSequence(t *testing.T) {
	h, profileID := setupStretchTest(t)

	t.Run("creates sequence successfully", func(t *testing.T) {
		body := `{"profileId":"` + profileID + `","name":"Alongamento Matinal","description":"Rotina da manha"}`
		req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateStretchSequence(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchSequence
		json.NewDecoder(w.Body).Decode(&result)

		if result.ID == "" {
			t.Error("expected ID to be set")
		}
		if result.Name != "Alongamento Matinal" {
			t.Errorf("expected name 'Alongamento Matinal', got '%s'", result.Name)
		}
		if result.ProfileID != profileID {
			t.Errorf("expected profileId %s, got %s", profileID, result.ProfileID)
		}
		if !result.IsActive {
			t.Error("expected isActive to be true")
		}
	})

	t.Run("requires profileId and name", func(t *testing.T) {
		body := `{"description":"missing fields"}`
		req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateStretchSequence(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestListStretchSequences(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create 2 sequences
	for _, name := range []string{"Matinal", "Pos-Treino"} {
		body := `{"profileId":"` + profileID + `","name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateStretchSequence(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: failed to create sequence: %d %s", w.Code, w.Body.String())
		}
	}

	t.Run("lists sequences for profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sequences?profileId="+profileID, nil)
		w := httptest.NewRecorder()
		h.ListStretchSequences(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []StretchSequence
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 2 {
			t.Fatalf("expected 2 sequences, got %d", len(results))
		}
	})

	t.Run("returns empty array for unknown profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sequences?profileId=nonexistent", nil)
		w := httptest.NewRecorder()
		h.ListStretchSequences(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var results []StretchSequence
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 0 {
			t.Errorf("expected 0 sequences, got %d", len(results))
		}
	})
}

func TestGetStretchSequence(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence
	body := `{"profileId":"` + profileID + `","name":"Test Sequence"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	// Add a movement
	movBody := `{"name":"Hamstring Stretch","durationSeconds":45,"orderIndex":0}`
	req = httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(movBody))
	req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
	w = httptest.NewRecorder()
	h.AddStretchMovement(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: failed to add movement: %d %s", w.Code, w.Body.String())
	}

	t.Run("returns sequence with movements", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sequences/"+seq.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
		w := httptest.NewRecorder()
		h.GetStretchSequence(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchSequence
		json.NewDecoder(w.Body).Decode(&result)

		if result.Name != "Test Sequence" {
			t.Errorf("expected name 'Test Sequence', got '%s'", result.Name)
		}
		if len(result.Movements) != 1 {
			t.Fatalf("expected 1 movement, got %d", len(result.Movements))
		}
		if result.Movements[0].Name != "Hamstring Stretch" {
			t.Errorf("expected movement 'Hamstring Stretch', got '%s'", result.Movements[0].Name)
		}
		if result.Movements[0].DurationSeconds != 45 {
			t.Errorf("expected 45 seconds, got %d", result.Movements[0].DurationSeconds)
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sequences/nonexistent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.GetStretchSequence(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestUpdateStretchSequence(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence
	body := `{"profileId":"` + profileID + `","name":"Original Name"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	t.Run("updates sequence", func(t *testing.T) {
		body := `{"name":"Updated Name","description":"new desc","isActive":false}`
		req := httptest.NewRequest("PUT", "/stretch-sequences/"+seq.ID, bytes.NewBufferString(body))
		req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
		w := httptest.NewRecorder()
		h.UpdateStretchSequence(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchSequence
		json.NewDecoder(w.Body).Decode(&result)

		if result.Name != "Updated Name" {
			t.Errorf("expected 'Updated Name', got '%s'", result.Name)
		}
		if result.Description == nil || *result.Description != "new desc" {
			t.Errorf("expected description 'new desc', got %v", result.Description)
		}
		if result.IsActive {
			t.Error("expected isActive false")
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		body := `{"name":"whatever"}`
		req := httptest.NewRequest("PUT", "/stretch-sequences/nonexistent", bytes.NewBufferString(body))
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.UpdateStretchSequence(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDeleteStretchSequence(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence with a movement
	body := `{"profileId":"` + profileID + `","name":"To Delete"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	t.Run("deletes sequence and cascades", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/stretch-sequences/"+seq.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
		w := httptest.NewRecorder()
		h.DeleteStretchSequence(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/stretch-sequences/nonexistent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.DeleteStretchSequence(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

// --- Movements ---

func TestAddStretchMovement(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence
	body := `{"profileId":"` + profileID + `","name":"Sequence for Movements"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	t.Run("adds movement to sequence", func(t *testing.T) {
		body := `{"name":"Quad Stretch","description":"Puxe o pe","durationSeconds":30,"orderIndex":0}`
		req := httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(body))
		req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
		w := httptest.NewRecorder()
		h.AddStretchMovement(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchMovement
		json.NewDecoder(w.Body).Decode(&result)

		if result.Name != "Quad Stretch" {
			t.Errorf("expected 'Quad Stretch', got '%s'", result.Name)
		}
		if result.DurationSeconds != 30 {
			t.Errorf("expected 30s, got %d", result.DurationSeconds)
		}
		if result.SequenceID != seq.ID {
			t.Errorf("expected sequenceId %s, got %s", seq.ID, result.SequenceID)
		}
	})

	t.Run("requires name", func(t *testing.T) {
		body := `{"durationSeconds":30}`
		req := httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(body))
		req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
		w := httptest.NewRecorder()
		h.AddStretchMovement(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestUpdateStretchMovement(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence + movement
	seqBody := `{"profileId":"` + profileID + `","name":"Seq"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(seqBody))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	movBody := `{"name":"Original Move","durationSeconds":30,"orderIndex":0}`
	req = httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(movBody))
	req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
	w = httptest.NewRecorder()
	h.AddStretchMovement(w, req)
	var mov StretchMovement
	json.NewDecoder(w.Body).Decode(&mov)

	t.Run("updates movement", func(t *testing.T) {
		body := `{"name":"Updated Move","durationSeconds":60,"orderIndex":1}`
		req := httptest.NewRequest("PUT", "/stretch-movements/"+mov.ID, bytes.NewBufferString(body))
		req = mux.SetURLVars(req, map[string]string{"id": mov.ID})
		w := httptest.NewRecorder()
		h.UpdateStretchMovement(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchMovement
		json.NewDecoder(w.Body).Decode(&result)

		if result.Name != "Updated Move" {
			t.Errorf("expected 'Updated Move', got '%s'", result.Name)
		}
		if result.DurationSeconds != 60 {
			t.Errorf("expected 60s, got %d", result.DurationSeconds)
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		body := `{"name":"whatever","durationSeconds":30}`
		req := httptest.NewRequest("PUT", "/stretch-movements/nonexistent", bytes.NewBufferString(body))
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.UpdateStretchMovement(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDeleteStretchMovement(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence + movement
	seqBody := `{"profileId":"` + profileID + `","name":"Seq"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(seqBody))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	movBody := `{"name":"To Delete","durationSeconds":20,"orderIndex":0}`
	req = httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(movBody))
	req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
	w = httptest.NewRecorder()
	h.AddStretchMovement(w, req)
	var mov StretchMovement
	json.NewDecoder(w.Body).Decode(&mov)

	t.Run("deletes movement", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/stretch-movements/"+mov.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": mov.ID})
		w := httptest.NewRecorder()
		h.DeleteStretchMovement(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/stretch-movements/nonexistent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.DeleteStretchMovement(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

// --- Sessions ---

func TestCreateStretchSession(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence + movements
	seqBody := `{"profileId":"` + profileID + `","name":"Session Seq"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(seqBody))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	movBody := `{"name":"Move A","durationSeconds":30,"orderIndex":0}`
	req = httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(movBody))
	req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
	w = httptest.NewRecorder()
	h.AddStretchMovement(w, req)
	var movA StretchMovement
	json.NewDecoder(w.Body).Decode(&movA)

	t.Run("creates session with movements", func(t *testing.T) {
		sessionBody := `{
			"profileId":"` + profileID + `",
			"sequenceId":"` + seq.ID + `",
			"sessionDate":"2026-02-19",
			"durationSeconds":120,
			"rating":4,
			"notes":"Felt great",
			"movements":[
				{"movementId":"` + movA.ID + `","movementName":"Move A","plannedDurationSeconds":30,"actualDurationSeconds":35,"reps":1,"skipped":false,"orderIndex":0}
			]
		}`
		req := httptest.NewRequest("POST", "/stretch-sessions", bytes.NewBufferString(sessionBody))
		w := httptest.NewRecorder()
		h.CreateStretchSession(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchSession
		json.NewDecoder(w.Body).Decode(&result)

		if result.ID == "" {
			t.Error("expected ID to be set")
		}
		if result.ProfileID != profileID {
			t.Errorf("expected profileId %s, got %s", profileID, result.ProfileID)
		}
		if result.SequenceID == nil || *result.SequenceID != seq.ID {
			t.Errorf("expected sequenceId %s, got %v", seq.ID, result.SequenceID)
		}
		if result.SessionDate != "2026-02-19" {
			t.Errorf("expected date 2026-02-19, got %s", result.SessionDate)
		}
		if result.DurationSeconds == nil || *result.DurationSeconds != 120 {
			t.Errorf("expected 120s, got %v", result.DurationSeconds)
		}
		if result.Rating == nil || *result.Rating != 4 {
			t.Errorf("expected rating 4, got %v", result.Rating)
		}
		if len(result.Movements) != 1 {
			t.Fatalf("expected 1 movement, got %d", len(result.Movements))
		}
		if result.Movements[0].ActualDurationSeconds == nil || *result.Movements[0].ActualDurationSeconds != 35 {
			t.Errorf("expected actual 35s, got %v", result.Movements[0].ActualDurationSeconds)
		}
	})

	t.Run("requires profileId and sessionDate", func(t *testing.T) {
		body := `{"durationSeconds":100}`
		req := httptest.NewRequest("POST", "/stretch-sessions", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateStretchSession(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestListStretchSessions(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence
	seqBody := `{"profileId":"` + profileID + `","name":"List Seq"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(seqBody))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	// Create 2 sessions
	for _, date := range []string{"2026-02-18", "2026-02-19"} {
		body := `{"profileId":"` + profileID + `","sequenceId":"` + seq.ID + `","sessionDate":"` + date + `","durationSeconds":100}`
		req := httptest.NewRequest("POST", "/stretch-sessions", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.CreateStretchSession(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: failed to create session: %d %s", w.Code, w.Body.String())
		}
	}

	t.Run("lists sessions for profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sessions?profileId="+profileID, nil)
		w := httptest.NewRecorder()
		h.ListStretchSessions(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var results []StretchSession
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(results))
		}
		// Most recent first
		if results[0].SessionDate != "2026-02-19" {
			t.Errorf("expected first session 2026-02-19, got %s", results[0].SessionDate)
		}
	})

	t.Run("filters by sequenceId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sessions?profileId="+profileID+"&sequenceId="+seq.ID, nil)
		w := httptest.NewRecorder()
		h.ListStretchSessions(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var results []StretchSession
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(results))
		}
	})

	t.Run("returns empty array for unknown profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sessions?profileId=nonexistent", nil)
		w := httptest.NewRecorder()
		h.ListStretchSessions(w, req)

		var results []StretchSession
		json.NewDecoder(w.Body).Decode(&results)
		if len(results) != 0 {
			t.Errorf("expected 0, got %d", len(results))
		}
	})
}

func TestGetStretchSession(t *testing.T) {
	h, profileID := setupStretchTest(t)

	// Create sequence + movement
	seqBody := `{"profileId":"` + profileID + `","name":"Get Session Seq"}`
	req := httptest.NewRequest("POST", "/stretch-sequences", bytes.NewBufferString(seqBody))
	w := httptest.NewRecorder()
	h.CreateStretchSequence(w, req)
	var seq StretchSequence
	json.NewDecoder(w.Body).Decode(&seq)

	movBody := `{"name":"Stretch X","durationSeconds":40,"orderIndex":0}`
	req = httptest.NewRequest("POST", "/stretch-sequences/"+seq.ID+"/movements", bytes.NewBufferString(movBody))
	req = mux.SetURLVars(req, map[string]string{"id": seq.ID})
	w = httptest.NewRecorder()
	h.AddStretchMovement(w, req)
	var mov StretchMovement
	json.NewDecoder(w.Body).Decode(&mov)

	// Create session with movement
	sessionBody := `{
		"profileId":"` + profileID + `",
		"sequenceId":"` + seq.ID + `",
		"sessionDate":"2026-02-19",
		"durationSeconds":50,
		"movements":[
			{"movementId":"` + mov.ID + `","movementName":"Stretch X","plannedDurationSeconds":40,"actualDurationSeconds":42,"reps":2,"skipped":false,"orderIndex":0}
		]
	}`
	req = httptest.NewRequest("POST", "/stretch-sessions", bytes.NewBufferString(sessionBody))
	w = httptest.NewRecorder()
	h.CreateStretchSession(w, req)
	var session StretchSession
	json.NewDecoder(w.Body).Decode(&session)

	t.Run("returns session with movements", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sessions/"+session.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": session.ID})
		w := httptest.NewRecorder()
		h.GetStretchSession(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result StretchSession
		json.NewDecoder(w.Body).Decode(&result)

		if result.SessionDate != "2026-02-19" {
			t.Errorf("expected date 2026-02-19, got %s", result.SessionDate)
		}
		if len(result.Movements) != 1 {
			t.Fatalf("expected 1 movement, got %d", len(result.Movements))
		}
		if result.Movements[0].Reps != 2 {
			t.Errorf("expected 2 reps, got %d", result.Movements[0].Reps)
		}
		if result.Movements[0].MovementName != "Stretch X" {
			t.Errorf("expected 'Stretch X', got '%s'", result.Movements[0].MovementName)
		}
	})

	t.Run("returns 404 for nonexistent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stretch-sessions/nonexistent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
		w := httptest.NewRecorder()
		h.GetStretchSession(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}
