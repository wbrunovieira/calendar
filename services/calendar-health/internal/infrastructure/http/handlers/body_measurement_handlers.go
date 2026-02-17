package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type BodyMeasurement struct {
	ID         string   `json:"id"`
	ProfileID  string   `json:"profileId"`
	MeasuredAt string   `json:"measuredAt"`
	Weight     *float64 `json:"weight,omitempty"`
	BodyFatPct *float64 `json:"bodyFatPct,omitempty"`
	Notes      *string  `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (h *HealthHandlers) CreateBodyMeasurement(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID  string   `json:"profileId"`
		MeasuredAt string   `json:"measuredAt"`
		Weight     *float64 `json:"weight"`
		BodyFatPct *float64 `json:"bodyFatPct"`
		Notes      *string  `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ProfileID == "" || input.MeasuredAt == "" {
		respondError(w, http.StatusBadRequest, "profileId and measuredAt are required")
		return
	}

	m := BodyMeasurement{
		ID:         uuid.New().String(),
		ProfileID:  input.ProfileID,
		MeasuredAt: input.MeasuredAt,
		Weight:     input.Weight,
		BodyFatPct: input.BodyFatPct,
		Notes:      input.Notes,
		CreatedAt:  time.Now(),
	}

	_, err := h.db.Exec(`
		INSERT INTO health.body_measurements (id, profile_id, measured_at, weight, body_fat_pct, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.ProfileID, m.MeasuredAt, m.Weight, m.BodyFatPct, m.Notes, m.CreatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, m)
}

func (h *HealthHandlers) ListBodyMeasurements(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")

	query := `SELECT id, profile_id, measured_at, weight, body_fat_pct, notes, created_at FROM health.body_measurements WHERE 1=1`
	var args []interface{}

	if profileID != "" {
		if _, err := uuid.Parse(profileID); err != nil {
			respondJSON(w, http.StatusOK, []BodyMeasurement{})
			return
		}
		query += " AND profile_id = $1"
		args = append(args, profileID)
	}
	query += " ORDER BY measured_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var measurements []BodyMeasurement
	for rows.Next() {
		var m BodyMeasurement
		var measuredAt time.Time
		if err := rows.Scan(&m.ID, &m.ProfileID, &measuredAt, &m.Weight, &m.BodyFatPct, &m.Notes, &m.CreatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		m.MeasuredAt = measuredAt.Format("2006-01-02")
		measurements = append(measurements, m)
	}

	if measurements == nil {
		measurements = []BodyMeasurement{}
	}
	respondJSON(w, http.StatusOK, measurements)
}

func (h *HealthHandlers) DeleteBodyMeasurement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "Body measurement not found")
		return
	}

	result, err := h.db.Exec(`DELETE FROM health.body_measurements WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Body measurement not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
