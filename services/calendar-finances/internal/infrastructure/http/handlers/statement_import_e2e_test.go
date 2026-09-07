//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/statement"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
)

// Against a real database, because the guarantees that matter here are the ones a fake
// cannot have: the UNIQUE that makes re-import idempotent, the CHECK that refuses an
// ignored line with no reason, and whether an upsert preserves reconciliation work.

const (
	stProfileID = "e2e00000-0000-0000-0000-0000000000a1"
	stAccountID = "e2e00000-0000-0000-0000-0000000000a2"
)

func seedStatementAccount(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM finance.bank_statement_lines WHERE account_id = $1`, stAccountID)
		db.Exec(`DELETE FROM finance.bank_accounts WHERE id = $1`, stAccountID)
		db.Exec(`DELETE FROM finance.profiles WHERE id = $1`, stProfileID)
	})
	exec(t, db, `INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1,$2,'E2E Extrato','BUSINESS') ON CONFLICT (id) DO NOTHING`, stProfileID, "e2e-statement")
	exec(t, db, `INSERT INTO finance.bank_accounts (id, profile_id, name, type, initial_balance, current_balance, currency)
		VALUES ($1,$2,'Conta Extrato','CHECKING',0,0,'BRL') ON CONFLICT (id) DO NOTHING`, stAccountID, stProfileID)
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
	stored, err := repo.FindByExternalID(statement.ProviderPluggy, "plg-2")
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	stored.MarkMatched()
	if err := repo.Update(stored); err != nil {
		t.Fatalf("marking matched: %v", err)
	}

	// The bank now reports a different description for the same line.
	refreshed := lineFor(t, "plg-2", 5558, "BRL", nil)
	refreshed.Description = "Cloudflare - descricao corrigida pelo banco"
	if _, _, err := repo.UpsertMany([]*statement.Line{refreshed}); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	after, err := repo.FindByExternalID(statement.ProviderPluggy, "plg-2")
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

	line, err := repo.FindByExternalID(statement.ProviderPluggy, "plg-3")
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if line.AmountMinor != 10754 || line.Currency != "USD" {
		t.Errorf("the original charge was lost: %d %s", line.AmountMinor, line.Currency)
	}
	if line.InAccountCurrency() != 57138 {
		t.Errorf("reconciling value = %d, want 57138 — never the face value of a foreign line", line.InAccountCurrency())
	}
	if !line.IsForeign() {
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
