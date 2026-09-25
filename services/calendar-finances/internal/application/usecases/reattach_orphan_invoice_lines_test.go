package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func orphanFixture(t *testing.T) (*ReattachOrphanInvoiceLinesUseCase, *fakeTransactionRepo, *fakeInvoiceRepo, string) {
	t.Helper()
	const cardID, profileID = "card", "personal"
	card := creditCardAccountWith(profileID, cardID, 9, 14)

	march := &invoice.Invoice{
		ID: "inv-march", BankAccountID: cardID,
		OpeningDate:   time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
		ReferenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Status:        invoice.StatusPaid,
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{march.ID: march}}

	credit := &transaction.Transaction{
		ID: "credit", ProfileID: profileID, BankAccountID: cardID,
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 139.93, Currency: "BRL", Description: "Credito concedido Mercado Pago",
		OccurredOn: time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{credit}}

	uc := NewReattachOrphanInvoiceLinesUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{cardID: card}},
		txRepo, invRepo,
	)
	return uc, txRepo, invRepo, cardID
}

// The repair itself. A credit that belongs to a bill but points at nothing gets
// attached to the bill whose cycle contains its date — no date is touched, because
// the date is what the bank statement says and it is not the thing that is wrong.
func TestReattach_AttachesAnOrphanCreditToItsCycle(t *testing.T) {
	uc, txRepo, _, cardID := orphanFixture(t)

	report, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Attached != 1 {
		t.Fatalf("attached = %d, want 1 (%+v)", report.Attached, report)
	}
	got := txRepo.created[0]
	if got.InvoiceID == nil || *got.InvoiceID != "inv-march" {
		t.Fatalf("invoiceID = %v, want inv-march", got.InvoiceID)
	}
	if !got.OccurredOn.Equal(time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("the repair moved the date; the date came from the bank and was never wrong")
	}
}

// Dry run is the default for a reason: this walks production money. It must report
// exactly what it would do and write nothing.
func TestReattach_DryRunWritesNothing(t *testing.T) {
	uc, txRepo, _, cardID := orphanFixture(t)

	report, err := uc.Execute(cardID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Attached != 1 {
		t.Fatalf("a dry run must still report what it would attach, got %+v", report)
	}
	if txRepo.created[0].InvoiceID != nil {
		t.Fatal("dry run wrote")
	}
	if txRepo.updates != 0 {
		t.Fatalf("dry run called Update %d times", txRepo.updates)
	}
}

// Running it twice must not do anything the second time. A repair that is not
// idempotent cannot be re-run after a partial failure, which is exactly when it
// needs to be.
func TestReattach_IsIdempotent(t *testing.T) {
	uc, _, _, cardID := orphanFixture(t)

	if _, err := uc.Execute(cardID, true); err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.Attached != 0 {
		t.Fatalf("second run attached %d; it must find nothing left to do", second.Attached)
	}
}

// An invoice payment is not a line of the bill. Attaching one would turn a settled
// bill into a smaller bill — the single most damaging thing this tool could do.
func TestReattach_NeverTouchesAnInvoicePayment(t *testing.T) {
	uc, txRepo, _, cardID := orphanFixture(t)
	paid := "inv-march"
	txRepo.created = append(txRepo.created, &transaction.Transaction{
		ID: "payment", ProfileID: "personal", BankAccountID: cardID,
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 500, Currency: "BRL", Description: "Pagamento fatura",
		OccurredOn:    time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		PaidInvoiceID: &paid,
	})

	report, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.SkippedPayments != 1 {
		t.Fatalf("skippedPayments = %d, want 1", report.SkippedPayments)
	}
	for _, txn := range txRepo.created {
		if txn.ID == "payment" && txn.InvoiceID != nil {
			t.Fatal("a payment was filed into the bill: the bill now reads as smaller instead of paid")
		}
	}
}

// No invoice covers the date: report it, do not invent one. Creating bills for old
// periods is a separate decision with its own consequences.
func TestReattach_ReportsWhatHasNoBillInsteadOfCreatingOne(t *testing.T) {
	uc, txRepo, invRepo, cardID := orphanFixture(t)
	txRepo.created[0].OccurredOn = time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)

	report, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Attached != 0 || report.NoInvoiceForDate != 1 {
		t.Fatalf("report = %+v, want nothing attached and one uncovered", report)
	}
	if len(invRepo.invoices) != 1 {
		t.Fatal("the repair created a bill for a period nobody asked it to")
	}
}

// A reversed or cancelled row is history, not a line of any bill.
func TestReattach_IgnoresRowsThatDoNotCount(t *testing.T) {
	uc, txRepo, _, cardID := orphanFixture(t)
	for _, status := range []transaction.Status{transaction.StatusReversed, transaction.StatusCancelled} {
		txRepo.created = append(txRepo.created, &transaction.Transaction{
			ID: "dead-" + string(status), ProfileID: "personal", BankAccountID: cardID,
			Type: transaction.TypeIncome, Status: status, Amount: 10, Currency: "BRL",
			Description: "estorno desfeito",
			OccurredOn:  time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
		})
	}
	report, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Attached != 1 {
		t.Fatalf("attached = %d, want only the live credit", report.Attached)
	}
}

func TestReattach_RefusesAnAccountThatIsNotACard(t *testing.T) {
	uc := NewReattachOrphanInvoiceLinesUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
			"checking": {ID: "checking", ProfileID: "personal", Name: "MP",
				Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true},
		}},
		&fakeTransactionRepo{}, &fakeInvoiceRepo{},
	)
	if _, err := uc.Execute("checking", true); err == nil {
		t.Fatal("a checking account has no bills; running this on one is a mistake worth refusing")
	}
}

func TestReattach_ReportsAFailedReadInsteadOfClaimingNothingToDo(t *testing.T) {
	uc := NewReattachOrphanInvoiceLinesUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{}},
		&fakeTransactionRepo{}, &fakeInvoiceRepo{},
	)
	_, err := uc.Execute("nao-existe", true)
	if err == nil {
		t.Fatal("a missing account was reported as a clean run")
	}
	if errors.Is(err, nil) {
		t.Fatal("unreachable")
	}
}

// Attaching a line changes what the bill is worth, and the STORED total is the
// number anyone actually reads. An OPEN bill is refreshed in place.
func TestReattach_RefreshesTheStoredTotalOfAnOpenBill(t *testing.T) {
	uc, _, invRepo, cardID := orphanFixture(t)
	march := invRepo.invoices["inv-march"]
	march.Status = invoice.StatusOpen
	march.Amount = 0 // stale: the credit was never counted

	report, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Recalculated) != 1 || report.Recalculated[0] != "inv-march" {
		t.Fatalf("recalculated = %+v, want inv-march", report.Recalculated)
	}
	if got := invRepo.invoices["inv-march"].Amount; got != -139.93 {
		t.Fatalf("stored total = %.2f, want -139.93 (a lone credit and nothing else)", got)
	}
}

// A SETTLED bill is reported, never rewritten. Recomputing the total of a paid bill
// rewrites history, and the recalculation refuses it on purpose — so the repair
// surfaces the gap with both numbers and leaves the decision to a person.
func TestReattach_ReportsASettledBillInsteadOfRewritingIt(t *testing.T) {
	uc, _, invRepo, cardID := orphanFixture(t)
	march := invRepo.invoices["inv-march"]
	march.Status = invoice.StatusPaid
	march.Amount = 2023.46

	report, err := uc.Execute(cardID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Recalculated) != 0 {
		t.Fatalf("a settled bill was rewritten: %+v", report.Recalculated)
	}
	if len(report.StaleTotals) != 1 {
		t.Fatalf("staleTotals = %+v, want the settled bill reported", report.StaleTotals)
	}
	stale := report.StaleTotals[0]
	if stale.Stored != 2023.46 || stale.Computed != -139.93 {
		t.Fatalf("reported stored=%.2f computed=%.2f", stale.Stored, stale.Computed)
	}
	if invRepo.invoices["inv-march"].Amount != 2023.46 {
		t.Fatal("the settled bill's stored total was changed")
	}
}

// Only ONE failure from the refresh is a normal outcome: the bill is settled. Any
// other failure is a failure, and must reach the caller — reporting it as a stale
// total would announce a clean repair over a broken one.
func TestReattach_PropagatesARefreshFailureThatIsNotASettledBill(t *testing.T) {
	uc, _, invRepo, cardID := orphanFixture(t)
	invRepo.invoices["inv-march"].Status = invoice.StatusOpen
	// Readable but not writable: the refresh fails on the WRITE, so the fallback
	// path that reports a settled bill would otherwise succeed and hide it.
	invRepo.updateErr = errors.New("dial tcp: connection refused")

	_, err := uc.Execute(cardID, true)
	if err == nil {
		t.Fatal("a failed refresh was reported as a successful repair")
	}
	if errors.Is(err, ErrInvoiceAlreadyPaid) {
		t.Fatal("an unrelated failure was classified as a settled bill")
	}
}
