package transaction

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
)

func acc(t bankaccount.AccountType) *bankaccount.BankAccount {
	return &bankaccount.BankAccount{Type: t}
}

// Paying a card invoice writes two legs: a TRANSFER out of the funding account into
// the card, and an INCOME credit on the card itself. Neither is real income or
// expense — the purchases already were — so both must drop out of a cashflow view.
func TestIsSettlement_CreditOnTheCardItself(t *testing.T) {
	tx := &Transaction{Type: TypeIncome}

	if !tx.IsSettlement(acc(bankaccount.AccountTypeCreditCard), nil) {
		t.Fatal("an INCOME landing on a credit card is the payment credit, not income")
	}
}

func TestIsSettlement_TransferIntoTheCard(t *testing.T) {
	tx := &Transaction{Type: TypeTransfer}

	if !tx.IsSettlement(acc(bankaccount.AccountTypeChecking), acc(bankaccount.AccountTypeCreditCard)) {
		t.Fatal("money moving into a credit card settles a bill, it is not an expense")
	}
}

func TestIsSettlement_OrdinaryExpenseIsNot(t *testing.T) {
	tx := &Transaction{Type: TypeExpense}

	if tx.IsSettlement(acc(bankaccount.AccountTypeCreditCard), nil) {
		t.Fatal("a purchase on the card is a real expense")
	}
	if tx.IsSettlement(acc(bankaccount.AccountTypeChecking), nil) {
		t.Fatal("a purchase on a checking account is a real expense")
	}
}

func TestIsSettlement_IncomeOnAChekingAccountIsNot(t *testing.T) {
	tx := &Transaction{Type: TypeIncome}

	if tx.IsSettlement(acc(bankaccount.AccountTypeChecking), nil) {
		t.Fatal("salary landing on a checking account is real income")
	}
}

// A transfer between the user's own accounts moves money without earning or spending
// it, so it stays out of the cashflow too.
func TestIsSettlement_TransferBetweenOwnAccounts(t *testing.T) {
	tx := &Transaction{Type: TypeTransfer}

	if !tx.IsSettlement(acc(bankaccount.AccountTypeChecking), acc(bankaccount.AccountTypeInvestment)) {
		t.Fatal("a transfer between own accounts is not income or expense")
	}
}

func TestIsSettlement_ToleratesMissingAccounts(t *testing.T) {
	tx := &Transaction{Type: TypeIncome}

	if tx.IsSettlement(nil, nil) {
		t.Fatal("with no account context the transaction must be treated as real")
	}
}

// Yield is an accounting distinction — money the balance produced by itself, versus
// money earned. The category's DRE bucket already answers it; the description does
// not, and "Rendimento da venda do carro" would fool a text match.
func TestIsYield_ComesFromTheCategoryClassification(t *testing.T) {
	financial := category.DREFinancial
	yieldCat := &category.Category{ClassificationDRE: &financial}

	tx := &Transaction{Type: TypeIncome, Description: "Pix recebido"}

	if !tx.IsYield(yieldCat) {
		t.Fatal("income in a FINANCIAL category is yield regardless of its wording")
	}
}

func TestIsYield_EarnedRevenueIsNot(t *testing.T) {
	revenue := category.DRERevenue
	revenueCat := &category.Category{ClassificationDRE: &revenue}

	tx := &Transaction{Type: TypeIncome, Description: "Rendimento do projeto X"}

	if tx.IsYield(revenueCat) {
		t.Fatal("a REVENUE category is earned income even when the text says rendimento")
	}
}

func TestIsYield_ExpenseIsNeverYield(t *testing.T) {
	financial := category.DREFinancial
	tx := &Transaction{Type: TypeExpense}

	if tx.IsYield(&category.Category{ClassificationDRE: &financial}) {
		t.Fatal("an expense in the FINANCIAL bucket is a fee, not yield")
	}
}

func TestIsYield_UnclassifiedCategoryIsNotYield(t *testing.T) {
	tx := &Transaction{Type: TypeIncome, Description: "Rendimento da conta"}

	if tx.IsYield(nil) {
		t.Fatal("without a classification the safe answer is: not yield")
	}
	if tx.IsYield(&category.Category{}) {
		t.Fatal("an unclassified category must not be guessed from the description")
	}
}
