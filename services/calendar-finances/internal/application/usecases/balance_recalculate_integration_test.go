//go:build integration
// +build integration

package usecases

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/database"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		// Skipping locally is a convenience. In CI the database is a service
		// container that is always there, so a skip would be a green tick over
		// nothing.
		if os.Getenv("CI") != "" {
			t.Fatalf("integration DB unavailable in CI: %v", err)
		}
		t.Skipf("integration DB unavailable (%v); skipping", err)
	}
	// A developer's database already has the schema; CI's is empty. Running the
	// migrations here is what lets the same test serve both.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}

func cleanIntegrationData(db *sql.DB, profileID string) {
	db.Exec("DELETE FROM finance.transaction_tags WHERE transaction_id IN (SELECT id FROM finance.transactions WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM finance.transaction_splits WHERE transaction_id IN (SELECT id FROM finance.transactions WHERE profile_id = $1)", profileID)
	db.Exec("DELETE FROM finance.transactions WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM finance.bank_accounts WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM finance.categories WHERE profile_id = $1", profileID)
	db.Exec("DELETE FROM finance.profiles WHERE id = $1", profileID)
}

func integrationSeedProfile(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type, created_at, updated_at)
		VALUES ($1::uuid, 'integration-' || $2, $2, 'PERSONAL', now(), now())
		ON CONFLICT (id) DO NOTHING
	`, id, id)
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}
}

func integrationSeedAccount(t *testing.T, db *sql.DB, acc *bankaccount.BankAccount) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO finance.bank_accounts (
			id, profile_id, name, type, initial_balance, current_balance, currency,
			is_active, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,true,now(),now())
	`, acc.ID, acc.ProfileID, acc.Name, acc.Type,
		acc.InitialBalance, acc.CurrentBalance, acc.Currency)
	if err != nil {
		t.Fatalf("seed account %s: %v", acc.ID, err)
	}
}

func integrationSeedCategory(t *testing.T, db *sql.DB, cat *category.Category) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO finance.categories (id, profile_id, name, type, created_at, updated_at)
		VALUES ($1,$2,$3,$4,now(),now())
		ON CONFLICT (id) DO NOTHING
	`, cat.ID, cat.ProfileID, cat.Name, cat.Type)
	if err != nil {
		t.Fatalf("seed category %s: %v", cat.ID, err)
	}
}

func dbCurrentBalance(t *testing.T, db *sql.DB, accountID string) float64 {
	t.Helper()
	var bal float64
	if err := db.QueryRow(`SELECT current_balance FROM finance.bank_accounts WHERE id = $1`, accountID).Scan(&bal); err != nil {
		t.Fatalf("read balance for %s: %v", accountID, err)
	}
	return bal
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

// TestIntegration_CreateConfirmedExpense_RecalculatesBalance proves that after
// creating a confirmed expense the balance is recomputed from scratch (not just
// decremented), so any pre-existing drift is corrected.
func TestIntegration_CreateConfirmedExpense_RecalculatesBalance(t *testing.T) {
	db := integrationDB(t)
	defer db.Close()

	pid := uuid.NewString()
	aid := uuid.NewString()
	cid := uuid.NewString()

	cleanIntegrationData(db, pid)
	defer cleanIntegrationData(db, pid)

	integrationSeedProfile(t, db, pid)
	integrationSeedAccount(t, db, &bankaccount.BankAccount{
		ID: aid, ProfileID: pid, Name: "checking",
		Type:           bankaccount.AccountTypeChecking,
		InitialBalance: 1000,
		CurrentBalance: 9999, // intentional drift — recalc must fix this
		Currency:       "BRL",
	})
	cat := &category.Category{ID: cid, ProfileID: pid, Name: "food", Type: category.TypeExpense}
	integrationSeedCategory(t, db, cat)

	profileRepo := persistence.NewProfileRepository(db)
	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	recalcUC := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	uc := NewCreateTransactionUseCase(
		profileRepo,
		accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{cid: cat}},
		txRepo,
		&fakeInvoiceRepo{},
		recalcUC,
		nil,
	)

	status := "CONFIRMED"
	_, err := uc.Execute(CreateTransactionInput{
		ProfileID:     pid,
		BankAccountID: aid,
		CategoryID:    strPtr(cid),
		Type:          "EXPENSE",
		Status:        &status,
		Amount:        200,
		Currency:      "BRL",
		Description:   "groceries",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// initialBalance(1000) + expense(-200) = 800
	// Without recalc: 9999 - 200 = 9799 (drift preserved)
	bal := dbCurrentBalance(t, db, aid)
	if bal != 800 {
		t.Errorf("expected balance 800, got %.2f", bal)
	}
}

// TestIntegration_DeleteConfirmedExpense_RecalculatesBalance verifies that
// deleting a confirmed expense re-computes the balance from scratch.
func TestIntegration_DeleteConfirmedExpense_RecalculatesBalance(t *testing.T) {
	db := integrationDB(t)
	defer db.Close()

	pid := uuid.NewString()
	aid := uuid.NewString()

	cleanIntegrationData(db, pid)
	defer cleanIntegrationData(db, pid)

	integrationSeedProfile(t, db, pid)
	integrationSeedAccount(t, db, &bankaccount.BankAccount{
		ID: aid, ProfileID: pid, Name: "checking",
		Type: bankaccount.AccountTypeChecking, InitialBalance: 500, CurrentBalance: 500,
		Currency: "BRL",
	})

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	recalcUC := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	// Seed a confirmed expense
	txID := uuid.NewString()
	tx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     pid,
		BankAccountID: aid,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        150,
		Currency:      "BRL",
		Description:   "to delete",
		OccurredOn:    time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := txRepo.Create(tx); err != nil {
		t.Fatalf("seed tx: %v", err)
	}
	// Introduce drift to prove recalc corrects it
	db.Exec(`UPDATE finance.bank_accounts SET current_balance = 9999 WHERE id = $1`, aid)

	deleteUC := NewDeleteTransactionUseCase(txRepo, accountRepo, recalcUC)
	if err := deleteUC.Execute(txID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// initialBalance(500) + 0 confirmed transactions = 500
	bal := dbCurrentBalance(t, db, aid)
	if bal != 500 {
		t.Errorf("expected balance 500 after delete, got %.2f", bal)
	}
}

// TestIntegration_UpdateStatus_PlannedToConfirmed_RecalculatesBalance verifies
// that confirming a planned transaction recalculates balance from scratch.
func TestIntegration_UpdateStatus_PlannedToConfirmed_RecalculatesBalance(t *testing.T) {
	db := integrationDB(t)
	defer db.Close()

	pid := uuid.NewString()
	aid := uuid.NewString()

	cleanIntegrationData(db, pid)
	defer cleanIntegrationData(db, pid)

	integrationSeedProfile(t, db, pid)
	integrationSeedAccount(t, db, &bankaccount.BankAccount{
		ID: aid, ProfileID: pid, Name: "checking",
		Type: bankaccount.AccountTypeChecking, InitialBalance: 300, CurrentBalance: 300,
		Currency: "BRL",
	})

	accountRepo := persistence.NewBankAccountRepository(db)
	txRepo := persistence.NewTransactionRepository(db)
	recalcUC := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	txID := uuid.NewString()
	tx := &transaction.Transaction{
		ID:            txID,
		ProfileID:     pid,
		BankAccountID: aid,
		Type:          transaction.TypeIncome,
		Status:        transaction.StatusPlanned,
		Amount:        700,
		Currency:      "BRL",
		Description:   "salary",
		OccurredOn:    time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := txRepo.Create(tx); err != nil {
		t.Fatalf("seed tx: %v", err)
	}
	db.Exec(`UPDATE finance.bank_accounts SET current_balance = 9999 WHERE id = $1`, aid)

	statusUC := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, recalcUC)
	_, err := statusUC.Execute(txID, UpdateTransactionStatusInput{Status: "CONFIRMED"})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	// initialBalance(300) + income(700) = 1000
	bal := dbCurrentBalance(t, db, aid)
	if bal != 1000 {
		t.Errorf("expected balance 1000, got %.2f", bal)
	}
}
