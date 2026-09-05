package app_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/app"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// openIdleDB returns a *sql.DB that never dials: sql.Open only parses the DSN.
// Wiring the API must not touch the database, and this test fails loudly if it
// ever starts to.
func openIdleDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", "postgres://unused:unused@127.0.0.1:1/unused?sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func registeredRoutes(t *testing.T, router *mux.Router) map[string]bool {
	t.Helper()
	routes := map[string]bool{}
	err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return nil // subrouters without a path of their own
		}
		methods, err := route.GetMethods()
		if err != nil {
			// A PathPrefix subrouter mount: no methods and no handler of its own.
			return nil
		}
		if route.GetHandler() == nil {
			return fmt.Errorf("route %s is registered with a nil handler", path)
		}
		for _, method := range methods {
			routes[method+" "+path] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}
	return routes
}

// The routes below are the ones that move money or report it. main.go used to own
// this wiring, so a route could exist in production and be absent from every
// test. Building the real router here is what closes that gap.
func TestNewRouter_RegistersTheRoutesThatMoveMoney(t *testing.T) {
	t.Setenv("KEY_BINANCE", "test-key")
	t.Setenv("SECRET_BINANCE", "test-secret")

	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	routes := registeredRoutes(t, application.Router)

	want := []string{
		"GET /health",
		"GET /api/v1/health/invariants",
		"GET /api/v1/profiles",
		"GET /api/v1/bank-accounts",
		"PUT /api/v1/bank-accounts/{id}",
		"POST /api/v1/bank-accounts/{id}/recalculate-balance",
		"POST /api/v1/bank-accounts/{id}/sell",
		"POST /api/v1/bank-accounts/close-month",
		"GET /api/v1/categories",
		"POST /api/v1/transactions",
		"GET /api/v1/transactions",
		"PUT /api/v1/transactions/{id}",
		"DELETE /api/v1/transactions/{id}",
		"PUT /api/v1/transactions/{id}/status",
		"GET /api/v1/transactions/daily-balances",
		"GET /api/v1/transactions/expense-analysis",
		"GET /api/v1/transactions/financial-summary",
		"GET /api/v1/invoices",
		"GET /api/v1/invoices/current",
		"POST /api/v1/invoices/auto-close",
		"POST /api/v1/invoices/{id}/close",
		"POST /api/v1/invoices/{id}/pay",
		"POST /api/v1/invoices/{id}/recalculate",
		"PUT /api/v1/invoices/{id}",
		"POST /api/v1/recurring-transactions/process",
		"GET /api/v1/budgets/summary",
		"GET /api/v1/goals",
		"POST /api/v1/crypto/sync-trades",
		"POST /api/v1/stocks/sync-prices",
		"POST /api/v1/stocks/sync-dividends",
		"GET /api/v1/capital-contributions/summary",
		"GET /api/v1/cost-centers",
		"GET /api/v1/marketing-campaigns",
		"GET /api/v1/company-assets",
		"GET /api/v1/benchmarks/returns",
		"GET /api/v1/fiis/market",
	}

	var missing []string
	for _, route := range want {
		if !routes[route] {
			missing = append(missing, route)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("routes missing from the wired router:\n  %s", strings.Join(missing, "\n  "))
	}
}

func TestNewRouter_SkipsTheBinanceRoutesWithoutCredentials(t *testing.T) {
	t.Setenv("KEY_BINANCE", "")
	t.Setenv("SECRET_BINANCE", "")

	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	routes := registeredRoutes(t, application.Router)

	if routes["POST /api/v1/crypto/sync-trades"] {
		t.Error("the Binance sync route must not exist without credentials")
	}
	if !routes["GET /api/v1/crypto/purchases"] {
		t.Error("manual crypto purchases do not need Binance and must stay registered")
	}
}

// The background loops in main.go run on the same use cases the routes use.
// Returning them from the wiring is what stops main.go from building a second,
// divergent copy.
func TestNewRouter_ExposesTheBackgroundJobs(t *testing.T) {
	t.Setenv("KEY_BINANCE", "test-key")
	t.Setenv("SECRET_BINANCE", "test-secret")

	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	if application.Jobs.AutoCloseInvoices == nil {
		t.Error("AutoCloseInvoices job is nil")
	}
	if application.Jobs.SyncTrades == nil {
		t.Error("SyncTrades job is nil when Binance credentials are set")
	}
	if application.Jobs.StockSync == nil {
		t.Error("StockSync job is nil")
	}
	if application.Jobs.DividendSync == nil {
		t.Error("DividendSync job is nil")
	}
	if application.Jobs.CloseMonth == nil {
		t.Error("CloseMonth job is nil")
	}
}

func TestNewRouter_HealthCheckAnswersWithoutADatabase(t *testing.T) {
	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	application.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /health = %d, want 200", rec.Code)
	}
}
