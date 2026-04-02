package brapi

import (
	"encoding/json"
	"fmt"
	"io"
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

// GetQuotes fetches current prices for one or more tickers (comma-separated in URL).
func (c *Client) GetQuotes(tickers ...string) ([]QuoteResult, error) {
	if len(tickers) == 0 {
		return nil, fmt.Errorf("at least one ticker is required")
	}

	url := fmt.Sprintf("%s/api/quote/%s", c.baseURL, strings.Join(tickers, ","))
	if c.token != "" {
		url += "?token=" + c.token
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brapi API error (%d): %s", resp.StatusCode, string(body))
	}

	var result QuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode quote response: %w", err)
	}

	return result.Results, nil
}

// GetDividends fetches dividend history for a ticker.
func (c *Client) GetDividends(ticker string) ([]CashDividend, error) {
	url := fmt.Sprintf("%s/api/quote/%s?dividends=true", c.baseURL, ticker)
	if c.token != "" {
		url += "&token=" + c.token
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dividends: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brapi API error (%d): %s", resp.StatusCode, string(body))
	}

	var result QuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode dividends response: %w", err)
	}

	if len(result.Results) == 0 || result.Results[0].DividendsData == nil {
		return []CashDividend{}, nil
	}

	return result.Results[0].DividendsData.CashDividends, nil
}
