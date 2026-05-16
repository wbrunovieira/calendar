package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/balancecheckpoint"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// --- helpers ---

func makeAccount(id string, initial, current float64) *bankaccount.BankAccount {
	now := time.Now()
	return &bankaccount.BankAccount{
		ID:             id,
		ProfileID:      "profile-1",
		Name:           id,
		Type:           bankaccount.AccountTypeChecking,
		InitialBalance: initial,
		CurrentBalance: current,
		Currency:       "BRL",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func confirmedTransfer(sourceID, destID string, amount float64) *transaction.Transaction {
	destIDCopy := destID
	now := time.Now()
	return &transaction.Transaction{
		ID:                   "tx-" + sourceID + "-" + destID,
		ProfileID:            "profile-1",
		BankAccountID:        sourceID,
		DestinationAccountID: &destIDCopy,
		Type:                 transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed,
		Amount:               amount,
		OccurredOn:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func confirmedIncome(accountID string, amount float64) *transaction.Transaction {
	now := time.Now()
	return &transaction.Transaction{
		ID:            "income-" + accountID,
		ProfileID:     "profile-1",
		BankAccountID: accountID,
		Type:          transaction.TypeIncome,
		Status:        transaction.StatusConfirmed,
		Amount:        amount,
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func confirmedExpense(accountID string, amount float64) *transaction.Transaction {
	now := time.Now()
	return &transaction.Transaction{
		ID:            "expense-" + accountID,
		ProfileID:     "profile-1",
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        amount,
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func plannedTransfer(sourceID, destID string, amount float64) *transaction.Transaction {
	t := confirmedTransfer(sourceID, destID, amount)
	t.ID = "planned-" + t.ID
	t.Status = transaction.StatusPlanned
	return t
}

// --- tests ---

// Given: caixinha with R$20000 initial, one outgoing TRANSFER of R$3000
// When: recalculate balance
// Then: new balance = 20000 - 3000 = 17000
func TestRecalculateBalance_OutgoingTransfer_DebitsSource(t *testing.T) {
	caixinhaID := "caixinha"
	contaID := "conta"

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		caixinhaID: makeAccount(caixinhaID, 20000, 20000),
	}}

	txRepo := &fakeTransactionRepo{
		created: []*transaction.Transaction{
			confirmedTransfer(caixinhaID, contaID, 3000),
		},
	}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	result, err := uc.Execute(caixinhaID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewBalance != 17000 {
		t.Fatalf("expected balance 17000, got %.2f", result.NewBalance)
	}
	if accountRepo.accounts[caixinhaID].CurrentBalance != 17000 {
		t.Fatalf("expected account persisted with 17000, got %.2f", accountRepo.accounts[caixinhaID].CurrentBalance)
	}
}

// Given: conta with R$0 initial, one incoming TRANSFER of R$3000
// When: recalculate balance
// Then: new balance = 0 + 3000 = 3000
func TestRecalculateBalance_IncomingTransfer_CreditsDestination(t *testing.T) {
	caixinhaID := "caixinha"
	contaID := "conta"

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		contaID: makeAccount(contaID, 0, 0),
	}}

	txRepo := &fakeTransactionRepo{
		created: []*transaction.Transaction{
			confirmedTransfer(caixinhaID, contaID, 3000),
		},
	}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	result, err := uc.Execute(contaID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewBalance != 3000 {
		t.Fatalf("expected balance 3000, got %.2f", result.NewBalance)
	}
}

// Given: caixinha with R$20000 initial, PLANNED transfer of R$3000 (not confirmed)
// When: recalculate balance
// Then: new balance = 20000 (planned doesn't count)
func TestRecalculateBalance_PlannedTransfer_DoesNotAffectBalance(t *testing.T) {
	caixinhaID := "caixinha"
	contaID := "conta"

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		caixinhaID: makeAccount(caixinhaID, 20000, 20000),
	}}

	txRepo := &fakeTransactionRepo{
		created: []*transaction.Transaction{
			plannedTransfer(caixinhaID, contaID, 3000),
		},
	}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	result, err := uc.Execute(caixinhaID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewBalance != 20000 {
		t.Fatalf("expected balance 20000 (planned transfer ignored), got %.2f", result.NewBalance)
	}
}

// Given: caixinha with R$0 initial, received R$26000 deposit, earned R$307.82,
//        spent R$4000 (expense), sent R$600+R$1000+R$550+R$3000 (transfers out)
// When: recalculate balance
// Then: new balance = 0 + 26000 + 307.82 - 4000 - 5150 = 17157.82
func TestRecalculateBalance_MixedTransactions_CaixinhaScenario(t *testing.T) {
	caixinhaID := "caixinha"
	depositSourceID := "source-extern"

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		caixinhaID: makeAccount(caixinhaID, 0, 0),
	}}

	txRepo := &fakeTransactionRepo{
		created: []*transaction.Transaction{
			confirmedTransfer(depositSourceID, caixinhaID, 26000),  // incoming
			confirmedIncome(caixinhaID, 307.82),                    // rendimento
			confirmedExpense(caixinhaID, 4000),                     // resgate via despesa
			confirmedTransfer(caixinhaID, "dest1", 600),            // outgoing
			confirmedTransfer(caixinhaID, "dest2", 1000),           // outgoing
			confirmedTransfer(caixinhaID, "dest3", 550),            // outgoing
			confirmedTransfer(caixinhaID, "dest4", 3000),           // outgoing (hoje)
		},
	}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	result, err := uc.Execute(caixinhaID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 17157.82
	if result.NewBalance != expected {
		t.Fatalf("expected balance %.2f, got %.2f", expected, result.NewBalance)
	}
}

// Given: account with wrong stale balance
// When: recalculate
// Then: OldBalance reports the stale value, NewBalance reports the correct one
func TestRecalculateBalance_ReportsOldAndNewBalance(t *testing.T) {
	accountID := "acc"

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: makeAccount(accountID, 1000, 9999), // current_balance out of sync
	}}

	txRepo := &fakeTransactionRepo{
		created: []*transaction.Transaction{
			confirmedIncome(accountID, 500),
		},
	}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	_, err := uc.Execute("wrong-id")
	if err == nil {
		t.Fatal("expected error for unknown account, got nil")
	}
	// now test the real account
	result2, err := uc.Execute(accountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result2.OldBalance != 9999 {
		t.Fatalf("expected OldBalance 9999, got %.2f", result2.OldBalance)
	}
	if result2.NewBalance != 1500 { // 1000 initial + 500 income
		t.Fatalf("expected NewBalance 1500, got %.2f", result2.NewBalance)
	}
}

// Given: account with initial=0, one retroactive income of R$100 added AFTER a checkpoint was created
//        (checkpoint has closing_balance=1000 but doesn't know about the retroactive income)
// When: recalculate
// Then: new balance = 100 (initialBalance + all confirmed txns), NOT 1000 (stale checkpoint)
func TestRecalculateBalance_StaleCheckpoint_MustBeIgnoredAndUseDirectCalculation(t *testing.T) {
	accountID := "acc"

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: makeAccount(accountID, 0, 0),
	}}

	retroactiveTx := &transaction.Transaction{
		ID:            "retro-income",
		ProfileID:     "profile-1",
		BankAccountID: accountID,
		Type:          transaction.TypeIncome,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		OccurredOn:    time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{retroactiveTx}}

	// Stale checkpoint: was created before the retroactive income was added.
	// closing_balance=1000 does not include the R$100 income above.
	staleCheckpoint := &balancecheckpoint.BalanceCheckpoint{
		AccountID:      accountID,
		ReferenceMonth: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ClosingBalance: 1000,
	}
	checkpointRepo := &fakeCheckpointRepo{latest: staleCheckpoint}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, checkpointRepo)
	result, err := uc.Execute(accountID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Correct: initialBalance(0) + CalculateBalanceByBankAccountID(100) = 100
	// Bug returns: staleCheckpoint(1000) + CalculateBalanceSince(0) = 1000
	if result.NewBalance != 100 {
		t.Errorf("expected balance 100 (all confirmed txns, no checkpoint), got %.2f", result.NewBalance)
	}
}

// Given: account not found
// When: recalculate
// Then: returns ErrBankAccountNotFound
func TestRecalculateBalance_AccountNotFound_ReturnsError(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{}}
	txRepo := &fakeTransactionRepo{}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	_, err := uc.Execute("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent account, got nil")
	}
	if err != ErrBankAccountNotFound {
		t.Fatalf("expected ErrBankAccountNotFound, got: %v", err)
	}
}
