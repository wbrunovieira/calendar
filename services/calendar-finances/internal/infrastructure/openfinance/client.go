// Package openfinance reads what the BANK says, so the ledger can be compared
// against it instead of only against itself.
package openfinance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.mcp.ai/api/openfinance"

// Client reads account balances and card limits over Open Finance.
//
// Read-only by construction: the only endpoint used lists accounts. Nothing here
// can move money, and nothing here writes to the ledger — the comparison that
// follows reports, it does not repair.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		// Generous: the provider refreshes upstream on some calls. The caller is a
		// daily job, so waiting beats failing.
		http: &http.Client{Timeout: 90 * time.Second},
	}
}

// Account is one account as the bank reports it.
type Account struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Balance string `json:"balance"`
	Credit  *struct {
		CreditLimit          string `json:"creditLimit"`
		AvailableCreditLimit string `json:"availableCreditLimit"`
		UsedAmount           string `json:"usedAmount"`
	} `json:"creditData,omitempty"`
}

// Owed is what this account represents as debt or cash, as one number.
//
// For a card the AVAILABLE limit is the figure to trust: `usedAmount` is not
// filled by every issuer, and the account `balance` means different things per
// connector — some report the open bill, others the whole revolving debt.
// Limit minus available is the one definition every issuer agrees on.
func (a Account) Owed() (float64, bool) {
	if a.Credit != nil {
		limit, okL := parseAmount(a.Credit.CreditLimit)
		avail, okA := parseAmount(a.Credit.AvailableCreditLimit)
		if okL && okA {
			return limit - avail, true
		}
		if used, ok := parseAmount(a.Credit.UsedAmount); ok {
			return used, true
		}
		return 0, false
	}
	return parseAmount(a.Balance)
}

func parseAmount(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%g", &f); err != nil {
		return 0, false
	}
	return f, true
}

type listResponse struct {
	Results []Account `json:"results"`
}

// ListAccounts returns every account across every linked bank.
func (c *Client) ListAccounts() ([]Account, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/accounts/list", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reaching the open finance provider: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		// The body carries the provider's own reason — a stale consent, a rate
		// limit — and a daily job that hides it leaves nobody able to act.
		return nil, fmt.Errorf("open finance answered %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed listResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("reading the open finance payload: %w", err)
	}
	return parsed.Results, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// OwedByProviderID adapts the client to what the reconciliation needs: one
// number per account, keyed by the id our own accounts store.
//
// An account the provider answers for but cannot put a figure on is LEFT OUT
// rather than entered as zero. Zero reads as "this account owes nothing", which
// is a lie that reconciles — and the comparison treats an absent key as an
// unanswered account, which is the truth.
func (c *Client) OwedByProviderID() (map[string]float64, error) {
	accounts, err := c.ListAccounts()
	if err != nil {
		return nil, err
	}
	owed := make(map[string]float64, len(accounts))
	for _, a := range accounts {
		if v, ok := a.Owed(); ok {
			owed[a.ID] = v
		}
	}
	return owed, nil
}
