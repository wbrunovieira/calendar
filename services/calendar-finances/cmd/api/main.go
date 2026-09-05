package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/brunovieira/calendar-finances/internal/app"
	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/database"
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

	log.Println("\u2713 Connected to PostgreSQL database")

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Wire the API. internal/app owns the wiring so tests can build the very
	// router this process serves, instead of assembling a lookalike.
	application, err := app.New(db)
	if err != nil {
		log.Fatalf("Failed to wire the application: %v", err)
	}

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
		Handler:      corsHandler.Handler(application.Router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Background job: auto-close invoices daily
	go func() {
		// Run once at startup
		result, err := application.Jobs.AutoCloseInvoices.Execute(time.Now())
		if err != nil {
			log.Printf("Auto-close invoices startup error: %v", err)
		} else if result.Closed > 0 {
			log.Printf("Auto-close invoices: closed %d invoices at startup", result.Closed)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			result, err := application.Jobs.AutoCloseInvoices.Execute(time.Now())
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
	if application.Jobs.SyncTrades != nil {
		go func() {
			profileID := os.Getenv("DEFAULT_PROFILE_ID")
			if profileID == "" {
				profileID = "259866ac-6f74-4f7f-98b3-9a4896fb6758"
			}

			// Run once at startup (after a short delay to let things settle)
			time.Sleep(5 * time.Second)
			result, err := application.Jobs.SyncTrades.Execute(profileID)
			if err != nil {
				log.Printf("Trade sync startup error: %v", err)
			} else {
				log.Printf("Trade sync: %d buys, %d sells, %d skipped, %d errors",
					result.NewBuys, result.NewSells, result.Skipped, result.Errors)
			}

			ticker := time.NewTicker(30 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				result, err := application.Jobs.SyncTrades.Execute(profileID)
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

	// Background job: sync B3 stock/FII prices every 2 hours and dividends daily
	// Rate limit: 15k requests/month, 1 asset per request, data updates every 30min
	go func() {
		profileID := os.Getenv("DEFAULT_PROFILE_ID")
		if profileID == "" {
			profileID = "259866ac-6f74-4f7f-98b3-9a4896fb6758"
		}

		// Run price sync at startup (after delay)
		time.Sleep(10 * time.Second)
		if result, err := application.Jobs.StockSync.Execute(profileID); err != nil {
			log.Printf("Stock price sync startup error: %v", err)
		} else if len(result.UpdatedAccounts) > 0 {
			log.Printf("Stock price sync: updated %d accounts", len(result.UpdatedAccounts))
		}

		// Run dividend sync at startup
		since := time.Now().AddDate(0, -3, 0)
		if result, err := application.Jobs.DividendSync.Execute(profileID, since); err != nil {
			log.Printf("Dividend sync startup error: %v", err)
		} else if result.NewDividends > 0 {
			log.Printf("Dividend sync: %d new dividends (R$%.2f)", result.NewDividends, result.TotalAmount)
		}

		priceTicker := time.NewTicker(2 * time.Hour)
		dividendTicker := time.NewTicker(24 * time.Hour)
		defer priceTicker.Stop()
		defer dividendTicker.Stop()

		for {
			select {
			case <-priceTicker.C:
				if result, err := application.Jobs.StockSync.Execute(profileID); err != nil {
					log.Printf("Stock price sync error: %v", err)
				} else if len(result.UpdatedAccounts) > 0 {
					log.Printf("Stock price sync: updated %d accounts", len(result.UpdatedAccounts))
				}
			case <-dividendTicker.C:
				since := time.Now().AddDate(0, -3, 0)
				if result, err := application.Jobs.DividendSync.Execute(profileID, since); err != nil {
					log.Printf("Dividend sync error: %v", err)
				} else if result.NewDividends > 0 {
					log.Printf("Dividend sync: %d new dividends (R$%.2f)", result.NewDividends, result.TotalAmount)
				}
			}
		}
	}()

	// Background job: close previous month on the 1st of each month (creates checkpoints)
	go func() {
		for {
			now := time.Now()
			// Next 1st of the month at 00:05 UTC
			nextRun := time.Date(now.Year(), now.Month()+1, 1, 0, 5, 0, 0, time.UTC)
			time.Sleep(time.Until(nextRun))

			prevMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			result, err := application.Jobs.CloseMonth.Execute(usecases.CloseMonthInput{ReferenceMonth: prevMonth})
			if err != nil {
				log.Printf("close-month error: %v", err)
			} else {
				log.Printf("close-month: created %d checkpoints for %s", result.CheckpointsCreated, prevMonth.Format("2006-01"))
			}
		}
	}()

	log.Printf("\U0001F680 Calendar Finances API starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
