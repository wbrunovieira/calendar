package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// --- Structs ---

type StretchSequence struct {
	ID          string           `json:"id"`
	ProfileID   string           `json:"profileId"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	IsActive    bool             `json:"isActive"`
	Movements   []StretchMovement `json:"movements"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type StretchMovement struct {
	ID              string    `json:"id"`
	SequenceID      string    `json:"sequenceId"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	DurationSeconds int       `json:"durationSeconds"`
	OrderIndex      int       `json:"orderIndex"`
	ImageURL        *string   `json:"imageUrl,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type StretchSession struct {
	ID              string                   `json:"id"`
	ProfileID       string                   `json:"profileId"`
	SequenceID      *string                  `json:"sequenceId,omitempty"`
	SequenceName    *string                  `json:"sequenceName,omitempty"`
	SessionDate     string                   `json:"sessionDate"`
	DurationSeconds *int                     `json:"durationSeconds,omitempty"`
	Rating          *int                     `json:"rating,omitempty"`
	Notes           *string                  `json:"notes,omitempty"`
	Movements       []StretchSessionMovement `json:"movements"`
	CreatedAt       time.Time                `json:"createdAt"`
}

type StretchSessionMovement struct {
	ID                     string `json:"id"`
	SessionID              string `json:"sessionId"`
	MovementID             *string `json:"movementId,omitempty"`
	MovementName           string `json:"movementName"`
	PlannedDurationSeconds int    `json:"plannedDurationSeconds"`
	ActualDurationSeconds  *int   `json:"actualDurationSeconds,omitempty"`
	Reps                   int    `json:"reps"`
	Skipped                bool   `json:"skipped"`
	OrderIndex             int    `json:"orderIndex"`
}

// --- Sequences ---

func (h *HealthHandlers) ListStretchSequences(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")

	query := `SELECT id, profile_id, name, description, is_active, created_at, updated_at FROM health.stretch_sequences WHERE 1=1`
	var args []any

	if profileID != "" {
		if _, err := uuid.Parse(profileID); err != nil {
			respondJSON(w, http.StatusOK, []StretchSequence{})
			return
		}
		query += " AND profile_id = $1"
		args = append(args, profileID)
	}
	query += " ORDER BY name ASC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var sequences []StretchSequence
	for rows.Next() {
		var s StretchSequence
		if err := rows.Scan(&s.ID, &s.ProfileID, &s.Name, &s.Description, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.Movements = []StretchMovement{}
		sequences = append(sequences, s)
	}

	if sequences == nil {
		sequences = []StretchSequence{}
	}

	// Load movements for each sequence
	for i := range sequences {
		movements, err := h.loadMovements(sequences[i].ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sequences[i].Movements = movements
	}

	respondJSON(w, http.StatusOK, sequences)
}

func (h *HealthHandlers) CreateStretchSequence(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID   string  `json:"profileId"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ProfileID == "" || input.Name == "" {
		respondError(w, http.StatusBadRequest, "profileId and name are required")
		return
	}

	now := time.Now()
	s := StretchSequence{
		ID:          uuid.New().String(),
		ProfileID:   input.ProfileID,
		Name:        input.Name,
		Description: input.Description,
		IsActive:    true,
		Movements:   []StretchMovement{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.stretch_sequences (id, profile_id, name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, s.ID, s.ProfileID, s.Name, s.Description, s.IsActive, s.CreatedAt, s.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, s)
}

func (h *HealthHandlers) GetStretchSequence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Sequence not found")
		return
	}

	var s StretchSequence
	err := h.db.QueryRow(`
		SELECT id, profile_id, name, description, is_active, created_at, updated_at
		FROM health.stretch_sequences WHERE id = $1
	`, id).Scan(&s.ID, &s.ProfileID, &s.Name, &s.Description, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Sequence not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	movements, err := h.loadMovements(s.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Movements = movements

	respondJSON(w, http.StatusOK, s)
}

func (h *HealthHandlers) UpdateStretchSequence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Sequence not found")
		return
	}

	var input struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"isActive"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	result, err := h.db.Exec(`
		UPDATE health.stretch_sequences
		SET name = $1, description = $2, is_active = $3, updated_at = $4
		WHERE id = $5
	`, input.Name, input.Description, isActive, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Sequence not found")
		return
	}

	h.GetStretchSequence(w, r)
}

func (h *HealthHandlers) DeleteStretchSequence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Sequence not found")
		return
	}

	result, err := h.db.Exec(`DELETE FROM health.stretch_sequences WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Sequence not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Movements ---

func (h *HealthHandlers) AddStretchMovement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sequenceID := vars["id"]

	var input struct {
		Name            string  `json:"name"`
		Description     *string `json:"description"`
		DurationSeconds int     `json:"durationSeconds"`
		OrderIndex      int     `json:"orderIndex"`
		ImageURL        *string `json:"imageUrl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	if input.DurationSeconds == 0 {
		input.DurationSeconds = 30
	}

	now := time.Now()
	m := StretchMovement{
		ID:              uuid.New().String(),
		SequenceID:      sequenceID,
		Name:            input.Name,
		Description:     input.Description,
		DurationSeconds: input.DurationSeconds,
		OrderIndex:      input.OrderIndex,
		ImageURL:        input.ImageURL,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.stretch_movements (id, sequence_id, name, description, duration_seconds, order_index, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, m.ID, m.SequenceID, m.Name, m.Description, m.DurationSeconds, m.OrderIndex, m.ImageURL, m.CreatedAt, m.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, m)
}

func (h *HealthHandlers) UpdateStretchMovement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Movement not found")
		return
	}

	var input struct {
		Name            string  `json:"name"`
		Description     *string `json:"description"`
		DurationSeconds int     `json:"durationSeconds"`
		OrderIndex      int     `json:"orderIndex"`
		ImageURL        *string `json:"imageUrl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.db.Exec(`
		UPDATE health.stretch_movements
		SET name = $1, description = $2, duration_seconds = $3, order_index = $4, image_url = $5, updated_at = $6
		WHERE id = $7
	`, input.Name, input.Description, input.DurationSeconds, input.OrderIndex, input.ImageURL, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Movement not found")
		return
	}

	// Return updated movement
	var m StretchMovement
	err = h.db.QueryRow(`
		SELECT id, sequence_id, name, description, duration_seconds, order_index, image_url, created_at, updated_at
		FROM health.stretch_movements WHERE id = $1
	`, id).Scan(&m.ID, &m.SequenceID, &m.Name, &m.Description, &m.DurationSeconds, &m.OrderIndex, &m.ImageURL, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, m)
}

func (h *HealthHandlers) DeleteStretchMovement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Movement not found")
		return
	}

	result, err := h.db.Exec(`DELETE FROM health.stretch_movements WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Movement not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Sessions ---

func (h *HealthHandlers) ListStretchSessions(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	sequenceID := r.URL.Query().Get("sequenceId")
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	query := `SELECT s.id, s.profile_id, s.sequence_id, sq.name, s.session_date, s.duration_seconds, s.rating, s.notes, s.created_at
		FROM health.stretch_sessions s
		LEFT JOIN health.stretch_sequences sq ON sq.id = s.sequence_id
		WHERE 1=1`
	var args []any
	argNum := 1

	if profileID != "" {
		if _, err := uuid.Parse(profileID); err != nil {
			respondJSON(w, http.StatusOK, []StretchSession{})
			return
		}
		query += fmt.Sprintf(" AND s.profile_id = $%d", argNum)
		args = append(args, profileID)
		argNum++
	}
	if sequenceID != "" {
		query += fmt.Sprintf(" AND s.sequence_id = $%d", argNum)
		args = append(args, sequenceID)
		argNum++
	}
	if startDate != "" {
		query += fmt.Sprintf(" AND s.session_date >= $%d", argNum)
		args = append(args, startDate)
		argNum++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND s.session_date <= $%d", argNum)
		args = append(args, endDate)
		argNum++
	}
	query += " ORDER BY s.session_date DESC, s.created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var sessions []StretchSession
	for rows.Next() {
		var s StretchSession
		var sessionDate time.Time
		if err := rows.Scan(&s.ID, &s.ProfileID, &s.SequenceID, &s.SequenceName, &sessionDate, &s.DurationSeconds, &s.Rating, &s.Notes, &s.CreatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.SessionDate = sessionDate.Format("2006-01-02")
		s.Movements = []StretchSessionMovement{}
		sessions = append(sessions, s)
	}

	if sessions == nil {
		sessions = []StretchSession{}
	}
	respondJSON(w, http.StatusOK, sessions)
}

func (h *HealthHandlers) CreateStretchSession(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID       string `json:"profileId"`
		SequenceID      *string `json:"sequenceId"`
		SessionDate     string `json:"sessionDate"`
		DurationSeconds *int   `json:"durationSeconds"`
		Rating          *int   `json:"rating"`
		Notes           *string `json:"notes"`
		Movements       []struct {
			MovementID             *string `json:"movementId"`
			MovementName           string  `json:"movementName"`
			PlannedDurationSeconds int     `json:"plannedDurationSeconds"`
			ActualDurationSeconds  *int    `json:"actualDurationSeconds"`
			Reps                   int     `json:"reps"`
			Skipped                bool    `json:"skipped"`
			OrderIndex             int     `json:"orderIndex"`
		} `json:"movements"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ProfileID == "" || input.SessionDate == "" {
		respondError(w, http.StatusBadRequest, "profileId and sessionDate are required")
		return
	}

	sessionID := uuid.New().String()
	now := time.Now()

	_, err := h.db.Exec(`
		INSERT INTO health.stretch_sessions (id, profile_id, sequence_id, session_date, duration_seconds, rating, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, sessionID, input.ProfileID, input.SequenceID, input.SessionDate, input.DurationSeconds, input.Rating, input.Notes, now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Insert session movements
	for _, m := range input.Movements {
		movID := uuid.New().String()
		_, err := h.db.Exec(`
			INSERT INTO health.stretch_session_movements (id, session_id, movement_id, movement_name, planned_duration_seconds, actual_duration_seconds, reps, skipped, order_index, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, movID, sessionID, m.MovementID, m.MovementName, m.PlannedDurationSeconds, m.ActualDurationSeconds, m.Reps, m.Skipped, m.OrderIndex, now)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Return full session with movements
	session, err := h.loadSession(sessionID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, session)
}

func (h *HealthHandlers) GetStretchSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Session not found")
		return
	}

	session, err := h.loadSession(id)
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Session not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, session)
}

// --- Helpers ---

func (h *HealthHandlers) loadMovements(sequenceID string) ([]StretchMovement, error) {
	rows, err := h.db.Query(`
		SELECT id, sequence_id, name, description, duration_seconds, order_index, image_url, created_at, updated_at
		FROM health.stretch_movements WHERE sequence_id = $1 ORDER BY order_index ASC
	`, sequenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movements := []StretchMovement{}
	for rows.Next() {
		var m StretchMovement
		if err := rows.Scan(&m.ID, &m.SequenceID, &m.Name, &m.Description, &m.DurationSeconds, &m.OrderIndex, &m.ImageURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	return movements, nil
}

func (h *HealthHandlers) loadSession(id string) (*StretchSession, error) {
	var s StretchSession
	var sessionDate time.Time
	err := h.db.QueryRow(`
		SELECT s.id, s.profile_id, s.sequence_id, sq.name, s.session_date, s.duration_seconds, s.rating, s.notes, s.created_at
		FROM health.stretch_sessions s
		LEFT JOIN health.stretch_sequences sq ON sq.id = s.sequence_id
		WHERE s.id = $1
	`, id).Scan(&s.ID, &s.ProfileID, &s.SequenceID, &s.SequenceName, &sessionDate, &s.DurationSeconds, &s.Rating, &s.Notes, &s.CreatedAt)

	if err != nil {
		return nil, err
	}

	s.SessionDate = sessionDate.Format("2006-01-02")

	// Load session movements
	rows, err := h.db.Query(`
		SELECT id, session_id, movement_id, movement_name, planned_duration_seconds, actual_duration_seconds, reps, skipped, order_index
		FROM health.stretch_session_movements WHERE session_id = $1 ORDER BY order_index ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	s.Movements = []StretchSessionMovement{}
	for rows.Next() {
		var m StretchSessionMovement
		if err := rows.Scan(&m.ID, &m.SessionID, &m.MovementID, &m.MovementName, &m.PlannedDurationSeconds, &m.ActualDurationSeconds, &m.Reps, &m.Skipped, &m.OrderIndex); err != nil {
			return nil, err
		}
		s.Movements = append(s.Movements, m)
	}

	return &s, nil
}
