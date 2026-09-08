//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Recalculating a balance silently is the single biggest risk to trusting this data.
// While it exists as a routine correction, nobody can tell whether the system is right
// or the number was merely stamped — and worse, the correction ERASES the evidence:
// after it runs there is no record of what the drift was, when it started, or how much
// it was worth.
//
// On 06/09/2026 it "saved" R$ 1.522,75 of drift and, in saving it, destroyed the fact
// that a confirmation bug existed. It was used three times that day.

const (
	adjProfileID = "e2e00000-0000-0000-0000-0000000000c1"
	adjAccountID = "e2e00000-0000-0000-0000-0000000000c2"
)

func seedDrifted(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.balance_adjustments WHERE account_id = $1`, adjAccountID)
		db.Exec(`DELETE FROM finance.transactions WHERE profile_id = $1`, adjProfileID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id = $1`, adjAccountID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, adjProfileID)
	})
	exec(t, db, `INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1,$2,'E2E Ajuste','BUSINESS') ON CONFLICT (id) DO NOTHING`, adjProfileID, "e2e-adjust")
	// Stored balance says 1000; no transaction justifies it. Drift of exactly 1000.
	acc := checkingAccount(adjProfileID, "Conta Deriva", 0, 1000)
	acc.ID = adjAccountID
	seedAccountThroughRepository(t, db, acc)
}

func TestE2E_RecalculateIsReadOnlyByDefault(t *testing.T) {
	db := testDB(t)
	seedDrifted(t, db)

	uc := usecases.NewRecalculateBalanceUseCase(
		persistence.NewBankAccountRepository(db), persistence.NewTransactionRepository(db), nil)

	result, err := uc.Execute(adjAccountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Drift != 1000 {
		t.Errorf("drift = %.2f, want 1000 reported", result.Drift)
	}

	var stored float64
	db.QueryRow(`SELECT current_balance FROM finance.bank_accounts WHERE id = $1`, adjAccountID).Scan(&stored)
	if stored != 1000 {
		t.Errorf("the balance was rewritten (%.2f) without anyone asking: reporting must not correct", stored)
	}
}

func TestE2E_ApplyingAnAdjustmentRequiresAReasonAndLeavesATrail(t *testing.T) {
	db := testDB(t)
	seedDrifted(t, db)

	uc := usecases.NewRecalculateBalanceUseCase(
		persistence.NewBankAccountRepository(db), persistence.NewTransactionRepository(db), nil)
	uc.SetAdjustmentLog(persistence.NewBalanceAdjustmentLog(db))

	if _, err := uc.Apply(adjAccountID, "", "teste"); err == nil {
		t.Error("applying without a reason must be refused: a correction with no motive cannot be reviewed")
	}

	result, err := uc.Apply(adjAccountID, "conferido contra o extrato de 07/09", "teste")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NewBalance != 0 {
		t.Errorf("newBalance = %.2f, want 0", result.NewBalance)
	}

	var before, after, delta float64
	var reason, by string
	if err := db.QueryRow(`SELECT balance_before, balance_after, delta, reason, adjusted_by
		FROM finance.balance_adjustments WHERE account_id = $1`, adjAccountID).
		Scan(&before, &after, &delta, &reason, &by); err == sql.ErrNoRows {
		t.Fatal("the adjustment left no trail: after it runs, what the drift was and when it started are gone")
	} else if err != nil {
		t.Fatalf("reading the trail: %v", err)
	}
	if before != 1000 || after != 0 || delta != -1000 {
		t.Errorf("trail = %.2f -> %.2f (%.2f), want 1000 -> 0 (-1000)", before, after, delta)
	}
	if reason == "" || by == "" {
		t.Error("who and why must both be on the record")
	}
}
