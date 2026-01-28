package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Workout struct {
	ID              string     `json:"id"`
	ProfileID       string     `json:"profileId"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	WorkoutDate     string     `json:"workoutDate"`
	StartTime       *string    `json:"startTime,omitempty"`
	EndTime         *string    `json:"endTime,omitempty"`
	DurationMinutes *int       `json:"durationMinutes,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	Rating          *int       `json:"rating,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (h *HealthHandlers) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	query := `SELECT id, profile_id, name, description, workout_date, start_time, end_time, duration_minutes, notes, rating, status, created_at, updated_at FROM health.workouts WHERE 1=1`
	var args []interface{}
	argNum := 1

	if profileID != "" {
		query += " AND profile_id = $1"
		args = append(args, profileID)
		argNum++
	}
	if startDate != "" {
		query += " AND workout_date >= $" + string(rune('0'+argNum))
		args = append(args, startDate)
		argNum++
	}
	if endDate != "" {
		query += " AND workout_date <= $" + string(rune('0'+argNum))
		args = append(args, endDate)
		argNum++
	}
	query += " ORDER BY workout_date DESC, created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var workouts []Workout
	for rows.Next() {
		var wo Workout
		var workoutDate time.Time
		if err := rows.Scan(&wo.ID, &wo.ProfileID, &wo.Name, &wo.Description, &workoutDate, &wo.StartTime, &wo.EndTime, &wo.DurationMinutes, &wo.Notes, &wo.Rating, &wo.Status, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		wo.WorkoutDate = workoutDate.Format("2006-01-02")
		workouts = append(workouts, wo)
	}

	if workouts == nil {
		workouts = []Workout{}
	}
	respondJSON(w, http.StatusOK, workouts)
}

func (h *HealthHandlers) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID       string  `json:"profileId"`
		Name            string  `json:"name"`
		Description     *string `json:"description"`
		WorkoutDate     string  `json:"workoutDate"`
		StartTime       *string `json:"startTime"`
		EndTime         *string `json:"endTime"`
		DurationMinutes *int    `json:"durationMinutes"`
		Notes           *string `json:"notes"`
		Rating          *int    `json:"rating"`
		Status          string  `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ProfileID == "" || input.Name == "" || input.WorkoutDate == "" {
		respondError(w, http.StatusBadRequest, "profileId, name, and workoutDate are required")
		return
	}

	if input.Status == "" {
		input.Status = "COMPLETED"
	}

	now := time.Now()
	wo := Workout{
		ID:              uuid.New().String(),
		ProfileID:       input.ProfileID,
		Name:            input.Name,
		Description:     input.Description,
		WorkoutDate:     input.WorkoutDate,
		StartTime:       input.StartTime,
		EndTime:         input.EndTime,
		DurationMinutes: input.DurationMinutes,
		Notes:           input.Notes,
		Rating:          input.Rating,
		Status:          input.Status,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.workouts (id, profile_id, name, description, workout_date, start_time, end_time, duration_minutes, notes, rating, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, wo.ID, wo.ProfileID, wo.Name, wo.Description, wo.WorkoutDate, wo.StartTime, wo.EndTime, wo.DurationMinutes, wo.Notes, wo.Rating, wo.Status, wo.CreatedAt, wo.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, wo)
}

func (h *HealthHandlers) GetWorkout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var wo Workout
	var workoutDate time.Time
	err := h.db.QueryRow(`
		SELECT id, profile_id, name, description, workout_date, start_time, end_time, duration_minutes, notes, rating, status, created_at, updated_at
		FROM health.workouts WHERE id = $1
	`, id).Scan(&wo.ID, &wo.ProfileID, &wo.Name, &wo.Description, &workoutDate, &wo.StartTime, &wo.EndTime, &wo.DurationMinutes, &wo.Notes, &wo.Rating, &wo.Status, &wo.CreatedAt, &wo.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Workout not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	wo.WorkoutDate = workoutDate.Format("2006-01-02")
	respondJSON(w, http.StatusOK, wo)
}

func (h *HealthHandlers) UpdateWorkout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input struct {
		Name            string  `json:"name"`
		Description     *string `json:"description"`
		WorkoutDate     string  `json:"workoutDate"`
		StartTime       *string `json:"startTime"`
		EndTime         *string `json:"endTime"`
		DurationMinutes *int    `json:"durationMinutes"`
		Notes           *string `json:"notes"`
		Rating          *int    `json:"rating"`
		Status          string  `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.db.Exec(`
		UPDATE health.workouts
		SET name = $1, description = $2, workout_date = $3, start_time = $4, end_time = $5, duration_minutes = $6, notes = $7, rating = $8, status = $9, updated_at = $10
		WHERE id = $11
	`, input.Name, input.Description, input.WorkoutDate, input.StartTime, input.EndTime, input.DurationMinutes, input.Notes, input.Rating, input.Status, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Workout not found")
		return
	}

	h.GetWorkout(w, r)
}

func (h *HealthHandlers) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := h.db.Exec(`DELETE FROM health.workouts WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Workout not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
