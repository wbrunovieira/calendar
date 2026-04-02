package brapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	Symbol                     string         `json:"symbol"`
	ShortName                  string         `json:"shortName"`
	LongName                   string         `json:"longName"`
	RegularMarketPrice         float64        `json:"regularMarketPrice"`
	RegularMarketChange        float64        `json:"regularMarketChange"`
	RegularMarketChangePercent float64        `json:"regularMarketChangePercent"`
	DividendsData              *DividendsData `json:"dividendsData,omitempty"`
}

type DividendsData struct {
	CashDividends []CashDividend `json:"cashDividends"`
}

type CashDividend struct {
	AssetIssued   string  `json:"assetIssued"`
	PaymentDate   string  `json:"paymentDate"`
	Rate          float64 `json:"rate"`
	Label         string  `json:"label"`
	LastDatePrior string  `json:"lastDatePrior"`
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
		quote, err := c.fetchQuote(ticker, false)
		if err != nil {
			log.Printf("brapi: failed to fetch %s: %v", ticker, err)
			continue
		}
		results = append(results, quote...)
	}

	return results, nil
}

func (c *Client) fetchQuote(ticker string, withDividends bool) ([]QuoteResult, error) {
	url := fmt.Sprintf("%s/api/quote/%s", c.baseURL, ticker)
	params := []string{}
	if c.token != "" {
		params = append(params, "token="+c.token)
	}
	if withDividends {
		params = append(params, "dividends=true")
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
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

// GetDividends fetches dividend history for a ticker.
func (c *Client) GetDividends(ticker string) ([]CashDividend, error) {
	results, err := c.fetchQuote(ticker, true)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 || results[0].DividendsData == nil {
		return []CashDividend{}, nil
	}

	return results[0].DividendsData.CashDividends, nil
}
