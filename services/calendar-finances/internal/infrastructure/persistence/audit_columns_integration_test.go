//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A column that is written and never read is not a record, it is a rumour.
//
// paid_invoice_id was added to the INSERT and to no SELECT, so every transaction
// loaded from the database came back with a nil link. The invariant built on it —
// live payments against a bill may not exceed what the bill is worth — could
// therefore never fire, and its unit tests passed only because the fake repository
// returns in-memory structs with the field already set.
//
// The reversal columns had the same shape: ReverseMany wrote reversed_at,
// reversal_reason, reversal_note and reversed_by, and nothing ever selected them.
// The audit trail that exists to answer "who reversed this, and why" was readable
// only by opening psql.
func TestIntegration_ThePaymentLinkSurvivesARoundTrip(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)

	invoiceID := seedInvoiceThroughRepository(t, db, &invoice.Invoice{
		BankAccountID: accountID, Amount: 799.57, Status: invoice.StatusClosed,
		ReferenceDate: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), OpeningDate: time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC),
		ClosingDate: time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
	})

	repo := NewTransactionRepository(db)
	txn, err := transaction.New(transaction.CreateParams{
		ProfileID: profileID, BankAccountID: accountID,
		Type:   transaction.TypeExpense,
		Amount: 799.57, Currency: "BRL", Description: "Pagamento fatura",
		OccurredOn:    time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
		PaidInvoiceID: &invoiceID,
	})
	if err != nil {
		t.Fatalf("build transaction: %v", err)
	}
	if err := repo.Create(txn); err != nil {
		t.Fatalf("create: %v", err)
	}

	loaded, err := repo.GetByID(txn.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.PaidInvoiceID == nil {
		t.Fatal("GetByID lost the invoice this payment settles")
	}
	if *loaded.PaidInvoiceID != invoiceID {
		t.Errorf("got invoice %s, want %s", *loaded.PaidInvoiceID, invoiceID)
	}

	listed, err := repo.List(transaction.ListFilter{ProfileID: profileID, BankAccountID: &accountID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *transaction.Transaction
	for _, candidate := range listed {
		if candidate.ID == txn.ID {
			found = candidate
		}
	}
	if found == nil {
		t.Fatal("the payment did not come back from List at all")
	}
	if found.PaidInvoiceID == nil || *found.PaidInvoiceID != invoiceID {
		t.Error("List lost the invoice this payment settles")
	}
}

func TestIntegration_TheReversalTrailSurvivesARoundTrip(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)

	repo := NewTransactionRepository(db)
	txn, err := transaction.New(transaction.CreateParams{
		ProfileID: profileID, BankAccountID: accountID,
		Type:   transaction.TypeExpense,
		Amount: 55.58, Currency: "BRL", Description: "Cloudflare",
		OccurredOn: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build transaction: %v", err)
	}
	if err := txn.SetStatus(transaction.StatusConfirmed); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := repo.Create(txn); err != nil {
		t.Fatalf("create: %v", err)
	}

	at := time.Date(2026, time.September, 7, 10, 0, 0, 0, time.UTC)
	if err := txn.Reverse(transaction.ReasonDuplicated, "lancada duas vezes", "bruno", at); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if err := repo.ReverseMany([]*transaction.Transaction{txn}); err != nil {
		t.Fatalf("persist reversal: %v", err)
	}

	loaded, err := repo.GetByID(txn.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.Status != transaction.StatusReversed {
		t.Fatalf("status came back %s", loaded.Status)
	}
	if loaded.ReversedBy == nil || *loaded.ReversedBy != "bruno" {
		t.Errorf("who reversed it is unreadable: %v", loaded.ReversedBy)
	}
	if loaded.ReversalReason == nil || *loaded.ReversalReason != transaction.ReasonDuplicated {
		t.Errorf("why it was reversed is unreadable: %v", loaded.ReversalReason)
	}
	if loaded.ReversedAt == nil {
		t.Error("when it was reversed is unreadable")
	}
	// The distinction the whole enum exists for: a correction retroacts, a new fact
	// does not. It is answerable only if the reason survives the round trip.
	if !loaded.ReversalRetroacts() {
		t.Error("a duplicate is a correction and must retroact")
	}
}

// Cancelling demands a motive and an actor at the door. Until this test, the
// persistence threw both away: the domain set them, UpdateStatus wrote status, date
// and notes, and nothing else reached the database. The requirement was real and the
// record behind it was not — a control that appears to operate while capturing
// nothing.
func TestIntegration_ACancellationKeepsItsMotiveAndItsActor(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)

	repo := NewTransactionRepository(db)
	txn, err := transaction.New(transaction.CreateParams{
		ProfileID: profileID, BankAccountID: accountID,
		Type: transaction.TypeExpense, Amount: 25, Currency: "BRL",
		Description: "Contabo - VPS",
		OccurredOn:  time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build transaction: %v", err)
	}
	if err := repo.Create(txn); err != nil {
		t.Fatalf("create: %v", err)
	}

	at := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)
	if err := txn.Cancel(transaction.ReasonNeverHappened, "assinatura cancelada antes de cobrar", "bruno", at); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := repo.CancelStatus(txn, txn.OccurredOn); err != nil {
		t.Fatalf("persist cancellation: %v", err)
	}

	loaded, err := repo.GetByID(txn.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.Status != transaction.StatusCancelled {
		t.Fatalf("status came back %s", loaded.Status)
	}
	if loaded.ReversedBy == nil || *loaded.ReversedBy != "bruno" {
		t.Errorf("who cancelled it is unreadable: %v", loaded.ReversedBy)
	}
	if loaded.ReversalReason == nil || *loaded.ReversalReason != transaction.ReasonNeverHappened {
		t.Errorf("why it was cancelled is unreadable: %v", loaded.ReversalReason)
	}
	if loaded.ReversedAt == nil {
		t.Error("when it was cancelled is unreadable")
	}
}

// List and Count must agree on what counts. Count rebuilt its conditions from scratch
// and never learned that a reversed row is hidden by default, so after the first
// reversal a page reported more items than it listed.
func TestIntegration_CountAgreesWithListAboutReversedRows(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)

	repo := NewTransactionRepository(db)
	for _, name := range []string{"fica", "estornada"} {
		txn, err := transaction.New(transaction.CreateParams{
			ProfileID: profileID, BankAccountID: accountID,
			Type: transaction.TypeExpense, Amount: 10, Currency: "BRL",
			Description: name,
			OccurredOn:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("build %s: %v", name, err)
		}
		if err := txn.SetStatus(transaction.StatusConfirmed); err != nil {
			t.Fatalf("confirm %s: %v", name, err)
		}
		if err := repo.Create(txn); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if name == "estornada" {
			if err := txn.Reverse(transaction.ReasonNeverHappened, "", "bruno", time.Now()); err != nil {
				t.Fatalf("reverse: %v", err)
			}
			if err := repo.ReverseMany([]*transaction.Transaction{txn}); err != nil {
				t.Fatalf("persist reversal: %v", err)
			}
		}
	}

	filter := transaction.ListFilter{ProfileID: profileID, BankAccountID: &accountID}
	listed, err := repo.List(filter)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	counted, err := repo.Count(filter)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if counted != len(listed) {
		t.Errorf("count says %d, list returns %d", counted, len(listed))
	}

	filter.IncludeReversed = true
	auditListed, _ := repo.List(filter)
	auditCounted, _ := repo.Count(filter)
	if auditCounted != len(auditListed) || auditCounted != 2 {
		t.Errorf("the audit view disagrees: count %d, list %d", auditCounted, len(auditListed))
	}
}

// A bill whose payment is reversed must stop claiming to be paid.
//
// Invoice.Pay accumulates, and nothing ever reduced what it recorded. Reversing an
// invoice payment therefore left the bill PAID with a paid_amount nobody had paid: it
// kept counting against the card limit, and the guard added to Reopen() — which
// refuses a bill that was ever settled — made the state unreachable through the API.
// The money came back to the checking account and the card never noticed.
func TestIntegration_ReversingAPaymentUnsettlesTheBill(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)

	paidAmount, paidAt := 799.57, time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC)
	invoiceID := seedInvoiceThroughRepository(t, db, &invoice.Invoice{
		BankAccountID: accountID, Amount: 799.57, Status: invoice.StatusPaid,
		PaidAmount: &paidAmount, PaidAt: &paidAt,
		ReferenceDate: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), OpeningDate: time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC),
		ClosingDate: time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
	})

	repo := NewTransactionRepository(db)
	payment, err := transaction.New(transaction.CreateParams{
		ProfileID: profileID, BankAccountID: accountID,
		Type: transaction.TypeExpense, Amount: 799.57, Currency: "BRL",
		Description:   "Pagamento fatura",
		OccurredOn:    time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
		PaidInvoiceID: &invoiceID,
	})
	if err != nil {
		t.Fatalf("build payment: %v", err)
	}
	if err := payment.SetStatus(transaction.StatusConfirmed); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := repo.Create(payment); err != nil {
		t.Fatalf("create: %v", err)
	}

	live, err := repo.SumLivePaymentsByInvoiceID(invoiceID)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	if live != 799.57 {
		t.Fatalf("a confirmed payment must count: got %v", live)
	}

	if err := payment.Reverse(transaction.ReasonNeverHappened, "paguei a fatura errada", "bruno", time.Now()); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if err := repo.ReverseMany([]*transaction.Transaction{payment}); err != nil {
		t.Fatalf("persist reversal: %v", err)
	}

	live, err = repo.SumLivePaymentsByInvoiceID(invoiceID)
	if err != nil {
		t.Fatalf("sum after reversal: %v", err)
	}
	if live != 0 {
		t.Fatalf("a reversed payment must stop counting: got %v", live)
	}
}

// seedInvoiceThroughRepository creates the bill with the same statement production
// uses. Seeding it with an INSERT written for the test leaves InvoiceRepository.Create
// — a central money path — with no coverage at all, which is how a duplicated column
// in the account repository broke every account creation while the suite stayed green.
//
// It matters more for invoices than for accounts: a balance is compared against the
// bank every day, and soon automatically. A bill is compared against nothing.
func seedInvoiceThroughRepository(t *testing.T, db *sql.DB, inv *invoice.Invoice) string {
	t.Helper()
	if inv.ID == "" {
		inv.ID = uuid.NewString()
	}
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	inv.UpdatedAt = time.Now()
	repo := NewInvoiceRepository(db)
	// Returns early when the bill is already there, the way the ON CONFLICT DO NOTHING
	// it replaced did: a run interrupted halfway must not make every later run fail on
	// a duplicate key.
	if existing, err := repo.FindByID(inv.ID); err == nil && existing != nil {
		return inv.ID
	}
	if err := repo.Create(inv); err != nil {
		t.Fatalf("seeding invoice through the repository: %v", err)
	}
	return inv.ID
}
