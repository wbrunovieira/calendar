package statement

import (
	"encoding/json"
	"os"
	"testing"
)

// Built from REAL Pluggy payloads, captured from the connected accounts. Written
// before any matching code, because every key the reconciliation will use is a bet on
// the payload's shape — and modelling against an imagined schema means writing the
// matcher twice.
//
// Three assumptions were wrong, and only reading a real payload showed it:
//
//  1. `amount` is a decimal STRING ("40.00"), not a number. Parsing it through
//     float64 reintroduces exactly the precision problem the minor-unit integers were
//     adopted to remove.
//  2. There is NO end-to-end id. `paymentData` carries only `paymentMethod` and
//     `payer`. The Pix E2E identifier — the one key that would reconcile both sides of
//     an internal transfer without guessing — is not exposed by this provider.
//  3. The provider's own PENDING/POSTED lives in a field called `status`, colliding
//     by name with our reconciliation status. They mean different things and must not
//     be conflated.

type pluggyFixture struct {
	Results []pluggyTransaction `json:"results"`
}

type pluggyTransaction struct {
	ID           string `json:"id"`
	Date         string `json:"date"`
	Description  string `json:"description"`
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
	// A STRING too, like Amount. Modelling it as float64 fails to parse — and would
	// have failed silently on exactly the foreign line the conversion exists for.
	AmountInAccountCurrency string `json:"amountInAccountCurrency"`
	Type                    string `json:"type"`
	Status                  string `json:"status"`
	OperationType           string `json:"operationType"`
}

func loadFixture(t *testing.T) pluggyFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/pluggy_transactions.json")
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	var f pluggyFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parsing the fixture: %v", err)
	}
	if len(f.Results) == 0 {
		t.Fatal("the fixture is empty")
	}
	return f
}

func TestFixture_AmountIsADecimalString(t *testing.T) {
	// If this ever parses as a number, the importer can stop doing string arithmetic.
	// Until then, going through float64 reintroduces the precision problem that the
	// minor-unit integers exist to remove.
	f := loadFixture(t)
	for _, tx := range f.Results {
		if tx.Amount == "" {
			t.Fatalf("%s: amount is not a string in the payload", tx.ID)
		}
	}
}

func TestParseMinor_ConvertsWithoutFloatingPoint(t *testing.T) {
	cases := map[string]int64{
		"40.00":    4000,
		"1.77":     177,
		"108.00":   10800,
		"-1018.18": -101818,
		"0.07":     7,
		"1234":     123400,
		"99.9":     9990,
	}
	for in, want := range cases {
		got, err := ParseMinor(in)
		if err != nil {
			t.Errorf("ParseMinor(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseMinor(%q) = %d, want %d", in, got, want)
		}
	}
	if _, err := ParseMinor("nao-e-numero"); err == nil {
		t.Error("a malformed amount must be refused, not silently zero")
	}
}

func TestFixture_ProviderStatusIsPendingOrPosted(t *testing.T) {
	// Confirms the values the domain's ProviderStatus was built around, and that the
	// field is called `status` — colliding by name with our own reconciliation status
	// while meaning something entirely different.
	f := loadFixture(t)
	seen := map[string]bool{}
	for _, tx := range f.Results {
		seen[tx.Status] = true
		if tx.Status != string(ProviderStatusPending) && tx.Status != string(ProviderStatusPosted) {
			t.Errorf("%s: unexpected provider status %q", tx.ID, tx.Status)
		}
	}
	if !seen["PENDING"] {
		t.Error("the fixture should include a PENDING line: it is the case that makes a match unverifiable")
	}
}

func TestFixture_ForeignLinesCarryTheConvertedValue(t *testing.T) {
	// The whole reason a foreign line is refused without it.
	f := loadFixture(t)
	foreign := 0
	for _, tx := range f.Results {
		if tx.CurrencyCode == "BRL" {
			continue
		}
		foreign++
		if tx.AmountInAccountCurrency == "" {
			t.Errorf("%s: %s line with no amountInAccountCurrency — reconciling it by face value is the bug this guards", tx.ID, tx.CurrencyCode)
		}
	}
	if foreign == 0 {
		t.Skip("no foreign line in this fixture")
	}
}

func TestFixture_NoEndToEndIdIsAvailable(t *testing.T) {
	// Documents a gap rather than a feature. The Pix end-to-end id appears on both
	// sides of an internal transfer and would reconcile one without guessing — it is
	// simply not in this provider's payload. `paymentData` carries paymentMethod and
	// payer only.
	//
	// Consequence: transfers between the owner's own accounts cannot be matched by a
	// network key here, and fall back to the composite deterministic key. When a
	// provider does expose it, this test is what says the column was waiting.
	raw, err := os.ReadFile("testdata/pluggy_transactions.json")
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	var probe struct {
		Results []map[string]any `json:"results"`
	}
	_ = json.Unmarshal(raw, &probe)
	for _, tx := range probe.Results {
		for _, key := range []string{"endToEndId", "endToEndID", "e2eId"} {
			if _, ok := tx[key]; ok {
				t.Errorf("the provider now exposes %s — wire it into the importer, it is the strongest key for internal transfers", key)
			}
		}
	}
}
