package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Invariants for the failures this system actually had. Each one names the bug it
// would have caught, so a future reader can tell whether it still earns its place.
//
// All read-only: a drift is a transaction to hunt down, never a number to overwrite.

func cycleFixture(t *testing.T) (*fakeAccountRepo, *fakeTransactionRepo, *cycleInvoiceRepo) {
	t.Helper()
	closing, due := 27, 3
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": {ID: "card", ProfileID: "p1", Name: "Cartao", Type: bankaccount.AccountTypeCreditCard,
			ClosingDay: &closing, DueDay: &due, Currency: "BRL"},
	}}
	return accounts, &fakeTransactionRepo{}, &cycleInvoiceRepo{}
}

type cycleInvoiceRepo struct {
	fakeInvoiceRepo
	list []*invoice.Invoice
}

func (r *cycleInvoiceRepo) FindByBankAccountID(string) ([]*invoice.Invoice, error) {
	return r.list, nil
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// I5: two invoices on the same card must not cover overlapping periods.
//
// The Nubank Juridica card had one invoice spanning 2026-07-28 to 2026-09-27 because
// a label collision made the code stretch a cycle instead of relabelling it. A real
// payment then had no invoice to land on.
func TestInvariants_OverlappingCyclesAreReported(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "a", BankAccountID: "card", OpeningDate: day(2026, time.July, 27), ClosingDate: day(2026, time.August, 27), DueDate: day(2026, time.September, 3)},
		{ID: "b", BankAccountID: "card", OpeningDate: day(2026, time.August, 1), ClosingDate: day(2026, time.September, 27), DueDate: day(2026, time.October, 3)},
	}

	got, err := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.CycleDrifts) == 0 {
		t.Fatal("two invoices covering the same days must be reported")
	}
	if got.OK {
		t.Error("overlapping cycles are actionable, so the check must not report OK")
	}
}

// I5 again, from the other side: consecutive cycles must not leave a gap. A purchase
// falling into one belongs to no invoice at all.
func TestInvariants_GapBetweenCyclesIsReported(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "a", BankAccountID: "card", OpeningDate: day(2026, time.June, 27), ClosingDate: day(2026, time.July, 27), DueDate: day(2026, time.August, 3)},
		// Opens a day late: 2026-07-27 belongs to nothing.
		{ID: "b", BankAccountID: "card", OpeningDate: day(2026, time.July, 28), ClosingDate: day(2026, time.August, 27), DueDate: day(2026, time.September, 3)},
	}

	got, _ := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if len(got.CycleDrifts) == 0 {
		t.Fatal("a day covered by no invoice must be reported")
	}
}

func TestInvariants_HealthyCyclesReportNothing(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "a", BankAccountID: "card", OpeningDate: day(2026, time.June, 27), ClosingDate: day(2026, time.July, 27), DueDate: day(2026, time.August, 3)},
		{ID: "b", BankAccountID: "card", OpeningDate: day(2026, time.July, 27), ClosingDate: day(2026, time.August, 27), DueDate: day(2026, time.September, 3)},
	}

	got, _ := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if len(got.CycleDrifts) != 0 {
		t.Errorf("cycles that tile the timeline must report nothing, got %v", got.CycleDrifts)
	}
}

// I7: an instalment series must be complete. The plan writes rows one at a time with
// no transaction around the loop, so a failure at instalment 7 of 12 leaves six
// committed and nobody looking.
func TestInvariants_IncompleteInstalmentSeriesIsReported(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	total := 3
	for _, n := range []int{1, 3} { // 2 never landed
		num := n
		txRepo.created = append(txRepo.created, &transaction.Transaction{
			ID: "i" + string(rune('0'+n)), ProfileID: "p1", BankAccountID: "card",
			Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
			Amount: 284.49, Currency: "BRL", Description: "Parcelamento fatura",
			OccurredOn:        day(2026, time.September, 3),
			InstallmentNumber: &num, InstallmentTotal: &total,
		})
	}

	got, _ := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if len(got.InstallmentDrifts) == 0 {
		t.Fatal("a series missing instalment 2 of 3 must be reported")
	}
	if got.OK {
		t.Error("a half-written instalment plan is actionable")
	}
}

func TestInvariants_CompleteInstalmentSeriesReportsNothing(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	total := 3
	for n := 1; n <= 3; n++ {
		num := n
		txRepo.created = append(txRepo.created, &transaction.Transaction{
			ID: "ok" + string(rune('0'+n)), ProfileID: "p1", BankAccountID: "card",
			Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
			Amount: 284.49, Currency: "BRL", Description: "Parcelamento fatura",
			OccurredOn:        day(2026, time.September, 3),
			InstallmentNumber: &num, InstallmentTotal: &total,
		})
	}

	got, _ := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if len(got.InstallmentDrifts) != 0 {
		t.Errorf("a complete series must report nothing, got %v", got.InstallmentDrifts)
	}
}

// A reversed instalment does not break the series: it was undone on purpose, and
// reporting it would train the reader to ignore the signal.
func TestInvariants_ReversedInstalmentDoesNotBreakTheSeries(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	total := 2
	one, two := 1, 2
	txRepo.created = append(txRepo.created,
		&transaction.Transaction{ID: "r1", ProfileID: "p1", BankAccountID: "card",
			Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 100, Currency: "BRL",
			Description: "Parcela", OccurredOn: day(2026, time.September, 3),
			InstallmentNumber: &one, InstallmentTotal: &total},
		&transaction.Transaction{ID: "r2", ProfileID: "p1", BankAccountID: "card",
			Type: transaction.TypeExpense, Status: transaction.StatusReversed, Amount: 100, Currency: "BRL",
			Description: "Parcela", OccurredOn: day(2026, time.October, 3),
			InstallmentNumber: &two, InstallmentTotal: &total},
	)

	got, _ := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	for _, d := range got.InstallmentDrifts {
		if d.Missing != nil && len(d.Missing) > 0 {
			t.Errorf("a deliberately reversed instalment must not read as missing: %v", d.Missing)
		}
	}
}

// The invariant the payment link unlocks, and which the reviewer argued is better than
// a natural uniqueness key: it does not block a legitimate duplicate, it is verifiable
// continuously, and it catches exactly the failure behind the R$ 8.863,07 phantom —
// invoices reading as paid far beyond what they were worth.
func TestInvariants_PaymentsBeyondTheBillAreReported(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "inv-1", BankAccountID: "card", Amount: 500,
			OpeningDate: day(2026, time.July, 27), ClosingDate: day(2026, time.August, 27), DueDate: day(2026, time.September, 3)},
	}
	// The same bill paid twice: 1000 against a bill of 500.
	invID := "inv-1"
	for i := 0; i < 2; i++ {
		txRepo.created = append(txRepo.created, &transaction.Transaction{
			ID: "pay" + string(rune('0'+i)), ProfileID: "p1", BankAccountID: "checking",
			DestinationAccountID: strPtr("card"),
			Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
			Amount: 500, Currency: "BRL", Description: "Pagamento fatura",
			OccurredOn: day(2026, time.September, 3), PaidInvoiceID: &invID,
		})
	}

	got, err := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.PaymentDrifts) == 0 {
		t.Fatal("a bill paid beyond its value must be reported")
	}
	if got.PaymentDrifts[0].PaidTotal != 1000 || got.PaymentDrifts[0].InvoiceAmount != 500 {
		t.Errorf("got paid %.2f against %.2f", got.PaymentDrifts[0].PaidTotal, got.PaymentDrifts[0].InvoiceAmount)
	}
	if got.OK {
		t.Error("paying a bill twice is actionable")
	}
}

func TestInvariants_AReversedPaymentDoesNotCountTowardsTheBill(t *testing.T) {
	accounts, txRepo, invoices := cycleFixture(t)
	invoices.list = []*invoice.Invoice{
		{ID: "inv-2", BankAccountID: "card", Amount: 500,
			OpeningDate: day(2026, time.July, 27), ClosingDate: day(2026, time.August, 27), DueDate: day(2026, time.September, 3)},
	}
	invID := "inv-2"
	txRepo.created = append(txRepo.created,
		&transaction.Transaction{ID: "live", ProfileID: "p1", BankAccountID: "checking",
			DestinationAccountID: strPtr("card"), Type: transaction.TypeTransfer,
			Status: transaction.StatusConfirmed, Amount: 500, Currency: "BRL",
			Description: "Pagamento fatura", OccurredOn: day(2026, time.September, 3), PaidInvoiceID: &invID},
		&transaction.Transaction{ID: "undone", ProfileID: "p1", BankAccountID: "checking",
			DestinationAccountID: strPtr("card"), Type: transaction.TypeTransfer,
			Status: transaction.StatusReversed, Amount: 500, Currency: "BRL",
			Description: "Pagamento fatura", OccurredOn: day(2026, time.September, 3), PaidInvoiceID: &invID},
	)

	got, _ := NewCheckInvariantsUseCase(accounts, txRepo, invoices).Execute()
	if len(got.PaymentDrifts) != 0 {
		t.Errorf("a reversed payment must not count: %v", got.PaymentDrifts)
	}
}
