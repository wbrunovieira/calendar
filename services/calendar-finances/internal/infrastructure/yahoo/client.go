package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
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

// FIIData holds market data for a Brazilian FII (Fundo de Investimento Imobiliário).
type FIIData struct {
	CurrentPrice     float64 `json:"current_price"`
	PriceChange12M   float64 `json:"price_change_12m"`
	Dividends12M     float64 `json:"dividends_12m"`
	DividendYield    float64 `json:"dividend_yield"`
	LastDividend     float64 `json:"last_dividend"`
	LastDividendDate string  `json:"last_dividend_date"`
}

type fiiChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Events     struct {
				Dividends map[string]struct {
					Amount float64 `json:"amount"`
					Date   int64   `json:"date"`
				} `json:"dividends"`
			} `json:"events"`
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

// GetFIIData fetches 12-month price and dividend data for a Brazilian FII ticker.
// The ticker should be provided without the .SA suffix (e.g. "HGLG11").
func (c *Client) GetFIIData(ticker string) (*FIIData, error) {
	yahooTicker := ticker
	if !strings.HasSuffix(strings.ToUpper(yahooTicker), ".SA") {
		yahooTicker = yahooTicker + ".SA"
	}

	now := time.Now()
	oneYearAgo := now.AddDate(-1, 0, 0)

	url := fmt.Sprintf(
		"%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&events=div",
		baseURL, yahooTicker, oneYearAgo.Unix(), now.Unix(),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo returned status %d for %s", resp.StatusCode, yahooTicker)
	}

	var result fiiChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo decode failed: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 || len(result.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no data for %s", yahooTicker)
	}

	r := result.Chart.Result[0]
	closes := r.Indicators.Quote[0].Close

	// Find first and last valid close prices.
	var startPrice, currentPrice float64
	for _, p := range closes {
		if p != nil {
			startPrice = *p
			break
		}
	}
	for i := len(closes) - 1; i >= 0; i-- {
		if closes[i] != nil {
			currentPrice = *closes[i]
			break
		}
	}

	if startPrice == 0 {
		return nil, fmt.Errorf("no valid prices for %s", yahooTicker)
	}

	priceChange12M := (currentPrice - startPrice) / startPrice * 100

	// Process dividends.
	var dividends12M float64
	var lastDividend float64
	var lastDividendDate int64

	type divEntry struct {
		Amount float64
		Date   int64
	}
	var divs []divEntry

	for _, d := range r.Events.Dividends {
		divs = append(divs, divEntry{Amount: d.Amount, Date: d.Date})
		dividends12M += d.Amount
	}

	if len(divs) > 0 {
		sort.Slice(divs, func(i, j int) bool {
			return divs[i].Date < divs[j].Date
		})
		last := divs[len(divs)-1]
		lastDividend = last.Amount
		lastDividendDate = last.Date
	}

	var dividendYield float64
	if currentPrice > 0 {
		dividendYield = dividends12M / currentPrice * 100
	}

	var lastDivDateStr string
	if lastDividendDate > 0 {
		lastDivDateStr = time.Unix(lastDividendDate, 0).Format("02/01/2006")
	}

	return &FIIData{
		CurrentPrice:     currentPrice,
		PriceChange12M:   priceChange12M,
		Dividends12M:     dividends12M,
		DividendYield:    dividendYield,
		LastDividend:     lastDividend,
		LastDividendDate: lastDivDateStr,
	}, nil
}
