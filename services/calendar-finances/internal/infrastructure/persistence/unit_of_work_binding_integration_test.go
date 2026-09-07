//go:build integration
// +build integration

package persistence

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func rollbackFixture(t *testing.T) (profileID, accountID string) {
	t.Helper()
	profileID, accountID = uuid.NewString(), uuid.NewString()
	return profileID, accountID
}

func charge(t *testing.T, profileID, accountID, description string) *transaction.Transaction {
	t.Helper()
	txn, err := transaction.New(transaction.CreateParams{
		ProfileID: profileID, BankAccountID: accountID,
		Type: transaction.TypeExpense, Amount: 100, Currency: "BRL",
		Description: description,
		OccurredOn:  time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build transaction: %v", err)
	}
	return txn
}

// Writing through the BOUND repositories is what makes a rollback undo anything.
//
// This is the shape the production wiring must have. The earlier version handed the
// use cases repositories built on the *sql.DB and merely opened a transaction around
// them: each Create then opened and committed a transaction of its own, the outer
// rollback found nothing to undo, and a series that failed halfway left its earlier
// rows behind. A twelve-instalment purchase failing at part seven committed six.
func TestIntegration_ARollbackUndoesEveryWriteMadeThroughTheBoundRepositories(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := rollbackFixture(t)
	seedProfileAndAccount(t, db, profileID, accountID)

	boom := errors.New("the last write failed")
	err := NewUnitOfWork(db).Do(func(r Repositories) error {
		for _, name := range []string{"parcela 1/3", "parcela 2/3"} {
			if createErr := r.Transactions.Create(charge(t, profileID, accountID, name)); createErr != nil {
				return createErr
			}
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected the failure to travel up, got %v", err)
	}

	var surviving int
	if err := db.QueryRow(
		`SELECT count(*) FROM finance.transactions WHERE bank_account_id = $1`, accountID,
	).Scan(&surviving); err != nil {
		t.Fatalf("count: %v", err)
	}
	if surviving != 0 {
		t.Errorf("a failed series left %d rows behind; the rollback undid nothing", surviving)
	}
}

// The negative control, kept on purpose: it documents the trap rather than the fix.
//
// A repository built on the pool escapes the surrounding transaction entirely. Nothing
// errors, nothing warns — the rows simply survive a rollback. That is why the unit of
// work hands its callback the bound repositories instead of trusting the caller to
// have used the right ones, and why the interface takes TxRepos so the mistake stops
// compiling rather than merely being discouraged.
func TestIntegration_ARepositoryOnThePoolEscapesTheTransaction(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := rollbackFixture(t)
	seedProfileAndAccount(t, db, profileID, accountID)

	escapee := NewTransactionRepository(db)

	boom := errors.New("rolled back")
	_ = NewUnitOfWork(db).Do(func(Repositories) error {
		if createErr := escapee.Create(charge(t, profileID, accountID, "escapou")); createErr != nil {
			return createErr
		}
		return boom
	})

	var surviving int
	if err := db.QueryRow(
		`SELECT count(*) FROM finance.transactions WHERE bank_account_id = $1`, accountID,
	).Scan(&surviving); err != nil {
		t.Fatalf("count: %v", err)
	}
	if surviving != 1 {
		t.Fatalf("expected the escaped write to survive the rollback, found %d rows — "+
			"if this now reports 0, repositories on the pool have started joining the "+
			"surrounding transaction and the comment above is out of date", surviving)
	}
}
