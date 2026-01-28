package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Profile struct {
	ID         string    `json:"id"`
	CalendarID string    `json:"calendarId"`
	Name       string    `json:"name"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// ListProfiles returns all profiles
func (h *HealthHandlers) ListProfiles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, calendar_id, name, is_active, created_at, updated_at
		FROM health.profiles
		ORDER BY created_at ASC
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.CalendarID, &p.Name, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		profiles = append(profiles, p)
	}

	if profiles == nil {
		profiles = []Profile{}
	}

	respondJSON(w, http.StatusOK, profiles)
}

// CreateProfile creates a new profile
func (h *HealthHandlers) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CalendarID string `json:"calendarId"`
		Name       string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.CalendarID == "" || input.Name == "" {
		respondError(w, http.StatusBadRequest, "calendarId and name are required")
		return
	}

	now := time.Now()
	p := Profile{
		ID:         uuid.New().String(),
		CalendarID: input.CalendarID,
		Name:       input.Name,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err := h.db.Exec(`
		INSERT INTO health.profiles (id, calendar_id, name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.ID, p.CalendarID, p.Name, p.IsActive, p.CreatedAt, p.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, p)
}

// GetProfile returns a single profile by ID
func (h *HealthHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var p Profile
	err := h.db.QueryRow(`
		SELECT id, calendar_id, name, is_active, created_at, updated_at
		FROM health.profiles
		WHERE id = $1
	`, id).Scan(&p.ID, &p.CalendarID, &p.Name, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Profile not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, p)
}

// UpdateProfile updates an existing profile
func (h *HealthHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"isActive"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	result, err := h.db.Exec(`
		UPDATE health.profiles
		SET name = $1, is_active = $2, updated_at = $3
		WHERE id = $4
	`, input.Name, isActive, time.Now(), id)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Profile not found")
		return
	}

	// Fetch and return updated profile
	h.GetProfile(w, r)
}

// DeleteProfile deletes a profile
func (h *HealthHandlers) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := h.db.Exec(`DELETE FROM health.profiles WHERE id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Profile not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
