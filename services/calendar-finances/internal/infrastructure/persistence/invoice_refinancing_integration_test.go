//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	_ "github.com/lib/pq"
)

// The in-memory fake mirrors the repository's UPDATE column list BY HAND. That
// caught the first version of this change — the columns were added to the real
// statement and not to the fake — but it only catches a forgotten write, never a
// forgotten migration or a column the SELECT does not read back. Both of those
// pass every unit test and lose the data on the way to Postgres.
//
// This exercises the real schema: convert a bill, read it back through the
// repository, and check the conversion survived the round trip.

const (
	refiProfileID = "5c3d0000-0000-4000-8000-000000000001"
	refiCardID    = "5c3d0000-0000-4000-8000-000000000002"
	refiBillID    = "5c3d0000-0000-4000-8000-000000000003"
	refiTxnID     = "5c3d0000-0000-4000-8000-000000000004"
)

func refiCleanup(db *sql.DB) {
	db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", refiProfileID)
	db.Exec("DELETE FROM finance.credit_card_invoices WHERE bank_account_id = $1", refiCardID)
	db.Exec("DELETE FROM finance.bank_accounts WHERE profile_id = $1", refiProfileID)
	db.Exec("DELETE FROM finance.profiles WHERE id = $1", refiProfileID)
}

func refiDB(t *testing.T) *sql.DB {
	t.Helper()
	db := getTestDB(t)
	if err := db.Ping(); err != nil {
		t.Skipf("integration DB unavailable (%v); skipping", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	refiCleanup(db)
	t.Cleanup(func() {
		refiCleanup(db)
		_ = db.Close()
	})
	return db
}

func refiSeed(t *testing.T, db *sql.DB) *invoice.Invoice {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'Refinancing', 'PERSONAL')`,
		refiProfileID, "refinancing-e2e"); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	closingDay, dueDay := 9, 14
	if err := NewBankAccountRepository(db).Create(&bankaccount.BankAccount{
		ID: refiCardID, ProfileID: refiProfileID, Name: "Cartão Refi",
		Type: bankaccount.AccountTypeCreditCard, ClosingDay: &closingDay, DueDay: &dueDay,
		Currency: "BRL", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	// The conversion credit has to exist: refinanced_by is a foreign key, which is
	// what stops a bill from naming a transaction that was never posted.
	if _, err := db.Exec(`
		INSERT INTO finance.transactions
			(id, profile_id, bank_account_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, $3, 'INCOME', 'CONFIRMED', 2205.86, 'Credito por parcelamento da fatura', '2026-10-11')`,
		refiTxnID, refiProfileID, refiCardID); err != nil {
		t.Fatalf("seed conversion credit: %v", err)
	}

	bill := &invoice.Invoice{
		ID: refiBillID, BankAccountID: refiCardID, Amount: 1649.35, Status: invoice.StatusClosed,
		ReferenceDate: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		OpeningDate:   time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(), UpdatedAt: time.Now(),
	}
	if err := NewInvoiceRepository(db).Create(bill); err != nil {
		t.Fatalf("seed bill: %v", err)
	}
	return bill
}

func TestRefinancingSurvivesTheRoundTripThroughPostgres(t *testing.T) {
	db := refiDB(t)
	bill := refiSeed(t, db)
	repo := NewInvoiceRepository(db)
	at := time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC)

	if err := bill.Refinance(refiTxnID, at); err != nil {
		t.Fatalf("Refinance: %v", err)
	}
	if err := repo.Update(bill); err != nil {
		t.Fatalf("Update: %v", err)
	}

	reloaded, err := repo.FindByID(refiBillID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if reloaded.Status != invoice.StatusRefinanced {
		t.Errorf("status = %q, want REFINANCED: the CHECK constraint or the UPDATE dropped it", reloaded.Status)
	}
	if reloaded.RefinancedBy == nil || *reloaded.RefinancedBy != refiTxnID {
		t.Errorf("refinanced by = %v, want %s", reloaded.RefinancedBy, refiTxnID)
	}
	if reloaded.RefinancedAt == nil || !reloaded.RefinancedAt.UTC().Equal(at) {
		t.Errorf("refinanced at = %v, want %v", reloaded.RefinancedAt, at)
	}
	// The column the cross-request cap is built on. Nothing read it back through
	// Postgres until this line: replacing it with NULL in the SELECT survived the
	// whole integration suite.
	if reloaded.RefinancedAmount == nil || *reloaded.RefinancedAmount != 1649.35 {
		t.Errorf("refinanced amount = %v, want 1649.35", reloaded.RefinancedAmount)
	}
	if got := reloaded.AmountRemaining(); got != 0 {
		t.Errorf("amount remaining = %.2f, want 0 after the round trip", got)
	}
}

// FindByRefinancedBy is what caps one credit across separate requests. It has its
// own SELECT, so a column missing from it reports zero already spent and the cap
// stops holding.
func TestFindByRefinancedByReadsBackWhatWasConverted(t *testing.T) {
	db := refiDB(t)
	bill := refiSeed(t, db)
	repo := NewInvoiceRepository(db)

	if err := bill.Refinance(refiTxnID, time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Refinance: %v", err)
	}
	if err := repo.Update(bill); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := repo.FindByRefinancedBy(refiTxnID)
	if err != nil {
		t.Fatalf("FindByRefinancedBy: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("got %d bills, want 1", len(found))
	}
	if found[0].RefinancedAmount == nil || *found[0].RefinancedAmount != 1649.35 {
		t.Errorf("refinanced amount = %v, want 1649.35 — the cap reads this", found[0].RefinancedAmount)
	}
	if found[0].ID != refiBillID {
		t.Errorf("id = %s, want %s", found[0].ID, refiBillID)
	}

	none, err := repo.FindByRefinancedBy("5c3d0000-0000-4000-8000-00000000ffff")
	if err != nil {
		t.Fatalf("FindByRefinancedBy for an unrelated credit: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("an unrelated credit reports %d converted bills, want 0", len(none))
	}
}

// The listing path has its own SELECT. A column added to FindByID and forgotten
// here reports every converted bill as still owing, which is the symptom this
// whole change set out to remove.
func TestRefinancingIsVisibleThroughTheListing(t *testing.T) {
	db := refiDB(t)
	bill := refiSeed(t, db)
	repo := NewInvoiceRepository(db)

	if err := bill.Refinance(refiTxnID, time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Refinance: %v", err)
	}
	if err := repo.Update(bill); err != nil {
		t.Fatalf("Update: %v", err)
	}

	list, err := repo.FindByBankAccountID(refiCardID)
	if err != nil {
		t.Fatalf("FindByBankAccountID: %v", err)
	}
	var found *invoice.Invoice
	for _, inv := range list {
		if inv.ID == refiBillID {
			found = inv
		}
	}
	if found == nil {
		t.Fatal("the converted bill vanished from the listing")
	}
	if found.Status != invoice.StatusRefinanced || found.RefinancedBy == nil {
		t.Errorf("listing lost the conversion: status=%q refinancedBy=%v", found.Status, found.RefinancedBy)
	}
	if got := found.AmountRemaining(); got != 0 {
		t.Errorf("listing reports %.2f still owed, want 0", got)
	}
}

// The constraint is the last line of defence: a status without the transaction
// behind it is a settlement nobody can audit.
func TestPostgresRefusesARefinancedBillWithNoTransaction(t *testing.T) {
	db := refiDB(t)
	refiSeed(t, db)

	// refinanced_amount is supplied so ONLY the missing transaction can be the
	// reason. The earlier version left both NULL, so either constraint satisfied
	// it and neither was independently verified.
	_, err := db.Exec(`
		UPDATE finance.credit_card_invoices
		SET status = 'REFINANCED', refinanced_by = NULL, refinanced_amount = 100
		WHERE id = $1`, refiBillID)

	if err == nil {
		t.Fatal("expected the CHECK constraint to refuse a conversion with no transaction")
	}
}

// The sibling constraint, isolated the same way: a conversion whose amount is
// unknown can be neither audited nor counted against the credit's cap.
func TestPostgresRefusesARefinancedBillWithNoAmount(t *testing.T) {
	db := refiDB(t)
	refiSeed(t, db)

	_, err := db.Exec(`
		UPDATE finance.credit_card_invoices
		SET status = 'REFINANCED', refinanced_by = $2, refinanced_amount = NULL
		WHERE id = $1`, refiBillID, refiTxnID)

	if err == nil {
		t.Fatal("expected the CHECK constraint to refuse a conversion with no amount")
	}
}

// Rows converted before refinanced_amount existed read NULL, and a NULL makes the
// whole bill look unconverted: it reports as drift and loosens the cap. The
// backfill runs on every startup and must repair them.
func TestMigrationsBackfillTheAmountOfAnOlderConversion(t *testing.T) {
	db := refiDB(t)
	bill := refiSeed(t, db)

	// A row as the earlier commits left it: converted, amount unknown. Written
	// with the constraint dropped, since that is the state it predates.
	if _, err := db.Exec(`ALTER TABLE finance.credit_card_invoices DROP CONSTRAINT IF EXISTS chk_refinanced_has_amount`); err != nil {
		t.Fatalf("dropping the constraint to recreate the legacy state: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE finance.credit_card_invoices
		SET status = 'REFINANCED', refinanced_by = $2, refinanced_amount = NULL, paid_amount = 400
		WHERE id = $1`, bill.ID, refiTxnID); err != nil {
		t.Fatalf("seeding the legacy row: %v", err)
	}

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("migrations failed over a legacy converted row: %v", err)
	}

	reloaded, err := NewInvoiceRepository(db).FindByID(bill.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	// 1.649,35 billed less 400,00 already paid in cash.
	if reloaded.RefinancedAmount == nil || *reloaded.RefinancedAmount != 1249.35 {
		t.Errorf("refinanced amount = %v, want 1249.35 backfilled", reloaded.RefinancedAmount)
	}
	if got := reloaded.UnconvertedDrift(); got != 0 {
		t.Errorf("drift = %v, want 0 — a backfilled row is not drifting", got)
	}
}

// The migration slice is replayed on EVERY startup, and the block that widened the
// status CHECK is not the only one that asserts its width: an older block drops and
// re-adds the same constraint, and it runs first. Left narrow, it failed against the
// REFINANCED rows already in the table, RunMigrations aborted on the first error, and
// the service died in log.Fatalf on boot. A restart after the feature was used once
// would not come back.
//
// Every test here calls RunMigrations on a table that its own cleanup has emptied,
// which is exactly the condition under which the defect is invisible. This one keeps
// the row.
func TestMigrationsReplayWithAConvertedBillAlreadyInTheTable(t *testing.T) {
	db := refiDB(t)
	bill := refiSeed(t, db)
	repo := NewInvoiceRepository(db)

	if err := bill.Refinance(refiTxnID, time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Refinance: %v", err)
	}
	if err := repo.Update(bill); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// What the next container start does.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("migrations failed with a converted bill present, so the service would not boot: %v", err)
	}

	reloaded, err := repo.FindByID(refiBillID)
	if err != nil {
		t.Fatalf("FindByID after replay: %v", err)
	}
	if reloaded.Status != invoice.StatusRefinanced {
		t.Errorf("status = %q after replaying migrations, want REFINANCED", reloaded.Status)
	}
}
