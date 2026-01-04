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
	p, _ := profile.NewProfile(
		uuid.New().String(), // calendarID
		name,
		profile.ProfileTypePersonal,
	)
	return p
}

func CreateBusinessProfile(name string) *profile.Profile {
	p, _ := profile.NewProfile(
		uuid.New().String(), // calendarID
		name,
		profile.ProfileTypeBusiness,
	)
	return p
}

// BankAccount Fixtures

func CreateTestBankAccount(profileID string) *bankaccount.BankAccount {
	acc, _ := bankaccount.NewBankAccount(
		profileID,
		"Test Account",
		bankaccount.AccountTypeChecking,
		1000.00,
		"BRL",
	)
	return acc
}

func CreateCreditCardAccount(profileID string) *bankaccount.BankAccount {
	acc, _ := bankaccount.NewBankAccount(
		profileID,
		"Credit Card",
		bankaccount.AccountTypeCreditCard,
		0,
		"BRL",
	)
	// Set optional fields
	limit := 5000.00
	dueDay := 15
	acc.CreditLimit = &limit
	acc.DueDay = &dueDay
	return acc
}

func CreateSavingsAccount(profileID string, balance float64) *bankaccount.BankAccount {
	acc, _ := bankaccount.NewBankAccount(
		profileID,
		"Savings Account",
		bankaccount.AccountTypeSavings,
		balance,
		"BRL",
	)
	return acc
}

// Category Fixtures

func CreateTestCategory(profileID string, name string, categoryType category.Type) *category.Category {
	color := "#3B82F6"
	cat, _ := category.NewCategory(category.CreateParams{
		ProfileID: profileID,
		Name:      name,
		Type:      categoryType,
		Color:     &color,
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
	catID := categoryID
	notes := "Test observation"
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &catID,
		Amount:        100.00,
		Type:          transaction.TypeExpense,
		Currency:      "BRL",
		Description:   "Test Transaction",
		OccurredOn:    FixedTime(),
		Notes:         &notes,
		Tags:          []string{"test"},
	})
	return tx
}

func CreateIncomeTransaction(profileID, accountID, categoryID string, amount float64) *transaction.Transaction {
	catID := categoryID
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &catID,
		Amount:        amount,
		Type:          transaction.TypeIncome,
		Currency:      "BRL",
		Description:   "Income Transaction",
		OccurredOn:    FixedTime(),
	})
	return tx
}

func CreateExpenseTransaction(profileID, accountID, categoryID string, amount float64) *transaction.Transaction {
	catID := categoryID
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &catID,
		Amount:        amount,
		Type:          transaction.TypeExpense,
		Currency:      "BRL",
		Description:   "Expense Transaction",
		OccurredOn:    FixedTime(),
	})
	return tx
}

func CreateTransferTransaction(profileID, fromAccountID, toAccountID, categoryID string, amount float64) *transaction.Transaction {
	catID := categoryID
	destID := toAccountID
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:            profileID,
		BankAccountID:        fromAccountID,
		DestinationAccountID: &destID,
		CategoryID:           &catID,
		Amount:               amount,
		Type:                 transaction.TypeTransfer,
		Currency:             "BRL",
		Description:          "Transfer Transaction",
		OccurredOn:           FixedTime(),
	})
	return tx
}

func CreatePendingTransaction(profileID, accountID, categoryID string) *transaction.Transaction {
	catID := categoryID
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &catID,
		Amount:        50.00,
		Type:          transaction.TypeExpense,
		Currency:      "BRL",
		Description:   "Pending Transaction",
		OccurredOn:    FixedTime(),
	})
	// Transaction starts as PLANNED (pending) by default
	return tx
}

func CreateTransactionWithSplits(profileID, accountID string, categoryIDs []string, amounts []float64) *transaction.Transaction {
	splits := make([]*transaction.Split, len(categoryIDs))
	for i, catID := range categoryIDs {
		catIDPtr := catID
		split, _ := transaction.NewSplit(&catIDPtr, amounts[i], nil)
		splits[i] = split
	}

	totalAmount := 0.0
	for _, amt := range amounts {
		totalAmount += amt
	}

	firstCatID := categoryIDs[0]
	tx, _ := transaction.New(transaction.CreateParams{
		ProfileID:     profileID,
		BankAccountID: accountID,
		CategoryID:    &firstCatID,
		Amount:        totalAmount,
		Type:          transaction.TypeExpense,
		Currency:      "BRL",
		Description:   "Transaction with Splits",
		OccurredOn:    FixedTime(),
		Splits:        splits,
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
