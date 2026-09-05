package brapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetQuotes_SingleTicker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/quote/HGLG11" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(QuoteResponse{
			Results: []QuoteResult{
				{
					Symbol:                     "HGLG11",
					ShortName:                  "FII CSHG LOG",
					RegularMarketPrice:         158.50,
					RegularMarketChange:        -0.30,
					RegularMarketChangePercent: -0.19,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("")
	client.baseURL = server.URL

	quotes, err := client.GetQuotes("HGLG11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].Symbol != "HGLG11" {
		t.Errorf("expected symbol HGLG11, got %s", quotes[0].Symbol)
	}
	if quotes[0].RegularMarketPrice != 158.50 {
		t.Errorf("expected price 158.50, got %f", quotes[0].RegularMarketPrice)
	}
}

func TestGetQuotes_MultipleTickers(t *testing.T) {
	// Each ticker is now fetched individually (1 asset per request limit)
	prices := map[string]float64{
		"HGLG11": 158.50,
		"KNCR11": 104.20,
		"PETR4":  38.75,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract ticker from path like /api/quote/HGLG11
		ticker := r.URL.Path[len("/api/quote/"):]
		price, ok := prices[ticker]
		if !ok {
			t.Errorf("unexpected ticker: %s", ticker)
			w.WriteHeader(404)
			return
		}
		json.NewEncoder(w).Encode(QuoteResponse{
			Results: []QuoteResult{{Symbol: ticker, RegularMarketPrice: price}},
		})
	}))
	defer server.Close()

	client := NewClient("")
	client.baseURL = server.URL

	quotes, err := client.GetQuotes("HGLG11", "KNCR11", "PETR4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 3 {
		t.Fatalf("expected 3 quotes, got %d", len(quotes))
	}
}

func TestGetQuotes_NoTickers(t *testing.T) {
	client := NewClient("")
	_, err := client.GetQuotes()
	if err == nil {
		t.Fatal("expected error for empty tickers")
	}
}

func TestGetQuotes_APIError_ReturnsEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := NewClient("")
	client.baseURL = server.URL

	// With individual requests, failed tickers are skipped (logged), result is empty
	quotes, err := client.GetQuotes("PETR4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("expected 0 quotes for failed request, got %d", len(quotes))
	}
}

func TestNewClient_WithToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != "my-token" {
			t.Error("expected token query param")
		}
		json.NewEncoder(w).Encode(QuoteResponse{
			Results: []QuoteResult{{Symbol: "MXRF11", RegularMarketPrice: 10.50}},
		})
	}))
	defer server.Close()

	client := NewClient("my-token")
	client.baseURL = server.URL

	quotes, err := client.GetQuotes("MXRF11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
}
