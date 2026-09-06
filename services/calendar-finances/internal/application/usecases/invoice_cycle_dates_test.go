package usecases

import (
	"strings"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
)

// Regression tests for the Nubank Juridica card (closingDay 27, dueDay 6), whose
// September invoice was created spanning TWO billing cycles: opening 2026-07-28,
// closing 2026-09-27, due 2026-10-06. The cycle that should have closed on
// 2026-08-27 and fallen due on 2026-09-06 never existed, so a real payment of
// R$ 119,94 made on 2026-09-03 had no invoice to land on.
//
// The trigger is a reference-month collision. Older rows label an invoice by the
// month it FALLS DUE, while invoice.New labels it by the month it CLOSES. When the
// two conventions meet, Create hits the unique constraint on
// (bank_account_id, reference_date), and the old fallback reacted by moving the
// whole cycle a month forward and stretching openingDate back to cover the gap —
// silently merging two cycles into one.

const (
	testClosingDay = 27
	testDueDay     = 6
)

func cardWithCycle(id string) *bankaccount.BankAccount {
	closing, due := testClosingDay, testDueDay
	return &bankaccount.BankAccount{
		ID:         id,
		Type:       bankaccount.AccountTypeCreditCard,
		ClosingDay: &closing,
		DueDay:     &due,
	}
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// legacyInvoice mirrors the row already in production: labelled by its DUE month
// (2026-08-01) while covering the cycle that closed on 2026-07-27.
func legacyInvoice(cardID string) *invoice.Invoice {
	return &invoice.Invoice{
		ID:            "legacy-invoice",
		BankAccountID: cardID,
		ReferenceDate: date(2026, time.August, 1),
		OpeningDate:   date(2026, time.June, 27),
		ClosingDate:   date(2026, time.July, 27),
		DueDate:       date(2026, time.August, 6),
		Status:        invoice.StatusClosed,
	}
}

// spansMoreThanOneCycle reports whether an invoice covers more than a single
// billing period. With closingDay 27, a healthy cycle is roughly one month;
// anything past ~45 days means two cycles were merged.
func spansMoreThanOneCycle(inv *invoice.Invoice) bool {
	return inv.ClosingDate.Sub(inv.OpeningDate) > 45*24*time.Hour
}

func TestGetOrCreateInvoice_NeverMergesTwoBillingCycles(t *testing.T) {
	cardID := "card-1"
	repo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}}
	repo.invoices["legacy-invoice"] = legacyInvoice(cardID)

	// A purchase inside the cycle that should close on 2026-08-27.
	txDate := date(2026, time.August, 10)

	inv, err := getOrCreateInvoiceForDate(repo, cardWithCycle(cardID), txDate)

	if err == nil && spansMoreThanOneCycle(inv) {
		t.Fatalf("invoice merged two billing cycles: opening %s, closing %s, due %s",
			inv.OpeningDate.Format("2006-01-02"),
			inv.ClosingDate.Format("2006-01-02"),
			inv.DueDate.Format("2006-01-02"))
	}

	// Whatever the outcome, no stored invoice may span two cycles.
	for _, stored := range repo.invoices {
		if spansMoreThanOneCycle(stored) {
			t.Fatalf("stored invoice %s spans two cycles: opening %s, closing %s",
				stored.ID,
				stored.OpeningDate.Format("2006-01-02"),
				stored.ClosingDate.Format("2006-01-02"))
		}
	}
}

func TestGetOrCreateInvoice_RelabelsInsteadOfMovingTheCycle(t *testing.T) {
	cardID := "card-1"
	repo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}}
	repo.invoices["legacy-invoice"] = legacyInvoice(cardID)

	inv, err := getOrCreateInvoiceForDate(repo, cardWithCycle(cardID), date(2026, time.August, 10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The billing period is what matters, and it must be the real one.
	if got, want := inv.OpeningDate, date(2026, time.July, 27); !got.Equal(want) {
		t.Errorf("openingDate = %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
	if got, want := inv.ClosingDate, date(2026, time.August, 27); !got.Equal(want) {
		t.Errorf("closingDate = %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
	if got, want := inv.DueDate, date(2026, time.September, 6); !got.Equal(want) {
		t.Errorf("dueDate = %s, want %s — this is the cycle whose absence left the 2026-09-03 payment homeless",
			got.Format("2006-01-02"), want.Format("2006-01-02"))
	}

	// Only the label moved, borrowing the older due-month convention.
	if got := inv.ReferenceDate; got.Month() != time.September || got.Year() != 2026 {
		t.Errorf("referenceDate = %s, want 2026-09 (labelled by due month to free the collision)",
			got.Format("2006-01"))
	}

	// And the legacy invoice was left exactly as it was.
	legacy := repo.invoices["legacy-invoice"]
	if !legacy.ClosingDate.Equal(date(2026, time.July, 27)) {
		t.Error("the conflicting invoice must not be modified")
	}
}

func TestGetOrCreateInvoice_ReportsConflictWhenBothLabelsAreTaken(t *testing.T) {
	cardID := "card-1"
	repo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}}
	repo.invoices["legacy-invoice"] = legacyInvoice(cardID)
	// Something else already squats on the due-month label with a different cycle,
	// so neither convention frees a slot.
	repo.invoices["squatter"] = &invoice.Invoice{
		ID:            "squatter",
		BankAccountID: cardID,
		ReferenceDate: date(2026, time.September, 1),
		OpeningDate:   date(2026, time.November, 27),
		ClosingDate:   date(2026, time.December, 27),
		DueDate:       date(2027, time.January, 6),
		Status:        invoice.StatusOpen,
	}

	_, err := getOrCreateInvoiceForDate(repo, cardWithCycle(cardID), date(2026, time.August, 10))
	if err == nil {
		t.Fatal("expected the unresolvable conflict to be reported, got nil error")
	}
	if !strings.Contains(err.Error(), "legacy-invoice") {
		t.Errorf("error should name the conflicting invoice so it can be repaired, got: %v", err)
	}
	for _, stored := range repo.invoices {
		if spansMoreThanOneCycle(stored) {
			t.Fatalf("no invoice may be left spanning two cycles, %s does", stored.ID)
		}
	}
}

func TestGetOrCreateInvoice_CreatesSingleCycleWhenNoConflict(t *testing.T) {
	cardID := "card-1"
	repo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}}

	inv, err := getOrCreateInvoiceForDate(repo, cardWithCycle(cardID), date(2026, time.August, 10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := inv.ClosingDate, date(2026, time.August, 27); !got.Equal(want) {
		t.Errorf("closingDate = %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
	if got, want := inv.DueDate, date(2026, time.September, 6); !got.Equal(want) {
		t.Errorf("dueDate = %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
	if !inv.ContainsDate(date(2026, time.August, 10)) {
		t.Error("invoice must contain the transaction date it was created for")
	}
}

func TestGetOrCreateInvoice_ReusesInvoiceCoveringTheDate(t *testing.T) {
	cardID := "card-1"
	repo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{}}
	existing := &invoice.Invoice{
		ID:            "already-there",
		BankAccountID: cardID,
		ReferenceDate: date(2026, time.August, 1),
		OpeningDate:   date(2026, time.July, 27),
		ClosingDate:   date(2026, time.August, 27),
		DueDate:       date(2026, time.September, 6),
		Status:        invoice.StatusOpen,
	}
	repo.invoices[existing.ID] = existing

	inv, err := getOrCreateInvoiceForDate(repo, cardWithCycle(cardID), date(2026, time.August, 10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.ID != existing.ID {
		t.Errorf("should reuse the invoice covering the date, got %s", inv.ID)
	}
	if len(repo.invoices) != 1 {
		t.Errorf("no new invoice should be created, have %d", len(repo.invoices))
	}
}
