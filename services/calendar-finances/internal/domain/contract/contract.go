// Package contract mirrors a deal that was decided in another system — today the
// CRM. Nothing here is a source of truth about the sale; the sale happened over
// there. What the ledger owns is what it DID about that deal, and for that it needs
// a stable local record to hang receivables and divergences on.
package contract

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status is the deal's lifecycle as the sender reports it.
type Status string

const (
	StatusOpen Status = "OPEN"
	StatusWon  Status = "WON"
	StatusLost Status = "LOST"
)

// ErrStale marks state that the contract has already moved past. It is not a
// failure: an out-of-order redelivery is expected traffic, and the correct answer
// is to drop it, not to retry it.
var ErrStale = errors.New("contract already carries newer remote state")

// ErrNotFound is returned by a repository when no contract mirrors that reference.
var ErrNotFound = errors.New("contract not found")

// Contract is the local mirror of a remote deal.
type Contract struct {
	ID           string `json:"id"`
	ProfileID    string `json:"profileId"`
	CostCenterID string `json:"costCenterId"`
	// Source names the system the deal came from, so two systems can never collide
	// on the same external id.
	Source     string `json:"source"`
	ExternalID string `json:"externalId"`
	Title      string `json:"title"`
	// TotalMinor is the value in minor units. Integer on purpose: the one question
	// this field exists to answer on a resync is "did the value change?", and
	// equality on floats is the wrong tool for it.
	TotalMinor *int64     `json:"totalMinor,omitempty"`
	Currency   string     `json:"currency,omitempty"`
	Status     Status     `json:"status"`
	ClosedAt   *time.Time `json:"closedAt,omitempty"`
	// RemoteUpdatedAt is the SENDER's clock. It is kept because our own UpdatedAt
	// says when we wrote, which says nothing about which of two deliveries carries
	// the newer state.
	RemoteUpdatedAt time.Time `json:"remoteUpdatedAt"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CreateParams is the full remote state plus the local references it resolves to.
type CreateParams struct {
	ProfileID       string
	CostCenterID    string
	Source          string
	ExternalID      string
	Title           string
	TotalMinor      *int64
	Currency        string
	Status          Status
	ClosedAt        *time.Time
	RemoteUpdatedAt time.Time
}

// RemoteState is what a later delivery carries: the deal as it stands now. The
// sender transmits complete state rather than a delta, so absence is meaningful —
// a nil TotalMinor means the value was removed, not that it was omitted.
type RemoteState struct {
	Title      string
	TotalMinor *int64
	Currency   string
	Status     Status
	ClosedAt   *time.Time
	UpdatedAt  time.Time
}

// Change records one field that moved, in a form a human report can print.
type Change struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// New validates the remote state and creates the local mirror.
func New(p CreateParams) (*Contract, error) {
	if err := validate(p.ProfileID, p.CostCenterID, p.Source, p.ExternalID, p.Title,
		p.TotalMinor, p.Currency, p.Status, p.RemoteUpdatedAt); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Contract{
		ID:              uuid.New().String(),
		ProfileID:       strings.TrimSpace(p.ProfileID),
		CostCenterID:    strings.TrimSpace(p.CostCenterID),
		Source:          strings.TrimSpace(p.Source),
		ExternalID:      strings.TrimSpace(p.ExternalID),
		Title:           strings.TrimSpace(p.Title),
		TotalMinor:      p.TotalMinor,
		Currency:        normaliseCurrency(p.Currency),
		Status:          p.Status,
		ClosedAt:        p.ClosedAt,
		RemoteUpdatedAt: p.RemoteUpdatedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// IsStale reports whether a delivery stamped remoteUpdatedAt carries state this
// contract has already passed. An EQUAL stamp counts as stale: the sender's
// updatedAt changes on every write, so the same stamp is the same write, and
// treating it as new would let a redelivery rewrite what is already stored.
func (c *Contract) IsStale(remoteUpdatedAt time.Time) bool {
	return !remoteUpdatedAt.After(c.RemoteUpdatedAt)
}

// Apply writes newer remote state onto the contract and reports what moved.
// It validates BEFORE mutating, so a rejected delivery leaves the contract exactly
// as it was rather than half-written.
func (c *Contract) Apply(r RemoteState) ([]Change, error) {
	if c.IsStale(r.UpdatedAt) {
		return nil, ErrStale
	}
	if err := validate(c.ProfileID, c.CostCenterID, c.Source, c.ExternalID, r.Title,
		r.TotalMinor, r.Currency, r.Status, r.UpdatedAt); err != nil {
		return nil, err
	}

	currency := normaliseCurrency(r.Currency)
	title := strings.TrimSpace(r.Title)

	changes := make([]Change, 0, 4)
	if title != c.Title {
		changes = append(changes, Change{"title", c.Title, title})
	}
	if !sameMinor(c.TotalMinor, r.TotalMinor) {
		changes = append(changes, Change{"totalValue", formatMinor(c.TotalMinor), formatMinor(r.TotalMinor)})
	}
	if currency != c.Currency {
		changes = append(changes, Change{"currency", c.Currency, currency})
	}
	if r.Status != c.Status {
		changes = append(changes, Change{"status", string(c.Status), string(r.Status)})
	}

	c.Title = title
	c.TotalMinor = r.TotalMinor
	c.Currency = currency
	c.Status = r.Status
	c.ClosedAt = r.ClosedAt
	c.RemoteUpdatedAt = r.UpdatedAt
	c.UpdatedAt = time.Now()
	return changes, nil
}

// Refile moves the contract to another client and reports the move.
//
// A deal reassigned to a different organization in the CRM has to follow here. The
// alternative is worse than it sounds: the contract stays filed under the previous
// client, so one company's revenue is attributed to another's, and nothing in the
// change list says it happened.
func (c *Contract) Refile(costCenterID string) *Change {
	id := strings.TrimSpace(costCenterID)
	if id == "" || id == c.CostCenterID {
		return nil
	}
	moved := &Change{"costCenterId", c.CostCenterID, id}
	c.CostCenterID = id
	c.UpdatedAt = time.Now()
	return moved
}

// ParseStatus reads the sender's spelling. The CRM writes lowercase; accepting it
// here keeps the translation in one place instead of at every call site.
func ParseStatus(raw string) (Status, error) {
	switch Status(strings.ToUpper(strings.TrimSpace(raw))) {
	case StatusOpen:
		return StatusOpen, nil
	case StatusWon:
		return StatusWon, nil
	case StatusLost:
		return StatusLost, nil
	default:
		return "", fmt.Errorf("unknown deal status %q", raw)
	}
}

// ValidateRemote checks everything the SENDER is responsible for, independently of
// the local references the ledger resolves. It is exported so a caller can reject a
// bad delivery BEFORE creating the client it would have been filed under: validating
// only inside New leaves an orphan cost center behind every rejected payload.
func ValidateRemote(r RemoteState) error {
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required")
	}
	switch r.Status {
	case StatusOpen, StatusWon, StatusLost:
	default:
		return fmt.Errorf("unknown deal status %q", r.Status)
	}
	if r.UpdatedAt.IsZero() {
		return errors.New("remote updatedAt is required: without it deliveries cannot be ordered")
	}
	if r.TotalMinor != nil && *r.TotalMinor < 0 {
		return errors.New("total value cannot be negative")
	}
	// The currency is checked whether or not there is a value to denominate. It used
	// to be checked only alongside a value, which let "DOLLAR" reach a CHAR(3) column
	// and come back as a database error — a payload that can never work, answered
	// with 500, which tells the sender to retry it forever.
	currency := normaliseCurrency(r.Currency)
	switch {
	case currency == "" && r.TotalMinor != nil:
		return errors.New("a valued contract needs a 3-letter currency")
	case currency != "" && !isCurrencyCode(currency):
		return fmt.Errorf("currency %q is not a 3-letter code", r.Currency)
	}
	return nil
}

func isCurrencyCode(c string) bool {
	if len(c) != 3 {
		return false
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func validate(profileID, costCenterID, source, externalID, title string,
	totalMinor *int64, currency string, status Status, remoteUpdatedAt time.Time) error {
	for _, f := range []struct {
		name, value string
	}{
		{"profileID", profileID},
		{"costCenterID", costCenterID},
		{"source", source},
		{"externalID", externalID},
	} {
		if strings.TrimSpace(f.value) == "" {
			return fmt.Errorf("%s is required", f.name)
		}
	}
	return ValidateRemote(RemoteState{
		Title: title, TotalMinor: totalMinor, Currency: currency,
		Status: status, UpdatedAt: remoteUpdatedAt,
	})
}

func normaliseCurrency(c string) string { return strings.ToUpper(strings.TrimSpace(c)) }

func sameMinor(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func formatMinor(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
