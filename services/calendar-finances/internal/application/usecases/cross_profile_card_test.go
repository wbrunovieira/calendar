package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// crossProfileFixture builds two profiles with one account each, so a transfer
// between them takes the cross-profile branch.
type crossProfileFixture struct {
	useCase     *CreateTransactionUseCase
	accountRepo *fakeAccountRepo
	sourceID    string
	destID      string
	sourceProf  string
	destProf    string
	incomeCatID string
}

func newCrossProfileFixture(t *testing.T, sourceType, destType bankaccount.AccountType) *crossProfileFixture {
	t.Helper()
	const (
		sourceProfile = "profile-personal"
		destProfile   = "profile-business"
		sourceAccount = "account-source"
		destAccount   = "account-dest"
		incomeCat     = "cat-income-business"
	)
	now := time.Now()

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		sourceProfile: {ID: sourceProfile, CalendarID: "cal-1", Name: "Bruno Pessoal", Type: profile.ProfileTypePersonal, IsActive: true, CreatedAt: now, UpdatedAt: now},
		destProfile:   {ID: destProfile, CalendarID: "cal-2", Name: "WB Digital", Type: profile.ProfileTypeBusiness, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}

	newAccount := func(id, profileID string, accType bankaccount.AccountType, balance float64) *bankaccount.BankAccount {
		account := &bankaccount.BankAccount{
			ID: id, ProfileID: profileID, Name: id, Type: accType,
			InitialBalance: balance, CurrentBalance: balance,
			Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now,
		}
		if accType == bankaccount.AccountTypeCreditCard {
			limit := 10000.0
			closingDay, dueDay := 5, 15
			account.CreditLimit = &limit
			account.ClosingDay = &closingDay
			account.DueDay = &dueDay
		}
		return account
	}

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		sourceAccount: newAccount(sourceAccount, sourceProfile, sourceType, 1000),
		destAccount:   newAccount(destAccount, destProfile, destType, 500),
	}}

	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		incomeCat: {ID: incomeCat, ProfileID: destProfile, Name: "Aporte Socio", Type: category.TypeIncome, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}
	txRepo := &fakeTransactionRepo{}

	return &crossProfileFixture{
		useCase: NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{},
			NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)),
		accountRepo: accountRepo,
		sourceID:    sourceAccount,
		destID:      destAccount,
		sourceProf:  sourceProfile,
		destProf:    destProfile,
		incomeCatID: incomeCat,
	}
}

func (f *crossProfileFixture) transfer(t *testing.T, amount float64) *transaction.Transaction {
	t.Helper()
	confirmed := "CONFIRMED"
	txn, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:             f.sourceProf,
		BankAccountID:         f.sourceID,
		DestinationAccountID:  &f.destID,
		DestinationCategoryID: &f.incomeCatID,
		Type:                  "TRANSFER",
		Status:                &confirmed,
		Amount:                amount,
		Currency:              "BRL",
		Description:           "Aporte",
		OccurredOn:            "2026-03-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return txn
}

// The charge and the guard are one change. The invoice link above is what makes
// skipping the card's balance safe: without it the expense would exist nowhere
// at all — not on a balance, not on any invoice. This test pins both halves
// together, because either one alone loses money.
func TestCrossProfileTransfer_ChargesTheCardsInvoiceAndLeavesItsBalanceAlone(t *testing.T) {
	f := newCrossProfileFixture(t, bankaccount.AccountTypeCreditCard, bankaccount.AccountTypeChecking)
	txn := f.transfer(t, 200)

	if txn.InvoiceID == nil {
		t.Error("the expense must land on the card's invoice; without it the charge exists nowhere")
	}
	if got := f.accountRepo.accounts[f.sourceID].CurrentBalance; got != 1000 {
		t.Errorf("card balance = %.2f, want 1000: a card is not debited when it is used", got)
	}
	if got := f.accountRepo.accounts[f.destID].CurrentBalance; got != 700 {
		t.Errorf("destination balance = %.2f, want 700: the money still arrived", got)
	}
}

func TestCrossProfileTransfer_DoesNotCreditACreditCardDestination(t *testing.T) {
	f := newCrossProfileFixture(t, bankaccount.AccountTypeChecking, bankaccount.AccountTypeCreditCard)
	f.transfer(t, 200)

	if got := f.accountRepo.accounts[f.destID].CurrentBalance; got != 500 {
		t.Errorf("card balance = %.2f, want 500: a card's balance is settled through its invoice", got)
	}
	if got := f.accountRepo.accounts[f.sourceID].CurrentBalance; got != 800 {
		t.Errorf("source balance = %.2f, want 800: the money still left", got)
	}
}

// The ordinary case must keep working exactly as before.
func TestCrossProfileTransfer_MovesBothCheckingBalances(t *testing.T) {
	f := newCrossProfileFixture(t, bankaccount.AccountTypeChecking, bankaccount.AccountTypeChecking)
	f.transfer(t, 200)

	if got := f.accountRepo.accounts[f.sourceID].CurrentBalance; got != 800 {
		t.Errorf("source balance = %.2f, want 800", got)
	}
	if got := f.accountRepo.accounts[f.destID].CurrentBalance; got != 700 {
		t.Errorf("destination balance = %.2f, want 700", got)
	}
}
