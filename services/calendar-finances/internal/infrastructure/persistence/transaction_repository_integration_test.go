//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/google/uuid"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
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
	if err := db.Ping(); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("integration DB unavailable in CI: %v", err)
		}
		t.Skipf("integration DB unavailable (%v); skipping", err)
	}
	// A developer's database already has the schema; CI's is empty.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return db
}

// seedProfileAndAccount creates the rows the transactions below point at. Every
// id column in this schema is a uuid and every foreign key is enforced, so a
// test cannot invent "account-1" and expect an insert to land.
// seedProfileAndAccount creates the rows through the REPOSITORY, not with raw SQL.
//
// It used to INSERT directly, and the cost of that was concrete: BankAccountRepository
// .Create was never executed by any test, so a column duplicated in its INSERT list
// broke every POST /bank-accounts while the whole suite stayed green. Seeding this way
// turns roughly every integration test into coverage of the write path, for free.
//
// The profile has no repository of its own yet, so it stays as SQL — and that is the
// remaining hole, not an oversight.
func seedProfileAndAccount(t *testing.T, db *sql.DB, profileID, accountID string) {
	t.Helper()
	seedProfile(t, db, profileID)
	seedAccount(t, db, profileID, accountID, "Conta Integration", bankaccount.AccountTypeChecking)
}

func seedProfile(t *testing.T, db *sql.DB, profileID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'Integration', 'PERSONAL')
		ON CONFLICT (id) DO NOTHING`, profileID, "integration-"+profileID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
}

// seedAccount goes through BankAccountRepository.Create, so the statement that writes
// an account in production is the one the tests exercise.
func seedAccount(t *testing.T, db *sql.DB, profileID, accountID, name string, kind bankaccount.AccountType) {
	t.Helper()
	existing, err := NewBankAccountRepository(db).FindByID(accountID)
	if err == nil && existing != nil {
		return
	}
	if err := NewBankAccountRepository(db).Create(&bankaccount.BankAccount{
		ID: accountID, ProfileID: profileID, Name: name, Type: kind,
		Currency: "BRL", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed account through the repository: %v", err)
	}
}

func cleanupTestData(db *sql.DB, profileID string) {
	db.Exec("DELETE FROM finance.bank_accounts WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM finance.cost_centers WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM finance.profiles WHERE id = $1", profileID)
	db.Exec("DELETE FROM finance.transaction_tags WHERE transaction_id IN (SELECT id FROM finance.transactions WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM finance.transaction_splits WHERE transaction_id IN (SELECT id FROM finance.transactions WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", profileID)
}

func TestTransactionRepository_List_FilterByDateRange_Integration(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewTransactionRepository(db)
	profileID := uuid.NewString()
	accountID := uuid.NewString()

	// Cleanup before and after test
	cleanupTestData(db, profileID)
	defer cleanupTestData(db, profileID)
	seedProfileAndAccount(t, db, profileID, accountID)

	// Create transactions in different months
	janTx := &transaction.Transaction{
		ID:            uuid.NewString(),
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "January transaction",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}

	febTx1 := &transaction.Transaction{
		ID:            uuid.NewString(),
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        200,
		Currency:      "BRL",
		Description:   "February transaction 1",
		OccurredOn:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	febTx2 := &transaction.Transaction{
		ID:            uuid.NewString(),
		ProfileID:     profileID,
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        300,
		Currency:      "BRL",
		Description:   "February transaction 2",
		OccurredOn:    time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	marTx := &transaction.Transaction{
		ID:            uuid.NewString(),
		ProfileID:     profileID,
		BankAccountID: accountID,
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
