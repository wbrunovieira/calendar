//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Against a real database, because the guarantees that matter here are the ones a fake
// cannot have: the UNIQUE that makes re-import idempotent, the CHECK that refuses an
// ignored line with no reason, and whether an upsert preserves reconciliation work.

const (
	stProfileID = "e2e00000-0000-0000-0000-0000000000a1"
	stAccountID = "e2e00000-0000-0000-0000-0000000000a2"
	stTxID      = "e2e00000-0000-0000-0000-0000000000a3"
)

func seedStatementAccount(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.bank_statement_lines WHERE account_id = $1`, stAccountID)
		db.Exec(`DELETE FROM finance.transactions WHERE id = $1`, stTxID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id = $1`, stAccountID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, stProfileID)
	})
	exec(t, db, `INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1,$2,'E2E Extrato','BUSINESS') ON CONFLICT (id) DO NOTHING`, stProfileID, "e2e-statement")
	exec(t, db, `INSERT INTO finance.bank_accounts (id, profile_id, name, type, initial_balance, current_balance, currency)
		VALUES ($1,$2,'Conta Extrato','CHECKING',0,0,'BRL') ON CONFLICT (id) DO NOTHING`, stAccountID, stProfileID)
	// A real entry to match against: the foreign key is the point — a match must name
	// something that exists.
	exec(t, db, `INSERT INTO finance.transactions
		(id, profile_id, bank_account_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1,$2,$3,'EXPENSE','CONFIRMED',55.58,'BRL','Cloudflare','2026-07-11')
		ON CONFLICT (id) DO NOTHING`, stTxID, stProfileID, stAccountID)
}

func lineFor(t *testing.T, externalID string, amountMinor int64, currency string, converted *int64) *statement.Line {
	t.Helper()
	l, err := statement.New(statement.CreateParams{
		AccountID:          stAccountID,
		Provider:           statement.ProviderPluggy,
		ExternalID:         externalID,
		BookedDate:         time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC),
		AmountMinor:        amountMinor,
		Currency:           currency,
		AmountAccountMinor: converted,
		Description:        "Anthropic* Claude Sub",
		AccountCurrency:    "BRL",
		Raw:                json.RawMessage(`{"id":"` + externalID + `","source":"test"}`),
	})
	if err != nil {
		t.Fatalf("building the line: %v", err)
	}
	return l
}

func TestE2E_StatementImportIsIdempotent(t *testing.T) {
	// The same statement window gets pulled again and again. Re-importing must not
	// duplicate: this codebase already paid for that lesson with 223 thousand
	// duplicated Binance rows.
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	converted := int64(57138)
	inserted, updated, err := repo.UpsertMany([]*statement.Line{lineFor(t, "plg-1", 10754, "USD", &converted)})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if inserted != 1 || updated != 0 {
		t.Errorf("first import: inserted=%d updated=%d, want 1/0", inserted, updated)
	}

	// Same provider id, brand new row object: the second pull of the same window.
	inserted, updated, err = repo.UpsertMany([]*statement.Line{lineFor(t, "plg-1", 10754, "USD", &converted)})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if inserted != 0 || updated != 1 {
		t.Errorf("second import: inserted=%d updated=%d, want 0/1 — the line must be recognised, not duplicated", inserted, updated)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM finance.bank_statement_lines WHERE account_id = $1`, stAccountID).Scan(&count)
	if count != 1 {
		t.Errorf("%d rows stored, want 1", count)
	}
}

func TestE2E_ReimportRefreshesTheBankSideButKeepsReconciliationWork(t *testing.T) {
	// Pluggy rewrites transactions. The refresh must land, but a line already
	// reconciled must not silently go back to unmatched — that is hours of work
	// erased by a routine sync.
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	if _, _, err := repo.UpsertMany([]*statement.Line{lineFor(t, "plg-2", 5558, "BRL", nil)}); err != nil {
		t.Fatalf("import: %v", err)
	}
	stored, err := repo.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-2")
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if err := stored.MarkMatched(5558); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if err := repo.Update(stored); err != nil {
		t.Fatalf("marking matched: %v", err)
	}

	// The bank now reports a different description for the same line.
	refreshed := lineFor(t, "plg-2", 5558, "BRL", nil)
	refreshed.Description = "Cloudflare - descricao corrigida pelo banco"
	if _, _, err := repo.UpsertMany([]*statement.Line{refreshed}); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	after, err := repo.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-2")
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if after.Description != "Cloudflare - descricao corrigida pelo banco" {
		t.Error("the bank's own fields must be refreshed on re-import")
	}
	if after.Status != statement.StatusMatched {
		t.Errorf("status = %s — a routine sync must not undo reconciliation work", after.Status)
	}
}

func TestE2E_ForeignLineKeepsBothAmounts(t *testing.T) {
	// Reading USD as BRL invented a 5.2x discrepancy with a supplier and had a support
	// ticket opened over nothing. Both numbers survive the round trip.
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	converted := int64(57138)
	if _, _, err := repo.UpsertMany([]*statement.Line{lineFor(t, "plg-3", 10754, "USD", &converted)}); err != nil {
		t.Fatalf("import: %v", err)
	}

	line, err := repo.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-3")
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if line.AmountMinor != 10754 || line.Currency != "USD" {
		t.Errorf("the original charge was lost: %d %s", line.AmountMinor, line.Currency)
	}
	if line.InAccountCurrency() != 57138 {
		t.Errorf("reconciling value = %d, want 57138 — never the face value of a foreign line", line.InAccountCurrency())
	}
	if !line.IsForeign("BRL") {
		t.Error("the line must know it was charged in another currency")
	}
	if len(line.Raw) == 0 {
		t.Error("the raw payload must survive the round trip")
	}
}

func TestE2E_DatabaseRefusesAnIgnoredLineWithoutReason(t *testing.T) {
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	if _, _, err := repo.UpsertMany([]*statement.Line{lineFor(t, "plg-4", 100, "BRL", nil)}); err != nil {
		t.Fatalf("import: %v", err)
	}

	if _, err := db.Exec(`UPDATE finance.bank_statement_lines SET status = 'IGNORED' WHERE external_id = 'plg-4'`); err == nil {
		t.Error("the database accepted an ignored line with no reason — it would be indistinguishable from one nobody looked at")
	}
	if _, err := db.Exec(`UPDATE finance.bank_statement_lines SET status = 'IGNORED', ignored_reason = 'par do rotativo' WHERE external_id = 'plg-4'`); err != nil {
		t.Fatalf("a properly justified ignore was rejected: %v", err)
	}
}

func TestE2E_ListNarrowsByAccountDateAndStatus(t *testing.T) {
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	older := lineFor(t, "plg-5", 100, "BRL", nil)
	older.BookedDate = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := repo.UpsertMany([]*statement.Line{older, lineFor(t, "plg-6", 200, "BRL", nil)}); err != nil {
		t.Fatalf("import: %v", err)
	}

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	unmatched := statement.StatusUnmatched
	got, err := repo.List(statement.ListFilter{AccountID: stAccountID, From: &from, Status: &unmatched})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(got) != 1 || got[0].ExternalID != "plg-6" {
		t.Errorf("got %d lines, want only the July one", len(got))
	}
}

func TestE2E_EveryVersionTheBankReportedIsKept(t *testing.T) {
	// Overwriting raw destroys the evidence that a change came from the bank and not
	// from us — and that evidence is the whole reason this table exists.
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	first := lineFor(t, "plg-rev", 5558, "BRL", nil)
	if _, _, err := repo.UpsertMany([]*statement.Line{first}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	second := lineFor(t, "plg-rev", 6000, "BRL", nil)
	second.Description = "valor reapresentado pelo banco"
	if _, _, err := repo.UpsertMany([]*statement.Line{second}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	stored, err := repo.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-rev")
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	revisions, err := repo.Revisions(stored.ID)
	if err != nil {
		t.Fatalf("reading revisions: %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("%d revisions kept, want 2 — what the bank said yesterday must survive", len(revisions))
	}
	if revisions[0].AmountMinor != 6000 || revisions[1].AmountMinor != 5558 {
		t.Errorf("revisions out of order or lost: %d then %d", revisions[0].AmountMinor, revisions[1].AmountMinor)
	}
}

func TestE2E_AMatchIsDroppedWhenTheBankSideChanges(t *testing.T) {
	// PENDING settling as POSTED routinely changes the amount, especially on foreign
	// purchases. A match whose bank side moved is by definition unverified: leaving it
	// MATCHED asserts a check nobody performed.
	db := testDB(t)
	seedStatementAccount(t, db)
	repo := persistence.NewStatementRepository(db)

	if _, _, err := repo.UpsertMany([]*statement.Line{lineFor(t, "plg-drop", 5558, "BRL", nil)}); err != nil {
		t.Fatalf("import: %v", err)
	}
	stored, _ := repo.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-drop")
	if err := stored.MarkMatched(5558); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if err := repo.Update(stored); err != nil {
		t.Fatalf("saving the match: %v", err)
	}

	// The bank now reports a different amount for the same line.
	if _, _, err := repo.UpsertMany([]*statement.Line{lineFor(t, "plg-drop", 6000, "BRL", nil)}); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	after, _ := repo.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-drop")
	if after.Status != statement.StatusUnmatched {
		t.Errorf("status = %s, want UNMATCHED — the amount moved, so the match was never verified against this value", after.Status)
	}
	// And the reason is discoverable: the previous version is still on file.
	revisions, _ := repo.Revisions(after.ID)
	if len(revisions) < 2 {
		t.Error("without the previous revision, a dropped match has no discoverable cause")
	}
}

func TestE2E_ReversalUndoesMatchesAndKeepsTheReason(t *testing.T) {
	// Undoing must leave the cause on file. A plain status flip left whoever
	// reconciles next seeing a line return to pending with no explanation.
	db := testDB(t)
	seedStatementAccount(t, db)
	lines := persistence.NewStatementRepository(db)
	matches := persistence.NewMatchRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	accountRepo := persistence.NewBankAccountRepository(db)

	if _, _, err := lines.UpsertMany([]*statement.Line{lineFor(t, "plg-match", 5558, "BRL", nil)}); err != nil {
		t.Fatalf("import: %v", err)
	}
	line, _ := lines.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-match")
	m, err := statement.NewMatch(line.ID, stTxID, 5558, statement.MethodExternalID, "teste")
	if err != nil {
		t.Fatalf("building the match: %v", err)
	}
	if err := matches.Create(m); err != nil {
		t.Fatalf("saving the match: %v", err)
	}
	if err := line.MarkMatched(5558); err != nil {
		t.Fatalf("marking: %v", err)
	}
	if err := lines.Update(line); err != nil {
		t.Fatalf("saving the line: %v", err)
	}

	recalc := usecases.NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	if err := usecases.NewDeleteTransactionUseCase(txRepo, accountRepo, recalc).
		ExecuteWithReason(usecases.ReverseTransactionInput{
			ID: stTxID, Reason: transaction.ReasonDuplicated, By: "teste",
		}); err != nil {
		t.Fatalf("reversing: %v", err)
	}

	stored, err := matches.ByLine(line.ID)
	if err != nil {
		t.Fatalf("reading matches: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("the match row must survive the undo, got %d", len(stored))
	}
	if !stored[0].IsUndone() {
		t.Error("the match must be undone")
	}
	if stored[0].UnmatchedReason == nil || *stored[0].UnmatchedReason != statement.UnmatchTransactionReversed {
		t.Errorf("reason = %v, want TRANSACTION_REVERSED — without it the line is pending with no explanation", stored[0].UnmatchedReason)
	}
	if statement.CoveredMinor(stored) != 0 {
		t.Error("an undone match must not count as coverage")
	}

	after, _ := lines.FindByExternalID(stAccountID, statement.ProviderPluggy, "plg-match")
	if after.Status != statement.StatusUnmatched {
		t.Errorf("line status = %s, want UNMATCHED: nothing covers it any more", after.Status)
	}
}
