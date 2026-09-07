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
// ProviderStatus is what the BANK says about the line's own lifecycle, as opposed to
// Status, which says what WE have done about it.
type ProviderStatus string

const (
	ProviderStatusPending ProviderStatus = "PENDING"
	ProviderStatusPosted  ProviderStatus = "POSTED"
)

// Matchable reports whether the line is settled enough to be reconciled. A pending
// authorisation still changes amount and date when it posts, so matching it asserts a
// check that the next sync invalidates.
func (l *Line) Matchable() bool {
	return l.ProviderStatus != ProviderStatusPending
}

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
	AmountAccountMinor *int64 `json:"amountAccountMinor,omitempty"`

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

	// ProviderStatus is the bank's own PENDING/POSTED. Without it, a settling
	// authorisation only reads as "the amount changed" — so every normal settlement
	// looks like an anomaly, and the rule that removes most of the noise cannot be
	// written: a PENDING line is not eligible for matching at all.
	ProviderStatus ProviderStatus `json:"providerStatus"`
	// BillID is the provider's invoice id. Card data in Open Finance comes grouped by
	// bill, not by account, and this is the join key the reconciliation will need on
	// the first card it touches.
	BillID *string `json:"billId,omitempty"`

	Status        Status  `json:"status"`
	IgnoredReason *string `json:"ignoredReason,omitempty"`
	// MatchedTransactionID names WHAT this line was reconciled against. MATCHED
	// without a referent is an assertion with nothing behind it: it cannot answer
	// "matched to which entry", and it cannot notice when that entry is later
	// reversed.
	MatchedTransactionID *string `json:"matchedTransactionId,omitempty"`

	ImportedAt time.Time `json:"importedAt"`
	// LastSeenAt is bumped by every import covering this line's window, so a row the
	// bank stopped reporting can be told apart from one nobody asked for.
	LastSeenAt time.Time `json:"lastSeenAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Revision is one version of what the provider reported for a line. Append-only:
// overwriting the payload destroys the proof that a change came from the bank.
type Revision struct {
	SeenAt             time.Time       `json:"seenAt"`
	BookedDate         time.Time       `json:"bookedDate"`
	AmountMinor        int64           `json:"amountMinor"`
	Currency           string          `json:"currency"`
	AmountAccountMinor *int64          `json:"amountAccountMinor,omitempty"`
	Description        string          `json:"description"`
	Raw                json.RawMessage `json:"raw"`
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
	Description        string
	EndToEndID         *string
	ProviderStatus     ProviderStatus
	BillID             *string
	Raw                json.RawMessage
	// AccountCurrency lets New refuse a foreign line with no converted value. Without
	// it, InAccountCurrency falls back to the face value and reconciles USD against
	// BRL — the exact bug this type was shaped to prevent, with a comment claiming
	// otherwise.
	AccountCurrency string
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

	currency := strings.ToUpper(strings.TrimSpace(params.Currency))
	accountCurrency := strings.ToUpper(strings.TrimSpace(params.AccountCurrency))
	if accountCurrency == "" {
		return nil, errors.New("accountCurrency is required to tell a foreign line from a domestic one")
	}
	if currency != accountCurrency && params.AmountAccountMinor == nil {
		return nil, errors.New("a line charged in another currency needs its converted value: reconciling by face value is how USD was read as BRL")
	}

	providerStatus := params.ProviderStatus
	if providerStatus == "" {
		providerStatus = ProviderStatusPosted
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
		Currency:           currency,
		AmountAccountMinor: params.AmountAccountMinor,
		Description:        params.Description,
		EndToEndID:         params.EndToEndID,
		ProviderStatus:     providerStatus,
		BillID:             params.BillID,
		Raw:                params.Raw,
		Status:             StatusUnmatched,
		ImportedAt:         now,
		LastSeenAt:         now,
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
// FXRate is DERIVED, never stored. The provider sends only the two amounts, so a
// stored rate would be a computed value written down — the exact pattern being removed
// from the rest of this system. Returns 0 for a domestic line.
func (l *Line) FXRate() float64 {
	if l.AmountAccountMinor == nil || l.AmountMinor == 0 {
		return 0
	}
	return float64(*l.AmountAccountMinor) / float64(l.AmountMinor)
}

// IsForeign compares currencies, not amounts: a conversion that happens to land at
// exactly 1:1 is still a foreign charge.
func (l *Line) IsForeign(accountCurrency string) bool {
	return !strings.EqualFold(l.Currency, accountCurrency)
}

// MarkMatched records what this line was reconciled against. The transaction id is
// required: a match with no referent cannot be verified, cannot be undone knowingly,
// and cannot notice when the entry behind it is reversed.
func (l *Line) MarkMatched(transactionID string) error {
	if !l.Matchable() {
		return errors.New("a pending line cannot be matched: its amount and date still move when it posts")
	}
	if strings.TrimSpace(transactionID) == "" {
		return errors.New("a match must name the transaction it was reconciled against")
	}
	l.Status = StatusMatched
	l.MatchedTransactionID = &transactionID
	l.IgnoredReason = nil
	l.UpdatedAt = time.Now()
	return nil
}

// MarkUnmatched undoes a match. Reconciliation only ever marks; it never destroys a
// line, so a wrong match costs a correction and not evidence.
func (l *Line) MarkUnmatched() {
	l.Status = StatusUnmatched
	l.MatchedTransactionID = nil
	// Cleared for the same reason MarkMatched clears it: an orphan reason left on an
	// unmatched line reads as a decision nobody made.
	l.IgnoredReason = nil
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
	l.MatchedTransactionID = nil
	l.UpdatedAt = time.Now()
	return nil
}
