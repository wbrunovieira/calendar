package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/brunovieira/calendar-finances/internal/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Connected to PostgreSQL database")

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
	router.HandleFunc("/", handlers.RootHandler).Methods("GET")

	// API v1 routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Accounts routes (placeholder)
	apiRouter.HandleFunc("/accounts", handlers.NotImplemented).Methods("GET", "POST")

	// Transactions routes (placeholder)
	apiRouter.HandleFunc("/transactions", handlers.NotImplemented).Methods("GET", "POST")

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "3335"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsHandler.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 Calendar Finances API starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
