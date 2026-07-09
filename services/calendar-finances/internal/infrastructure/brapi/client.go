package brapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const defaultBaseURL = "https://brapi.dev"

type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

type QuoteResponse struct {
	Results []QuoteResult `json:"results"`
}

type QuoteResult struct {
	Symbol                     string  `json:"symbol"`
	ShortName                  string  `json:"shortName"`
	LongName                   string  `json:"longName"`
	RegularMarketPrice         float64 `json:"regularMarketPrice"`
	RegularMarketChange        float64 `json:"regularMarketChange"`
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
}

func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// GetQuotes fetches current prices for one or more tickers.
// Each ticker is fetched individually (brapi.dev free plan: 1 asset per request).
func (c *Client) GetQuotes(tickers ...string) ([]QuoteResult, error) {
	if len(tickers) == 0 {
		return nil, fmt.Errorf("at least one ticker is required")
	}

	var results []QuoteResult
	for _, ticker := range tickers {
		quote, err := c.fetchQuote(ticker)
		if err != nil {
			log.Printf("brapi: failed to fetch %s: %v", ticker, err)
			continue
		}
		results = append(results, quote...)
	}

	return results, nil
}

func (c *Client) fetchQuote(ticker string) ([]QuoteResult, error) {
	url := fmt.Sprintf("%s/api/quote/%s", c.baseURL, ticker)
	if c.token != "" {
		url += "?token=" + c.token
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote %s: %w", ticker, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brapi API error (%d) for %s: %s", resp.StatusCode, ticker, string(body))
	}

	var result QuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode quote %s: %w", ticker, err)
	}

	return result.Results, nil
}
