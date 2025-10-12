package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/brunovieira/calendar-finances/internal/handlers"
	profileHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
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

	// Initialize Profile repository and use cases
	profileRepo := persistence.NewProfileRepository(db)
	createProfileUC := usecases.NewCreateProfileUseCase(profileRepo)
	listProfilesUC := usecases.NewListProfilesUseCase(profileRepo)
	getProfileUC := usecases.NewGetProfileUseCase(profileRepo)
	updateProfileUC := usecases.NewUpdateProfileUseCase(profileRepo)
	deleteProfileUC := usecases.NewDeleteProfileUseCase(profileRepo)

	// Initialize Profile handlers
	profileHandler := profileHandlers.NewProfileHandlers(
		createProfileUC,
		listProfilesUC,
		getProfileUC,
		updateProfileUC,
		deleteProfileUC,
	)

	// Initialize Bank Account repository and use cases
	bankAccountRepo := persistence.NewBankAccountRepository(db)
	createBankAccountUC := usecases.NewCreateBankAccountUseCase(bankAccountRepo)
	listBankAccountsUC := usecases.NewListBankAccountsUseCase(bankAccountRepo)
	getBankAccountUC := usecases.NewGetBankAccountUseCase(bankAccountRepo)
	updateBankAccountUC := usecases.NewUpdateBankAccountUseCase(bankAccountRepo)
	deleteBankAccountUC := usecases.NewDeleteBankAccountUseCase(bankAccountRepo)

	// Initialize Bank Account handlers
	bankAccountHandler := profileHandlers.NewBankAccountHandlers(
		createBankAccountUC,
		listBankAccountsUC,
		getBankAccountUC,
		updateBankAccountUC,
		deleteBankAccountUC,
	)

	// API v1 routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Profile routes
	apiRouter.HandleFunc("/profiles", profileHandler.List).Methods("GET")
	apiRouter.HandleFunc("/profiles", profileHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/profiles/{id}", profileHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/profiles/{id}", profileHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/profiles/{id}", profileHandler.Delete).Methods("DELETE")

	// Bank Account routes
	apiRouter.HandleFunc("/bank-accounts", bankAccountHandler.List).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts", bankAccountHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Delete).Methods("DELETE")

	// Accounts routes (placeholder)
	apiRouter.HandleFunc("/accounts", handlers.NotImplemented).Methods("GET", "POST")

	// Transactions routes (placeholder)
	apiRouter.HandleFunc("/transactions", handlers.NotImplemented).Methods("GET", "POST")

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3002", "http://localhost:3003"},
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
