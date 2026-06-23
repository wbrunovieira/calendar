package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func incTx(amount float64, catID, accID string, y int, m time.Month) *transaction.Transaction {
	t := expTx(amount, "receita", catID, accID, y, m)
	t.Type = transaction.TypeIncome
	return t
}

func TestAnalyzeFinancialSummary(t *testing.T) {
	periods := []string{"2026-05", "2026-06"}
	categories := []*category.Category{
		{ID: "serv", Name: "Serviços", Type: category.TypeIncome},
		{ID: "ali", Name: "Alimentação", Type: category.TypeExpense},
		{ID: "transf", Name: "Transferências", Type: category.TypeTransfer},
	}
	accounts := []*bankaccount.BankAccount{{ID: "mp", Type: bankaccount.AccountTypeChecking}}
	txs := []*transaction.Transaction{
		incTx(5000, "serv", "mp", 2026, time.May),
		incTx(3000, "serv", "mp", 2026, time.June),
		expTx(2000, "mercado", "ali", "mp", 2026, time.May),
		expTx(1000, "mercado", "ali", "mp", 2026, time.June),
		incTx(9999, "transf", "mp", 2026, time.June),      // transfer-in: excluded from revenue
		expTx(8888, "x", "transf", "mp", 2026, time.June), // transfer-out: excluded from expense
	}

	out := analyzeFinancialSummary(txs, categories, accounts, periods)

	if out.RevenueByMonth[0] != 5000 || out.RevenueByMonth[1] != 3000 {
		t.Errorf("revenue = %v, want [5000 3000] (transfers excluded)", out.RevenueByMonth)
	}
	if out.ExpenseByMonth[0] != 2000 || out.ExpenseByMonth[1] != 1000 {
		t.Errorf("expense = %v, want [2000 1000] (transfers excluded)", out.ExpenseByMonth)
	}
	if out.ResultByMonth[0] != 3000 || out.ResultByMonth[1] != 2000 {
		t.Errorf("result = %v, want [3000 2000]", out.ResultByMonth)
	}
	if out.MarginByMonth[0] != 60 {
		t.Errorf("margin[0] = %v, want 60", out.MarginByMonth[0])
	}
	if out.TotalRevenue != 8000 || out.TotalExpense != 3000 || out.TotalResult != 5000 {
		t.Errorf("totals = %v/%v/%v, want 8000/3000/5000", out.TotalRevenue, out.TotalExpense, out.TotalResult)
	}
	if out.AvgMargin != 62.5 {
		t.Errorf("avgMargin = %v, want 62.5", out.AvgMargin)
	}
	if len(out.RevenueCategories) != 1 || out.RevenueCategories[0].Name != "Serviços" {
		t.Errorf("revenueCategories = %+v, want [Serviços]", out.RevenueCategories)
	}
	if len(out.ExpenseCategories) != 1 || out.ExpenseCategories[0].Name != "Alimentação" {
		t.Errorf("expenseCategories = %+v, want [Alimentação]", out.ExpenseCategories)
	}
}

func TestAnalyzeFinancialSummary_LossAndZeroRevenue(t *testing.T) {
	// Expense with zero revenue: result negative, margin 0 (no divide-by-zero), slices non-nil.
	periods := []string{"2026-04"}
	categories := []*category.Category{{ID: "ali", Name: "Alimentação", Type: category.TypeExpense}}
	txs := []*transaction.Transaction{expTx(500, "x", "ali", "mp", 2026, time.April)}

	out := analyzeFinancialSummary(txs, categories, nil, periods)
	if out.ResultByMonth[0] != -500 || out.MarginByMonth[0] != 0 {
		t.Errorf("result/margin = %v/%v, want -500/0", out.ResultByMonth[0], out.MarginByMonth[0])
	}
	if out.RevenueCategories == nil || out.ExpenseCategories == nil {
		t.Errorf("category slices must be non-nil (JSON [] not null)")
	}
}

func TestAnalyzeFinancialSummary_AporteExcludedRendimentoSeparate(t *testing.T) {
	periods := []string{"2026-05"}
	categories := []*category.Category{
		{ID: "serv", Name: "Projetos/Servicos", Type: category.TypeIncome},
		{ID: "rend", Name: "Rendimentos Caixinha", Type: category.TypeIncome},
		{ID: "aporte", Name: "Aporte Socio", Type: category.TypeIncome},
		{ID: "infra", Name: "Infraestrutura", Type: category.TypeExpense},
	}
	txs := []*transaction.Transaction{
		incTx(5000, "serv", "mp", 2026, time.May),   // operational revenue
		incTx(180, "rend", "mp", 2026, time.May),    // financial income (shown apart)
		incTx(1200, "aporte", "mp", 2026, time.May), // owner capital -> excluded entirely
		expTx(1000, "infra", "infra", "mp", 2026, time.May),
	}

	out := analyzeFinancialSummary(txs, categories, nil, periods)

	if out.RevenueByMonth[0] != 5000 {
		t.Errorf("operational revenue = %v, want 5000 (rendimento + aporte excluded)", out.RevenueByMonth[0])
	}
	if out.FinancialIncomeByMonth[0] != 180 || out.TotalFinancialIncome != 180 {
		t.Errorf("financial income = %v / %v, want 180", out.FinancialIncomeByMonth[0], out.TotalFinancialIncome)
	}
	// result = operational 5000 + financial 180 - expense 1000 = 4180 (aporte 1200 dropped)
	if out.ResultByMonth[0] != 4180 {
		t.Errorf("result = %v, want 4180 (aporte excluded)", out.ResultByMonth[0])
	}
}

func TestAnalyzeFinancialSummary_SubcategoryClassification(t *testing.T) {
	// Aporte / Rendimentos are sub-categories under "Receitas" -> classification must
	// walk the chain (leaf to root), not just the root category.
	periods := []string{"2026-05"}
	categories := []*category.Category{
		{ID: "rec", Name: "Receitas", Type: category.TypeIncome},
		{ID: "serv", Name: "Projetos/Servicos", Type: category.TypeIncome, ParentID: sp("rec")},
		{ID: "rend", Name: "Rendimentos Caixinha", Type: category.TypeIncome, ParentID: sp("rec")},
		{ID: "aporte", Name: "Aporte Socio", Type: category.TypeIncome, ParentID: sp("rec")},
	}
	txs := []*transaction.Transaction{
		incTx(5000, "serv", "mp", 2026, time.May),
		incTx(180, "rend", "mp", 2026, time.May),
		incTx(1200, "aporte", "mp", 2026, time.May),
	}

	out := analyzeFinancialSummary(txs, categories, nil, periods)

	if out.RevenueByMonth[0] != 5000 {
		t.Errorf("operational revenue = %v, want 5000 (sub-cat aporte/rendimento handled)", out.RevenueByMonth[0])
	}
	if out.FinancialIncomeByMonth[0] != 180 {
		t.Errorf("financial income = %v, want 180", out.FinancialIncomeByMonth[0])
	}
	if out.ResultByMonth[0] != 5180 {
		t.Errorf("result = %v, want 5180 (5000 + 180, aporte excluded)", out.ResultByMonth[0])
	}
}

func TestAnalyzeFinancialSummary_DRELines(t *testing.T) {
	dre := func(s string) *category.ClassificationDRE { v := category.ClassificationDRE(s); return &v }
	periods := []string{"2026-05"}
	categories := []*category.Category{
		{ID: "rec", Name: "Receitas", Type: category.TypeIncome, ClassificationDRE: dre("REVENUE")},
		{ID: "serv", Name: "Projetos", Type: category.TypeIncome, ParentID: sp("rec")}, // inherits REVENUE
		{ID: "rend", Name: "Rendimentos", Type: category.TypeIncome, ParentID: sp("rec"), ClassificationDRE: dre("FINANCIAL")},
		{ID: "tax", Name: "Impostos", Type: category.TypeExpense, ClassificationDRE: dre("TAX")},
		{ID: "fix", Name: "Pessoal", Type: category.TypeExpense, ClassificationDRE: dre("FIXED_COST")},
		{ID: "other", Name: "Algo", Type: category.TypeExpense}, // unclassified -> OUTROS
	}
	txs := []*transaction.Transaction{
		incTx(5000, "serv", "mp", 2026, time.May),
		incTx(180, "rend", "mp", 2026, time.May),
		expTx(300, "imposto", "tax", "mp", 2026, time.May),
		expTx(2000, "folha", "fix", "mp", 2026, time.May),
		expTx(50, "x", "other", "mp", 2026, time.May),
	}

	out := analyzeFinancialSummary(txs, categories, nil, periods)

	got := map[string]float64{}
	for _, l := range out.Dre {
		got[l.Classification] = l.Total
	}
	if got["REVENUE"] != 5000 || got["FINANCIAL"] != 180 || got["TAX"] != 300 || got["FIXED_COST"] != 2000 || got["OUTROS"] != 50 {
		t.Errorf("dre lines = %+v", out.Dre)
	}
	if len(out.Dre) == 0 || out.Dre[0].Classification != "REVENUE" {
		t.Errorf("dre must start with REVENUE: %+v", out.Dre)
	}
}
