package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/balancecheckpoint"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeCheckpointRepo struct {
	saved  []*balancecheckpoint.BalanceCheckpoint
	latest *balancecheckpoint.BalanceCheckpoint // controlled by test
}

func (r *fakeCheckpointRepo) Save(cp *balancecheckpoint.BalanceCheckpoint) error {
	for i, s := range r.saved {
		if s.AccountID == cp.AccountID && s.ReferenceMonth.Equal(cp.ReferenceMonth) {
			r.saved[i] = cp
			return nil
		}
	}
	r.saved = append(r.saved, cp)
	return nil
}

func (r *fakeCheckpointRepo) FindLatestBefore(accountID string, before time.Time) (*balancecheckpoint.BalanceCheckpoint, error) {
	return r.latest, nil
}

func (r *fakeCheckpointRepo) FindByAccountID(accountID string) ([]*balancecheckpoint.BalanceCheckpoint, error) {
	var result []*balancecheckpoint.BalanceCheckpoint
	for _, cp := range r.saved {
		if cp.AccountID == accountID {
			result = append(result, cp)
		}
	}
	return result, nil
}

// balanceUpToRepo tracks CalculateBalanceUpTo calls and returns a preset value.
type balanceUpToRepo struct {
	fakeTransactionRepo
	upToResult map[string]float64 // accountID → balance
	upToCalls  []string
}

func (r *balanceUpToRepo) CalculateBalanceUpTo(accountID string, upTo time.Time) (float64, error) {
	r.upToCalls = append(r.upToCalls, accountID)
	if v, ok := r.upToResult[accountID]; ok {
		return v, nil
	}
	return 0, nil
}

func (r *balanceUpToRepo) CalculateBalanceSince(accountID string, since time.Time) (float64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// CloseMonth tests
// ---------------------------------------------------------------------------

func TestCloseMonth_CreatesCheckpointForEachCheckingAccount(t *testing.T) {
	acc1 := checkingAccount(testProfile, "acc-a", 0)
	acc1.InitialBalance = 500
	acc2 := checkingAccount(testProfile, "acc-b", 0)
	acc2.InitialBalance = 200

	accounts := map[string]*bankaccount.BankAccount{
		"acc-a": acc1,
		"acc-b": acc2,
	}

	txRepo := &balanceUpToRepo{
		upToResult: map[string]float64{
			"acc-a": 300,  // net tx impact for acc-a
			"acc-b": -100, // net tx impact for acc-b
		},
	}
	cpRepo := &fakeCheckpointRepo{}

	allAccounts := []*bankaccount.BankAccount{acc1, acc2}
	accountRepo := &fakeAccountRepoWithAll{fakeAccountRepo: fakeAccountRepo{accounts: accounts}, all: allAccounts}

	uc := NewCloseMonthUseCase(accountRepo, txRepo, cpRepo)
	result, err := uc.Execute(CloseMonthInput{ReferenceMonth: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CheckpointsCreated != 2 {
		t.Errorf("expected 2 checkpoints, got %d", result.CheckpointsCreated)
	}
	if len(cpRepo.saved) != 2 {
		t.Errorf("expected 2 saved checkpoints, got %d", len(cpRepo.saved))
	}

	for _, cp := range cpRepo.saved {
		switch cp.AccountID {
		case "acc-a":
			// 500 (initial) + 300 (tx) = 800
			if cp.ClosingBalance != 800 {
				t.Errorf("acc-a: expected closing_balance 800, got %.2f", cp.ClosingBalance)
			}
		case "acc-b":
			// 200 (initial) + (-100) = 100
			if cp.ClosingBalance != 100 {
				t.Errorf("acc-b: expected closing_balance 100, got %.2f", cp.ClosingBalance)
			}
		}
	}
}

func TestCloseMonth_SkipsCreditCardAccounts(t *testing.T) {
	cc := creditCardAccount(testProfile, "cc-x")
	checking := checkingAccount(testProfile, "chk-x", 0)

	allAccounts := []*bankaccount.BankAccount{cc, checking}
	accountRepo := &fakeAccountRepoWithAll{
		fakeAccountRepo: fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
			"cc-x":  cc,
			"chk-x": checking,
		}},
		all: allAccounts,
	}
	txRepo := &balanceUpToRepo{upToResult: map[string]float64{"chk-x": 0}}
	cpRepo := &fakeCheckpointRepo{}

	uc := NewCloseMonthUseCase(accountRepo, txRepo, cpRepo)
	result, _ := uc.Execute(CloseMonthInput{ReferenceMonth: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)})

	if result.CheckpointsCreated != 1 {
		t.Errorf("expected only 1 checkpoint (checking), got %d", result.CheckpointsCreated)
	}
	for _, cp := range cpRepo.saved {
		if cp.AccountID == "cc-x" {
			t.Errorf("credit card must not get a checkpoint")
		}
	}
}

func TestCloseMonth_NormalizesReferenceMonthToFirstDay(t *testing.T) {
	acc := checkingAccount(testProfile, "acc-norm", 0)
	allAccounts := []*bankaccount.BankAccount{acc}
	accountRepo := &fakeAccountRepoWithAll{
		fakeAccountRepo: fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-norm": acc}},
		all:             allAccounts,
	}
	txRepo := &balanceUpToRepo{upToResult: map[string]float64{"acc-norm": 0}}
	cpRepo := &fakeCheckpointRepo{}

	uc := NewCloseMonthUseCase(accountRepo, txRepo, cpRepo)
	// Pass the 15th — should be normalized to the 1st
	result, _ := uc.Execute(CloseMonthInput{ReferenceMonth: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)})

	if result.ReferenceMonth.Day() != 1 {
		t.Errorf("expected reference_month normalized to day 1, got day %d", result.ReferenceMonth.Day())
	}
	if len(cpRepo.saved) > 0 && cpRepo.saved[0].ReferenceMonth.Day() != 1 {
		t.Errorf("checkpoint reference_month must be day 1")
	}
}

func TestCloseMonth_UpsertExistingCheckpoint(t *testing.T) {
	acc := checkingAccount(testProfile, "acc-upsert", 0)
	allAccounts := []*bankaccount.BankAccount{acc}
	accountRepo := &fakeAccountRepoWithAll{
		fakeAccountRepo: fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-upsert": acc}},
		all:             allAccounts,
	}
	txRepo := &balanceUpToRepo{upToResult: map[string]float64{"acc-upsert": 100}}
	cpRepo := &fakeCheckpointRepo{}

	uc := NewCloseMonthUseCase(accountRepo, txRepo, cpRepo)
	refMonth := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	uc.Execute(CloseMonthInput{ReferenceMonth: refMonth})
	// Run again (simulates a re-run / correction)
	txRepo.upToResult["acc-upsert"] = 200
	uc.Execute(CloseMonthInput{ReferenceMonth: refMonth})

	// Should still be only 1 checkpoint (upserted, not duplicated)
	if len(cpRepo.saved) != 1 {
		t.Errorf("expected 1 checkpoint after upsert, got %d", len(cpRepo.saved))
	}
	if cpRepo.saved[0].ClosingBalance != 200 {
		t.Errorf("expected updated closing_balance 200, got %.2f", cpRepo.saved[0].ClosingBalance)
	}
}

// ---------------------------------------------------------------------------
// RecalculateBalance — with checkpoint tests
// ---------------------------------------------------------------------------

func TestRecalculateBalance_WithCheckpoint_OnlySumsRecentTransactions(t *testing.T) {
	acc := checkingAccount(testProfile, "acc-cp", 1000)
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-cp": acc}}

	// Checkpoint: end of Feb 2026, balance = 1500
	cp := &balancecheckpoint.BalanceCheckpoint{
		ID:             "cp-1",
		AccountID:      "acc-cp",
		ReferenceMonth: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		ClosingBalance: 1500,
	}
	cpRepo := &fakeCheckpointRepo{latest: cp}

	// Transactions since March 1st net to -200 (some expenses in March/April)
	txRepo := &balanceSinceRepo{sinceResult: map[string]float64{"acc-cp": -200}}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, cpRepo)
	result, err := uc.Execute("acc-cp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 1500 (checkpoint) + (-200) since March = 1300
	if result.NewBalance != 1300 {
		t.Errorf("expected new balance 1300, got %.2f", result.NewBalance)
	}
	// Must NOT call CalculateBalanceByBankAccountID (full scan)
	if txRepo.fullScanCalled {
		t.Errorf("full scan must NOT be called when checkpoint exists")
	}
}

func TestRecalculateBalance_NoCheckpoint_FallsBackToFullRecalc(t *testing.T) {
	acc := checkingAccount(testProfile, "acc-full", 0)
	acc.InitialBalance = 500
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"acc-full": acc}}

	cpRepo := &fakeCheckpointRepo{latest: nil}
	txRepo := &balanceSinceRepo{fullResult: map[string]float64{"acc-full": 800}}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, cpRepo)
	result, err := uc.Execute("acc-full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 500 (initial) + 800 (full scan) = 1300
	if result.NewBalance != 1300 {
		t.Errorf("expected new balance 1300, got %.2f", result.NewBalance)
	}
	if !txRepo.fullScanCalled {
		t.Errorf("full scan must be called when no checkpoint exists")
	}
}

// balanceSinceRepo tracks which path was taken.
type balanceSinceRepo struct {
	fakeTransactionRepo
	sinceResult    map[string]float64
	fullResult     map[string]float64
	fullScanCalled bool
	sinceCalls     []string
}

func (r *balanceSinceRepo) CalculateBalanceByBankAccountID(accountID string) (float64, error) {
	r.fullScanCalled = true
	if v, ok := r.fullResult[accountID]; ok {
		return v, nil
	}
	return 0, nil
}

func (r *balanceSinceRepo) CalculateBalanceSince(accountID string, since time.Time) (float64, error) {
	r.sinceCalls = append(r.sinceCalls, accountID)
	if v, ok := r.sinceResult[accountID]; ok {
		return v, nil
	}
	return 0, nil
}

func (r *balanceSinceRepo) CalculateBalanceUpTo(accountID string, upTo time.Time) (float64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// fakeAccountRepoWithAll — extends fakeAccountRepo with FindAll
// ---------------------------------------------------------------------------

type fakeAccountRepoWithAll struct {
	fakeAccountRepo
	all []*bankaccount.BankAccount
}

func (f *fakeAccountRepoWithAll) FindAll() ([]*bankaccount.BankAccount, error) {
	return f.all, nil
}

// ---------------------------------------------------------------------------
// Ensure existing fakeTransactionRepo satisfies extended interface
// ---------------------------------------------------------------------------

func (f *fakeTransactionRepo) CalculateBalanceSince(accountID string, since time.Time) (float64, error) {
	return 0, nil
}

func (f *fakeTransactionRepo) CalculateBalanceUpTo(accountID string, upTo time.Time) (float64, error) {
	// Sum confirmed transactions up to the given date
	var balance float64
	for _, tx := range f.created {
		if tx.Status != transaction.StatusConfirmed {
			continue
		}
		if tx.OccurredOn.After(upTo) {
			continue
		}
		if tx.BankAccountID == accountID {
			switch tx.Type {
			case transaction.TypeIncome:
				balance += tx.Amount
			case transaction.TypeExpense:
				balance -= tx.Amount
			case transaction.TypeTransfer:
				balance -= tx.Amount
			}
		}
		if tx.DestinationAccountID != nil && *tx.DestinationAccountID == accountID && tx.Type == transaction.TypeTransfer {
			balance += tx.Amount
		}
	}
	return balance, nil
}
