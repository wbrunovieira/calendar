// Package openfinance reads what the BANK says, so the ledger can be compared
// against it instead of only against itself.
package openfinance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
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
		// Under the server's 15s WriteTimeout, deliberately. A longer budget here
		// does not buy patience — it buys a response write killed mid-flight, so
		// the caller sees an aborted connection instead of the 503 this design
		// promises.
		http: &http.Client{Timeout: 12 * time.Second},
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

// parseAmount reads a money string, or refuses.
//
// Sscanf was wrong here twice over: it accepts a PREFIX, so "718.36abc" read as
// 718.36 and the junk vanished, and it read "1.234,56" as 1.234 — a thousandfold
// error that still reconciles against nothing. It also accepted NaN and Inf,
// which then break JSON encoding and leave a 200 with an empty body, which a
// status-code cron reads as healthy.
func parseAmount(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// The provider wraps every answer: {"ok":true,"tool":"...","result":<payload>}.
// The first version of this read `results` from the TOP level, which
// encoding/json happily satisfies with nothing — unknown fields are ignored, so
// it returned zero accounts and a nil error. Zero accounts reconciles against
// anything, so the check would have reported perfect health having compared
// nothing, every day, which is the exact failure it exists to prevent.
//
// Both shapes are accepted: the envelope the REST API uses, and the bare one, so
// a provider that stops wrapping does not silently empty this out again.
type listResponse struct {
	Results []Account `json:"results"`
	Result  *struct {
		Results []Account `json:"results"`
	} `json:"result"`
}

// accounts returns whichever shape carried them, and says which.
func (r listResponse) accounts() ([]Account, bool) {
	if r.Result != nil && len(r.Result.Results) > 0 {
		return r.Result.Results, true
	}
	if len(r.Results) > 0 {
		return r.Results, true
	}
	return nil, false
}

// ListAccounts returns every account across every linked bank.
func (c *Client) ListAccounts() ([]Account, error) {
	return c.ListAccountsContext(context.Background())
}

// ListAccountsContext is the same call, cancellable: a caller that goes away
// should not leave this one waiting on the provider.
func (c *Client) ListAccountsContext(ctx context.Context) ([]Account, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/accounts/list", bytes.NewReader([]byte(`{}`)))
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
	accounts, found := parsed.accounts()
	if !found {
		// An answer carrying no accounts is an ERROR, never an empty list. An
		// empty list compares clean against everything; a changed envelope would
		// otherwise be indistinguishable from a healthy bank.
		return nil, fmt.Errorf("open finance returned no accounts; the payload shape may have changed: %s", truncate(string(body), 300))
	}
	return accounts, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// FiguresByProviderID adapts the client to what the reconciliation needs, keyed
// by the provider id our own accounts store: what each account is worth, and
// what the provider says each account IS.
//
// Both come back from ONE call on purpose. Asking twice would either cost a
// second round trip or leave the second answer served from state left behind by
// the first, which is a stale read waiting to happen.
//
// An account the provider answers for but cannot put a figure on is LEFT OUT of
// the amounts rather than entered as zero. Zero reads as "this account owes
// nothing", which is a lie that reconciles — and the comparison treats an absent
// key as an unanswered account, which is the truth. Its KIND is still reported:
// a transposed mapping is worth naming even when the figure is unreadable.
func (c *Client) FiguresByProviderID() (map[string]float64, map[string]string, error) {
	accounts, err := c.ListAccounts()
	if err != nil {
		return nil, nil, err
	}
	owed := make(map[string]float64, len(accounts))
	kinds := make(map[string]string, len(accounts))
	for _, a := range accounts {
		kinds[a.ID] = a.Type
		if v, ok := a.Owed(); ok {
			owed[a.ID] = v
		}
	}
	return owed, kinds, nil
}
