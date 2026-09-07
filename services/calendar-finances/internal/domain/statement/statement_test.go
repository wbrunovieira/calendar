package statement

import (
	"encoding/json"
	"testing"
	"time"
)

// A statement line is what the BANK said, stored verbatim. Everything the
// reconciliation of 06-07/09/2026 discovered lived only in a chat transcript and
// evaporated with it — so the same months would be reconciled again, repeating the
// same mistakes.
//
// Three of those mistakes are prevented by the shape of this type alone: the amount
// carries its currency (a USD line read as BRL invented a 5.2x discrepancy), the raw
// payload is kept (Pluggy rewrites transactions, and without the original it looks
// like rows are vanishing), and matching keys off the provider's id instead of the
// amount (three R$40 charges in one month are indistinguishable by value).

func TestNew_KeepsWhatTheBankSaid(t *testing.T) {
	raw := json.RawMessage(`{"id":"plg-1","amount":"107.54","currencyCode":"USD","amountInAccountCurrency":571.38}`)

	line, err := New(CreateParams{
		AccountID:          "acc-1",
		Provider:           ProviderPluggy,
		ExternalID:         "plg-1",
		BookedDate:         time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC),
		AmountMinor:        10754,
		Currency:           "USD",
		AmountAccountMinor: func() *int64 { v := int64(57138); return &v }(),
		Raw:                raw,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if line.Status != StatusUnmatched {
		t.Errorf("status = %s, want UNMATCHED — a freshly imported line is not reconciled", line.Status)
	}
	if line.Currency != "USD" {
		t.Error("the original currency must survive: reading USD as BRL is what invented a 5.2x discrepancy with a supplier")
	}
	if line.AmountAccountMinor == nil || *line.AmountAccountMinor != 57138 {
		t.Error("the converted amount must be stored alongside, not instead of, the original")
	}
	if len(line.Raw) == 0 {
		t.Error("the raw payload must be kept: Pluggy rewrites transactions, and without it that looks like rows vanishing")
	}
}

func TestNew_RejectsALineWithoutCurrency(t *testing.T) {
	_, err := New(CreateParams{
		AccountID: "acc-1", Provider: ProviderPluggy, ExternalID: "x",
		BookedDate: time.Now(), AmountMinor: 100,
	})
	if err == nil {
		t.Error("no monetary column may exist without its currency beside it")
	}
}

func TestNew_RejectsALineWithoutAProviderID(t *testing.T) {
	// The provider id is the strong matching key. Without it a line can only be
	// matched by amount, which is how a legitimate entry was deleted.
	_, err := New(CreateParams{
		AccountID: "acc-1", Provider: ProviderPluggy,
		BookedDate: time.Now(), AmountMinor: 100, Currency: "BRL",
	})
	if err == nil {
		t.Error("a statement line without the provider's id cannot be reconciled safely")
	}
}

func TestAmountInAccountCurrency_FallsBackToTheOriginalWhenSameCurrency(t *testing.T) {
	line, err := New(CreateParams{
		AccountID: "acc-1", Provider: ProviderPluggy, ExternalID: "y",
		BookedDate: time.Now(), AmountMinor: 9990, Currency: "BRL",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := line.InAccountCurrency(); got != 9990 {
		t.Errorf("= %d, want 9990: a domestic line needs no conversion", got)
	}
}

func TestInAccountCurrency_UsesTheConvertedValueForForeignLines(t *testing.T) {
	converted := int64(57138)
	line, _ := New(CreateParams{
		AccountID: "acc-1", Provider: ProviderPluggy, ExternalID: "z",
		BookedDate: time.Now(), AmountMinor: 10754, Currency: "USD",
		AmountAccountMinor: &converted,
	})
	if got := line.InAccountCurrency(); got != 57138 {
		t.Errorf("= %d, want 57138 — never reconcile a foreign line by its face value", got)
	}
}

func TestMarkMatched_AndUnmatch(t *testing.T) {
	line, _ := New(CreateParams{
		AccountID: "acc-1", Provider: ProviderPluggy, ExternalID: "m",
		BookedDate: time.Now(), AmountMinor: 100, Currency: "BRL",
	})

	line.MarkMatched()
	if line.Status != StatusMatched {
		t.Errorf("status = %s, want MATCHED", line.Status)
	}
	line.MarkUnmatched()
	if line.Status != StatusUnmatched {
		t.Errorf("status = %s, want UNMATCHED — a match can be undone without losing the line", line.Status)
	}
}

func TestMarkIgnored_RequiresAReason(t *testing.T) {
	// Ignoring a line is a decision someone made. Without the motive it is
	// indistinguishable from a line nobody looked at.
	line, _ := New(CreateParams{
		AccountID: "acc-1", Provider: ProviderPluggy, ExternalID: "i",
		BookedDate: time.Now(), AmountMinor: 100, Currency: "BRL",
	})
	if err := line.MarkIgnored(""); err == nil {
		t.Error("ignoring a statement line without a reason must be refused")
	}
	if err := line.MarkIgnored("mecanica do rotativo, par que se anula"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line.Status != StatusIgnored || line.IgnoredReason == nil {
		t.Error("the reason must survive alongside the status")
	}
}
