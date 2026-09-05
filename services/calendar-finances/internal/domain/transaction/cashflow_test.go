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

	if !tx.IsYieldIn(yieldCat, map[string]*category.Category{yieldCat.ID: yieldCat}) {
		t.Fatal("income in a FINANCIAL category is yield regardless of its wording")
	}
}

func TestIsYield_EarnedRevenueIsNot(t *testing.T) {
	revenue := category.DRERevenue
	revenueCat := &category.Category{ClassificationDRE: &revenue}

	tx := &Transaction{Type: TypeIncome, Description: "Rendimento do projeto X"}

	if tx.IsYieldIn(revenueCat, map[string]*category.Category{revenueCat.ID: revenueCat}) {
		t.Fatal("a REVENUE category is earned income even when the text says rendimento")
	}
}

func TestIsYield_ExpenseIsNeverYield(t *testing.T) {
	financial := category.DREFinancial
	tx := &Transaction{Type: TypeExpense}

	if tx.IsYieldIn(&category.Category{ClassificationDRE: &financial}, nil) {
		t.Fatal("an expense in the FINANCIAL bucket is a fee, not yield")
	}
}

func TestIsYield_UnclassifiedCategoryIsNotYield(t *testing.T) {
	tx := &Transaction{Type: TypeIncome, Description: "Rendimento da conta"}

	if tx.IsYieldIn(nil, nil) {
		t.Fatal("without a classification the safe answer is: not yield")
	}
	if tx.IsYieldIn(&category.Category{}, nil) {
		t.Fatal("an unclassified category must not be guessed from the description")
	}
}

// A card bill paid by hand — typed as an ordinary expense on the funding account,
// with no destination — is still a settlement. Seventeen such rows exist in this
// database, R$ 26.044,85 worth: counting them as expenses double-counts the purchases
// they pay for. The wording varies ("Pagamento fatura cartao Nubank (venc 03/08)"),
// so the description cannot be the test — the invoice link is.
func TestIsSettlement_HandEnteredInvoicePayment(t *testing.T) {
	invoiceID := "inv-1"
	tx := &Transaction{Type: TypeExpense, InvoiceID: &invoiceID}

	if !tx.IsSettlement(acc(bankaccount.AccountTypeChecking), nil) {
		t.Fatal("an expense that pays an invoice settles it, however it was typed in")
	}
}

// An EXPENSE carrying a destination is the source leg of a cross-profile transfer.
// Money genuinely left this profile, so it is not a settlement.
func TestIsSettlement_ExpenseWithDestinationIsRealSpending(t *testing.T) {
	tx := &Transaction{Type: TypeExpense}

	if tx.IsSettlement(acc(bankaccount.AccountTypeChecking), acc(bankaccount.AccountTypeChecking)) {
		t.Fatal("a cross-profile expense is real spending, not an internal settlement")
	}
}

// Only a TRANSFER moves money between the owner's own accounts.
func TestIsSettlement_OnlyTransfersCountAsInternalMovement(t *testing.T) {
	income := &Transaction{Type: TypeIncome}

	if income.IsSettlement(acc(bankaccount.AccountTypeChecking), acc(bankaccount.AccountTypeInvestment)) {
		t.Fatal("an INCOME with a destination is not an internal transfer")
	}
}

// Yield classification must agree with the DRE report, which lets a subcategory
// inherit its parent's bucket. "Rendimento CDB" under "Rendimentos" is FINANCIAL in
// the report; reading only the leaf made it not-yield here.
func TestIsYield_InheritsTheParentClassification(t *testing.T) {
	financial := category.DREFinancial
	parent := &category.Category{ID: "root", ClassificationDRE: &financial}
	leaf := &category.Category{ID: "leaf", ParentID: &parent.ID}

	tx := &Transaction{Type: TypeIncome, Description: "CDB Arbi"}

	if !tx.IsYieldIn(leaf, map[string]*category.Category{"root": parent, "leaf": leaf}) {
		t.Fatal("a subcategory of a FINANCIAL parent is yield, as the DRE report already says")
	}
}

func TestIsYield_DoesNotInheritFromANonFinancialParent(t *testing.T) {
	revenue := category.DRERevenue
	parent := &category.Category{ID: "root", ClassificationDRE: &revenue}
	leaf := &category.Category{ID: "leaf", ParentID: &parent.ID}

	tx := &Transaction{Type: TypeIncome, Description: "Rendimento do projeto"}

	if tx.IsYieldIn(leaf, map[string]*category.Category{"root": parent, "leaf": leaf}) {
		t.Fatal("a REVENUE branch is earned income")
	}
}

// A cycle in the parent chain must not hang the request.
func TestIsYield_SurvivesACycleInTheCategoryTree(t *testing.T) {
	a := &category.Category{ID: "a"}
	b := &category.Category{ID: "b", ParentID: &a.ID}
	a.ParentID = &b.ID

	tx := &Transaction{Type: TypeIncome}

	if tx.IsYieldIn(a, map[string]*category.Category{"a": a, "b": b}) {
		t.Fatal("a cycle must resolve to not-yield rather than loop forever")
	}
}
