package yahoo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func dividendsChartJSON(divs map[string]struct {
	Amount float64
	Date   int64
}) string {
	entries := ""
	first := true
	for k, d := range divs {
		if !first {
			entries += ","
		}
		first = false
		entries += fmt.Sprintf(`"%s":{"amount":%f,"date":%d}`, k, d.Amount, d.Date)
	}
	return fmt.Sprintf(`{"chart":{"result":[{"timestamp":[],"events":{"dividends":{%s}},"indicators":{"quote":[{"close":[]}]}}],"error":null}}`, entries)
}

func TestGetDividends_ParsesAndSortsByDate(t *testing.T) {
	// 2026-04-01 13:10 UTC and 2026-03-02 13:10 UTC (Yahoo uses market-open timestamps)
	apr := time.Date(2026, 4, 1, 13, 10, 0, 0, time.UTC).Unix()
	mar := time.Date(2026, 3, 2, 13, 10, 0, 0, time.UTC).Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/finance/chart/HGLG11.SA" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("events") != "div" {
			t.Error("expected events=div query param")
		}
		fmt.Fprint(w, dividendsChartJSON(map[string]struct {
			Amount float64
			Date   int64
		}{
			fmt.Sprint(apr): {Amount: 1.10, Date: apr},
			fmt.Sprint(mar): {Amount: 1.05, Date: mar},
		}))
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	dividends, err := client.GetDividends("HGLG11", time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dividends) != 2 {
		t.Fatalf("expected 2 dividends, got %d", len(dividends))
	}
	// Sorted ascending by date, normalized to midnight UTC
	if !dividends[0].Date.Equal(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected first dividend on 2026-03-02, got %v", dividends[0].Date)
	}
	if dividends[0].Amount != 1.05 {
		t.Errorf("expected amount 1.05, got %f", dividends[0].Amount)
	}
	if !dividends[1].Date.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected second dividend on 2026-04-01, got %v", dividends[1].Date)
	}
}

func TestGetDividends_NoDividends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"chart":{"result":[{"timestamp":[],"events":{},"indicators":{"quote":[{"close":[]}]}}],"error":null}}`)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	dividends, err := client.GetDividends("PETR4", time.Now().AddDate(0, -3, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dividends) != 0 {
		t.Errorf("expected 0 dividends, got %d", len(dividends))
	}
}

func TestGetDividends_YahooError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	_, err := client.GetDividends("XXXX11", time.Now().AddDate(0, -3, 0))
	if err == nil {
		t.Fatal("expected error for yahoo error response")
	}
}

func TestGetDividends_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	_, err := client.GetDividends("HGLG11", time.Now().AddDate(0, -3, 0))
	if err == nil {
		t.Fatal("expected error for HTTP 429")
	}
}
