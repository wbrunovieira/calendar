package helpers

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/google/uuid"
)

// FixedTime returns a fixed time for deterministic tests
func FixedTime() time.Time {
	return time.Date(2024, 11, 16, 10, 0, 0, 0, time.UTC)
}

// Profile Fixtures

func CreateTestProfile(name string) *profile.Profile {
	p, _ := profile.NewProfile(profile.CreateParams{
		Name:         name,
		Type:         profile.TypePersonal,
		CalendarUser: stringPtr("test-calendar-user"),
	})
	return p
}

func CreateBusinessProfile(name string) *profile.Profile {
	p, _ := profile.NewProfile(profile.CreateParams{
		Name: name,
		Type: profile.TypeBusiness,
	})
	return p
}

// BankAccount Fixtures

func CreateTestBankAccount(profileID string) *bankaccount.BankAccount {
	acc, _ := bankaccount.NewBankAccount(bankaccount.CreateParams{
		ProfileID:      profileID,
		Name:           "Test Account",
		Type:           bankaccount.TypeChecking,
		InitialBalance: 1000.00,
	})
	return acc
}

func CreateCreditCardAccount(profileID string) *bankaccount.BankAccount {
	limit := 5000.00
	dueDay := 15
	acc, _ := bankaccount.NewBankAccount(bankaccount.CreateParams{
		ProfileID:     profileID,
		Name:          "Credit Card",
		Type:          bankaccount.TypeCreditCard,
		CreditLimit:   &limit,
		BillingDueDay: &dueDay,
	})
	return acc
}

func CreateSavingsAccount(profileID string, balance float64) *bankaccount.BankAccount {
	acc, _ := bankaccount.NewBankAccount(bankaccount.CreateParams{
		ProfileID:      profileID,
		Name:           "Savings Account",
		Type:           bankaccount.TypeSavings,
		InitialBalance: balance,
	})
	return acc
}

// Category Fixtures

func CreateTestCategory(profileID string, name string, categoryType category.Type) *category.Category {
	cat, _ := category.NewCategory(category.CreateParams{
		ProfileID: profileID,
		Name:      name,
		Type:      categoryType,
		Color:     stringPtr("#3B82F6"),
	})
	return cat
}

func CreateExpenseCategory(profileID string, name string) *category.Category {
	return CreateTestCategory(profileID, name, category.TypeExpense)
}

func CreateIncomeCategory(profileID string, name string) *category.Category {
	return CreateTestCategory(profileID, name, category.TypeIncome)
}

// Transaction Fixtures

func CreateTestTransaction(profileID, accountID, categoryID string) *transaction.Transaction {
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:    profileID,
		AccountID:    accountID,
		CategoryID:   categoryID,
		Amount:       100.00,
		Type:         transaction.TypeExpense,
		Description:  "Test Transaction",
		OccurredAt:   FixedTime(),
		IsPending:    false,
		Tags:         []string{"test"},
		Observations: stringPtr("Test observation"),
	})
	return tx
}

func CreateIncomeTransaction(profileID, accountID, categoryID string, amount float64) *transaction.Transaction {
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:   profileID,
		AccountID:   accountID,
		CategoryID:  categoryID,
		Amount:      amount,
		Type:        transaction.TypeIncome,
		Description: "Income Transaction",
		OccurredAt:  FixedTime(),
		IsPending:   false,
	})
	return tx
}

func CreateExpenseTransaction(profileID, accountID, categoryID string, amount float64) *transaction.Transaction {
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:   profileID,
		AccountID:   accountID,
		CategoryID:  categoryID,
		Amount:      amount,
		Type:        transaction.TypeExpense,
		Description: "Expense Transaction",
		OccurredAt:  FixedTime(),
		IsPending:   false,
	})
	return tx
}

func CreateTransferTransaction(profileID, fromAccountID, toAccountID, categoryID string, amount float64) *transaction.Transaction {
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:            profileID,
		AccountID:            fromAccountID,
		DestinationAccountID: &toAccountID,
		CategoryID:           categoryID,
		Amount:               amount,
		Type:                 transaction.TypeTransfer,
		Description:          "Transfer Transaction",
		OccurredAt:           FixedTime(),
		IsPending:            false,
	})
	return tx
}

func CreatePendingTransaction(profileID, accountID, categoryID string) *transaction.Transaction {
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:   profileID,
		AccountID:   accountID,
		CategoryID:  categoryID,
		Amount:      50.00,
		Type:        transaction.TypeExpense,
		Description: "Pending Transaction",
		OccurredAt:  FixedTime(),
		IsPending:   true,
	})
	return tx
}

func CreateTransactionWithSplits(profileID, accountID string, categoryIDs []string, amounts []float64) *transaction.Transaction {
	splits := make([]transaction.SplitParams, len(categoryIDs))
	for i, catID := range categoryIDs {
		splits[i] = transaction.SplitParams{
			CategoryID: catID,
			Amount:     amounts[i],
		}
	}

	totalAmount := 0.0
	for _, amt := range amounts {
		totalAmount += amt
	}

	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:   profileID,
		AccountID:   accountID,
		CategoryID:  categoryIDs[0], // Main category
		Amount:      totalAmount,
		Type:        transaction.TypeExpense,
		Description: "Transaction with Splits",
		OccurredAt:  FixedTime(),
		IsPending:   false,
		Splits:      splits,
	})
	return tx
}

// Helper Functions

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

// UUID helpers

func NewUUID() string {
	return uuid.New().String()
}

func ValidUUID() string {
	return "123e4567-e89b-12d3-a456-426614174000"
}

// Date helpers

func DateAfter(days int) time.Time {
	return FixedTime().AddDate(0, 0, days)
}

func DateBefore(days int) time.Time {
	return FixedTime().AddDate(0, 0, -days)
}

// Month helpers for budget testing

func FirstDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func LastDayOfMonth(t time.Time) time.Time {
	return FirstDayOfMonth(t).AddDate(0, 1, -1)
}
