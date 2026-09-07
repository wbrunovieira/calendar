//go:build integration
// +build integration

package persistence

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

func line(t *testing.T, accountID, externalID string, minor int64, day int) *statement.Line {
	t.Helper()
	l, err := statement.New(statement.CreateParams{
		AccountID:   accountID,
		Provider:    statement.ProviderPluggy,
		ExternalID:  externalID,
		BookedDate:  time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC),
		AmountMinor: minor, Currency: "BRL", AccountCurrency: "BRL",
		Description: "linha " + externalID,
		Raw:         json.RawMessage(`{"id":"` + externalID + `"}`),
	})
	if err != nil {
		t.Fatalf("build line: %v", err)
	}
	return l
}

// The cron pulls an overlapping window every morning, so re-importing is the normal
// case. It must refresh what the bank now says without duplicating a row and without
// losing the reconciliation already recorded against it.
func TestIntegration_ReimportingAStatementDoesNotDuplicate(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)
	repo := NewStatementRepository(db)

	first := []*statement.Line{
		line(t, accountID, "ext-a", -4000, 1),
		line(t, accountID, "ext-b", -9990, 2),
	}
	inserted, updated, err := repo.UpsertMany(first)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if inserted != 2 || updated != 0 {
		t.Fatalf("first import: inserted %d, updated %d", inserted, updated)
	}

	// The same window again, with one line the bank has since restated.
	again := []*statement.Line{
		line(t, accountID, "ext-a", -4000, 1),
		line(t, accountID, "ext-b", -10500, 2),
		line(t, accountID, "ext-c", -1500, 3),
	}
	inserted, updated, err = repo.UpsertMany(again)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if inserted != 1 {
		t.Errorf("only the new line is new, got %d inserted", inserted)
	}
	if updated != 2 {
		t.Errorf("the two known lines must be refreshed, got %d updated", updated)
	}

	var total int
	if err := db.QueryRow(
		`SELECT count(*) FROM finance.bank_statement_lines WHERE account_id = $1`, accountID,
	).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected three rows, found %d — the unique key did not hold", total)
	}

	restated, err := repo.FindByExternalID(accountID, statement.ProviderPluggy, "ext-b")
	if err != nil || restated == nil {
		t.Fatalf("find restated line: %v", err)
	}
	if restated.AmountMinor != -10500 {
		t.Errorf("the refreshed value must be what the bank now says, got %d", restated.AmountMinor)
	}
}

// Two different accounts can carry the same provider id — a transfer between the
// owner's own accounts is the everyday case. The key is (account, provider, external),
// and getting that wrong would silently drop one side of every internal transfer.
func TestIntegration_TheSameProviderIdOnTwoAccountsIsTwoLines(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountA := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountA)
	accountB := uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountB)

	repo := NewStatementRepository(db)
	inserted, _, err := repo.UpsertMany([]*statement.Line{
		line(t, accountA, "shared-id", -5000, 4),
		line(t, accountB, "shared-id", 5000, 4),
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if inserted != 2 {
		t.Fatalf("both sides must land, got %d", inserted)
	}
}
