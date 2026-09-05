//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	_ "github.com/lib/pq"
)

func getTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}

func cleanupTestData(db *sql.DB, profileID string) {
	db.Exec("DELETE FROM finance.transaction_tags WHERE transaction_id IN (SELECT id FROM finance.transactions WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM finance.transaction_splits WHERE transaction_id IN (SELECT id FROM finance.transactions WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", profileID)
}

func TestTransactionRepository_List_FilterByDateRange_Integration(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewTransactionRepository(db)
	profileID := "test-profile-date-filter"

	// Cleanup before and after test
	cleanupTestData(db, profileID)
	defer cleanupTestData(db, profileID)

	// Create transactions in different months
	janTx := &transaction.Transaction{
		ID:            "tx-jan-integration",
		ProfileID:     profileID,
		BankAccountID: "account-1",
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "January transaction",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}

	febTx1 := &transaction.Transaction{
		ID:            "tx-feb-1-integration",
		ProfileID:     profileID,
		BankAccountID: "account-1",
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200,
		Currency:      "BRL",
		Description:   "February transaction 1",
		OccurredOn:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	febTx2 := &transaction.Transaction{
		ID:            "tx-feb-2-integration",
		ProfileID:     profileID,
		BankAccountID: "account-1",
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        300,
		Currency:      "BRL",
		Description:   "February transaction 2",
		OccurredOn:    time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	marTx := &transaction.Transaction{
		ID:            "tx-mar-integration",
		ProfileID:     profileID,
		BankAccountID: "account-1",
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        400,
		Currency:      "BRL",
		Description:   "March transaction",
		OccurredOn:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	}

	// Create all transactions
	for _, tx := range []*transaction.Transaction{janTx, febTx1, febTx2, marTx} {
		if err := repo.Create(tx); err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}
	}

	// Test: Filter for February only
	fromDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := repo.List(transaction.ListFilter{
		ProfileID:    profileID,
		OccurredFrom: &fromDate,
		OccurredTo:   &toDate,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 transactions for February, got %d", len(result))
		for _, tx := range result {
			t.Logf("Got transaction: %s, date: %v", tx.Description, tx.OccurredOn)
		}
	}

	// Verify all returned transactions are in February
	for _, tx := range result {
		if tx.OccurredOn.Month() != time.February {
			t.Errorf("Expected transaction in February, got %v", tx.OccurredOn)
		}
	}
}
