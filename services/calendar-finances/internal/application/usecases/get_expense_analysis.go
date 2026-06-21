package usecases

import (
	"sort"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// GetExpenseAnalysisInput is the request for the cost-of-living report.
type GetExpenseAnalysisInput struct {
	ProfileID string
	From      string // YYYY-MM-DD
	To        string // YYYY-MM-DD
}

// CategoryTrend is one top-level (or sub) category's spend over the months.
type CategoryTrend struct {
	CategoryID string          `json:"categoryId"`
	Name       string          `json:"name"`
	ByMonth    []float64       `json:"byMonth"`
	Total      float64         `json:"total"`
	Average    float64         `json:"average"`
	DeltaVsAvg float64         `json:"deltaVsAvg"` // last month vs its own average
	Trend      string          `json:"trend"`      // "up" | "down" | "flat"
	Children   []CategoryTrend `json:"children,omitempty"`
}

// Mover is a category whose latest month deviates most from its average.
type Mover struct {
	Name  string  `json:"name"`
	Delta float64 `json:"delta"`
}

// ExpenseAnalysisOutput answers "is my cost of living rising/falling and where".
type ExpenseAnalysisOutput struct {
	Periods         []string        `json:"periods"`
	TotalByMonth    []float64       `json:"totalByMonth"`
	Average         float64         `json:"average"`
	FixedByMonth    []float64       `json:"fixedByMonth"`
	VariableByMonth []float64       `json:"variableByMonth"`
	FixedAverage    float64         `json:"fixedAverage"`
	VariableAverage float64         `json:"variableAverage"`
	Categories      []CategoryTrend `json:"categories"`
	TopUp           []Mover         `json:"topUp"`
	TopDown         []Mover         `json:"topDown"`
}

type GetExpenseAnalysisUseCase struct {
	transactionRepo transaction.Repository
	categoryRepo    category.Repository
	recurringRepo   recurringtransaction.Repository
	accountRepo     bankaccount.Repository
}

func NewGetExpenseAnalysisUseCase(
	txRepo transaction.Repository,
	catRepo category.Repository,
	recRepo recurringtransaction.Repository,
	accRepo bankaccount.Repository,
) *GetExpenseAnalysisUseCase {
	return &GetExpenseAnalysisUseCase{
		transactionRepo: txRepo,
		categoryRepo:    catRepo,
		recurringRepo:   recRepo,
		accountRepo:     accRepo,
	}
}

func (uc *GetExpenseAnalysisUseCase) Execute(input GetExpenseAnalysisInput) (*ExpenseAnalysisOutput, error) {
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

	expense := transaction.TypeExpense
	confirmed := transaction.StatusConfirmed
	txs, err := uc.transactionRepo.List(transaction.ListFilter{
		ProfileID:    input.ProfileID,
		Type:         &expense,
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
	recurrings, err := uc.recurringRepo.ListByProfile(input.ProfileID)
	if err != nil {
		return nil, err
	}
	accounts, err := uc.accountRepo.FindByProfileID(input.ProfileID)
	if err != nil {
		return nil, err
	}

	return analyzeExpenses(txs, categories, recurrings, accounts, monthsBetween(from, to)), nil
}

// ---- aggregation (business rule, unit-tested) ----

func monthsBetween(from, to time.Time) []string {
	var out []string
	y, m := from.Year(), from.Month()
	for {
		cur := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
		if cur.After(time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)) {
			break
		}
		out = append(out, cur.Format("2006-01"))
		m++
		if m > 12 {
			m = 1
			y++
		}
	}
	return out
}

func isInvoiceSettlement(desc string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(desc)), "pagamento fatura")
}

func expenseTrend(byMonth []float64, average float64) string {
	if len(byMonth) < 2 || average == 0 {
		return "flat"
	}
	last := byMonth[len(byMonth)-1]
	switch {
	case last > average*1.1:
		return "up"
	case last < average*0.9:
		return "down"
	default:
		return "flat"
	}
}

func analyzeExpenses(
	txs []*transaction.Transaction,
	categories []*category.Category,
	recurrings []*recurringtransaction.RecurringTransaction,
	accounts []*bankaccount.BankAccount,
	periods []string,
) *ExpenseAnalysisOutput {
	n := len(periods)
	idxOf := make(map[string]int, n)
	for i, p := range periods {
		idxOf[p] = i
	}

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
	// A category is "fixed" if an ACTIVE recurring expense uses it.
	fixedCat := make(map[string]bool)
	for _, r := range recurrings {
		if r.Status == recurringtransaction.StatusActive && r.Type == string(transaction.TypeExpense) && r.CategoryID != nil {
			fixedCat[*r.CategoryID] = true
		}
	}

	totalByMonth := make([]float64, n)
	fixedByMonth := make([]float64, n)
	variableByMonth := make([]float64, n)

	type bucket struct {
		name     string
		byMonth  []float64
		children map[string]*bucket
	}
	roots := map[string]*bucket{}

	for _, tx := range txs {
		if tx.Type != transaction.TypeExpense || tx.Status != transaction.StatusConfirmed {
			continue
		}
		if exchange[tx.BankAccountID] || isInvoiceSettlement(tx.Description) {
			continue
		}
		i, ok := idxOf[tx.OccurredOn.Format("2006-01")]
		if !ok {
			continue
		}
		rootID, rootName := rootCategoryOf(catByID, tx.CategoryID)
		if isNonConsumptionRoot(catByID, rootID, rootName) {
			continue
		}
		totalByMonth[i] += tx.Amount
		if tx.CategoryID != nil && fixedCat[*tx.CategoryID] {
			fixedByMonth[i] += tx.Amount
		} else {
			variableByMonth[i] += tx.Amount
		}

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

	toRow := func(id, name string, byMonth []float64, children []CategoryTrend) CategoryTrend {
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

	out := &ExpenseAnalysisOutput{Periods: periods}
	for rootID, root := range roots {
		var children []CategoryTrend
		for cid, c := range root.children {
			// hide a single child that is just the top-level category itself
			if len(root.children) == 1 && cid == rootID {
				continue
			}
			children = append(children, toRow(cid, c.name, c.byMonth, nil))
		}
		sort.Slice(children, func(a, b int) bool { return children[a].Total > children[b].Total })
		out.Categories = append(out.Categories, toRow(rootID, root.name, root.byMonth, children))
	}
	sort.Slice(out.Categories, func(a, b int) bool { return out.Categories[a].Total > out.Categories[b].Total })

	for i := range periods {
		totalByMonth[i] = round2(totalByMonth[i])
		fixedByMonth[i] = round2(fixedByMonth[i])
		variableByMonth[i] = round2(variableByMonth[i])
	}
	out.TotalByMonth = totalByMonth
	out.FixedByMonth = fixedByMonth
	out.VariableByMonth = variableByMonth
	out.Average = round2(sumf(totalByMonth) / float64(max(n, 1)))
	out.FixedAverage = round2(sumf(fixedByMonth) / float64(max(n, 1)))
	out.VariableAverage = round2(sumf(variableByMonth) / float64(max(n, 1)))

	for _, c := range out.Categories {
		if c.DeltaVsAvg > 0.005 {
			out.TopUp = append(out.TopUp, Mover{Name: c.Name, Delta: c.DeltaVsAvg})
		} else if c.DeltaVsAvg < -0.005 {
			out.TopDown = append(out.TopDown, Mover{Name: c.Name, Delta: c.DeltaVsAvg})
		}
	}
	sort.Slice(out.TopUp, func(a, b int) bool { return out.TopUp[a].Delta > out.TopUp[b].Delta })
	sort.Slice(out.TopDown, func(a, b int) bool { return out.TopDown[a].Delta < out.TopDown[b].Delta })
	out.TopUp = trim(out.TopUp, 3)
	out.TopDown = trim(out.TopDown, 3)

	// Never return nil slices: the API contract is arrays, so the client can
	// safely call .length/.map (a nil slice would serialize to JSON null).
	if out.Categories == nil {
		out.Categories = []CategoryTrend{}
	}
	if out.TopUp == nil {
		out.TopUp = []Mover{}
	}
	if out.TopDown == nil {
		out.TopDown = []Mover{}
	}
	return out
}

func sumf(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s
}

func trim(m []Mover, k int) []Mover {
	if len(m) > k {
		return m[:k]
	}
	return m
}
