package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ---------------------------------------------------------------------------
// Tracking fake for BalanceRecalculator
// ---------------------------------------------------------------------------

type trackingRecalculator struct {
	calls []string
}

func (r *trackingRecalculator) Execute(accountID string) (*RecalculateBalanceResult, error) {
	r.calls = append(r.calls, accountID)
	return &RecalculateBalanceResult{}, nil
}

func (r *trackingRecalculator) calledWith(accountID string) bool {
	for _, id := range r.calls {
		if id == accountID {
			return true
		}
	}
	return false
}

func (r *trackingRecalculator) callCount() int { return len(r.calls) }

// ---------------------------------------------------------------------------
// Helpers shared across test groups
// ---------------------------------------------------------------------------

func setupProfile(id string) *fakeProfileRepo {
	p := &profile.Profile{ID: id, Name: id}
	return &fakeProfileRepo{profiles: map[string]*profile.Profile{id: p}}
}

func expenseCat(profileID, id string) *category.Category {
	return &category.Category{ID: id, ProfileID: profileID, Name: id, Type: category.TypeExpense}
}

func incomeCat(profileID, id string) *category.Category {
	return &category.Category{ID: id, ProfileID: profileID, Name: id, Type: category.TypeIncome}
}

func checkingAccount(profileID, id string, balance float64) *bankaccount.BankAccount {
	return &bankaccount.BankAccount{
		ID:             id,
		ProfileID:      profileID,
		Name:           id,
		Type:           bankaccount.AccountTypeChecking,
		InitialBalance: 0,
		CurrentBalance: balance,
		Currency:       "BRL",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func creditCardAccount(profileID, id string) *bankaccount.BankAccount {
	limit := 5000.0
	closingDay := 10
	dueDay := 20
	return &bankaccount.BankAccount{
		ID:          id,
		ProfileID:   profileID,
		Name:        id,
		Type:        bankaccount.AccountTypeCreditCard,
		CreditLimit: &limit,
		ClosingDay:  &closingDay,
		DueDay:      &dueDay,
		Currency:    "BRL",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

const (
	testProfile = "profile-1"
	testAccount = "acc-1"
	testDest    = "acc-2"
	testCC      = "cc-1"
	testCat     = "cat-expense"
	testCatInc  = "cat-income"
)

func baseCreateUC(
	accounts map[string]*bankaccount.BankAccount,
	recalculator BalanceRecalculator,
) *CreateTransactionUseCase {
	return NewCreateTransactionUseCase(
		setupProfile(testProfile),
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{categories: map[string]*category.Category{
			testCat:    expenseCat(testProfile, testCat),
			testCatInc: incomeCat(testProfile, testCatInc),
		}},
		&fakeTransactionRepo{},
		&fakeInvoiceRepo{},
		recalculator,
		nil,
	)
}

func confirmedStr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// CreateTransactionUseCase — recalculate tests
// ---------------------------------------------------------------------------

// Given a CONFIRMED expense, the recalculator must be called for the account.
func TestCreateTransaction_ConfirmedExpense_TriggersRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 500),
	}
	uc := baseCreateUC(accounts, recalc)

	status := "CONFIRMED"
	_, err := uc.Execute(CreateTransactionInput{
		ProfileID:     testProfile,
		BankAccountID: testAccount,
		CategoryID:    strPtr(testCat),
		Type:          "EXPENSE",
		Status:        &status,
		Amount:        100,
		Currency:      "BRL",
		Description:   "test",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculator to be called for account %s, calls=%v", testAccount, recalc.calls)
	}
}

// Given a PLANNED expense, the recalculator must NOT be called (balance unaffected).
func TestCreateTransaction_PlannedExpense_DoesNotTriggerRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 500),
	}
	uc := baseCreateUC(accounts, recalc)

	_, err := uc.Execute(CreateTransactionInput{
		ProfileID:     testProfile,
		BankAccountID: testAccount,
		CategoryID:    strPtr(testCat),
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "test",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recalc.callCount() > 0 {
		t.Errorf("expected no recalculator calls for PLANNED transaction, got %v", recalc.calls)
	}
}

// Given a CONFIRMED transfer, recalculator must be called for BOTH source and destination.
func TestCreateTransaction_ConfirmedTransfer_RecalculatesBothAccounts(t *testing.T) {
	recalc := &trackingRecalculator{}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 1000),
		testDest:    checkingAccount(testProfile, testDest, 0),
	}
	uc := baseCreateUC(accounts, recalc)

	status := "CONFIRMED"
	dest := testDest
	_, err := uc.Execute(CreateTransactionInput{
		ProfileID:            testProfile,
		BankAccountID:        testAccount,
		DestinationAccountID: &dest,
		Type:                 "TRANSFER",
		Status:               &status,
		Amount:               300,
		Currency:             "BRL",
		Description:          "transfer",
		OccurredOn:           "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate for source %s, calls=%v", testAccount, recalc.calls)
	}
	if !recalc.calledWith(testDest) {
		t.Errorf("expected recalculate for destination %s, calls=%v", testDest, recalc.calls)
	}
}

// Credit card expenses must NOT trigger recalculation (CC balance is managed by invoices).
func TestCreateTransaction_CreditCardExpense_DoesNotTriggerRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	accounts := map[string]*bankaccount.BankAccount{
		testCC: creditCardAccount(testProfile, testCC),
	}
	uc := NewCreateTransactionUseCase(
		setupProfile(testProfile),
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{categories: map[string]*category.Category{
			testCat: expenseCat(testProfile, testCat),
		}},
		&fakeTransactionRepo{},
		&fakeInvoiceRepo{},
		recalc,
		nil,
	)

	status := "CONFIRMED"
	_, err := uc.Execute(CreateTransactionInput{
		ProfileID:     testProfile,
		BankAccountID: testCC,
		CategoryID:    strPtr(testCat),
		Type:          "EXPENSE",
		Status:        &status,
		Amount:        150,
		Currency:      "BRL",
		Description:   "cc purchase",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recalc.calledWith(testCC) {
		t.Errorf("credit card account must NOT trigger recalculate, calls=%v", recalc.calls)
	}
}

// After a CONFIRMED income is created, the account balance must reflect
// initialBalance + income (validated via the fake repo's CalculateBalance).
func TestCreateTransaction_ConfirmedIncome_BalanceIsCorrect(t *testing.T) {
	acc := checkingAccount(testProfile, testAccount, 0)
	acc.InitialBalance = 500
	acc.CurrentBalance = 500

	txRepo := &fakeTransactionRepo{}
	accRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testAccount: acc}}
	recalc := NewRecalculateBalanceUseCase(accRepo, txRepo, nil)

	uc := NewCreateTransactionUseCase(
		setupProfile(testProfile),
		accRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{
			testCatInc: incomeCat(testProfile, testCatInc),
		}},
		txRepo,
		&fakeInvoiceRepo{},
		recalc,
		nil,
	)

	status := "CONFIRMED"
	_, err := uc.Execute(CreateTransactionInput{
		ProfileID:     testProfile,
		BankAccountID: testAccount,
		CategoryID:    strPtr(testCatInc),
		Type:          "INCOME",
		Status:        &status,
		Amount:        200,
		Currency:      "BRL",
		Description:   "salary",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotBalance := accRepo.accounts[testAccount].CurrentBalance
	wantBalance := 700.0 // 500 initial + 200 income
	if gotBalance != wantBalance {
		t.Errorf("expected balance %.2f after confirmed income, got %.2f", wantBalance, gotBalance)
	}
}

// ---------------------------------------------------------------------------
// UpdateTransactionUseCase — recalculate tests
// ---------------------------------------------------------------------------

func setupUpdateUC(
	accounts map[string]*bankaccount.BankAccount,
	txs []*transaction.Transaction,
	recalculator BalanceRecalculator,
) *UpdateTransactionUseCase {
	return NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{categories: map[string]*category.Category{
			testCat:    expenseCat(testProfile, testCat),
			testCatInc: incomeCat(testProfile, testCatInc),
		}},
		&fakeTransactionRepo{created: txs},
		&fakeInvoiceRepo{},
		recalculator,
	)
}

func existingConfirmedExpense(id, accountID string, amount float64) *transaction.Transaction {
	return &transaction.Transaction{
		ID:            id,
		ProfileID:     testProfile,
		BankAccountID: accountID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        amount,
		Currency:      "BRL",
		Description:   "original",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// Updating a transaction always triggers recalculate for the affected account.
func TestUpdateTransaction_TriggersRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := existingConfirmedExpense("tx-1", testAccount, 100)
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 400),
	}
	uc := setupUpdateUC(accounts, []*transaction.Transaction{tx}, recalc)

	_, err := uc.Execute("tx-1", UpdateTransactionInput{
		BankAccountID: testAccount,
		Type:          "EXPENSE",
		Amount:        150,
		Currency:      "BRL",
		Description:   "updated",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate for %s after update, calls=%v", testAccount, recalc.calls)
	}
}

// When the bank account changes, recalculate must be called for BOTH old and new accounts.
func TestUpdateTransaction_AccountChange_RecalculatesBothAccounts(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := existingConfirmedExpense("tx-1", testAccount, 100)
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 400),
		testDest:    checkingAccount(testProfile, testDest, 200),
	}
	uc := setupUpdateUC(accounts, []*transaction.Transaction{tx}, recalc)

	_, err := uc.Execute("tx-1", UpdateTransactionInput{
		BankAccountID: testDest, // changed from testAccount to testDest
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "moved",
		OccurredOn:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate for old account %s, calls=%v", testAccount, recalc.calls)
	}
	if !recalc.calledWith(testDest) {
		t.Errorf("expected recalculate for new account %s, calls=%v", testDest, recalc.calls)
	}
}

// ---------------------------------------------------------------------------
// UpdateTransactionStatusUseCase — recalculate tests
// ---------------------------------------------------------------------------

func setupStatusUC(
	accounts map[string]*bankaccount.BankAccount,
	txs []*transaction.Transaction,
	recalculator BalanceRecalculator,
) *UpdateTransactionStatusUseCase {
	return NewUpdateTransactionStatusUseCase(
		&fakeTransactionRepo{created: txs},
		&fakeAccountRepo{accounts: accounts},
		recalculator,
	)
}

// Confirming a PLANNED transaction triggers recalculate.
func TestUpdateTransactionStatus_PlannedToConfirmed_TriggersRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := &transaction.Transaction{
		ID:            "tx-1",
		ProfileID:     testProfile,
		BankAccountID: testAccount,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        100,
		Currency:      "BRL",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 500),
	}
	uc := setupStatusUC(accounts, []*transaction.Transaction{tx}, recalc)

	_, err := uc.Execute("tx-1", UpdateTransactionStatusInput{Status: "CONFIRMED"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate after PLANNED→CONFIRMED, calls=%v", recalc.calls)
	}
}

// Cancelling a CONFIRMED transaction triggers recalculate.
func TestUpdateTransactionStatus_ConfirmedToCancelled_TriggersRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := &transaction.Transaction{
		ID:            "tx-1",
		ProfileID:     testProfile,
		BankAccountID: testAccount,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 400),
	}
	uc := setupStatusUC(accounts, []*transaction.Transaction{tx}, recalc)

	_, err := uc.Execute("tx-1", UpdateTransactionStatusInput{Status: "CANCELLED"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate after CONFIRMED→CANCELLED, calls=%v", recalc.calls)
	}
}

// Credit card transactions do NOT trigger recalculate on status change.
func TestUpdateTransactionStatus_CreditCard_DoesNotTriggerRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := &transaction.Transaction{
		ID:            "tx-cc",
		ProfileID:     testProfile,
		BankAccountID: testCC,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        100,
		Currency:      "BRL",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	accounts := map[string]*bankaccount.BankAccount{
		testCC: creditCardAccount(testProfile, testCC),
	}
	uc := setupStatusUC(accounts, []*transaction.Transaction{tx}, recalc)

	_, err := uc.Execute("tx-cc", UpdateTransactionStatusInput{Status: "CONFIRMED"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recalc.calledWith(testCC) {
		t.Errorf("credit card must NOT trigger recalculate, calls=%v", recalc.calls)
	}
}

// ---------------------------------------------------------------------------
// DeleteTransactionUseCase — recalculate tests
// ---------------------------------------------------------------------------

func setupDeleteUC(
	accounts map[string]*bankaccount.BankAccount,
	txs []*transaction.Transaction,
	recalculator BalanceRecalculator,
) *DeleteTransactionUseCase {
	return NewDeleteTransactionUseCase(
		&fakeTransactionRepo{created: txs},
		&fakeAccountRepo{accounts: accounts},
		recalculator,
	)
}

// Deleting a CONFIRMED transaction triggers recalculate.
func TestDeleteTransaction_Confirmed_TriggersRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := existingConfirmedExpense("tx-1", testAccount, 100)
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 400),
	}
	uc := setupDeleteUC(accounts, []*transaction.Transaction{tx}, recalc)

	if err := uc.Execute("tx-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate after confirmed tx deletion, calls=%v", recalc.calls)
	}
}

// Deleting a PLANNED transaction does NOT trigger recalculate (balance unaffected).
func TestDeleteTransaction_Planned_DoesNotTriggerRecalculate(t *testing.T) {
	recalc := &trackingRecalculator{}
	tx := &transaction.Transaction{
		ID:            "tx-planned",
		ProfileID:     testProfile,
		BankAccountID: testAccount,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        100,
		Currency:      "BRL",
		OccurredOn:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 500),
	}
	uc := setupDeleteUC(accounts, []*transaction.Transaction{tx}, recalc)

	if err := uc.Execute("tx-planned"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recalc.callCount() > 0 {
		t.Errorf("expected no recalculate calls for PLANNED deletion, calls=%v", recalc.calls)
	}
}

// Deleting a CONFIRMED transfer triggers recalculate on BOTH source and destination.
func TestDeleteTransaction_ConfirmedTransfer_RecalculatesBothAccounts(t *testing.T) {
	recalc := &trackingRecalculator{}
	dest := testDest
	tx := &transaction.Transaction{
		ID:                   "tx-transfer",
		ProfileID:            testProfile,
		BankAccountID:        testAccount,
		DestinationAccountID: &dest,
		Type:                 transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed,
		Amount:               300,
		Currency:             "BRL",
		OccurredOn:           time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 700),
		testDest:    checkingAccount(testProfile, testDest, 300),
	}
	uc := setupDeleteUC(accounts, []*transaction.Transaction{tx}, recalc)

	if err := uc.Execute("tx-transfer"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !recalc.calledWith(testAccount) {
		t.Errorf("expected recalculate for source %s, calls=%v", testAccount, recalc.calls)
	}
	if !recalc.calledWith(testDest) {
		t.Errorf("expected recalculate for destination %s, calls=%v", testDest, recalc.calls)
	}
}
