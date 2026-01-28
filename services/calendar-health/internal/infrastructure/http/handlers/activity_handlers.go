package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Activity struct {
	ID              string          `json:"id"`
	ProfileID       string          `json:"profileId"`
	ExerciseID      *string         `json:"exerciseId,omitempty"`
	Name            string          `json:"name"`
	ActivityType    string          `json:"activityType"`
	ActivityDate    string          `json:"activityDate"`
	StartTime       *string         `json:"startTime,omitempty"`
	DurationMinutes *int            `json:"durationMinutes,omitempty"`
	Rounds          *int            `json:"rounds,omitempty"`
	Notes           *string         `json:"notes,omitempty"`
	Metrics         json.RawMessage `json:"metrics,omitempty"`
	Rating          *int            `json:"rating,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

func (h *HealthHandlers) ListActivities(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	activityType := r.URL.Query().Get("activityType")
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	query := `SELECT id, profile_id, exercise_id, name, activity_type, activity_date, start_time, duration_minutes, rounds, notes, metrics, rating, created_at, updated_at FROM health.activities WHERE 1=1`
	var args []interface{}
	argNum := 1

	if profileID != "" {
		query += " AND profile_id = $1"
		args = append(args, profileID)
		argNum++
	}
	if activityType != "" {
		query += " AND activity_type = $" + string(rune('0'+argNum))
		args = append(args, activityType)
		argNum++
	}
	if startDate != "" {
		query += " AND activity_date >= $" + string(rune('0'+argNum))
		args = append(args, startDate)
		argNum++
	}
	if endDate != "" {
		query += " AND activity_date <= $" + string(rune('0'+argNum))
		args = append(args, endDate)
		argNum++
	}
	query += " ORDER BY activity_date DESC, created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		var activityDate time.Time
		var metricsBytes []byte
		if err := rows.Scan(&a.ID, &a.ProfileID, &a.ExerciseID, &a.Name, &a.ActivityType, &activityDate, &a.StartTime, &a.DurationMinutes, &a.Rounds, &a.Notes, &metricsBytes, &a.Rating, &a.CreatedAt, &a.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.ActivityDate = activityDate.Format("2006-01-02")
		if metricsBytes != nil {
			a.Metrics = metricsBytes
		}
		activities = append(activities, a)
	}

	if activities == nil {
		activities = []Activity{}
	}
	respondJSON(w, http.StatusOK, activities)
}

func (h *HealthHandlers) CreateActivity(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID       string          `json:"profileId"`
		ExerciseID      *string         `json:"exerciseId"`
		Name            string          `json:"name"`
		ActivityType    string          `json:"activityType"`
		ActivityDate    string          `json:"activityDate"`
		StartTime       *string         `json:"startTime"`
		DurationMinutes *int            `json:"durationMinutes"`
		Rounds          *int            `json:"rounds"`
		Notes           *string         `json:"notes"`
		Metrics         json.RawMessage `json:"metrics"`
		Rating          *int            `json:"rating"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ProfileID == "" || input.Name == "" || input.ActivityType == "" || input.ActivityDate == "" {
		respondError(w, http.StatusBadRequest, "profileId, name, activityType, and activityDate are required")
		return
	}

	now := time.Now()
	a := Activity{
		ID:              uuid.New().String(),
		ProfileID:       input.ProfileID,
		ExerciseID:      input.ExerciseID,
		Name:            input.Name,
		ActivityType:    input.ActivityType,
		ActivityDate:    input.ActivityDate,
		StartTime:       input.StartTime,
		DurationMinutes: input.DurationMinutes,
		Rounds:          input.Rounds,
		Notes:           input.Notes,
		Metrics:         input.Metrics,
		Rating:          input.Rating,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.activities (id, profile_id, exercise_id, name, activity_type, activity_date, start_time, duration_minutes, rounds, notes, metrics, rating, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, a.ID, a.ProfileID, a.ExerciseID, a.Name, a.ActivityType, a.ActivityDate, a.StartTime, a.DurationMinutes, a.Rounds, a.Notes, a.Metrics, a.Rating, a.CreatedAt, a.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, a)
}

func (h *HealthHandlers) GetActivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var a Activity
	var activityDate time.Time
	var metricsBytes []byte
	err := h.db.QueryRow(`
		SELECT id, profile_id, exercise_id, name, activity_type, activity_date, start_time, duration_minutes, rounds, notes, metrics, rating, created_at, updated_at
		FROM health.activities WHERE id = $1
	`, id).Scan(&a.ID, &a.ProfileID, &a.ExerciseID, &a.Name, &a.ActivityType, &activityDate, &a.StartTime, &a.DurationMinutes, &a.Rounds, &a.Notes, &metricsBytes, &a.Rating, &a.CreatedAt, &a.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Activity not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.ActivityDate = activityDate.Format("2006-01-02")
	if metricsBytes != nil {
		a.Metrics = metricsBytes
	}
	respondJSON(w, http.StatusOK, a)
}

func (h *HealthHandlers) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input struct {
		ExerciseID      *string         `json:"exerciseId"`
		Name            string          `json:"name"`
		ActivityType    string          `json:"activityType"`
		ActivityDate    string          `json:"activityDate"`
		StartTime       *string         `json:"startTime"`
		DurationMinutes *int            `json:"durationMinutes"`
		Rounds          *int            `json:"rounds"`
		Notes           *string         `json:"notes"`
		Metrics         json.RawMessage `json:"metrics"`
		Rating          *int            `json:"rating"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.db.Exec(`
		UPDATE health.activities
		SET exercise_id = $1, name = $2, activity_type = $3, activity_date = $4, start_time = $5, duration_minutes = $6, rounds = $7, notes = $8, metrics = $9, rating = $10, updated_at = $11
		WHERE id = $12
	`, input.ExerciseID, input.Name, input.ActivityType, input.ActivityDate, input.StartTime, input.DurationMinutes, input.Rounds, input.Notes, input.Metrics, input.Rating, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Activity not found")
		return
	}

	h.GetActivity(w, r)
}

func (h *HealthHandlers) DeleteActivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := h.db.Exec(`DELETE FROM health.activities WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Activity not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
