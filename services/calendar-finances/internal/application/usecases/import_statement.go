package usecases

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

// issuerLocation is the timezone the card and bank statements are printed in.
//
// Pluggy returns UTC timestamps and this system runs in Sao Paulo, so a purchase made
// at 23:30 on the 30th arrives as 02:30 on the 31st. Comparing that naively makes the
// charge missing from one month and extra in the next — which is the exact shape of a
// phantom, and the thing a reconciliation then spends an hour chasing.
var issuerLocation = mustLoadIssuerLocation()

func mustLoadIssuerLocation() *time.Location {
	if loc, err := time.LoadLocation("America/Sao_Paulo"); err == nil {
		return loc
	}
	// A container without tzdata must not silently fall back to UTC, which would
	// reintroduce the off-by-one-day this exists to prevent.
	return time.FixedZone("BRT", -3*60*60)
}

// ImportStatementInput is one pull of one account's statement.
type ImportStatementInput struct {
	// AccountID is the account in THIS system, already resolved from the provider's.
	AccountID string
	Provider  statement.Provider
	// AccountKind decides how to read the provider's sign: the same DEBIT means
	// opposite things on a card and on a checking account.
	AccountKind statement.AccountKind
	// AccountCurrency is what the account is denominated in, so a foreign line can be
	// told from a domestic one.
	AccountCurrency string
	// Payload is the provider's response, untouched.
	Payload []byte
}

// RejectedLine is a line the importer would not store, and why.
type RejectedLine struct {
	ExternalID  string `json:"externalId"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
}

// ImportStatementOutput separates "the import ran" from "the import brought
// anything" — different facts that a single count would blur.
type ImportStatementOutput struct {
	Inserted int            `json:"inserted"`
	Updated  int            `json:"updated"`
	Rejected []RejectedLine `json:"rejected"`
}

// ImportStatementUseCase records what the bank said, and only that.
//
// It classifies nothing and posts nothing: importing is the half that cannot be wrong,
// and keeping it separate is what makes the other half safe to be careful about. A
// line nobody can categorise is still stored, so it shows up on a list instead of
// living in a chat transcript — which is how a R$ 53,90 subscription and a R$ 116,03
// phone bill sat unnoticed for weeks.
type ImportStatementUseCase struct {
	lines statement.Repository
}

func NewImportStatementUseCase(lines statement.Repository) *ImportStatementUseCase {
	return &ImportStatementUseCase{lines: lines}
}

// providerLine is the subset of the provider's record this importer reads. Everything
// else survives in Raw.
type providerLine struct {
	ID                      string          `json:"id"`
	Date                    string          `json:"date"`
	Description             string          `json:"description"`
	Amount                  json.RawMessage `json:"amount"`
	CurrencyCode            string          `json:"currencyCode"`
	Type                    string          `json:"type"`
	Status                  string          `json:"status"`
	AmountInAccountCurrency json.RawMessage `json:"amountInAccountCurrency"`
	PaymentData             *struct {
		EndToEndID string `json:"endToEndId"`
	} `json:"paymentData"`
	BillID string `json:"billId"`
}

func (uc *ImportStatementUseCase) Execute(input ImportStatementInput) (*ImportStatementOutput, error) {
	if strings.TrimSpace(input.AccountID) == "" {
		return nil, errors.New("accountID is required: a statement line belongs to an account")
	}
	if strings.TrimSpace(input.AccountCurrency) == "" {
		return nil, errors.New("accountCurrency is required to tell a foreign line from a domestic one")
	}

	records, err := decodeProviderRecords(input.Payload)
	if err != nil {
		return nil, err
	}

	out := &ImportStatementOutput{Rejected: []RejectedLine{}}
	lines := make([]*statement.Line, 0, len(records))

	for _, raw := range records {
		var rec providerLine
		if err := json.Unmarshal(raw, &rec); err != nil {
			out.Rejected = append(out.Rejected, RejectedLine{Reason: "unreadable record: " + err.Error()})
			continue
		}

		line, err := uc.toLine(input, rec, raw)
		if err != nil {
			// Reported, never dropped. A line the importer cannot trust is exactly
			// the one somebody has to look at.
			out.Rejected = append(out.Rejected, RejectedLine{
				ExternalID: rec.ID, Description: rec.Description, Reason: err.Error(),
			})
			continue
		}
		lines = append(lines, line)
	}

	if len(lines) > 0 {
		inserted, updated, err := uc.lines.UpsertMany(lines)
		if err != nil {
			return nil, err
		}
		out.Inserted, out.Updated = inserted, updated
	}
	return out, nil
}

func (uc *ImportStatementUseCase) toLine(input ImportStatementInput, rec providerLine, raw json.RawMessage) (*statement.Line, error) {
	bookedDate, err := parseIssuerDate(rec.Date)
	if err != nil {
		return nil, err
	}

	faceMinor, err := statement.ParseMinor(decodeAmount(rec.Amount))
	if err != nil {
		return nil, fmt.Errorf("amount: %w", err)
	}
	// Normalise before storing, so everything downstream compares one convention:
	// money leaving is negative, money arriving is positive, on any account.
	faceMinor = statement.NormalizeSign(faceMinor, input.AccountKind, strings.ToUpper(rec.Type))

	currency := strings.ToUpper(strings.TrimSpace(rec.CurrencyCode))
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(input.AccountCurrency))
	}

	var accountMinor *int64
	if converted := decodeAmount(rec.AmountInAccountCurrency); converted != "" {
		value, err := statement.ParseMinor(converted)
		if err != nil {
			return nil, fmt.Errorf("amountInAccountCurrency: %w", err)
		}
		value = statement.NormalizeSign(value, input.AccountKind, strings.ToUpper(rec.Type))
		accountMinor = &value
	}

	var endToEnd *string
	if rec.PaymentData != nil && strings.TrimSpace(rec.PaymentData.EndToEndID) != "" {
		e2e := rec.PaymentData.EndToEndID
		endToEnd = &e2e
	}
	var billID *string
	if strings.TrimSpace(rec.BillID) != "" {
		b := rec.BillID
		billID = &b
	}

	providerStatus := statement.ProviderStatusPosted
	if strings.EqualFold(rec.Status, string(statement.ProviderStatusPending)) {
		providerStatus = statement.ProviderStatusPending
	}

	return statement.New(statement.CreateParams{
		AccountID:          input.AccountID,
		Provider:           input.Provider,
		ExternalID:         rec.ID,
		BookedDate:         bookedDate,
		AmountMinor:        faceMinor,
		Currency:           currency,
		AmountAccountMinor: accountMinor,
		Description:        strings.TrimSpace(rec.Description),
		EndToEndID:         endToEnd,
		ProviderStatus:     providerStatus,
		BillID:             billID,
		Raw:                raw,
		AccountCurrency:    input.AccountCurrency,
	})
}

// decodeProviderRecords accepts both the wrapped shape the API returns and a bare
// array, because the same payload reaches here from a live call and from a fixture.
func decodeProviderRecords(payload []byte) ([]json.RawMessage, error) {
	if len(payload) == 0 {
		return nil, errors.New("empty payload: nothing to import")
	}
	var wrapped struct {
		Results      []json.RawMessage `json:"results"`
		Transactions []json.RawMessage `json:"transactions"`
		Result       *struct {
			Results []json.RawMessage `json:"results"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &wrapped); err == nil {
		switch {
		case len(wrapped.Results) > 0:
			return wrapped.Results, nil
		case len(wrapped.Transactions) > 0:
			return wrapped.Transactions, nil
		case wrapped.Result != nil && len(wrapped.Result.Results) > 0:
			return wrapped.Result.Results, nil
		}
	}
	var bare []json.RawMessage
	if err := json.Unmarshal(payload, &bare); err == nil {
		return bare, nil
	}
	return nil, errors.New("payload carries no recognisable list of transactions")
}

// decodeAmount reads a value the provider may send as a string or as a number, and
// returns it as text so ParseMinor can read it without a float ever touching it.
func decodeAmount(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" {
		return ""
	}
	return trimmed
}

// parseIssuerDate converts the provider's UTC instant to the calendar day the issuer
// would print, and keeps only that day.
func parseIssuerDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New("date is required")
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		local := parsed.In(issuerLocation)
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("date %q is in no format this importer reads", value)
}
