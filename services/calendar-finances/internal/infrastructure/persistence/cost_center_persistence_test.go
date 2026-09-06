//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// A cost center that survives the round trip is the whole point: the column and
// its foreign key already existed, and nothing was writing them.
func TestTransactionRepository_PersistsTheCostCenter(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID := uuid.NewString()
	accountID := uuid.NewString()
	centerID := uuid.NewString()

	cleanupTestData(db, profileID)
	defer cleanupTestData(db, profileID)
	seedProfileAndAccount(t, db, profileID, accountID)
	seedCostCenter(t, db, centerID, profileID)

	repo := NewTransactionRepository(db)
	tx := &transaction.Transaction{
		ID:            uuid.NewString(),
		ProfileID:     profileID,
		BankAccountID: accountID,
		CostCenterID:  &centerID,
		Type:          transaction.TypeIncome,
		Status:        transaction.StatusConfirmed,
		Amount:        1500,
		Currency:      "BRL",
		Description:   "Pix cliente Acme",
		OccurredOn:    time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.Create(tx); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(tx.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.CostCenterID == nil {
		t.Fatal("the cost center did not survive the round trip")
	}
	if *got.CostCenterID != centerID {
		t.Errorf("CostCenterID = %q, want %q", *got.CostCenterID, centerID)
	}
}

// Listing has to carry it too, or a revenue-by-client report reads nothing.
func TestTransactionRepository_ListReturnsTheCostCenter(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID := uuid.NewString()
	accountID := uuid.NewString()
	centerID := uuid.NewString()

	cleanupTestData(db, profileID)
	defer cleanupTestData(db, profileID)
	seedProfileAndAccount(t, db, profileID, accountID)
	seedCostCenter(t, db, centerID, profileID)

	repo := NewTransactionRepository(db)
	if err := repo.Create(&transaction.Transaction{
		ID: uuid.NewString(), ProfileID: profileID, BankAccountID: accountID,
		CostCenterID: &centerID, Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 1500, Currency: "BRL", Description: "Pix cliente Acme",
		OccurredOn: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := repo.List(transaction.ListFilter{ProfileID: profileID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(list))
	}
	if list[0].CostCenterID == nil || *list[0].CostCenterID != centerID {
		t.Errorf("the listed transaction lost its cost center: %v", list[0].CostCenterID)
	}
}

// Omitting it must stay legal: every existing caller does.
func TestTransactionRepository_CostCenterIsOptional(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID := uuid.NewString()
	accountID := uuid.NewString()

	cleanupTestData(db, profileID)
	defer cleanupTestData(db, profileID)
	seedProfileAndAccount(t, db, profileID, accountID)

	repo := NewTransactionRepository(db)
	tx := &transaction.Transaction{
		ID: uuid.NewString(), ProfileID: profileID, BankAccountID: accountID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 10, Currency: "BRL", Description: "Sem centro de custo",
		OccurredOn: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.Create(tx); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByID(tx.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.CostCenterID != nil {
		t.Errorf("a cost center appeared out of nowhere: %v", *got.CostCenterID)
	}
}

func seedCostCenter(t *testing.T, db *sql.DB, id, profileID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO finance.cost_centers (id, profile_id, name, type)
		VALUES ($1, $2, 'Acme', 'CLIENT')`, id, profileID); err != nil {
		t.Fatalf("seed cost center: %v", err)
	}
}
