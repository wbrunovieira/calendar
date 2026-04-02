package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://query1.finance.yahoo.com"

type Client struct {
	http *http.Client
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// GetReturn calculates the percentage return for a ticker between two dates.
func (c *Client) GetReturn(ticker string, from, to time.Time) (float64, error) {
	url := fmt.Sprintf(
		"%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		baseURL, ticker, from.Unix(), to.Unix(),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("yahoo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("yahoo returned status %d", resp.StatusCode)
	}

	var result chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("yahoo decode failed: %w", err)
	}

	if result.Chart.Error != nil {
		return 0, fmt.Errorf("yahoo error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 || len(result.Chart.Result[0].Indicators.Quote) == 0 {
		return 0, fmt.Errorf("no data for %s", ticker)
	}

	closes := result.Chart.Result[0].Indicators.Quote[0].Close

	var startPrice, endPrice float64
	for _, p := range closes {
		if p != nil {
			startPrice = *p
			break
		}
	}
	for i := len(closes) - 1; i >= 0; i-- {
		if closes[i] != nil {
			endPrice = *closes[i]
			break
		}
	}

	if startPrice == 0 {
		return 0, fmt.Errorf("no valid prices for %s", ticker)
	}

	return (endPrice - startPrice) / startPrice * 100, nil
}
