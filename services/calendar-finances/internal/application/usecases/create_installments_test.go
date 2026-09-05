package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

// installmentFixture wires the create use case over the real balance
// recalculator, so a balance assertion here is the same arithmetic production
// runs rather than a fake's opinion of it.
type installmentFixture struct {
	useCase     *CreateTransactionUseCase
	accountRepo *fakeAccountRepo
	txRepo      *fakeTransactionRepo
	accountID   string
	categoryID  string
	profileID   string
}

func newInstallmentFixture(t *testing.T, accType bankaccount.AccountType, initialBalance float64) *installmentFixture {
	t.Helper()
	const (
		profileID  = "profile-1"
		accountID  = "account-1"
		categoryID = "cat-1"
	)
	now := time.Now()

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {ID: profileID, CalendarID: "cal-1", Name: "Bruno", Type: profile.ProfileTypePersonal, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	limit := 10000.0
	closingDay, dueDay := 5, 15
	account := &bankaccount.BankAccount{
		ID:             accountID,
		ProfileID:      profileID,
		Name:           "Conta",
		Type:           accType,
		InitialBalance: initialBalance,
		CurrentBalance: initialBalance,
		Currency:       "BRL",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if accType == bankaccount.AccountTypeCreditCard {
		account.CreditLimit = &limit
		account.ClosingDay = &closingDay
		account.DueDay = &dueDay
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{accountID: account}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {ID: categoryID, ProfileID: profileID, Name: "Mercado", Type: category.TypeExpense, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}
	txRepo := &fakeTransactionRepo{}
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	return &installmentFixture{
		useCase:     NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, recalculator),
		accountRepo: accountRepo,
		txRepo:      txRepo,
		accountID:   accountID,
		categoryID:  categoryID,
		profileID:   profileID,
	}
}

// A purchase split across categories and paid in installments used to reuse the
// very same *Split pointers for every installment. setSplits only assigns an ID
// when it is empty, so all N installments ended up sharing one primary key —
// and each installment carried the whole purchase's split amounts, which alone
// exceeds the installment and is rejected.
func TestCreateInstallments_GivesEachInstallmentItsOwnSplits(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeChecking, 5000)

	total := 3
	planned := "PLANNED"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		Type:             "EXPENSE",
		Status:           &planned,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra dividida",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &total,
		Splits: []CreateTransactionSplitInput{
			{CategoryID: &f.categoryID, Amount: 200},
			{CategoryID: &f.categoryID, Amount: 100},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(f.txRepo.created) != 3 {
		t.Fatalf("expected 3 installments, got %d", len(f.txRepo.created))
	}

	seen := map[string]bool{}
	for i, txn := range f.txRepo.created {
		if len(txn.Splits) != 2 {
			t.Fatalf("installment %d carries %d splits, want 2", i+1, len(txn.Splits))
		}

		var sum float64
		for _, split := range txn.Splits {
			if split.ID == "" {
				t.Errorf("installment %d has a split with no id", i+1)
			}
			if seen[split.ID] {
				t.Errorf("split id %s is shared with another installment; the primary key would collide", split.ID)
			}
			seen[split.ID] = true
			sum += split.Amount
		}

		if diff := sum - txn.Amount; diff > 0.005 || diff < -0.005 {
			t.Errorf("installment %d: splits sum to %.2f but the installment is %.2f", i+1, sum, txn.Amount)
		}
	}
}

// The splits keep their proportions: two thirds of every installment stay on
// the first category.
func TestCreateInstallments_KeepsTheProportionOfEachSplit(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeChecking, 5000)

	total := 2
	planned := "PLANNED"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		Type:             "EXPENSE",
		Status:           &planned,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra dividida",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &total,
		Splits: []CreateTransactionSplitInput{
			{CategoryID: &f.categoryID, Amount: 200},
			{CategoryID: &f.categoryID, Amount: 100},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, txn := range f.txRepo.created {
		if txn.Splits[0].Amount != 100 {
			t.Errorf("installment %d: first split = %.2f, want 100", i+1, txn.Splits[0].Amount)
		}
		if txn.Splits[1].Amount != 50 {
			t.Errorf("installment %d: second split = %.2f, want 50", i+1, txn.Splits[1].Amount)
		}
	}
}

// createInstallments returned right after writing the rows, without ever
// recalculating: a confirmed installment purchase on a checking account left
// the balance untouched.
func TestCreateInstallments_ConfirmedInstallmentsMoveTheBalance(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeChecking, 5000)

	total := 3
	confirmed := "CONFIRMED"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		CategoryID:       &f.categoryID,
		Type:             "EXPENSE",
		Status:           &confirmed,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra parcelada",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &total,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := f.accountRepo.accounts[f.accountID].CurrentBalance; got != 4700 {
		t.Errorf("balance = %.2f, want 4700 (5000 - 300 confirmed)", got)
	}
}

// Planned installments are a forecast. They must not move anything.
func TestCreateInstallments_PlannedInstallmentsLeaveTheBalanceAlone(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeChecking, 5000)

	total := 3
	planned := "PLANNED"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		CategoryID:       &f.categoryID,
		Type:             "EXPENSE",
		Status:           &planned,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra parcelada",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &total,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := f.accountRepo.accounts[f.accountID].CurrentBalance; got != 5000 {
		t.Errorf("balance = %.2f, want 5000: a planned installment is not money", got)
	}
}

// A credit card's balance is settled when the invoice is paid, not when the
// card is used. Recalculating it here would double-count the purchase.
func TestCreateInstallments_CreditCardInstallmentsLeaveTheCardBalanceAlone(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeCreditCard, 0)

	total := 3
	confirmed := "CONFIRMED"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		CategoryID:       &f.categoryID,
		Type:             "EXPENSE",
		Status:           &confirmed,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra parcelada no cartão",
		OccurredOn:       "2026-03-01",
		InstallmentTotal: &total,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := f.accountRepo.accounts[f.accountID].CurrentBalance; got != 0 {
		t.Errorf("card balance = %.2f, want 0", got)
	}
}
