package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type cashflowTxRepo struct {
	mockTransactionRepo
	txs        []*transaction.Transaction
	lastFilter transaction.ListFilter
}

func (r *cashflowTxRepo) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	r.lastFilter = filter
	return r.txs, nil
}

type cashflowAccountRepo struct {
	fakeAccountRepo
	list []*bankaccount.BankAccount
}

func (r *cashflowAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return r.list, nil
}

type cashflowCategoryRepo struct {
	fakeCategoryRepo
	list []*category.Category
}

func (r *cashflowCategoryRepo) ListByProfile(string) ([]*category.Category, error) {
	return r.list, nil
}

// One ordinary month: a salary, the account's own yield, a purchase on the card, and
// the two legs of paying that card's bill. Only the salary, the yield and the purchase
// are cashflow; the bill payment just moves money that was already spent.
func TestGetCashflowSummary_SeparatesYieldAndDropsSettlements(t *testing.T) {
	financial := category.DREFinancial
	yieldCat := &category.Category{ID: "cat-yield", ProfileID: "p1", ClassificationDRE: &financial}
	cardID := "card"

	accounts := []*bankaccount.BankAccount{
		{ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking},
		{ID: cardID, ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard},
	}
	txs := []*transaction.Transaction{
		{BankAccountID: "checking", Type: transaction.TypeIncome, Amount: 5000, Description: "Salario"},
		{BankAccountID: "checking", Type: transaction.TypeIncome, Amount: 12.50,
			CategoryID: strPtr("cat-yield"), Description: "Rendimento da conta"},
		{BankAccountID: cardID, Type: transaction.TypeExpense, Amount: 200, Description: "Supermercado"},
		{BankAccountID: "checking", Type: transaction.TypeTransfer, Amount: 800,
			DestinationAccountID: &cardID, Description: "Pagamento fatura"},
		{BankAccountID: cardID, Type: transaction.TypeIncome, Amount: 800, Description: "Pagamento fatura"},
	}

	uc := NewGetCashflowSummaryUseCase(
		&cashflowTxRepo{txs: txs},
		&cashflowAccountRepo{list: accounts},
		&cashflowCategoryRepo{list: []*category.Category{yieldCat}},
	)

	out, err := uc.Execute(CashflowSummaryInput{ProfileID: "p1", From: "2026-09-01", To: "2026-09-30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.IncomeYield != 12.50 {
		t.Errorf("expected yield 12.50, got %v", out.IncomeYield)
	}
	if out.IncomeOther != 5000 {
		t.Errorf("expected other income 5000, got %v", out.IncomeOther)
	}
	if out.Income != 5012.50 {
		t.Errorf("expected total income 5012.50, got %v", out.Income)
	}
	if out.Expense != 200 {
		t.Errorf("expected expense 200 (the bill payment is not an expense), got %v", out.Expense)
	}
	if out.Net != 4812.50 {
		t.Errorf("expected net 4812.50, got %v", out.Net)
	}
}

// Crypto trades on an exchange are not household cashflow — the money never left.
func TestGetCashflowSummary_IgnoresExchangeAccounts(t *testing.T) {
	accounts := []*bankaccount.BankAccount{
		{ID: "binance", ProfileID: "p1", Type: bankaccount.AccountTypeExchange},
	}
	txs := []*transaction.Transaction{
		{BankAccountID: "binance", Type: transaction.TypeIncome, Amount: 300, Description: "Venda BTC"},
		{BankAccountID: "binance", Type: transaction.TypeExpense, Amount: 120, Description: "Compra ETH"},
	}

	uc := NewGetCashflowSummaryUseCase(
		&cashflowTxRepo{txs: txs},
		&cashflowAccountRepo{list: accounts},
		&cashflowCategoryRepo{},
	)

	out, err := uc.Execute(CashflowSummaryInput{ProfileID: "p1", From: "2026-09-01", To: "2026-09-30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Income != 0 || out.Expense != 0 {
		t.Fatalf("expected an exchange account to be ignored, got income %v expense %v", out.Income, out.Expense)
	}
}

func TestGetCashflowSummary_RequiresAProfile(t *testing.T) {
	uc := NewGetCashflowSummaryUseCase(&cashflowTxRepo{}, &cashflowAccountRepo{}, &cashflowCategoryRepo{})

	if _, err := uc.Execute(CashflowSummaryInput{From: "2026-09-01", To: "2026-09-30"}); err == nil {
		t.Fatal("expected an error when no profile is given")
	}
}

func TestGetCashflowSummary_RejectsAMalformedPeriod(t *testing.T) {
	uc := NewGetCashflowSummaryUseCase(&cashflowTxRepo{}, &cashflowAccountRepo{}, &cashflowCategoryRepo{})

	if _, err := uc.Execute(CashflowSummaryInput{ProfileID: "p1", From: "setembro", To: "2026-09-30"}); err == nil {
		t.Fatal("expected an error for a date that is not YYYY-MM-DD")
	}
}

// The seventeen hand-entered card payments in this database are typed as ordinary
// expenses with no destination. Counting them double-counts the purchases they pay.
func TestGetCashflowSummary_DropsHandEnteredInvoicePayments(t *testing.T) {
	invoiceID := "inv-1"
	accounts := []*bankaccount.BankAccount{
		{ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking},
		{ID: "card", ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard},
	}
	txs := []*transaction.Transaction{
		{BankAccountID: "card", Type: transaction.TypeExpense, Amount: 200, Description: "Supermercado"},
		{BankAccountID: "checking", Type: transaction.TypeExpense, Amount: 200,
			InvoiceID: &invoiceID, Description: "Pagamento fatura cartao Nubank (venc 03/08)"},
	}

	uc := NewGetCashflowSummaryUseCase(
		&cashflowTxRepo{txs: txs},
		&cashflowAccountRepo{list: accounts},
		&cashflowCategoryRepo{},
	)

	out, err := uc.Execute(CashflowSummaryInput{ProfileID: "p1", From: "2026-09-01", To: "2026-09-30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Expense != 200 {
		t.Fatalf("expected only the purchase counted (200), got %.2f", out.Expense)
	}
}

// Yield must follow the same inheritance the DRE report uses.
func TestGetCashflowSummary_YieldInheritsTheParentBucket(t *testing.T) {
	financial := category.DREFinancial
	parent := &category.Category{ID: "root", ProfileID: "p1", ClassificationDRE: &financial}
	leaf := &category.Category{ID: "leaf", ProfileID: "p1", ParentID: &parent.ID}

	accounts := []*bankaccount.BankAccount{{ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking}}
	txs := []*transaction.Transaction{
		{BankAccountID: "checking", Type: transaction.TypeIncome, Amount: 40,
			CategoryID: strPtr("leaf"), Description: "CDB Arbi"},
	}

	uc := NewGetCashflowSummaryUseCase(
		&cashflowTxRepo{txs: txs},
		&cashflowAccountRepo{list: accounts},
		&cashflowCategoryRepo{list: []*category.Category{parent, leaf}},
	)

	out, err := uc.Execute(CashflowSummaryInput{ProfileID: "p1", From: "2026-09-01", To: "2026-09-30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.IncomeYield != 40 {
		t.Fatalf("expected 40 of yield via the parent bucket, got %.2f", out.IncomeYield)
	}
}

// The query itself must be constrained: only confirmed transactions, only the period,
// only this profile. The fake used to throw the filter away, so nothing checked it.
func TestGetCashflowSummary_ConstrainsTheQuery(t *testing.T) {
	repo := &cashflowTxRepo{}
	uc := NewGetCashflowSummaryUseCase(repo, &cashflowAccountRepo{}, &cashflowCategoryRepo{})

	if _, err := uc.Execute(CashflowSummaryInput{ProfileID: "p1", From: "2026-09-01", To: "2026-09-30"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.ProfileID != "p1" {
		t.Errorf("expected the profile in the filter, got %q", repo.lastFilter.ProfileID)
	}
	if repo.lastFilter.Status == nil || *repo.lastFilter.Status != transaction.StatusConfirmed {
		t.Error("expected only CONFIRMED transactions to be requested")
	}
	if repo.lastFilter.OccurredFrom == nil || repo.lastFilter.OccurredFrom.Format("2006-01-02") != "2026-09-01" {
		t.Error("expected the period start in the filter")
	}
	if repo.lastFilter.OccurredTo == nil || repo.lastFilter.OccurredTo.Format("2006-01-02") != "2026-09-30" {
		t.Error("expected the period end in the filter")
	}
}
