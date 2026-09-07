//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Against the real database, because this is exactly the class of bug an in-memory
// fake cannot see: the CHECK constraint on status, the atomicity of the two-leg
// update, and whether the row is really still there afterwards.
//
// The behaviour under test exists because a delete destroyed a legitimate R$ 55,58
// entry during the reconciliation of 06/09/2026, and nothing could say what was lost.

const (
	revProfileID  = "e1e00000-0000-0000-0000-000000000001"
	revAccountID  = "e1e00000-0000-0000-0000-000000000002"
	revCategoryID = "e1e00000-0000-0000-0000-000000000003"
	revTxID       = "e1e00000-0000-0000-0000-000000000004"
)

func seedReversal(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.transactions WHERE id = $1`, revTxID)
		db.Exec(`DELETE FROM finance.categories WHERE id = $1`, revCategoryID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id = $1`, revAccountID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, revProfileID)
	})

	exec(t, db, `INSERT INTO finance.profiles (id, calendar_id, name, type) VALUES ($1,$2,'E2E Estorno','BUSINESS')
		ON CONFLICT (id) DO NOTHING`, revProfileID, "e2e-reversal")
	// initial_balance and current_balance must be coherent: the account opens at
	// 1055.58 and the single confirmed expense of 55.58 brings it to 1000. Seeding an
	// opening of 0 with a balance of 1000 describes money that no transaction explains,
	// and the derived balance would rightly disagree.
	exec(t, db, `INSERT INTO finance.bank_accounts (id, profile_id, name, type, initial_balance, current_balance, currency)
		VALUES ($1,$2,'Conta E2E','CHECKING',1055.58,1000,'BRL') ON CONFLICT (id) DO NOTHING`, revAccountID, revProfileID)
	exec(t, db, `INSERT INTO finance.categories (id, profile_id, name, type) VALUES ($1,$2,'Servidores','EXPENSE')
		ON CONFLICT (id) DO NOTHING`, revCategoryID, revProfileID)
	exec(t, db, `INSERT INTO finance.transactions
		(id, profile_id, bank_account_id, category_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1,$2,$3,$4,'EXPENSE','CONFIRMED',55.58,'BRL','Cloudflare - dominio','2026-07-11')
		ON CONFLICT (id) DO NOTHING`, revTxID, revProfileID, revAccountID, revCategoryID)
}

func TestE2E_DeleteReversesInsteadOfDestroying(t *testing.T) {
	db := testDB(t)
	seedReversal(t, db)

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	recalc := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalc).Execute(revTxID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. The row survived, with its content intact.
	var status, description string
	var amount float64
	var reversedAt sql.NullTime
	err := db.QueryRow(`SELECT status, description, amount, reversed_at
		FROM finance.transactions WHERE id = $1`, revTxID).
		Scan(&status, &description, &amount, &reversedAt)
	if err == sql.ErrNoRows {
		t.Fatal("the row was DESTROYED — a ledger must reverse, not delete")
	}
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if status != string(transaction.StatusReversed) {
		t.Errorf("status = %s, want REVERSED", status)
	}
	if description != "Cloudflare - dominio" || amount != 55.58 {
		t.Errorf("the reversed row lost its content: %q / %.2f", description, amount)
	}
	if !reversedAt.Valid {
		t.Error("reversed_at must record when it was undone")
	}

	// 2. The balance moved as if it had been deleted: an expense was undone.
	var balance float64
	if err := db.QueryRow(`SELECT current_balance FROM finance.bank_accounts WHERE id = $1`, revAccountID).
		Scan(&balance); err != nil {
		t.Fatalf("reading the balance: %v", err)
	}
	if balance < 1055.575 || balance > 1055.585 {
		t.Errorf("balance = %.2f, want 1055.58", balance)
	}

	// 3. A reversed row must not count towards the balance any more.
	var confirmedSum sql.NullFloat64
	if err := db.QueryRow(`SELECT SUM(amount) FROM finance.transactions
		WHERE bank_account_id = $1 AND status = 'CONFIRMED'`, revAccountID).Scan(&confirmedSum); err != nil {
		t.Fatalf("summing: %v", err)
	}
	if confirmedSum.Valid && confirmedSum.Float64 != 0 {
		t.Errorf("reversed row still counts as CONFIRMED: sum = %.2f", confirmedSum.Float64)
	}
}

func TestE2E_ReversingTwiceIsRefusedByTheDatabase(t *testing.T) {
	db := testDB(t)
	seedReversal(t, db)

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	recalc := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	uc := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalc)

	if err := uc.Execute(revTxID); err != nil {
		t.Fatalf("first reversal: %v", err)
	}
	var before float64
	db.QueryRow(`SELECT current_balance FROM finance.bank_accounts WHERE id = $1`, revAccountID).Scan(&before)

	if err := uc.Execute(revTxID); err == nil {
		t.Error("reversing twice must be refused")
	}

	var after float64
	db.QueryRow(`SELECT current_balance FROM finance.bank_accounts WHERE id = $1`, revAccountID).Scan(&after)
	if before != after {
		t.Errorf("a refused reversal moved the balance: %.2f -> %.2f", before, after)
	}
}

// The database must both accept REVERSED and refuse it without an audit trail. A
// reason column that can be left NULL is a control that exists in the schema and does
// not operate.
func TestE2E_DatabaseRequiresAuditOnReversedRows(t *testing.T) {
	db := testDB(t)
	seedReversal(t, db)

	if _, err := db.Exec(`UPDATE finance.transactions SET status = 'REVERSED' WHERE id = $1`, revTxID); err == nil {
		t.Error("the database accepted REVERSED with no reason and no actor — the audit constraint is missing")
	}

	if _, err := db.Exec(`UPDATE finance.transactions
		SET status = 'REVERSED', reversal_reason = 'DUPLICADO', reversed_by = 'teste', reversed_at = NOW()
		WHERE id = $1`, revTxID); err != nil {
		t.Fatalf("the CHECK rejects a properly audited reversal — migration wrong: %v", err)
	}
}

// The three findings that blocked the first review round were the same failure: the
// rest of the system did not know REVERSED existed. One test per consumer.

func TestE2E_ReversedPurchaseLeavesTheInvoice(t *testing.T) {
	// A reversed card purchase that still counts makes the invoice claim a debt with
	// no counterpart — and the recalculation that runs right after a reversal would
	// rewrite the total with the reversed line still in it.
	db := testDB(t)
	seedReversal(t, db)

	const invoiceID = "e1e00000-0000-0000-0000-000000000009"
	t.Cleanup(func() { db.Exec(`DELETE FROM finance.credit_card_invoices WHERE id = $1`, invoiceID) })
	exec(t, db, `INSERT INTO finance.credit_card_invoices
		(id, bank_account_id, reference_date, opening_date, closing_date, due_date, amount, status)
		VALUES ($1,$2,'2026-07-01','2026-06-27','2026-07-27','2026-08-03',55.58,'CLOSED')
		ON CONFLICT (id) DO NOTHING`, invoiceID, revAccountID)
	exec(t, db, `UPDATE finance.transactions SET invoice_id = $1 WHERE id = $2`, invoiceID, revTxID)

	txRepo := persistence.NewTransactionRepository(db)
	accountRepo := persistence.NewBankAccountRepository(db)
	recalc := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	before, err := txRepo.SumByInvoiceID(invoiceID)
	if err != nil {
		t.Fatalf("summing before: %v", err)
	}
	if before != 55.58 {
		t.Fatalf("setup: invoice should total 55.58, got %.2f", before)
	}

	if err := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalc).
		ExecuteWithReason(usecases.ReverseTransactionInput{
			ID: revTxID, Reason: transaction.ReasonNeverHappened, By: "teste",
		}); err != nil {
		t.Fatalf("reversing: %v", err)
	}

	after, err := txRepo.SumByInvoiceID(invoiceID)
	if err != nil {
		t.Fatalf("summing after: %v", err)
	}
	if after != 0 {
		t.Errorf("invoice still totals %.2f — a reversed purchase must leave the bill", after)
	}
}

func TestE2E_ReversedRowCannotBeConfirmedBackToLife(t *testing.T) {
	// PUT /transactions/{id}/status was a second write path with no guard: it brought
	// a reversed row back to CONFIRMED and applied its balance a second time.
	db := testDB(t)
	seedReversal(t, db)

	txRepo := persistence.NewTransactionRepository(db)
	accountRepo := persistence.NewBankAccountRepository(db)
	recalc := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalc).
		ExecuteWithReason(usecases.ReverseTransactionInput{
			ID: revTxID, Reason: transaction.ReasonNeverHappened, By: "teste",
		}); err != nil {
		t.Fatalf("reversing: %v", err)
	}

	txn, err := txRepo.GetByID(revTxID)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if err := txn.Confirm(txn.OccurredOn); err == nil {
		t.Error("confirming a reversed row must be refused — REVERSED is terminal")
	}
	if err := txn.CanTransitionTo(transaction.StatusCancelled); err == nil {
		t.Error("cancelling a reversed row must be refused too")
	}
}

func TestE2E_ReversedRowIsHiddenFromTheDefaultListing(t *testing.T) {
	// Showing it by default is how the next reconciliation finds it on the system
	// side as "extra" and hunts the very phantom this feature prevents.
	db := testDB(t)
	seedReversal(t, db)

	txRepo := persistence.NewTransactionRepository(db)
	accountRepo := persistence.NewBankAccountRepository(db)
	recalc := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	if err := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalc).
		ExecuteWithReason(usecases.ReverseTransactionInput{
			ID: revTxID, Reason: transaction.ReasonDuplicated, By: "teste",
		}); err != nil {
		t.Fatalf("reversing: %v", err)
	}

	account := revAccountID
	visible, err := txRepo.List(transaction.ListFilter{ProfileID: revProfileID, BankAccountID: &account})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	for _, tx := range visible {
		if tx.ID == revTxID {
			t.Error("a reversed row must not appear in the default listing")
		}
	}

	audit, err := txRepo.List(transaction.ListFilter{ProfileID: revProfileID, BankAccountID: &account, IncludeReversed: true})
	if err != nil {
		t.Fatalf("listing for audit: %v", err)
	}
	found := false
	for _, tx := range audit {
		if tx.ID == revTxID {
			found = true
		}
	}
	if !found {
		t.Error("the audit view must still see it — otherwise the row is lost in practice")
	}
}
