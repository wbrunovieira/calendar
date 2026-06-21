package usecases

import (
	"sort"
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ---- shared category helpers (used by expense analysis and financial summary) ----

// Money that merely moves between own accounts (TRANSFER-typed categories) or is
// saved/lent (investments, loans) is not operational consumption/revenue.
var nonConsumptionCategoryNames = map[string]bool{
	"transferências": true, "transferencias": true,
	"empréstimos": true, "emprestimos": true,
	"investimentos": true,
}

func rootCategoryOf(catByID map[string]*category.Category, categoryID *string) (string, string) {
	if categoryID == nil {
		return "", "Sem categoria"
	}
	cur := catByID[*categoryID]
	if cur == nil {
		return "", "Sem categoria"
	}
	for cur.ParentID != nil {
		parent := catByID[*cur.ParentID]
		if parent == nil {
			break
		}
		cur = parent
	}
	return cur.ID, cur.Name
}

// walkCategoryChain visits a category and its ancestors (leaf -> root), returning
// true as soon as match does. Classification must look at the whole chain because
// the relevant categories (Aporte, Rendimentos, ...) are often sub-categories.
func walkCategoryChain(catByID map[string]*category.Category, categoryID *string, match func(*category.Category) bool) bool {
	if categoryID == nil {
		return false
	}
	cur := catByID[*categoryID]
	for cur != nil {
		if match(cur) {
			return true
		}
		if cur.ParentID == nil {
			break
		}
		cur = catByID[*cur.ParentID]
	}
	return false
}

// isNonConsumptionCategory: transfers between own accounts, loans, investments and
// owner capital (aporte) are not operational money - excluded from revenue/expense.
func isNonConsumptionCategory(catByID map[string]*category.Category, categoryID *string) bool {
	return walkCategoryChain(catByID, categoryID, func(c *category.Category) bool {
		if c.Type == category.TypeTransfer {
			return true
		}
		lower := strings.ToLower(c.Name)
		return strings.HasPrefix(lower, "aporte") || nonConsumptionCategoryNames[lower]
	})
}

// isFinancialIncomeCategory flags yield/interest income (rendimentos): still income,
// but shown apart from operational revenue.
func isFinancialIncomeCategory(catByID map[string]*category.Category, categoryID *string) bool {
	return walkCategoryChain(catByID, categoryID, func(c *category.Category) bool {
		return strings.Contains(strings.ToLower(c.Name), "rendiment")
	})
}

func toCategoryRow(id, name string, byMonth []float64, children []CategoryTrend, n int) CategoryTrend {
	bm := make([]float64, len(byMonth))
	var total float64
	for i, v := range byMonth {
		bm[i] = round2(v)
		total += v
	}
	avg := round2(total / float64(max(n, 1)))
	var last float64
	if len(bm) > 0 {
		last = bm[len(bm)-1]
	}
	return CategoryTrend{
		CategoryID: id, Name: name, ByMonth: bm,
		Total: round2(total), Average: avg,
		DeltaVsAvg: round2(last - avg), Trend: expenseTrend(byMonth, avg),
		Children: children,
	}
}

// rollupCategories buckets CONFIRMED transactions of one type into a top-level
// category x month matrix (with sub-category drill-down), excluding settlements,
// EXCHANGE-account movements and non-consumption categories.
func rollupCategories(
	txs []*transaction.Transaction,
	txType transaction.Type,
	catByID map[string]*category.Category,
	exchange map[string]bool,
	periods []string,
	extraSkip func(categoryID *string) bool,
) ([]float64, []CategoryTrend) {
	n := len(periods)
	idxOf := make(map[string]int, n)
	for i, p := range periods {
		idxOf[p] = i
	}
	totalByMonth := make([]float64, n)

	type bucket struct {
		name     string
		byMonth  []float64
		children map[string]*bucket
	}
	roots := map[string]*bucket{}

	for _, tx := range txs {
		if tx.Type != txType || tx.Status != transaction.StatusConfirmed {
			continue
		}
		if exchange[tx.BankAccountID] || isInvoiceSettlement(tx.Description) {
			continue
		}
		i, ok := idxOf[tx.OccurredOn.Format("2006-01")]
		if !ok {
			continue
		}
		if isNonConsumptionCategory(catByID, tx.CategoryID) {
			continue
		}
		if extraSkip != nil && extraSkip(tx.CategoryID) {
			continue
		}
		rootID, rootName := rootCategoryOf(catByID, tx.CategoryID)
		totalByMonth[i] += tx.Amount

		r := roots[rootID]
		if r == nil {
			r = &bucket{name: rootName, byMonth: make([]float64, n), children: map[string]*bucket{}}
			roots[rootID] = r
		}
		r.byMonth[i] += tx.Amount

		childID := ""
		childName := "Sem categoria"
		if tx.CategoryID != nil {
			childID = *tx.CategoryID
			if c := catByID[childID]; c != nil {
				childName = c.Name
			}
		}
		ch := r.children[childID]
		if ch == nil {
			ch = &bucket{name: childName, byMonth: make([]float64, n)}
			r.children[childID] = ch
		}
		ch.byMonth[i] += tx.Amount
	}

	var cats []CategoryTrend
	for rootID, root := range roots {
		var children []CategoryTrend
		for cid, c := range root.children {
			if len(root.children) == 1 && cid == rootID {
				continue
			}
			children = append(children, toCategoryRow(cid, c.name, c.byMonth, nil, n))
		}
		sort.Slice(children, func(a, b int) bool { return children[a].Total > children[b].Total })
		cats = append(cats, toCategoryRow(rootID, root.name, root.byMonth, children, n))
	}
	sort.Slice(cats, func(a, b int) bool { return cats[a].Total > cats[b].Total })
	for i := range totalByMonth {
		totalByMonth[i] = round2(totalByMonth[i])
	}
	return totalByMonth, cats
}

// ---- financial summary (DRE / result) for BUSINESS profiles ----

// GetFinancialSummaryInput requests the result-oriented report.
type GetFinancialSummaryInput struct {
	ProfileID string
	From      string
	To        string
}

// FinancialSummaryOutput answers "is the company making money, and where".
type FinancialSummaryOutput struct {
	Periods                []string        `json:"periods"`
	RevenueByMonth         []float64       `json:"revenueByMonth"`
	FinancialIncomeByMonth []float64       `json:"financialIncomeByMonth"`
	ExpenseByMonth         []float64       `json:"expenseByMonth"`
	ResultByMonth          []float64       `json:"resultByMonth"`
	MarginByMonth          []float64       `json:"marginByMonth"` // percent
	TotalRevenue           float64         `json:"totalRevenue"`
	TotalFinancialIncome   float64         `json:"totalFinancialIncome"`
	TotalExpense           float64         `json:"totalExpense"`
	TotalResult            float64         `json:"totalResult"`
	AvgMargin              float64         `json:"avgMargin"` // percent (result / total income)
	ExpenseCategories      []CategoryTrend `json:"expenseCategories"`
	RevenueCategories      []CategoryTrend `json:"revenueCategories"`
}

func analyzeFinancialSummary(
	txs []*transaction.Transaction,
	categories []*category.Category,
	accounts []*bankaccount.BankAccount,
	periods []string,
) *FinancialSummaryOutput {
	n := len(periods)
	catByID := make(map[string]*category.Category, len(categories))
	for _, c := range categories {
		catByID[c.ID] = c
	}
	exchange := make(map[string]bool)
	for _, a := range accounts {
		if a.Type == bankaccount.AccountTypeExchange {
			exchange[a.ID] = true
		}
	}

	// Operational revenue (services) and financial income (rendimentos) are split so
	// the company result is not flattered by yield; owner capital (aporte) is dropped.
	revenueByMonth, revenueCats := rollupCategories(txs, transaction.TypeIncome, catByID, exchange, periods,
		func(catID *string) bool { return isFinancialIncomeCategory(catByID, catID) })
	financialIncomeByMonth, _ := rollupCategories(txs, transaction.TypeIncome, catByID, exchange, periods,
		func(catID *string) bool { return !isFinancialIncomeCategory(catByID, catID) })
	expenseByMonth, expenseCats := rollupCategories(txs, transaction.TypeExpense, catByID, exchange, periods, nil)

	resultByMonth := make([]float64, n)
	marginByMonth := make([]float64, n)
	for i := range periods {
		income := revenueByMonth[i] + financialIncomeByMonth[i]
		resultByMonth[i] = round2(income - expenseByMonth[i])
		if income != 0 {
			marginByMonth[i] = round2(resultByMonth[i] / income * 100)
		}
	}

	totalRevenue := round2(sumf(revenueByMonth))
	totalFinancialIncome := round2(sumf(financialIncomeByMonth))
	totalExpense := round2(sumf(expenseByMonth))
	totalResult := round2(totalRevenue + totalFinancialIncome - totalExpense)
	avgMargin := 0.0
	if totalIncome := totalRevenue + totalFinancialIncome; totalIncome != 0 {
		avgMargin = round2(totalResult / totalIncome * 100)
	}

	if expenseCats == nil {
		expenseCats = []CategoryTrend{}
	}
	if revenueCats == nil {
		revenueCats = []CategoryTrend{}
	}

	return &FinancialSummaryOutput{
		Periods:                periods,
		RevenueByMonth:         revenueByMonth,
		FinancialIncomeByMonth: financialIncomeByMonth,
		ExpenseByMonth:         expenseByMonth,
		ResultByMonth:          resultByMonth,
		MarginByMonth:          marginByMonth,
		TotalRevenue:           totalRevenue,
		TotalFinancialIncome:   totalFinancialIncome,
		TotalExpense:           totalExpense,
		TotalResult:            totalResult,
		AvgMargin:              avgMargin,
		ExpenseCategories:      expenseCats,
		RevenueCategories:      revenueCats,
	}
}

type GetFinancialSummaryUseCase struct {
	transactionRepo transaction.Repository
	categoryRepo    category.Repository
	accountRepo     bankaccount.Repository
}

func NewGetFinancialSummaryUseCase(
	txRepo transaction.Repository,
	catRepo category.Repository,
	accRepo bankaccount.Repository,
) *GetFinancialSummaryUseCase {
	return &GetFinancialSummaryUseCase{transactionRepo: txRepo, categoryRepo: catRepo, accountRepo: accRepo}
}

func (uc *GetFinancialSummaryUseCase) Execute(input GetFinancialSummaryInput) (*FinancialSummaryOutput, error) {
	if strings.TrimSpace(input.ProfileID) == "" {
		return nil, ErrInvalidInput
	}
	from, err := parseDate(input.From)
	if err != nil {
		return nil, ErrInvalidInput
	}
	to, err := parseDate(input.To)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if to.Before(from) {
		return nil, ErrInvalidInput
	}

	confirmed := transaction.StatusConfirmed
	txs, err := uc.transactionRepo.List(transaction.ListFilter{
		ProfileID:    input.ProfileID,
		Status:       &confirmed,
		OccurredFrom: &from,
		OccurredTo:   &to,
	})
	if err != nil {
		return nil, err
	}
	categories, err := uc.categoryRepo.ListByProfile(input.ProfileID)
	if err != nil {
		return nil, err
	}
	accounts, err := uc.accountRepo.FindByProfileID(input.ProfileID)
	if err != nil {
		return nil, err
	}

	return analyzeFinancialSummary(txs, categories, accounts, monthsBetween(from, to)), nil
}
