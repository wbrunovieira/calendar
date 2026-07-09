package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const defaultBaseURL = "https://query1.finance.yahoo.com"

type Client struct {
	baseURL string
	http    *http.Client
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
	CurrentPrice     float64            `json:"current_price"`
	PriceChange12M   float64            `json:"price_change_12m"`
	Dividends12M     float64            `json:"dividends_12m"`
	DividendYield    float64            `json:"dividend_yield"`
	LastDividend     float64            `json:"last_dividend"`
	LastDividendDate string             `json:"last_dividend_date"`
	PriceToBook      *float64           `json:"price_to_book"`
	BookValue        *float64           `json:"book_value"`
	TotalReturn12M   float64            `json:"total_return_12m"`
	MonthlyDividends []MonthlyDividend  `json:"monthly_dividends"`
}

// MonthlyDividend holds dividend data for a single month.
type MonthlyDividend struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
	Yield  float64 `json:"yield"`
}

// quoteSummaryResponse for Yahoo v10 quoteSummary endpoint.
type quoteSummaryResponse struct {
	QuoteSummary struct {
		Result []struct {
			DefaultKeyStatistics struct {
				PriceToBook struct {
					Raw float64 `json:"raw"`
				} `json:"priceToBook"`
				BookValue struct {
					Raw float64 `json:"raw"`
				} `json:"bookValue"`
			} `json:"defaultKeyStatistics"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"quoteSummary"`
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
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Dividend is a single dividend payment. Date is the ex-dividend date
// (Yahoo does not expose the payment date), normalized to midnight UTC.
type Dividend struct {
	Date   time.Time
	Amount float64
}

// GetDividends fetches dividends per quota for a B3 ticker since the given date.
// The ticker should be provided without the .SA suffix (e.g. "HGLG11").
// Results are sorted by date ascending.
func (c *Client) GetDividends(ticker string, from time.Time) ([]Dividend, error) {
	yahooTicker := ticker
	if !strings.HasSuffix(strings.ToUpper(yahooTicker), ".SA") {
		yahooTicker = yahooTicker + ".SA"
	}

	url := fmt.Sprintf(
		"%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&events=div",
		c.baseURL, yahooTicker, from.Unix(), time.Now().Unix(),
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

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data for %s", yahooTicker)
	}

	var dividends []Dividend
	for _, d := range result.Chart.Result[0].Events.Dividends {
		t := time.Unix(d.Date, 0).UTC()
		dividends = append(dividends, Dividend{
			Date:   time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
			Amount: d.Amount,
		})
	}

	sort.Slice(dividends, func(i, j int) bool {
		return dividends[i].Date.Before(dividends[j].Date)
	})

	return dividends, nil
}

// GetReturn calculates the percentage return for a ticker between two dates.
func (c *Client) GetReturn(ticker string, from, to time.Time) (float64, error) {
	url := fmt.Sprintf(
		"%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		c.baseURL, ticker, from.Unix(), to.Unix(),
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
		c.baseURL, yahooTicker, oneYearAgo.Unix(), now.Unix(),
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

	// Build monthly dividend breakdown.
	monthlyMap := make(map[string]float64)
	for _, d := range divs {
		t := time.Unix(d.Date, 0)
		key := t.Format("2006-01")
		monthlyMap[key] += d.Amount
	}
	var monthlyDivs []MonthlyDividend
	var months []string
	for m := range monthlyMap {
		months = append(months, m)
	}
	sort.Strings(months)
	for _, m := range months {
		yld := float64(0)
		if currentPrice > 0 {
			yld = monthlyMap[m] / currentPrice * 100
		}
		monthlyDivs = append(monthlyDivs, MonthlyDividend{
			Month:  m,
			Amount: monthlyMap[m],
			Yield:  yld,
		})
	}

	totalReturn := priceChange12M + dividendYield

	data := &FIIData{
		CurrentPrice:     currentPrice,
		PriceChange12M:   priceChange12M,
		Dividends12M:     dividends12M,
		DividendYield:    dividendYield,
		LastDividend:     lastDividend,
		LastDividendDate: lastDivDateStr,
		TotalReturn12M:   totalReturn,
		MonthlyDividends: monthlyDivs,
	}

	// Try to fetch P/VP from quoteSummary (best effort).
	if pvp, bv, err := c.getQuoteSummary(yahooTicker); err == nil {
		data.PriceToBook = pvp
		data.BookValue = bv
	}

	return data, nil
}

// getQuoteSummary fetches P/VP and book value from Yahoo v10 quoteSummary.
func (c *Client) getQuoteSummary(yahooTicker string) (*float64, *float64, error) {
	url := fmt.Sprintf(
		"%s/v10/finance/quoteSummary/%s?modules=defaultKeyStatistics",
		c.baseURL, yahooTicker,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("yahoo quoteSummary request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("yahoo quoteSummary returned status %d", resp.StatusCode)
	}

	var result quoteSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("yahoo quoteSummary decode failed: %w", err)
	}

	if result.QuoteSummary.Error != nil {
		return nil, nil, fmt.Errorf("yahoo quoteSummary error: %s", result.QuoteSummary.Error.Description)
	}

	if len(result.QuoteSummary.Result) == 0 {
		return nil, nil, fmt.Errorf("no quoteSummary data for %s", yahooTicker)
	}

	stats := result.QuoteSummary.Result[0].DefaultKeyStatistics
	var pvp, bv *float64
	if stats.PriceToBook.Raw > 0 {
		v := stats.PriceToBook.Raw
		pvp = &v
	}
	if stats.BookValue.Raw > 0 {
		v := stats.BookValue.Raw
		bv = &v
	}

	return pvp, bv, nil
}
