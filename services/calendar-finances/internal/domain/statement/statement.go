// Package statement holds what the BANK said, verbatim, before anyone interprets it.
//
// It exists because the reconciliation of 06-07/09/2026 lived entirely in a chat
// transcript: what had been checked, against which criterion, and what was left over
// evaporated with the session. The same months would be reconciled again, and the same
// mistakes made again.
package statement

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Provider names where a line came from. Kept explicit so a line's meaning does not
// depend on guessing which importer produced it.
type Provider string

const (
	ProviderPluggy  Provider = "PLUGGY"
	ProviderOFX     Provider = "OFX"
	ProviderCSV     Provider = "CSV"
	ProviderBinance Provider = "BINANCE"
)

// Status tracks a line through reconciliation. It describes the LINE, not the money:
// the bank's word is a fact, and these say what we have done about it.
type Status string

const (
	StatusUnmatched Status = "UNMATCHED"
	StatusMatched   Status = "MATCHED"
	// StatusIgnored is for lines that are deliberately not expected to have a
	// counterpart — the mirrored legs of a revolving-credit rollover, for instance,
	// which cancel each other out. It always carries a reason.
	StatusIgnored Status = "IGNORED"
)

// Line is one row of a bank statement, as the bank reported it.
//
// Amounts are stored in MINOR UNITS (centavos) as integers. Money in float64 was
// already forcing half-a-centavo tolerances elsewhere in this codebase, and a
// tolerance on value is how a real difference becomes "acceptable rounding" and
// disappears.
type Line struct {
	ID         string   `json:"id"`
	AccountID  string   `json:"accountId"`
	Provider   Provider `json:"provider"`
	ExternalID string   `json:"externalId"`

	// BookedDate is the date in the ISSUER's timezone, computed at import. Pluggy
	// returns UTC timestamps while this system runs in America/Sao_Paulo, so a
	// purchase at 21h on the 30th becomes the 1st of the next month if compared
	// naively — showing up as "missing" in one month and "extra" in the next.
	BookedDate time.Time  `json:"bookedDate"`
	ValueDate  *time.Time `json:"valueDate,omitempty"`

	// AmountMinor is in Currency, the currency the bank charged in.
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
	// AmountAccountMinor is the same movement in the ACCOUNT's currency. Present only
	// for foreign lines. Reconcile with this, never with AmountMinor.
	AmountAccountMinor *int64   `json:"amountAccountMinor,omitempty"`
	FXRate             *float64 `json:"fxRate,omitempty"`

	Description string `json:"description"`
	// EndToEndID is the Pix network identifier, when there is one. It appears on BOTH
	// sides of an internal transfer, which makes it the only key that reconciles a
	// transfer between two of the owner's own accounts without guessing.
	EndToEndID *string `json:"endToEndId,omitempty"`

	// Raw is the provider's payload, untouched. Pluggy rewrites transactions —
	// categories change, PENDING becomes POSTED sometimes under a new id, rows vanish
	// when the bank reverses them. Without the original, that reads as our system
	// losing data.
	Raw json.RawMessage `json:"raw"`

	Status        Status  `json:"status"`
	IgnoredReason *string `json:"ignoredReason,omitempty"`

	ImportedAt time.Time `json:"importedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type CreateParams struct {
	AccountID          string
	Provider           Provider
	ExternalID         string
	BookedDate         time.Time
	ValueDate          *time.Time
	AmountMinor        int64
	Currency           string
	AmountAccountMinor *int64
	FXRate             *float64
	Description        string
	EndToEndID         *string
	Raw                json.RawMessage
}

func New(params CreateParams) (*Line, error) {
	if params.AccountID == "" {
		return nil, errors.New("accountID is required")
	}
	if params.Provider == "" {
		return nil, errors.New("provider is required")
	}
	// The provider's id is the strong matching key: deterministic, one-to-one, and it
	// covers the overwhelming majority of lines. Without it a line can only be matched
	// by amount, which is what led to deleting a legitimate entry.
	if strings.TrimSpace(params.ExternalID) == "" {
		return nil, errors.New("externalID is required: a line without it can only be matched by amount")
	}
	if params.BookedDate.IsZero() {
		return nil, errors.New("bookedDate is required")
	}
	// No monetary value without its currency beside it.
	if strings.TrimSpace(params.Currency) == "" {
		return nil, errors.New("currency is required")
	}

	now := time.Now()
	return &Line{
		ID:                 uuid.New().String(),
		AccountID:          params.AccountID,
		Provider:           params.Provider,
		ExternalID:         params.ExternalID,
		BookedDate:         params.BookedDate,
		ValueDate:          params.ValueDate,
		AmountMinor:        params.AmountMinor,
		Currency:           strings.ToUpper(params.Currency),
		AmountAccountMinor: params.AmountAccountMinor,
		FXRate:             params.FXRate,
		Description:        params.Description,
		EndToEndID:         params.EndToEndID,
		Raw:                params.Raw,
		Status:             StatusUnmatched,
		ImportedAt:         now,
		UpdatedAt:          now,
	}, nil
}

// InAccountCurrency is the value to reconcile against, always in the account's own
// currency. A foreign line reconciled by its face value is how USD 107,54 was read as
// R$ 107,54 and a 5.2x discrepancy with a supplier was invented.
func (l *Line) InAccountCurrency() int64 {
	if l.AmountAccountMinor != nil {
		return *l.AmountAccountMinor
	}
	return l.AmountMinor
}

// IsForeign reports whether the line was charged in a currency other than the
// account's, meaning InAccountCurrency carries a conversion.
func (l *Line) IsForeign() bool {
	return l.AmountAccountMinor != nil && *l.AmountAccountMinor != l.AmountMinor
}

func (l *Line) MarkMatched() {
	l.Status = StatusMatched
	l.IgnoredReason = nil
	l.UpdatedAt = time.Now()
}

// MarkUnmatched undoes a match. Reconciliation only ever marks; it never destroys a
// line, so a wrong match costs a correction and not evidence.
func (l *Line) MarkUnmatched() {
	l.Status = StatusUnmatched
	l.UpdatedAt = time.Now()
}

// MarkIgnored records that this line is deliberately not expected to have a
// counterpart. The reason is required: without it, an ignored line is
// indistinguishable from one nobody looked at.
func (l *Line) MarkIgnored(reason string) error {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return errors.New("ignoring a statement line requires a reason")
	}
	l.Status = StatusIgnored
	l.IgnoredReason = &trimmed
	l.UpdatedAt = time.Now()
	return nil
}
