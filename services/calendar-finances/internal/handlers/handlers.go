package handlers

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthCheck returns the service health status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Calendar Finances API is healthy",
		Data: map[string]string{
			"service": "calendar-finances",
			"version": "1.0.0",
			"status":  "running",
		},
	})
}

// RootHandler returns basic API information
func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Calendar Finances API",
		Data: map[string]string{
			"service":     "calendar-finances",
			"version":     "1.0.0",
			"description": "Financial management service for calendar application",
			"docs":        "/api/v1/docs",
		},
	})
}

// NotImplemented is a placeholder for routes not yet implemented
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: "This endpoint is not yet implemented",
	})
}
