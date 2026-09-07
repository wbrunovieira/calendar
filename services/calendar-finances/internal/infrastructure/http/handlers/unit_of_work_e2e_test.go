//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// The instalment loop writes rows one at a time with no database transaction around
// it, so a failure at part 7 of 12 leaves six committed and the caller is told the
// whole thing failed. Nobody goes looking, and the ledger carries half a plan.
//
// Against a real database, because atomicity cannot be faked: an in-memory repository
// has no rollback to get wrong.

const (
	uowProfileID = "e2e00000-0000-0000-0000-0000000000b1"
	uowAccountID = "e2e00000-0000-0000-0000-0000000000b2"
)

func seedUnitOfWork(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.transactions WHERE profile_id = $1`, uowProfileID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id = $1`, uowAccountID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, uowProfileID)
	})
	exec(t, db, `INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1,$2,'E2E UoW','BUSINESS') ON CONFLICT (id) DO NOTHING`, uowProfileID, "e2e-uow")
	exec(t, db, `INSERT INTO finance.bank_accounts (id, profile_id, name, type, initial_balance, current_balance, currency)
		VALUES ($1,$2,'Conta UoW','CHECKING',0,0,'BRL') ON CONFLICT (id) DO NOTHING`, uowAccountID, uowProfileID)
}

func uowTransaction(n int) *transaction.Transaction {
	total := 3
	num := n
	return &transaction.Transaction{
		ProfileID: uowProfileID, BankAccountID: uowAccountID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 284.49, Currency: "BRL", Description: "Parcelamento fatura",
		OccurredOn:        time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
		InstallmentNumber: &num, InstallmentTotal: &total,
	}
}

func countUoWRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM finance.transactions WHERE profile_id = $1`, uowProfileID).Scan(&n); err != nil {
		t.Fatalf("counting: %v", err)
	}
	return n
}

func TestE2E_UnitOfWorkCommitsEverythingOrNothing(t *testing.T) {
	db := testDB(t)
	seedUnitOfWork(t, db)
	uow := persistence.NewUnitOfWork(db)

	err := uow.Do(func(r persistence.Repositories) error {
		for n := 1; n <= 3; n++ {
			txn, nerr := transaction.New(transaction.CreateParams{
				ProfileID: uowProfileID, BankAccountID: uowAccountID,
				Type: transaction.TypeExpense, Amount: 284.49, Currency: "BRL",
				Description: "Parcelamento fatura", OccurredOn: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
			})
			if nerr != nil {
				return nerr
			}
			if cerr := r.Transactions.Create(txn); cerr != nil {
				return cerr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := countUoWRows(t, db); got != 3 {
		t.Errorf("%d rows committed, want 3", got)
	}
}

func TestE2E_UnitOfWorkRollsBackAPartialSeries(t *testing.T) {
	// The whole point: failing at part 3 must leave zero rows, not two.
	db := testDB(t)
	seedUnitOfWork(t, db)
	uow := persistence.NewUnitOfWork(db)

	boom := errors.New("falha na parcela 3")
	err := uow.Do(func(r persistence.Repositories) error {
		for n := 1; n <= 3; n++ {
			if n == 3 {
				return boom
			}
			txn, _ := transaction.New(transaction.CreateParams{
				ProfileID: uowProfileID, BankAccountID: uowAccountID,
				Type: transaction.TypeExpense, Amount: 284.49, Currency: "BRL",
				Description: "Parcelamento fatura", OccurredOn: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
			})
			if cerr := r.Transactions.Create(txn); cerr != nil {
				return cerr
			}
		}
		return nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the original failure surfaced", err)
	}
	if got := countUoWRows(t, db); got != 0 {
		t.Errorf("%d rows survived a failed series, want 0 — a half-written plan is what this exists to prevent", got)
	}
}

func TestE2E_UnitOfWorkRollsBackOnPanic(t *testing.T) {
	// A panic mid-series must not commit what came before it.
	db := testDB(t)
	seedUnitOfWork(t, db)
	uow := persistence.NewUnitOfWork(db)

	func() {
		defer func() { _ = recover() }()
		_ = uow.Do(func(r persistence.Repositories) error {
			txn, _ := transaction.New(transaction.CreateParams{
				ProfileID: uowProfileID, BankAccountID: uowAccountID,
				Type: transaction.TypeExpense, Amount: 100, Currency: "BRL",
				Description: "Antes do panico", OccurredOn: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
			})
			_ = r.Transactions.Create(txn)
			panic("boom")
		})
	}()

	if got := countUoWRows(t, db); got != 0 {
		t.Errorf("%d rows survived a panic, want 0", got)
	}
}

// Paying an invoice is FIVE writes: the invoice, the transfer, both account balances,
// and the credit on the card. With no transaction around them and errors discarded in
// the middle, this is the exact shape of the failure that left R$ 8.863,07 in "paid"
// invoices against R$ 769,51 of credit on the card — a phantom of thousands.
//
// The instalment loop was covered first, but the instalment loop never lost money in
// production. This one did.
func TestE2E_InvoicePaymentIsAllOrNothing(t *testing.T) {
	db := testDB(t)
	seedUnitOfWork(t, db)

	const cardID = "e2e00000-0000-0000-0000-0000000000b3"
	const invoiceID = "e2e00000-0000-0000-0000-0000000000b4"
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.transactions WHERE bank_account_id = $1 OR destination_account_id = $1`, cardID)
		db.Exec(`DELETE FROM finance.credit_card_invoices WHERE id = $1`, invoiceID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id = $1`, cardID)
	})
	exec(t, db, `INSERT INTO finance.bank_accounts
		(id, profile_id, name, type, initial_balance, current_balance, currency, closing_day, due_day, linked_account_id)
		VALUES ($1,$2,'Cartao UoW','CREDIT_CARD',0,-500,'BRL',27,3,$3) ON CONFLICT (id) DO NOTHING`,
		cardID, uowProfileID, uowAccountID)
	exec(t, db, `INSERT INTO finance.credit_card_invoices
		(id, bank_account_id, reference_date, opening_date, closing_date, due_date, amount, status)
		VALUES ($1,$2,'2026-09-01','2026-07-27','2026-08-27','2026-09-03',500,'CLOSED')
		ON CONFLICT (id) DO NOTHING`, invoiceID, cardID)

	countAll := func() (invoiceStatus string, txCount int) {
		db.QueryRow(`SELECT status FROM finance.credit_card_invoices WHERE id = $1`, invoiceID).Scan(&invoiceStatus)
		db.QueryRow(`SELECT COUNT(*) FROM finance.transactions
			WHERE bank_account_id = $1 OR destination_account_id = $1`, cardID).Scan(&txCount)
		return
	}

	statusBefore, txBefore := countAll()
	if statusBefore != "CLOSED" || txBefore != 0 {
		t.Fatalf("setup: status=%s tx=%d", statusBefore, txBefore)
	}

	// A failure after the invoice is marked must leave the invoice unmarked too.
	uow := persistence.NewUnitOfWork(db)
	boom := errors.New("falha depois de marcar a fatura")
	err := uow.Do(func(r persistence.Repositories) error {
		inv, ferr := r.Invoices.FindByID(invoiceID)
		if ferr != nil {
			return ferr
		}
		if perr := inv.Pay(500, time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)); perr != nil {
			return perr
		}
		if uerr := r.Invoices.Update(inv); uerr != nil {
			return uerr
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the failure surfaced", err)
	}

	statusAfter, txAfter := countAll()
	if statusAfter != "CLOSED" {
		t.Errorf("invoice status = %s, want CLOSED — a bill marked paid with no payment behind it is the phantom this prevents", statusAfter)
	}
	if txAfter != 0 {
		t.Errorf("%d transactions survived a failed payment, want 0", txAfter)
	}
}
