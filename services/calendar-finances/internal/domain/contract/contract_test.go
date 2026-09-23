package contract

import (
	"testing"
	"time"
)

func minor(v int64) *int64 { return &v }

func validParams() CreateParams {
	return CreateParams{
		ProfileID:       "profile-1",
		CostCenterID:    "cc-1",
		Source:          "wb-crm",
		ExternalID:      "deal-1",
		Title:           "Site institucional",
		TotalMinor:      minor(35000),
		Currency:        "BRL",
		Status:          StatusOpen,
		RemoteUpdatedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
	}
}

func TestNewContractAcceptsAValidDeal(t *testing.T) {
	c, err := New(validParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == "" {
		t.Fatal("a contract without an id cannot be referenced by anything")
	}
	if c.Status != StatusOpen {
		t.Fatalf("status = %q, want OPEN", c.Status)
	}
}

func TestNewContractRejectsWhatCannotBeResolvedLater(t *testing.T) {
	cases := map[string]func(*CreateParams){
		"no profile":     func(p *CreateParams) { p.ProfileID = "  " },
		"no cost center": func(p *CreateParams) { p.CostCenterID = "" },
		"no source":      func(p *CreateParams) { p.Source = "" },
		"no external id": func(p *CreateParams) { p.ExternalID = " " },
		"no title":       func(p *CreateParams) { p.Title = "" },
		"bad status":     func(p *CreateParams) { p.Status = "pending" },
		"negative value": func(p *CreateParams) { p.TotalMinor = minor(-1) },
		"no stamp":       func(p *CreateParams) { p.RemoteUpdatedAt = time.Time{} },
		"valued, no ccy": func(p *CreateParams) { p.Currency = "" },
		"malformed ccy":  func(p *CreateParams) { p.Currency = "brazil" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := validParams()
			mutate(&p)
			if _, err := New(p); err == nil {
				t.Fatalf("%s was accepted", name)
			}
		})
	}
}

// A deal with no value yet is legitimate — the CRM sends totalValue null while the
// proposal is still being written — and it must not require a currency it has no
// amount to denominate.
func TestNewContractAcceptsADealWithNoValueYet(t *testing.T) {
	p := validParams()
	p.TotalMinor = nil
	p.Currency = ""
	c, err := New(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.TotalMinor != nil {
		t.Fatal("a valueless deal must not invent a zero")
	}
}

func TestNewContractNormalises(t *testing.T) {
	p := validParams()
	p.Title = "  Site institucional  "
	p.Currency = "brl"
	c, err := New(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Title != "Site institucional" {
		t.Fatalf("title = %q, not trimmed", c.Title)
	}
	if c.Currency != "BRL" {
		t.Fatalf("currency = %q, want BRL", c.Currency)
	}
}

func TestParseStatusAcceptsTheSendersLowercase(t *testing.T) {
	for in, want := range map[string]Status{
		"open": StatusOpen, "won": StatusWon, "lost": StatusLost,
		"OPEN": StatusOpen, " Won ": StatusWon,
	} {
		got, err := ParseStatus(in)
		if err != nil {
			t.Fatalf("ParseStatus(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseStatus(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := ParseStatus("abandoned"); err == nil {
		t.Fatal("an unknown status was accepted; the ledger would act on a state it cannot read")
	}
}

// Ordering is the whole reason RemoteUpdatedAt is stored. An equal stamp counts as
// already seen: the sender's updatedAt changes on every write, so the same stamp is
// the same write, and treating it as new would make a redelivery rewrite state.
func TestIsStale(t *testing.T) {
	c, _ := New(validParams())
	base := c.RemoteUpdatedAt

	if !c.IsStale(base.Add(-time.Second)) {
		t.Fatal("an older delivery must be dropped")
	}
	if !c.IsStale(base) {
		t.Fatal("a redelivery of the same write must be dropped")
	}
	if c.IsStale(base.Add(time.Second)) {
		t.Fatal("a newer delivery must be applied")
	}
}

func TestApplyReportsWhatActuallyChanged(t *testing.T) {
	c, _ := New(validParams())
	newer := c.RemoteUpdatedAt.Add(time.Hour)
	closed := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)

	changes, err := c.Apply(RemoteState{
		Title:      "Site institucional",
		TotalMinor: minor(42000),
		Currency:   "BRL",
		Status:     StatusWon,
		ClosedAt:   &closed,
		UpdatedAt:  newer,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := map[string]Change{}
	for _, ch := range changes {
		got[ch.Field] = ch
	}
	if _, ok := got["title"]; ok {
		t.Fatal("title did not change; reporting it as a change would bury the real ones")
	}
	if got["totalValue"].From != "35000" || got["totalValue"].To != "42000" {
		t.Fatalf("totalValue change = %+v", got["totalValue"])
	}
	if got["status"].From != "OPEN" || got["status"].To != "WON" {
		t.Fatalf("status change = %+v", got["status"])
	}
	if c.TotalMinor == nil || *c.TotalMinor != 42000 {
		t.Fatal("Apply did not write the new value")
	}
	if c.Status != StatusWon || c.ClosedAt == nil || !c.ClosedAt.Equal(closed) {
		t.Fatal("Apply did not write the new status/closedAt")
	}
	if !c.RemoteUpdatedAt.Equal(newer) {
		t.Fatal("Apply did not advance the ordering stamp; the next redelivery would be applied twice")
	}
}

func TestApplyOnIdenticalStateReportsNothing(t *testing.T) {
	c, _ := New(validParams())
	changes, err := c.Apply(RemoteState{
		Title:      c.Title,
		TotalMinor: minor(35000),
		Currency:   "BRL",
		Status:     StatusOpen,
		UpdatedAt:  c.RemoteUpdatedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("changes = %+v, want none", changes)
	}
}

// Dropping a value is a change, and it is the one most worth reporting: a contract
// that had R$350 and now has none silently removes a receivable from the forecast.
// The sender keeps the currency on the deal even when the value is cleared, so this
// is what a real "value removed" delivery looks like.
func TestApplyReportsAValueThatDisappeared(t *testing.T) {
	c, _ := New(validParams())
	changes, err := c.Apply(RemoteState{
		Title:     c.Title,
		Currency:  "BRL",
		Status:    StatusOpen,
		UpdatedAt: c.RemoteUpdatedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 || changes[0].Field != "totalValue" || changes[0].To != "" {
		t.Fatalf("changes = %+v, want totalValue -> empty", changes)
	}
	if c.TotalMinor != nil {
		t.Fatal("Apply did not clear the value")
	}
}

// State is complete, never a delta, so an absent currency means absent — it is not
// silently carried over from what was stored. Reporting the wipe is the point: a
// contract that loses its denomination is a defect at the sender worth seeing.
func TestApplyDoesNotCarryOverAnAbsentCurrency(t *testing.T) {
	c, _ := New(validParams())
	changes, err := c.Apply(RemoteState{
		Title:     c.Title,
		Status:    StatusOpen,
		UpdatedAt: c.RemoteUpdatedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Currency != "" {
		t.Fatalf("currency = %q, want it cleared rather than merged from the stored state", c.Currency)
	}
	var sawCurrency bool
	for _, ch := range changes {
		if ch.Field == "currency" {
			sawCurrency = true
		}
	}
	if !sawCurrency {
		t.Fatal("the currency was wiped without being reported")
	}
}

func TestApplyRefusesStaleState(t *testing.T) {
	c, _ := New(validParams())
	before := *c
	_, err := c.Apply(RemoteState{
		Title: "rewritten by an old delivery", Status: StatusLost,
		UpdatedAt: c.RemoteUpdatedAt.Add(-time.Hour),
	})
	if err != ErrStale {
		t.Fatalf("err = %v, want ErrStale", err)
	}
	if c.Title != before.Title || c.Status != before.Status {
		t.Fatal("a refused Apply still mutated the contract")
	}
}

func TestApplyRejectsStateItCannotStore(t *testing.T) {
	c, _ := New(validParams())
	_, err := c.Apply(RemoteState{
		Title: "", Status: StatusOpen, UpdatedAt: c.RemoteUpdatedAt.Add(time.Hour),
	})
	if err == nil {
		t.Fatal("an empty title was accepted on update but rejected on create")
	}
}
