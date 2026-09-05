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
			// A PathPrefix subrouter mount has no methods and no handler of its
			// own. Anything else without methods is a route with no verb.
			if route.GetHandler() != nil {
				return fmt.Errorf("route %s is registered without any method", path)
			}
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

// wantRoutes is the complete route table the API serves, not a sample of it.
// main.go used to own this wiring, so a route could exist in production and be
// absent from every test, and a later refactor could drop one with nothing
// noticing. Comparing the whole set in both directions is what makes the
// extraction verifiable rather than trusted.
//
// The list was checked against origin/main's cmd/api/main.go, not merely
// generated from the code it tests: main.go registered 89 routes with methods,
// and this is those 89 plus /api/v1/health/invariants. Adding a route here
// should be a deliberate line, never a paste of the failure message.
var wantRoutes = []string{
	"DELETE /api/v1/bank-accounts/{id}",
	"DELETE /api/v1/budgets/{id}",
	"DELETE /api/v1/capital-contributions/{id}",
	"DELETE /api/v1/categories/{id}",
	"DELETE /api/v1/company-assets/{id}",
	"DELETE /api/v1/cost-centers/{id}",
	"DELETE /api/v1/goals/{id}",
	"DELETE /api/v1/marketing-campaigns/{id}",
	"DELETE /api/v1/profiles/{id}",
	"DELETE /api/v1/recurring-transactions/{id}",
	"DELETE /api/v1/transactions/{id}",
	"GET /",
	"GET /api/v1/accounts",
	"GET /api/v1/bank-accounts",
	"GET /api/v1/bank-accounts/maturities",
	"GET /api/v1/bank-accounts/{id}",
	"GET /api/v1/benchmarks/returns",
	"GET /api/v1/budgets",
	"GET /api/v1/budgets/summary",
	"GET /api/v1/capital-contributions",
	"GET /api/v1/capital-contributions/summary",
	"GET /api/v1/capital-contributions/{id}",
	"GET /api/v1/categories",
	"GET /api/v1/company-assets",
	"GET /api/v1/company-assets/{id}",
	"GET /api/v1/cost-centers",
	"GET /api/v1/cost-centers/{id}",
	"GET /api/v1/crypto/purchases",
	"GET /api/v1/fiis/market",
	"GET /api/v1/goals",
	"GET /api/v1/health/invariants",
	"GET /api/v1/invoices",
	"GET /api/v1/invoices/current",
	"GET /api/v1/invoices/{id}",
	"GET /api/v1/marketing-campaigns",
	"GET /api/v1/marketing-campaigns/{id}",
	"GET /api/v1/marketing-campaigns/{id}/metrics",
	"GET /api/v1/profiles",
	"GET /api/v1/profiles/{id}",
	"GET /api/v1/recurring-transactions",
	"GET /api/v1/transactions",
	"GET /api/v1/transactions/daily-balances",
	"GET /api/v1/transactions/expense-analysis",
	"GET /api/v1/transactions/financial-summary",
	"GET /api/v1/transactions/{id}",
	"GET /health",
	"PATCH /api/v1/goals/{id}/status",
	"PATCH /api/v1/recurring-transactions/{id}/status",
	"POST /api/v1/accounts",
	"POST /api/v1/bank-accounts",
	"POST /api/v1/bank-accounts/close-month",
	"POST /api/v1/bank-accounts/{id}/recalculate-balance",
	"POST /api/v1/bank-accounts/{id}/sell",
	"POST /api/v1/budgets",
	"POST /api/v1/capital-contributions",
	"POST /api/v1/categories",
	"POST /api/v1/company-assets",
	"POST /api/v1/cost-centers",
	"POST /api/v1/crypto/purchases",
	"POST /api/v1/crypto/sync",
	"POST /api/v1/crypto/sync-trades",
	"POST /api/v1/goals",
	"POST /api/v1/goals/{id}/add-amount",
	"POST /api/v1/invoices",
	"POST /api/v1/invoices/auto-close",
	"POST /api/v1/invoices/{id}/close",
	"POST /api/v1/invoices/{id}/pay",
	"POST /api/v1/invoices/{id}/recalculate",
	"POST /api/v1/marketing-campaigns",
	"POST /api/v1/profiles",
	"POST /api/v1/recurring-transactions",
	"POST /api/v1/recurring-transactions/process",
	"POST /api/v1/stocks/sync-dividends",
	"POST /api/v1/stocks/sync-prices",
	"POST /api/v1/transactions",
	"PUT /api/v1/bank-accounts/reorder",
	"PUT /api/v1/bank-accounts/{id}",
	"PUT /api/v1/budgets/{id}",
	"PUT /api/v1/capital-contributions/{id}",
	"PUT /api/v1/categories/{id}",
	"PUT /api/v1/company-assets/{id}",
	"PUT /api/v1/cost-centers/{id}",
	"PUT /api/v1/goals/reorder",
	"PUT /api/v1/goals/{id}",
	"PUT /api/v1/invoices/{id}",
	"PUT /api/v1/marketing-campaigns/{id}",
	"PUT /api/v1/profiles/{id}",
	"PUT /api/v1/recurring-transactions/{id}",
	"PUT /api/v1/transactions/{id}",
	"PUT /api/v1/transactions/{id}/status",
}

func TestNewRouter_RegistersExactlyTheRoutesTheAPIServes(t *testing.T) {
	t.Setenv("KEY_BINANCE", "test-key")
	t.Setenv("SECRET_BINANCE", "test-secret")

	application, err := app.New(openIdleDB(t))
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	got := registeredRoutes(t, application.Router)

	want := make(map[string]bool, len(wantRoutes))
	for _, route := range wantRoutes {
		want[route] = true
	}

	var missing, unexpected []string
	for route := range want {
		if !got[route] {
			missing = append(missing, route)
		}
	}
	for route := range got {
		if !want[route] {
			unexpected = append(unexpected, route)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("routes missing from the wired router:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		t.Errorf("routes registered but not declared here, add them deliberately:\n  %s", strings.Join(unexpected, "\n  "))
	}
}

func TestNewRouter_RefusesToWireWithoutADatabase(t *testing.T) {
	if _, err := app.New(nil); err == nil {
		t.Error("app.New(nil) returned no error")
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
