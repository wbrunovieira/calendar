package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A card whose stored cycles are shifted five days from the closing day it actually
// has — the shape found on the real Nubank Juridica card.
func shiftedCycleFixture(t *testing.T) (*ApplyInvoiceCyclePlanUseCase, *fakeInvoiceRepo, *fakeTransactionRepo, string) {
	t.Helper()
	const cardID, profileID = "card", "wb"
	card := creditCardAccountWith(profileID, cardID, 27, 3)

	shifted := &invoice.Invoice{
		ID: "inv-shifted", BankAccountID: cardID,
		OpeningDate:   day(2026, 1, 2),
		ClosingDate:   day(2026, 2, 1),
		DueDate:       day(2026, 2, 3),
		ReferenceDate: day(2026, 1, 1),
		Status:        invoice.StatusOpen,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{shifted.ID: shifted}}

	// Dated 29/01: inside the canonical window (27/12–27/01 closes before it, so it
	// belongs to the NEXT cycle) — this is a purchase that has to move.
	moving := &transaction.Transaction{
		ID: "moving", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 98, Currency: "BRL", Description: "Google Workspace",
		OccurredOn: day(2026, 1, 29), InvoiceID: &shifted.ID,
	}
	staying := &transaction.Transaction{
		ID: "staying", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 10, Currency: "BRL", Description: "Tavily",
		OccurredOn: day(2026, 1, 10), InvoiceID: &shifted.ID,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{moving, staying}}

	uc := NewApplyInvoiceCyclePlanUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo, invRepo,
	)
	return uc, invRepo, txRepo, cardID
}

// Nothing is applied that was not named. Same model as the reattach repair: the plan
// enumerates, a person reads it, and apply names the cycles.
func TestApplyCycles_RefusesWithoutAnExplicitList(t *testing.T) {
	uc, invRepo, _, cardID := shiftedCycleFixture(t)
	before := *invRepo.invoices["inv-shifted"]

	if _, err := uc.Execute(cardID, nil); err == nil {
		t.Fatal("apply ran with no list: it was allowed to reshape settled bills on its own")
	}
	if !invRepo.invoices["inv-shifted"].ClosingDate.Equal(before.ClosingDate) {
		t.Fatal("a refused apply still moved a window")
	}
}

// Reshaping the window is only half the repair. A purchase that now falls outside
// the window it was filed under has to follow, or it stays on a bill that no longer
// covers its date — which is the very defect the reshape exists to fix.
func TestApplyCycles_ReshapesTheWindowAndMovesWhatNoLongerFits(t *testing.T) {
	uc, invRepo, txRepo, cardID := shiftedCycleFixture(t)

	report, err := uc.Execute(cardID, []string{"2026-01-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Reshaped != 1 {
		t.Fatalf("reshaped = %d, want 1 (%+v)", report.Reshaped, report)
	}

	got := invRepo.invoices["inv-shifted"]
	if !got.OpeningDate.Equal(day(2025, 12, 27)) || !got.ClosingDate.Equal(day(2026, 1, 27)) {
		t.Fatalf("window is %s..%s, want 2025-12-27..2026-01-27",
			got.OpeningDate.Format("2006-01-02"), got.ClosingDate.Format("2006-01-02"))
	}

	var moving, staying *transaction.Transaction
	for _, txn := range txRepo.created {
		switch txn.ID {
		case "moving":
			moving = txn
		case "staying":
			staying = txn
		}
	}
	if moving.InvoiceID != nil && *moving.InvoiceID == "inv-shifted" {
		t.Fatal("a purchase dated after the new closing stayed on the bill that no longer covers it")
	}
	if staying.InvoiceID == nil || *staying.InvoiceID != "inv-shifted" {
		t.Fatal("a purchase still inside the window was moved off its bill")
	}
	if report.Moved != 1 {
		t.Fatalf("moved = %d, want 1", report.Moved)
	}
}

// A cycle whose only difference is the DUE date reflects the card's terms as they
// were. Rewriting it replaces a historical fact with today's configuration, so the
// plan marks it informational and apply must refuse it even when named.
func TestApplyCycles_RefusesAnInformationalAction(t *testing.T) {
	const cardID, profileID = "card", "wb"
	card := creditCardAccountWith(profileID, cardID, 27, 3)
	// Right days, only the due date differs from what today's settings would produce.
	termsOnly := &invoice.Invoice{
		ID: "inv-terms", BankAccountID: cardID,
		OpeningDate:   day(2025, 11, 27),
		ClosingDate:   day(2025, 12, 27),
		DueDate:       day(2026, 1, 6),
		ReferenceDate: day(2025, 12, 1),
		Status:        invoice.StatusPaid,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{termsOnly.ID: termsOnly}}
	uc := NewApplyInvoiceCyclePlanUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		&fakeTransactionRepo{}, invRepo,
	)

	report, err := uc.Execute(cardID, []string{"2025-12-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Reshaped != 0 {
		t.Fatal("an informational action was applied: a historical due date was rewritten with today's settings")
	}
	if report.RefusedInformational != 1 {
		t.Fatalf("refusedInformational = %d, want 1", report.RefusedInformational)
	}
	if !invRepo.invoices["inv-terms"].DueDate.Equal(day(2026, 1, 6)) {
		t.Fatal("the stored due date was changed")
	}
}

// Naming one cycle must not reshape the others the plan found alongside it.
func TestApplyCycles_TouchesOnlyWhatWasNamed(t *testing.T) {
	uc, invRepo, _, cardID := shiftedCycleFixture(t)
	other := &invoice.Invoice{
		ID: "inv-other", BankAccountID: cardID,
		OpeningDate:   day(2026, 2, 2),
		ClosingDate:   day(2026, 3, 1),
		DueDate:       day(2026, 3, 3),
		ReferenceDate: day(2026, 2, 1),
		Status:        invoice.StatusOpen,
	}
	invRepo.invoices[other.ID] = other

	if _, err := uc.Execute(cardID, []string{"2026-01-01"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !invRepo.invoices["inv-other"].ClosingDate.Equal(day(2026, 3, 1)) {
		t.Fatal("a cycle nobody named was reshaped")
	}
}

// Running it twice changes nothing the second time: the windows are already
// canonical, so the plan has nothing left to propose for them.
func TestApplyCycles_IsIdempotent(t *testing.T) {
	uc, _, _, cardID := shiftedCycleFixture(t)

	if _, err := uc.Execute(cardID, []string{"2026-01-01"}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := uc.Execute(cardID, []string{"2026-01-01"})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.Reshaped != 0 || second.Moved != 0 {
		t.Fatalf("second run reshaped=%d moved=%d; it must find nothing left to do",
			second.Reshaped, second.Moved)
	}
}

func TestApplyCycles_RefusesAnAccountThatIsNotACard(t *testing.T) {
	uc := NewApplyInvoiceCyclePlanUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
			"checking": {ID: "checking", ProfileID: "wb", Name: "Nubank",
				Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true},
		}},
		&fakeTransactionRepo{}, &fakeInvoiceRepo{},
	)
	_, err := uc.Execute("checking", []string{"2026-01-01"})
	if err == nil {
		t.Fatal("a checking account has no cycles; running this on one is a mistake worth refusing")
	}
	// The planner would fail here too, but with a message about cycles rather than
	// about the mistake the caller actually made. Asserting the sentinel is what
	// makes the guard mean something.
	if !errors.Is(err, ErrNotACreditCard) {
		t.Fatalf("err = %v, want ErrNotACreditCard", err)
	}
}
