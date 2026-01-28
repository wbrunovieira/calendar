package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterHealthRoutes registers all health-related routes
func RegisterHealthRoutes(router *mux.Router, db *sql.DB) {
	h := &HealthHandlers{db: db}

	// Health check
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")

	// Profiles
	router.HandleFunc("/profiles", h.ListProfiles).Methods("GET")
	router.HandleFunc("/profiles", h.CreateProfile).Methods("POST")
	router.HandleFunc("/profiles/{id}", h.GetProfile).Methods("GET")
	router.HandleFunc("/profiles/{id}", h.UpdateProfile).Methods("PUT")
	router.HandleFunc("/profiles/{id}", h.DeleteProfile).Methods("DELETE")

	// Exercise Types
	router.HandleFunc("/exercise-types", h.ListExerciseTypes).Methods("GET")
	router.HandleFunc("/exercise-types", h.CreateExerciseType).Methods("POST")
	router.HandleFunc("/exercise-types/{id}", h.UpdateExerciseType).Methods("PUT")
	router.HandleFunc("/exercise-types/{id}", h.DeleteExerciseType).Methods("DELETE")

	// Exercises
	router.HandleFunc("/exercises", h.ListExercises).Methods("GET")
	router.HandleFunc("/exercises", h.CreateExercise).Methods("POST")
	router.HandleFunc("/exercises/{id}", h.GetExercise).Methods("GET")
	router.HandleFunc("/exercises/{id}", h.UpdateExercise).Methods("PUT")
	router.HandleFunc("/exercises/{id}", h.DeleteExercise).Methods("DELETE")

	// Workouts
	router.HandleFunc("/workouts", h.ListWorkouts).Methods("GET")
	router.HandleFunc("/workouts", h.CreateWorkout).Methods("POST")
	router.HandleFunc("/workouts/{id}", h.GetWorkout).Methods("GET")
	router.HandleFunc("/workouts/{id}", h.UpdateWorkout).Methods("PUT")
	router.HandleFunc("/workouts/{id}", h.DeleteWorkout).Methods("DELETE")

	// Activities (Wim Hof, HIT, etc.)
	router.HandleFunc("/activities", h.ListActivities).Methods("GET")
	router.HandleFunc("/activities", h.CreateActivity).Methods("POST")
	router.HandleFunc("/activities/{id}", h.GetActivity).Methods("GET")
	router.HandleFunc("/activities/{id}", h.UpdateActivity).Methods("PUT")
	router.HandleFunc("/activities/{id}", h.DeleteActivity).Methods("DELETE")
}

type HealthHandlers struct {
	db *sql.DB
}

// HealthCheck returns a simple health check response
func (h *HealthHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "calendar-health"})
}

// Helper function to send JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Helper function to send error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
