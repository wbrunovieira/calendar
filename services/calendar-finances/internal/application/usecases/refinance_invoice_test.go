package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// The fixture is the Mercado Pago card as it actually stood on 11/10/2026. The
// bank settled two cycles at once: a cash payment of 2.700,00 left 92,96 owed on
// the September bill, and a single credit of 2.205,86 — "Crédito por parcelamento
// da fatura" — cleared that remainder together with the whole October bill,
// charging the money again as six instalments of 633,35.
//
// Nothing was paid by that credit. Recording it as a payment debits an account
// that reconciles to the cent against the bank; leaving it unrecorded leaves two
// bills owing 1.742,31 that the bank considers closed, while the instalments it
// became are also on the books. The same debt, twice.
func refinanceFixture(t *testing.T) (*RefinanceInvoiceUseCase, *fakeInvoiceRepo) {
	t.Helper()
	const cardID, profileID = "card", "personal"
	card := creditCardAccountWith(profileID, cardID, 9, 14)

	setembro := &invoice.Invoice{
		ID: "set", BankAccountID: cardID,
		OpeningDate: day(2026, 8, 10), ClosingDate: day(2026, 9, 9),
		DueDate: day(2026, 9, 14), ReferenceDate: day(2026, 9, 1),
		Status: invoice.StatusPartiallyPaid, Amount: 3292.96,
		PaidAmount: ptrFloat(3200.00),
	}
	outubro := &invoice.Invoice{
		ID: "out", BankAccountID: cardID,
		OpeningDate: day(2026, 9, 10), ClosingDate: day(2026, 10, 9),
		DueDate: day(2026, 10, 14), ReferenceDate: day(2026, 10, 1),
		Status: invoice.StatusClosed, Amount: 1649.35,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		setembro.ID: setembro, outubro.ID: outubro,
	}}

	credito := &transaction.Transaction{
		ID: "credito", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 2205.86, Currency: "BRL", Description: "Credito por parcelamento da fatura",
		OccurredOn: day(2026, 10, 11),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{credito}}

	uc := NewRefinanceInvoiceUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo, invRepo,
	)
	return uc, invRepo
}

func ptrFloat(v float64) *float64 { return &v }

func TestRefinanceSettlesEveryBillTheConversionCovered(t *testing.T) {
	uc, invRepo := refinanceFixture(t)

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, id := range []string{"set", "out"} {
		got := invRepo.invoices[id]
		if got.Status != invoice.StatusRefinanced {
			t.Errorf("%s: status = %q, want REFINANCED", id, got.Status)
		}
		if got.AmountRemaining() != 0 {
			t.Errorf("%s: still owes %.2f, want 0", id, got.AmountRemaining())
		}
		if got.RefinancedBy == nil || *got.RefinancedBy != "credito" {
			t.Errorf("%s: refinanced by = %v, want credito", id, got.RefinancedBy)
		}
	}
	// The cash payment really happened and must survive the conversion.
	if p := invRepo.invoices["set"].PaidAmount; p == nil || *p != 3200.00 {
		t.Errorf("september paid amount = %v, want 3200.00 kept", p)
	}
}

// The guard that makes this safe to expose. Converting more debt than the bank
// actually credited is how a forced settlement would hide a bill nobody cleared.
func TestRefinanceRefusesMoreDebtThanTheCreditCovers(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	invRepo.invoices["out"].Amount = 9000 // far beyond the 2.205,86 credit

	err := uc.Execute("credito", []string{"set", "out"})

	if !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	for _, id := range []string{"set", "out"} {
		if invRepo.invoices[id].Status == invoice.StatusRefinanced {
			t.Errorf("%s was converted despite the refusal", id)
		}
	}
}

// All or nothing. A conversion that settles the first bill and then refuses the
// second leaves the ledger in a state the bank never produced.
func TestRefinanceLeavesNothingHalfDoneWhenOneBillIsIneligible(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	invRepo.invoices["out"].Status = invoice.StatusOpen // still accruing charges

	err := uc.Execute("credito", []string{"set", "out"})

	if err == nil {
		t.Fatal("expected a refusal")
	}
	if invRepo.invoices["set"].Status == invoice.StatusRefinanced {
		t.Error("september was converted even though the batch failed")
	}
}

func TestRefinanceRefusesABillFromAnotherCard(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	invRepo.invoices["out"].BankAccountID = "another-card"

	if err := uc.Execute("credito", []string{"out"}); !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
}

func TestRefinanceRefusesADebitAsTheConversion(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	_ = invRepo

	if err := uc.Execute("missing", []string{"out"}); err == nil {
		t.Fatal("expected an unknown transaction to be refused")
	}
}

func TestRefinanceRequiresAtLeastOneBill(t *testing.T) {
	uc, _ := refinanceFixture(t)

	if err := uc.Execute("credito", nil); !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
}

// Re-running must not convert a second time, and must not fail either: a retried
// request should land on the same state as the first.
func TestRefinanceIsIdempotentForTheSameConversion(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if *invRepo.invoices["out"].RefinancedBy != "credito" {
		t.Error("the recorded conversion changed on the retry")
	}
}

// A different credit must not take over a bill another one already converted.
func TestRefinanceRefusesToReassignABillToAnotherConversion(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	if err := uc.Execute("credito", []string{"out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	invRepo.invoices["out"].RefinancedBy = strPtr("outro-credito")

	if err := uc.Execute("credito", []string{"out"}); err == nil {
		t.Fatal("expected a bill converted by another credit to be refused")
	}
}

// The cap has to hold across REQUESTS, not only within one. Spending the same
// credit a few bills at a time passed every batch while settling far more debt
// than the bank ever credited — a forced settlement in instalments.
func TestRefinanceCapsOneCreditAcrossSeparateRequests(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	// 2.205,86 of credit against 92,96 + 1.649,35 = 1.742,31 owed: the first two
	// fit. A third bill of 1.000,00 does not, even though on its own it would.
	invRepo.invoices["nov"] = &invoice.Invoice{
		ID: "nov", BankAccountID: "card",
		OpeningDate: day(2026, 10, 10), ClosingDate: day(2026, 11, 9),
		DueDate: day(2026, 11, 14), ReferenceDate: day(2026, 11, 1),
		Status: invoice.StatusClosed, Amount: 1000,
	}

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("first request: %v", err)
	}

	err := uc.Execute("credito", []string{"nov"})

	if !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("second request: error = %v, want a refusal", err)
	}
	if invRepo.invoices["nov"].Status == invoice.StatusRefinanced {
		t.Error("the third bill was converted beyond what the credit covered")
	}
}

// Spending the rest of a credit later is legitimate, and must still work.
func TestRefinanceAllowsASecondRequestWithinWhatIsLeft(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	invRepo.invoices["nov"] = &invoice.Invoice{
		ID: "nov", BankAccountID: "card",
		OpeningDate: day(2026, 10, 10), ClosingDate: day(2026, 11, 9),
		DueDate: day(2026, 11, 14), ReferenceDate: day(2026, 11, 1),
		Status: invoice.StatusClosed, Amount: 400,
	}

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("first request: %v", err)
	}
	// 2.205,86 - 1.742,31 = 463,55 left, which covers 400,00.
	if err := uc.Execute("credito", []string{"nov"}); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if invRepo.invoices["nov"].Status != invoice.StatusRefinanced {
		t.Error("a conversion within the remaining credit was refused")
	}
}

// Money that never moved cannot settle debt that did.
func TestRefinanceRefusesACreditThatIsNotConfirmed(t *testing.T) {
	for _, status := range []transaction.Status{
		transaction.StatusPlanned, transaction.StatusCancelled, transaction.StatusReversed,
	} {
		uc, invRepo := refinanceFixture(t)
		txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{{
			ID: "credito", ProfileID: "personal", BankAccountID: "card",
			Type: transaction.TypeIncome, Status: status, Amount: 2205.86,
			Currency: "BRL", OccurredOn: day(2026, 10, 11),
		}}}
		uc.txRepo = txRepo

		if err := uc.Execute("credito", []string{"out"}); !errors.Is(err, ErrPaymentRefused) {
			t.Errorf("%s: error = %v, want a refusal", status, err)
		}
		if invRepo.invoices["out"].Status == invoice.StatusRefinanced {
			t.Errorf("%s: a bill was converted by a credit that is not confirmed", status)
		}
	}
}

func TestRefinanceRefusesADebitOrATransferAsTheConversion(t *testing.T) {
	for _, kind := range []transaction.Type{transaction.TypeExpense, transaction.TypeTransfer} {
		uc, invRepo := refinanceFixture(t)
		uc.txRepo = &fakeTransactionRepo{created: []*transaction.Transaction{{
			ID: "credito", ProfileID: "personal", BankAccountID: "card",
			Type: kind, Status: transaction.StatusConfirmed, Amount: 2205.86,
			Currency: "BRL", OccurredOn: day(2026, 10, 11),
		}}}

		if err := uc.Execute("credito", []string{"out"}); !errors.Is(err, ErrPaymentRefused) {
			t.Errorf("%s: error = %v, want a refusal", kind, err)
		}
		if invRepo.invoices["out"].Status == invoice.StatusRefinanced {
			t.Errorf("%s: a bill was converted by something that is not a credit", kind)
		}
	}
}

// The conversion is dated by the bank's own posting, not by the clock.
func TestRefinanceIsDatedByTheCredit(t *testing.T) {
	uc, invRepo := refinanceFixture(t)

	if err := uc.Execute("credito", []string{"out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	at := invRepo.invoices["out"].RefinancedAt
	if at == nil || !at.Equal(day(2026, 10, 11)) {
		t.Errorf("refinanced at = %v, want the credit's own date 11/10/2026", at)
	}
}

// The conversion credit is INCOME on the card, so the date rule files it as a LINE
// of whatever cycle is open — where SumByInvoiceID scores a credit as -amount. The
// November bill came out R$ 2.205,86 lighter: phantom CREDIT, the mirror image of
// the phantom debt this whole change removes.
//
// It is the same failure MarkInvoicePayment documents and fixes by detaching the
// row. A credit that settles bills is not a line of one.
func TestRefinanceDetachesTheCreditFromTheCycleItLandedIn(t *testing.T) {
	uc, _ := refinanceFixture(t)
	novID := "nov"
	uc.txRepo.(*fakeTransactionRepo).created[0].InvoiceID = &novID

	if err := uc.Execute("credito", []string{"out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Read it back through the repository: the fake hands out copies, so asserting
	// on the seeded pointer would pass whether or not anything was persisted.
	stored, err := uc.txRepo.GetByID("credito")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if stored.InvoiceID != nil {
		t.Errorf("the credit is still a line of bill %v; it would deflate that cycle by its full value", *stored.InvoiceID)
	}
}

func TestRefinanceLeavesACreditThatWasNotFiledAsALineAlone(t *testing.T) {
	uc, _ := refinanceFixture(t)

	if err := uc.Execute("credito", []string{"out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	stored, err := uc.txRepo.GetByID("credito")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if stored.InvoiceID != nil {
		t.Error("expected the credit to stay detached")
	}
}
