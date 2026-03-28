package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "https://api.binance.com"

type Client struct {
	apiKey    string
	secretKey string
	http      *http.Client
}

type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type AccountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

type AccountInfo struct {
	Balances []AccountBalance `json:"balances"`
}

type Trade struct {
	ID              int64  `json:"id"`
	Symbol          string `json:"symbol"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	QuoteQty        string `json:"quoteQty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	IsBuyer         bool   `json:"isBuyer"`
}

type CryptoPrice struct {
	Symbol   string  `json:"symbol"`
	PriceUSD float64 `json:"priceUsd"`
	PriceBRL float64 `json:"priceBrl"`
}

type SyncResult struct {
	Prices   []CryptoPrice            `json:"prices"`
	Balances []map[string]interface{} `json:"balances,omitempty"`
	UsdBrl   float64                  `json:"usdBrl"`
}

func NewClient(apiKey, secretKey string) *Client {
	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// GetTickerPrice fetches current price for a symbol (public, no auth)
func (c *Client) GetTickerPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/api/v3/ticker/price?symbol=%s", baseURL, symbol)
	resp, err := c.http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch ticker %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("binance API error for %s: %s", symbol, string(body))
	}

	var ticker TickerPrice
	if err := json.NewDecoder(resp.Body).Decode(&ticker); err != nil {
		return 0, fmt.Errorf("failed to decode ticker %s: %w", symbol, err)
	}

	price, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price %s: %w", ticker.Price, err)
	}
	return price, nil
}

// GetAccountBalances fetches account balances (authenticated)
func (c *Client) GetAccountBalances() ([]AccountBalance, error) {
	timestamp := time.Now().UnixMilli()
	queryString := fmt.Sprintf("timestamp=%d", timestamp)

	signature := c.sign(queryString)
	url := fmt.Sprintf("%s/api/v3/account?%s&signature=%s", baseURL, queryString, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance account API error: %s", string(body))
	}

	var account AccountInfo
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode account: %w", err)
	}

	// Filter only non-zero balances
	var nonZero []AccountBalance
	for _, b := range account.Balances {
		free, _ := strconv.ParseFloat(b.Free, 64)
		locked, _ := strconv.ParseFloat(b.Locked, 64)
		if free > 0 || locked > 0 {
			nonZero = append(nonZero, b)
		}
	}
	return nonZero, nil
}

// FetchPrices gets current prices for common crypto pairs and USD/BRL
func (c *Client) FetchPrices() (*SyncResult, error) {
	result := &SyncResult{}

	// USD/BRL via USDT/BRL
	usdtBrl, err := c.GetTickerPrice("USDTBRL")
	if err != nil {
		return nil, fmt.Errorf("failed to get USDT/BRL: %w", err)
	}
	result.UsdBrl = usdtBrl

	// SOL/USDT
	solUsdt, err := c.GetTickerPrice("SOLUSDT")
	if err != nil {
		return nil, fmt.Errorf("failed to get SOL/USDT: %w", err)
	}
	result.Prices = append(result.Prices, CryptoPrice{
		Symbol:   "SOL",
		PriceUSD: solUsdt,
		PriceBRL: solUsdt * usdtBrl,
	})

	// Get account balances
	balances, err := c.GetAccountBalances()
	if err == nil {
		for _, b := range balances {
			free, _ := strconv.ParseFloat(b.Free, 64)
			locked, _ := strconv.ParseFloat(b.Locked, 64)
			result.Balances = append(result.Balances, map[string]interface{}{
				"asset":  b.Asset,
				"free":   free,
				"locked": locked,
				"total":  free + locked,
			})
		}
	}

	return result, nil
}

// GetMyTrades fetches trade history for a symbol (authenticated)
func (c *Client) GetMyTrades(symbol string) ([]Trade, error) {
	timestamp := time.Now().UnixMilli()
	queryString := fmt.Sprintf("symbol=%s&timestamp=%d", symbol, timestamp)

	signature := c.sign(queryString)
	url := fmt.Sprintf("%s/api/v3/myTrades?%s&signature=%s", baseURL, queryString, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance trades API error: %s", string(body))
	}

	var trades []Trade
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		return nil, fmt.Errorf("failed to decode trades: %w", err)
	}
	return trades, nil
}

func (c *Client) sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
