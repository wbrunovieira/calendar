//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	_ "github.com/lib/pq"
)

const (
	invoiceSumProfileID = "4b2c0000-0000-4000-8000-000000000001"
	invoiceSumCardID    = "4b2c0000-0000-4000-8000-000000000002"
	invoiceSumInvoiceID = "4b2c0000-0000-4000-8000-000000000003"
)

func invoiceSumDB(t *testing.T) *sql.DB {
	t.Helper()
	db := getTestDB(t)
	if err := db.Ping(); err != nil {
		t.Skipf("integration DB unavailable (%v); skipping", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	invoiceSumCleanup(db)
	t.Cleanup(func() {
		invoiceSumCleanup(db)
		_ = db.Close()
	})
	return db
}

func invoiceSumCleanup(db *sql.DB) {
	db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", invoiceSumProfileID)
	db.Exec("DELETE FROM finance.credit_card_invoices WHERE bank_account_id = $1", invoiceSumCardID)
	db.Exec("DELETE FROM finance.bank_accounts WHERE profile_id = $1", invoiceSumProfileID)
	db.Exec("DELETE FROM finance.profiles WHERE id = $1", invoiceSumProfileID)
}

func invoiceSumSeed(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'Invoice Sum', 'PERSONAL')`,
		invoiceSumProfileID, "invoice-sum-e2e"); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO finance.bank_accounts (id, profile_id, name, type, closing_day, due_day)
		VALUES ($1, $2, 'Cartão E2E', 'CREDIT_CARD', 5, 15)`,
		invoiceSumCardID, invoiceSumProfileID); err != nil {
		t.Fatalf("seed card: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO finance.credit_card_invoices
			(id, bank_account_id, reference_date, opening_date, closing_date, due_date, amount, status)
		VALUES ($1, $2, '2026-03-01', '2026-02-06', '2026-03-05', '2026-03-15', 0, 'OPEN')`,
		invoiceSumInvoiceID, invoiceSumCardID); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
}

func invoiceSumCharge(t *testing.T, db *sql.DB, txType, status string, amount float64, description string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO finance.transactions
			(profile_id, bank_account_id, invoice_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, '2026-02-20')`,
		invoiceSumProfileID, invoiceSumCardID, invoiceSumInvoiceID, txType, status, amount, description); err != nil {
		t.Fatalf("seed %s: %v", description, err)
	}
}

// A refund is credited back onto the card and carries the same invoice_id as
// the purchase it reverses. Summing every row's amount regardless of type made
// the refund *raise* the bill it was cancelling out.
func TestSumByInvoiceID_SubtractsCreditsInsteadOfAddingThem(t *testing.T) {
	db := invoiceSumDB(t)
	invoiceSumSeed(t, db)

	invoiceSumCharge(t, db, "EXPENSE", "CONFIRMED", 100, "Compra")
	invoiceSumCharge(t, db, "INCOME", "CONFIRMED", 30, "Estorno parcial")

	total, err := NewTransactionRepository(db).SumByInvoiceID(invoiceSumInvoiceID)
	if err != nil {
		t.Fatalf("SumByInvoiceID: %v", err)
	}

	if total != 70 {
		t.Errorf("invoice total = %.2f, want 70 (100 charged, 30 refunded)", total)
	}
}

func TestSumByInvoiceID_StillIgnoresCancelledRows(t *testing.T) {
	db := invoiceSumDB(t)
	invoiceSumSeed(t, db)

	invoiceSumCharge(t, db, "EXPENSE", "CONFIRMED", 100, "Compra")
	invoiceSumCharge(t, db, "EXPENSE", "CANCELLED", 999, "Compra cancelada")
	invoiceSumCharge(t, db, "INCOME", "CANCELLED", 999, "Estorno cancelado")

	total, err := NewTransactionRepository(db).SumByInvoiceID(invoiceSumInvoiceID)
	if err != nil {
		t.Fatalf("SumByInvoiceID: %v", err)
	}

	if total != 100 {
		t.Errorf("invoice total = %.2f, want 100", total)
	}
}

// A refund is not always confirmed on the same day it is posted, so the
// per-status sum has to net the same way the overall one does.
func TestSumByInvoiceIDByStatus_AlsoSubtractsCredits(t *testing.T) {
	db := invoiceSumDB(t)
	invoiceSumSeed(t, db)

	invoiceSumCharge(t, db, "EXPENSE", "CONFIRMED", 100, "Compra")
	invoiceSumCharge(t, db, "INCOME", "CONFIRMED", 30, "Estorno parcial")
	invoiceSumCharge(t, db, "EXPENSE", "PLANNED", 40, "Parcela futura")

	repo := NewTransactionRepository(db)

	confirmed, err := repo.SumByInvoiceIDByStatus(invoiceSumInvoiceID, transaction.StatusConfirmed)
	if err != nil {
		t.Fatalf("SumByInvoiceIDByStatus(CONFIRMED): %v", err)
	}
	if confirmed != 70 {
		t.Errorf("confirmed total = %.2f, want 70", confirmed)
	}

	planned, err := repo.SumByInvoiceIDByStatus(invoiceSumInvoiceID, transaction.StatusPlanned)
	if err != nil {
		t.Fatalf("SumByInvoiceIDByStatus(PLANNED): %v", err)
	}
	if planned != 40 {
		t.Errorf("planned total = %.2f, want 40", planned)
	}
}
