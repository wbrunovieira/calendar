package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/brunovieira/calendar-finances/internal/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/binance"
	httpHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
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
	profileHandler := httpHandlers.NewProfileHandlers(
		createProfileUC,
		listProfilesUC,
		getProfileUC,
		updateProfileUC,
		deleteProfileUC,
	)

	// Initialize repositories
	bankAccountRepo := persistence.NewBankAccountRepository(db)
	categoryRepo := persistence.NewCategoryRepository(db)
	invoiceRepo := persistence.NewInvoiceRepository(db)
	transactionRepo := persistence.NewTransactionRepository(db)

	// Initialize Bank Account use cases and handlers
	createBankAccountUC := usecases.NewCreateBankAccountUseCase(bankAccountRepo)
	listBankAccountsUC := usecases.NewListBankAccountsUseCase(bankAccountRepo)
	getBankAccountUC := usecases.NewGetBankAccountUseCase(bankAccountRepo)
	updateBankAccountUC := usecases.NewUpdateBankAccountUseCase(bankAccountRepo)
	deleteBankAccountUC := usecases.NewDeleteBankAccountUseCase(bankAccountRepo)
	reorderBankAccountsUC := usecases.NewReorderBankAccountsUseCase(bankAccountRepo)
	recalculateBalanceUC := usecases.NewRecalculateBalanceUseCase(bankAccountRepo, transactionRepo)
	bankAccountHandler := httpHandlers.NewBankAccountHandlers(
		createBankAccountUC,
		listBankAccountsUC,
		getBankAccountUC,
		updateBankAccountUC,
		deleteBankAccountUC,
		reorderBankAccountsUC,
		recalculateBalanceUC,
	)

	// Initialize Category use cases and handlers
	createCategoryUC := usecases.NewCreateCategoryUseCase(profileRepo, categoryRepo)
	listCategoriesUC := usecases.NewListCategoriesUseCase(categoryRepo)
	updateCategoryUC := usecases.NewUpdateCategoryUseCase(categoryRepo)
	deleteCategoryUC := usecases.NewDeleteCategoryUseCase(categoryRepo)
	categoryHandler := httpHandlers.NewCategoryHandlers(
		createCategoryUC,
		listCategoriesUC,
		updateCategoryUC,
		deleteCategoryUC,
	)

	// Initialize Transaction use cases and handlers
	createTransactionUC := usecases.NewCreateTransactionUseCase(profileRepo, bankAccountRepo, categoryRepo, transactionRepo, invoiceRepo)
	listTransactionsUC := usecases.NewListTransactionsUseCase(transactionRepo)
	getTransactionUC := usecases.NewGetTransactionUseCase(transactionRepo)
	updateTransactionUC := usecases.NewUpdateTransactionUseCase(bankAccountRepo, categoryRepo, transactionRepo, invoiceRepo)
	updateTransactionStatusUC := usecases.NewUpdateTransactionStatusUseCase(transactionRepo, bankAccountRepo)
	deleteTransactionUC := usecases.NewDeleteTransactionUseCase(transactionRepo, bankAccountRepo)
	dailyBalancesUC := usecases.NewGetDailyBalancesUseCase(transactionRepo, bankAccountRepo)
	transactionHandler := httpHandlers.NewTransactionHandlers(
		createTransactionUC,
		listTransactionsUC,
		getTransactionUC,
		updateTransactionUC,
		updateTransactionStatusUC,
		deleteTransactionUC,
	)
	transactionHandler.SetDailyBalancesUseCase(dailyBalancesUC)

	// Initialize Invoice use cases and handlers
	createInvoiceUC := usecases.NewCreateInvoiceUseCase(invoiceRepo, bankAccountRepo)
	listInvoicesUC := usecases.NewListInvoicesUseCase(invoiceRepo, bankAccountRepo, transactionRepo)
	getInvoiceUC := usecases.NewGetInvoiceUseCase(invoiceRepo, transactionRepo)
	getCurrentInvoiceUC := usecases.NewGetCurrentInvoiceUseCase(invoiceRepo, bankAccountRepo, transactionRepo)
	closeInvoiceUC := usecases.NewCloseInvoiceUseCase(invoiceRepo)
	// PayInvoiceUseCaseV2: Creates payment transaction on linked checking account when invoice is paid
	payInvoiceUC := usecases.NewPayInvoiceUseCaseV2(invoiceRepo, bankAccountRepo, transactionRepo)
	addAmountToInvoiceUC := usecases.NewAddAmountToInvoiceUseCase(invoiceRepo)
	recalculateInvoiceUC := usecases.NewRecalculateInvoiceAmountUseCase(invoiceRepo, transactionRepo)
	updateInvoiceUC := usecases.NewUpdateInvoiceUseCase(invoiceRepo)
	invoiceHandler := httpHandlers.NewInvoiceHandlers(
		createInvoiceUC,
		listInvoicesUC,
		getInvoiceUC,
		getCurrentInvoiceUC,
		closeInvoiceUC,
		payInvoiceUC,
		addAmountToInvoiceUC,
		recalculateInvoiceUC,
	)
	invoiceHandler.SetUpdateUseCase(updateInvoiceUC)
	autoCloseInvoicesUC := usecases.NewAutoCloseInvoicesUseCase(invoiceRepo)
	invoiceHandler.SetAutoCloseUseCase(autoCloseInvoicesUC)

	recurringRepo := persistence.NewRecurringTransactionRepository(db)
	recurringService := usecases.NewRecurringTransactionsService(recurringRepo)
	processRecurringUC := usecases.NewProcessRecurringTransactionsUseCase(recurringRepo, createTransactionUC)
	recurringHandler := httpHandlers.NewRecurringTransactionHandlers(recurringService, processRecurringUC)

	budgetRepo := persistence.NewBudgetTargetRepository(db)
	budgetService := usecases.NewBudgetTargetsService(budgetRepo, transactionRepo, categoryRepo)
	budgetHandler := httpHandlers.NewBudgetHandlers(budgetService)

	goalRepo := persistence.NewGoalRepository(db)
	goalService := usecases.NewGoalsService(goalRepo)
	goalHandler := httpHandlers.NewGoalHandlers(goalService)

	// Initialize Binance client and crypto sync
	cryptoPurchaseRepo := persistence.NewCryptoPurchaseRepository(db)
	binanceKey := os.Getenv("KEY_BINANCE")
	binanceSecret := os.Getenv("SECRET_BINANCE")
	var cryptoHandler *httpHandlers.CryptoHandlers
	var cryptoPurchaseHandler *httpHandlers.CryptoPurchaseHandlers
	var binanceClient *binance.Client
	var syncTradesUC *usecases.SyncTradesUseCase
	if binanceKey != "" && binanceSecret != "" {
		binanceClient = binance.NewClient(binanceKey, binanceSecret)
		cryptoSyncUC := usecases.NewCryptoSyncUseCase(binanceClient, bankAccountRepo)
		cryptoHandler = httpHandlers.NewCryptoHandlers(cryptoSyncUC)
		cryptoPurchaseHandler = httpHandlers.NewCryptoPurchaseHandlers(cryptoPurchaseRepo, binanceClient)

		// Initialize trade sync
		syncTradesUC = usecases.NewSyncTradesUseCase(
			binanceClient, bankAccountRepo, transactionRepo, cryptoPurchaseRepo,
			[]string{"SOLBRL", "ETHBRL", "BTCBRL", "USDCBRL"},
		)
		cryptoHandler.SetSyncTradesUseCase(syncTradesUC)

		log.Println("✓ Binance API integration enabled")
	} else {
		cryptoPurchaseHandler = httpHandlers.NewCryptoPurchaseHandlers(cryptoPurchaseRepo, nil)
	}

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
	apiRouter.HandleFunc("/bank-accounts/reorder", bankAccountHandler.Reorder).Methods("PUT")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Delete).Methods("DELETE")
	apiRouter.HandleFunc("/bank-accounts/{id}/recalculate-balance", bankAccountHandler.RecalculateBalance).Methods("POST")

	// Accounts routes (placeholder)
	apiRouter.HandleFunc("/accounts", handlers.NotImplemented).Methods("GET", "POST")

	// Category routes
	apiRouter.HandleFunc("/categories", categoryHandler.List).Methods("GET")
	apiRouter.HandleFunc("/categories", categoryHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/categories/{id}", categoryHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/categories/{id}", categoryHandler.Delete).Methods("DELETE")

	// Transaction routes
	apiRouter.HandleFunc("/transactions/daily-balances", transactionHandler.DailyBalances).Methods("GET")
	apiRouter.HandleFunc("/transactions", transactionHandler.List).Methods("GET")
	apiRouter.HandleFunc("/transactions", transactionHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/transactions/{id}", transactionHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/transactions/{id}", transactionHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/transactions/{id}/status", transactionHandler.UpdateStatus).Methods("PUT")
	apiRouter.HandleFunc("/recurring-transactions", recurringHandler.List).Methods("GET")
	apiRouter.HandleFunc("/recurring-transactions", recurringHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/recurring-transactions/process", recurringHandler.Process).Methods("POST")
	apiRouter.HandleFunc("/recurring-transactions/{id}", recurringHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/recurring-transactions/{id}/status", recurringHandler.UpdateStatus).Methods("PATCH")
	apiRouter.HandleFunc("/recurring-transactions/{id}", recurringHandler.Delete).Methods("DELETE")

	apiRouter.HandleFunc("/budgets/summary", budgetHandler.Summary).Methods("GET")
	apiRouter.HandleFunc("/budgets", budgetHandler.List).Methods("GET")
	apiRouter.HandleFunc("/budgets", budgetHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/budgets/{id}", budgetHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/budgets/{id}", budgetHandler.Delete).Methods("DELETE")

	// Goal routes
	apiRouter.HandleFunc("/goals", goalHandler.List).Methods("GET")
	apiRouter.HandleFunc("/goals", goalHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/goals/{id}", goalHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/goals/{id}", goalHandler.Delete).Methods("DELETE")
	apiRouter.HandleFunc("/goals/{id}/add-amount", goalHandler.AddAmount).Methods("POST")
	apiRouter.HandleFunc("/goals/{id}/status", goalHandler.UpdateStatus).Methods("PATCH")

	apiRouter.HandleFunc("/transactions/{id}", transactionHandler.Delete).Methods("DELETE")

	// Invoice routes
	apiRouter.HandleFunc("/invoices", invoiceHandler.List).Methods("GET")
	apiRouter.HandleFunc("/invoices", invoiceHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/invoices/auto-close", invoiceHandler.AutoClose).Methods("POST")
	apiRouter.HandleFunc("/invoices/current", invoiceHandler.GetCurrent).Methods("GET")
	apiRouter.HandleFunc("/invoices/{id}", invoiceHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/invoices/{id}", invoiceHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/invoices/{id}/close", invoiceHandler.Close).Methods("POST")
	apiRouter.HandleFunc("/invoices/{id}/pay", invoiceHandler.Pay).Methods("POST")
	apiRouter.HandleFunc("/invoices/{id}/add-amount", invoiceHandler.AddAmount).Methods("POST")
	apiRouter.HandleFunc("/invoices/{id}/recalculate", invoiceHandler.Recalculate).Methods("POST")

	// Crypto routes
	if cryptoHandler != nil {
		apiRouter.HandleFunc("/crypto/sync", cryptoHandler.Sync).Methods("POST")
		apiRouter.HandleFunc("/crypto/sync-trades", cryptoHandler.SyncTrades).Methods("POST")
	}
	apiRouter.HandleFunc("/crypto/purchases", cryptoPurchaseHandler.List).Methods("GET")
	apiRouter.HandleFunc("/crypto/purchases", cryptoPurchaseHandler.Create).Methods("POST")

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3002", "http://localhost:3003", "https://finances.wbdigitalsolutions.com", "https://calendar.wbdigitalsolutions.com", "https://finances.app.localhost", "https://calendar.app.localhost"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
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

	// Background job: auto-close invoices daily
	go func() {
		// Run once at startup
		result, err := autoCloseInvoicesUC.Execute(time.Now())
		if err != nil {
			log.Printf("Auto-close invoices startup error: %v", err)
		} else if result.Closed > 0 {
			log.Printf("Auto-close invoices: closed %d invoices at startup", result.Closed)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			result, err := autoCloseInvoicesUC.Execute(time.Now())
			if err != nil {
				log.Printf("Auto-close invoices error: %v", err)
				continue
			}
			if result.Closed > 0 {
				log.Printf("Auto-close invoices: closed %d invoices", result.Closed)
			}
		}
	}()

	// Background job: sync Binance trades every 30 minutes
	if syncTradesUC != nil {
		go func() {
			profileID := os.Getenv("DEFAULT_PROFILE_ID")
			if profileID == "" {
				profileID = "259866ac-6f74-4f7f-98b3-9a4896fb6758"
			}

			// Run once at startup (after a short delay to let things settle)
			time.Sleep(5 * time.Second)
			result, err := syncTradesUC.Execute(profileID)
			if err != nil {
				log.Printf("Trade sync startup error: %v", err)
			} else {
				log.Printf("Trade sync: %d buys, %d sells, %d skipped, %d errors",
					result.NewBuys, result.NewSells, result.Skipped, result.Errors)
			}

			ticker := time.NewTicker(30 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				result, err := syncTradesUC.Execute(profileID)
				if err != nil {
					log.Printf("Trade sync error: %v", err)
					continue
				}
				if result.NewBuys > 0 || result.NewSells > 0 {
					log.Printf("Trade sync: %d buys, %d sells, %d skipped",
						result.NewBuys, result.NewSells, result.Skipped)
				}
			}
		}()
	}

	log.Printf("🚀 Calendar Finances API starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
