package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type ExerciseType struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profileId"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	Icon      *string   `json:"icon,omitempty"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Exercise struct {
	ID             string    `json:"id"`
	ProfileID      string    `json:"profileId"`
	ExerciseTypeID *string   `json:"exerciseTypeId,omitempty"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	MuscleGroup    *string   `json:"muscleGroup,omitempty"`
	Equipment      *string   `json:"equipment,omitempty"`
	Instructions   *string   `json:"instructions,omitempty"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Exercise Types

func (h *HealthHandlers) ListExerciseTypes(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")

	query := `SELECT id, profile_id, name, color, icon, is_active, created_at, updated_at FROM health.exercise_types`
	var args []interface{}

	if profileID != "" {
		query += " WHERE profile_id = $1"
		args = append(args, profileID)
	}
	query += " ORDER BY name ASC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var types []ExerciseType
	for rows.Next() {
		var t ExerciseType
		if err := rows.Scan(&t.ID, &t.ProfileID, &t.Name, &t.Color, &t.Icon, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		types = append(types, t)
	}

	if types == nil {
		types = []ExerciseType{}
	}
	respondJSON(w, http.StatusOK, types)
}

func (h *HealthHandlers) CreateExerciseType(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID string  `json:"profileId"`
		Name      string  `json:"name"`
		Color     *string `json:"color"`
		Icon      *string `json:"icon"`
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
	t := ExerciseType{
		ID:        uuid.New().String(),
		ProfileID: input.ProfileID,
		Name:      input.Name,
		Color:     input.Color,
		Icon:      input.Icon,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.exercise_types (id, profile_id, name, color, icon, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.ID, t.ProfileID, t.Name, t.Color, t.Icon, t.IsActive, t.CreatedAt, t.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, t)
}

func (h *HealthHandlers) UpdateExerciseType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input struct {
		Name  string  `json:"name"`
		Color *string `json:"color"`
		Icon  *string `json:"icon"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.db.Exec(`
		UPDATE health.exercise_types
		SET name = $1, color = $2, icon = $3, updated_at = $4
		WHERE id = $5
	`, input.Name, input.Color, input.Icon, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Exercise type not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *HealthHandlers) DeleteExerciseType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := h.db.Exec(`DELETE FROM health.exercise_types WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Exercise type not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Exercises

func (h *HealthHandlers) ListExercises(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	typeID := r.URL.Query().Get("exerciseTypeId")

	query := `SELECT id, profile_id, exercise_type_id, name, description, muscle_group, equipment, instructions, is_active, created_at, updated_at FROM health.exercises WHERE 1=1`
	var args []interface{}
	argNum := 1

	if profileID != "" {
		query += " AND profile_id = $" + string(rune('0'+argNum))
		args = append(args, profileID)
		argNum++
	}
	if typeID != "" {
		query += " AND exercise_type_id = $" + string(rune('0'+argNum))
		args = append(args, typeID)
		argNum++
	}
	query += " ORDER BY name ASC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var e Exercise
		if err := rows.Scan(&e.ID, &e.ProfileID, &e.ExerciseTypeID, &e.Name, &e.Description, &e.MuscleGroup, &e.Equipment, &e.Instructions, &e.IsActive, &e.CreatedAt, &e.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		exercises = append(exercises, e)
	}

	if exercises == nil {
		exercises = []Exercise{}
	}
	respondJSON(w, http.StatusOK, exercises)
}

func (h *HealthHandlers) CreateExercise(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ProfileID      string  `json:"profileId"`
		ExerciseTypeID *string `json:"exerciseTypeId"`
		Name           string  `json:"name"`
		Description    *string `json:"description"`
		MuscleGroup    *string `json:"muscleGroup"`
		Equipment      *string `json:"equipment"`
		Instructions   *string `json:"instructions"`
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
	e := Exercise{
		ID:             uuid.New().String(),
		ProfileID:      input.ProfileID,
		ExerciseTypeID: input.ExerciseTypeID,
		Name:           input.Name,
		Description:    input.Description,
		MuscleGroup:    input.MuscleGroup,
		Equipment:      input.Equipment,
		Instructions:   input.Instructions,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.exercises (id, profile_id, exercise_type_id, name, description, muscle_group, equipment, instructions, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, e.ID, e.ProfileID, e.ExerciseTypeID, e.Name, e.Description, e.MuscleGroup, e.Equipment, e.Instructions, e.IsActive, e.CreatedAt, e.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, e)
}

func (h *HealthHandlers) GetExercise(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var e Exercise
	err := h.db.QueryRow(`
		SELECT id, profile_id, exercise_type_id, name, description, muscle_group, equipment, instructions, is_active, created_at, updated_at
		FROM health.exercises WHERE id = $1
	`, id).Scan(&e.ID, &e.ProfileID, &e.ExerciseTypeID, &e.Name, &e.Description, &e.MuscleGroup, &e.Equipment, &e.Instructions, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Exercise not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, e)
}

func (h *HealthHandlers) UpdateExercise(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input struct {
		ExerciseTypeID *string `json:"exerciseTypeId"`
		Name           string  `json:"name"`
		Description    *string `json:"description"`
		MuscleGroup    *string `json:"muscleGroup"`
		Equipment      *string `json:"equipment"`
		Instructions   *string `json:"instructions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.db.Exec(`
		UPDATE health.exercises
		SET exercise_type_id = $1, name = $2, description = $3, muscle_group = $4, equipment = $5, instructions = $6, updated_at = $7
		WHERE id = $8
	`, input.ExerciseTypeID, input.Name, input.Description, input.MuscleGroup, input.Equipment, input.Instructions, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Exercise not found")
		return
	}

	h.GetExercise(w, r)
}

func (h *HealthHandlers) DeleteExercise(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := h.db.Exec(`DELETE FROM health.exercises WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Exercise not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
