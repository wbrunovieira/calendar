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
	// The bills' own lines. Without them the stored totals disagree with what the
	// transactions say, which is precisely the state the use case must refuse.
	linha := func(id, billID string, amount float64) *transaction.Transaction {
		b := billID
		return &transaction.Transaction{
			ID: id, ProfileID: profileID, BankAccountID: cardID,
			Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
			Amount: amount, Currency: "BRL", Description: "compras do ciclo",
			OccurredOn: day(2026, 9, 20), InvoiceID: &b,
		}
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		credito, linha("l-set", "set", 3292.96), linha("l-out", "out", 1649.35),
	}}

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
	// Raise BOTH the stored total and the lines, or checkRestatable fires first and
	// the test passes on the wrong refusal — which is what it was doing.
	invRepo.invoices["out"].Amount = 9000
	seedBillLine(uc, "out", 9000-1649.35)

	err := uc.Execute("credito", []string{"set", "out"})

	if err == nil || err.Error() != ErrConversionExceeded.Error() {
		t.Fatalf("error = %v, want %v", err, ErrConversionExceeded)
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

	// Named, not merely refused: an open cycle is a different problem from a bill
	// that owes nothing, and the caller is told which. Letting the domain's generic
	// refusal answer instead loses that.
	if !errors.Is(err, ErrPaymentRefused) || err.Error() != ErrInvoiceStillOpen.Error() {
		t.Fatalf("error = %v, want %v", err, ErrInvoiceStillOpen)
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
	seedBillLine(uc, "nov", 1000)

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
	seedBillLine(uc, "nov", 400)

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

// Converting must not rewrite the bills' totals. Their lines did not change, and
// `amount` is a statement figure that a blind refresh replaces with whatever the
// lines happen to sum to — in production that is frequently zero.
func TestRefinanceDoesNotRestateTheBillsItSettles(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	uc.SetInvoiceRecalculator(&recordingRecalculator{})

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	rec := uc.invoiceRecalculator.(*recordingRecalculator)
	for _, id := range rec.refreshed {
		if id == "set" || id == "out" {
			t.Errorf("bill %s was restated; its lines did not change and its total is a statement figure", id)
		}
	}
	if !moneyNear(invRepo.invoices["out"].Amount, 1649.35) {
		t.Errorf("october's total is now %v, want 1649.35 preserved", invRepo.invoices["out"].Amount)
	}
}

// The cycle the credit was removed FROM did change, and must be refreshed.
func TestRefinanceRestatesTheCycleTheCreditLeft(t *testing.T) {
	uc, _ := refinanceFixture(t)
	uc.SetInvoiceRecalculator(&recordingRecalculator{})
	novID := "nov"
	uc.txRepo.(*fakeTransactionRepo).created[0].InvoiceID = &novID

	if err := uc.Execute("credito", []string{"out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	rec := uc.invoiceRecalculator.(*recordingRecalculator)
	found := false
	for _, id := range rec.refreshed {
		if id == "nov" {
			found = true
		}
	}
	if !found {
		t.Error("the cycle the credit left was not refreshed; it is still short by the credit's value")
	}
}

// seedBillLine gives a bill the charges its stored total claims. A bill with a
// total and no lines is the out-of-sync state the use case refuses, so a fixture
// without them tests the guard rather than the behaviour under it.
func seedBillLine(uc *RefinanceInvoiceUseCase, billID string, amount float64) {
	b := billID
	repo := uc.txRepo.(*fakeTransactionRepo)
	repo.created = append(repo.created, &transaction.Transaction{
		ID: "l-" + billID, ProfileID: "personal", BankAccountID: "card",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: amount, Currency: "BRL", Description: "compras do ciclo",
		OccurredOn: day(2026, 10, 20), InvoiceID: &b,
	})
}

type recordingRecalculator struct{ refreshed []string }

func (r *recordingRecalculator) RefreshTotal(id string) (*invoice.Invoice, error) {
	r.refreshed = append(r.refreshed, id)
	return nil, nil
}

func (r *recordingRecalculator) Execute(id string) (*invoice.Invoice, error) {
	return nil, nil
}

func (r *recordingRecalculator) RestatePayments(id string) (*invoice.Invoice, error) {
	return nil, nil
}

func moneyNear(got, want float64) bool { return got-want < 0.005 && want-got < 0.005 }

// The cap reads the bill's stored total, and that column is NOT the source of
// truth: it is written as zero on creation and refreshed by a manual endpoint, so
// it drifts from the lines underneath it. A credit of R$ 100 could settle a bill
// storing 100 while carrying 3.000 of confirmed charges — and RefinancedAmount
// would freeze at 100, so the cross-request cap would still believe the rest of
// the credit was unspent. The same unbounded settlement through a different door.
//
// Refusing to settle against a number that disagrees with itself is the only
// honest answer: it is the same guard MarkInvoicePayment carries.
func TestRefinanceRefusesABillWhoseTotalDisagreesWithItsLines(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	// The lines say 1.649,35; the stored total drifted to 100,00.
	invRepo.invoices["out"].Amount = 100

	err := uc.Execute("credito", []string{"out"})

	if !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal naming both numbers", err)
	}
	if invRepo.invoices["out"].Status == invoice.StatusRefinanced {
		t.Error("a bill was settled against a total nothing agrees with")
	}
}

func TestRefinanceProceedsWhenTheTotalAgreesWithItsLines(t *testing.T) {
	uc, invRepo := refinanceFixture(t)

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if invRepo.invoices["out"].Status != invoice.StatusRefinanced {
		t.Error("a bill whose total matches its lines was refused")
	}
}

// Release had no coverage above the domain: deleting its Update, or its restate,
// left the whole suite green — integration included.
func TestReleasePersistsAndRestores(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	uc.SetInvoiceRecalculator(&recordingRecalculator{})
	if err := uc.Execute("credito", []string{"out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if err := uc.Release("out"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Read through the repository: asserting on an in-memory object would pass
	// whether or not Update was called.
	stored := invRepo.invoices["out"]
	if stored.Status == invoice.StatusRefinanced {
		t.Error("the release was not persisted")
	}
	if stored.RefinancedBy != nil || stored.RefinancedAmount != nil {
		t.Error("the conversion record survived the release in storage")
	}
	if !moneyNear(stored.AmountRemaining(), 1649.35) {
		t.Errorf("remaining = %v, want the whole bill owed again", stored.AmountRemaining())
	}

	rec := uc.invoiceRecalculator.(*recordingRecalculator)
	found := false
	for _, id := range rec.refreshed {
		if id == "out" {
			found = true
		}
	}
	if !found {
		t.Error("the released bill's total was not refreshed")
	}
}

// Releasing frees the credit again: the cap must stop counting what it converted.
func TestReleaseReturnsTheCreditToTheCap(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	invRepo.invoices["nov"] = &invoice.Invoice{
		ID: "nov", BankAccountID: "card",
		OpeningDate: day(2026, 10, 10), ClosingDate: day(2026, 11, 9),
		DueDate: day(2026, 11, 14), ReferenceDate: day(2026, 11, 1),
		Status: invoice.StatusClosed, Amount: 1000,
	}
	seedBillLine(uc, "nov", 1000)

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if err := uc.Execute("credito", []string{"nov"}); err == nil {
		t.Fatal("precondition: the third bill should not fit")
	}

	if err := uc.Release("out"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if err := uc.Execute("credito", []string{"nov"}); err != nil {
		t.Fatalf("after releasing 1.649,35 the credit should cover 1.000,00: %v", err)
	}
}

func TestReleaseRefusesABillThatWasNotConverted(t *testing.T) {
	uc, _ := refinanceFixture(t)

	if err := uc.Release("out"); !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
}

// BLOCKER 1 (review 3). checkRestatable guarded the bills being converted but not
// the one bill that actually gets REWRITTEN: the cycle the credit is detached
// from. Restating a drifted bill replaces a statement figure with an incomplete
// sum — reproduced destroying R$ 54,69 of a real Mercado Pago total.
//
// The guard has to run BEFORE the detach, where the credit is still one of the
// lines; after it, the sum has legitimately changed and proves nothing.
func TestRefinanceRefusesWhenTheCreditsOwnCycleIsOutOfSync(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	novID := "nov"
	invRepo.invoices[novID] = &invoice.Invoice{
		ID: novID, BankAccountID: "card",
		OpeningDate: day(2026, 10, 10), ClosingDate: day(2026, 11, 9),
		DueDate: day(2026, 11, 14), ReferenceDate: day(2026, 11, 1),
		Status: invoice.StatusClosed, Amount: 2702.66, // statement figure
	}
	seedBillLine(uc, novID, 497.00) // lines sum 497,00 - 2.205,86 = -1.708,86: drifted
	uc.txRepo.(*fakeTransactionRepo).created[0].InvoiceID = &novID

	err := uc.Execute("credito", []string{"out"})

	if !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if invRepo.invoices["out"].Status == invoice.StatusRefinanced {
		t.Error("a conversion went through that would have rewritten a drifted statement total")
	}
	if !moneyNear(invRepo.invoices[novID].Amount, 2702.66) {
		t.Errorf("the statement total was changed to %v", invRepo.invoices[novID].Amount)
	}
}

// BLOCKER 2 (review 3). When the bill being converted is ALSO the cycle the credit
// landed in, its total nets the credit against its own charges: the cap reads the
// netted figure and lets a credit settle far more than it covers, and the bill is
// then restated back to the gross — ending REFINANCED while reporting the credit's
// full value as debt. The phantom this change exists to delete, reintroduced.
func TestRefinanceRefusesABillThatIsAlsoTheCreditsOwnCycle(t *testing.T) {
	uc, invRepo := refinanceFixture(t)
	outID := "out"
	uc.txRepo.(*fakeTransactionRepo).created[0].InvoiceID = &outID
	// Already restated after the credit landed, so the stored total IS the netted
	// figure and checkRestatable passes. That is the state the reviewer found:
	// 3.000,00 of purchases less a 2.205,86 credit stored as 794,14.
	seedBillLine(uc, outID, 1350.65) // 1.649,35 + 1.350,65 = 3.000,00 of charges
	invRepo.invoices[outID].Amount = 794.14

	err := uc.Execute("credito", []string{"out"})

	if !errors.Is(err, ErrPaymentRefused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if invRepo.invoices["out"].Status == invoice.StatusRefinanced {
		t.Error("the bill hosting the credit was converted by it")
	}
}

// The cap's boundary. One cent either side decides whether a bill is settled.
func TestRefinanceCapIsInclusiveToTheCent(t *testing.T) {
	// Exactly the credit: 2.205,86 of debt against 2.205,86 of credit.
	uc, invRepo := refinanceFixture(t)
	invRepo.invoices["out"].Amount = 2205.86 - 92.96
	seedBillLine(uc, "out", (2205.86-92.96)-1649.35)

	if err := uc.Execute("credito", []string{"set", "out"}); err != nil {
		t.Fatalf("a conversion that exactly exhausts the credit was refused: %v", err)
	}

	// One cent over.
	uc2, invRepo2 := refinanceFixture(t)
	invRepo2.invoices["out"].Amount = 2205.86 - 92.96 + 0.02
	seedBillLine(uc2, "out", (2205.86-92.96+0.02)-1649.35)

	if err := uc2.Execute("credito", []string{"set", "out"}); err == nil {
		t.Error("a conversion two cents over the credit was allowed")
	}
}

// The same bill named twice in one request must be counted once, or the cap sees
// double the debt and refuses a conversion that fits.
func TestRefinanceCountsARepeatedBillOnce(t *testing.T) {
	uc, invRepo := refinanceFixture(t)

	if err := uc.Execute("credito", []string{"out", "out"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if invRepo.invoices["out"].Status != invoice.StatusRefinanced {
		t.Error("the bill was not converted")
	}
	if invRepo.invoices["out"].RefinancedAmount == nil || !moneyNear(*invRepo.invoices["out"].RefinancedAmount, 1649.35) {
		t.Errorf("recorded conversion = %v, want 1649.35 counted once", invRepo.invoices["out"].RefinancedAmount)
	}
}
