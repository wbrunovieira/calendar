package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/brunovieira/calendar-health/internal/database"
	"github.com/brunovieira/calendar-health/internal/infrastructure/http/handlers"
	"github.com/brunovieira/calendar-health/internal/infrastructure/http/middleware"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://calendar:calendar123@postgres:5432/calendar_db?sslmode=disable"
	}

	// Connect to database
	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Initialize handlers
	handlers.RegisterHealthRoutes(api, db)

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3004", "https://health.wbdigitalsolutions.com", "https://health.app.localhost"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-API-Key"},
		AllowCredentials: true,
	})

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "3336"
	}

	// Token check sits inside CORS so preflights and error responses still carry CORS headers.
	apiToken := os.Getenv("API_TOKEN")
	if strings.TrimSpace(apiToken) == "" {
		log.Printf("API_TOKEN not set: API is open")
	}
	protected := middleware.APIToken(apiToken, "/api/v1/health")(router)

	// Start server
	handler := c.Handler(protected)
	log.Printf("Health service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
