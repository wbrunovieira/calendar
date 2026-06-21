package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func sp(s string) *string { return &s }

func expTx(amount float64, desc, catID, accID string, y int, m time.Month) *transaction.Transaction {
	return &transaction.Transaction{
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        amount,
		Description:   desc,
		CategoryID:    sp(catID),
		BankAccountID: accID,
		OccurredOn:    time.Date(y, m, 15, 0, 0, 0, 0, time.UTC),
	}
}

func TestAnalyzeExpenses(t *testing.T) {
	periods := []string{"2026-05", "2026-06"}
	categories := []*category.Category{
		{ID: "ali", Name: "Alimentação"},
		{ID: "rest", Name: "Restaurante", ParentID: sp("ali")},
		{ID: "mor", Name: "Moradia"},
	}
	accounts := []*bankaccount.BankAccount{
		{ID: "mp", Type: bankaccount.AccountTypeChecking},
		{ID: "bin", Type: bankaccount.AccountTypeExchange},
	}
	recurrings := []*recurringtransaction.RecurringTransaction{
		{Status: recurringtransaction.StatusActive, Type: "EXPENSE", CategoryID: sp("mor")},
	}
	txs := []*transaction.Transaction{
		expTx(100, "Restaurante", "rest", "mp", 2026, time.May),
		expTx(800, "Aluguel", "mor", "mp", 2026, time.May),
		expTx(150, "Restaurante", "rest", "mp", 2026, time.June),
		expTx(800, "Aluguel", "mor", "mp", 2026, time.June),
		expTx(500, "Pagamento fatura Cartão Mercado Pago", "rest", "mp", 2026, time.June), // settlement -> excluded
		expTx(999, "compra cripto", "rest", "bin", 2026, time.June),                       // EXCHANGE -> excluded
	}

	out := analyzeExpenses(txs, categories, recurrings, accounts, periods)

	// Total per month excludes settlement + exchange.
	if got := out.TotalByMonth; got[0] != 900 || got[1] != 950 {
		t.Fatalf("totalByMonth = %v, want [900 950]", got)
	}
	if out.Average != 925 {
		t.Errorf("average = %.2f, want 925", out.Average)
	}

	// Fixed (Moradia has a recurring) vs variable (Restaurante).
	if out.FixedByMonth[0] != 800 || out.FixedByMonth[1] != 800 {
		t.Errorf("fixedByMonth = %v, want [800 800]", out.FixedByMonth)
	}
	if out.VariableByMonth[0] != 100 || out.VariableByMonth[1] != 150 {
		t.Errorf("variableByMonth = %v, want [100 150]", out.VariableByMonth)
	}

	// Categories rolled up to top level, sorted by total (Moradia 1600 > Alimentação 250).
	if len(out.Categories) != 2 {
		t.Fatalf("categories = %d, want 2", len(out.Categories))
	}
	if out.Categories[0].Name != "Moradia" || out.Categories[0].Total != 1600 {
		t.Errorf("cat[0] = %+v, want Moradia 1600", out.Categories[0])
	}
	ali := out.Categories[1]
	if ali.Name != "Alimentação" || ali.Total != 250 {
		t.Errorf("cat[1] = %+v, want Alimentação 250", ali)
	}
	// Alimentação drills down into Restaurante.
	if len(ali.Children) != 1 || ali.Children[0].Name != "Restaurante" || ali.Children[0].Total != 250 {
		t.Errorf("alimentação children = %+v, want [Restaurante 250]", ali.Children)
	}
	// Moradia is flat, has no sub-breakdown.
	if out.Categories[0].Trend != "flat" || len(out.Categories[0].Children) != 0 {
		t.Errorf("Moradia trend/children = %s / %d", out.Categories[0].Trend, len(out.Categories[0].Children))
	}
	// Alimentação rose (150 last vs 125 avg).
	if ali.Trend != "up" {
		t.Errorf("Alimentação trend = %s, want up", ali.Trend)
	}

	// Movers: Alimentação is the only one above its average.
	if len(out.TopUp) != 1 || out.TopUp[0].Name != "Alimentação" || out.TopUp[0].Delta != 25 {
		t.Errorf("topUp = %+v, want [Alimentação +25]", out.TopUp)
	}
}

func TestAnalyzeExpenses_Empty(t *testing.T) {
	out := analyzeExpenses(nil, nil, nil, nil, []string{"2026-06"})
	if out.TotalByMonth[0] != 0 || out.Average != 0 || len(out.Categories) != 0 {
		t.Errorf("empty analysis not zeroed: %+v", out)
	}
	// Slices must be non-nil so the JSON is [] (not null) and the client is safe.
	if out.Categories == nil || out.TopUp == nil || out.TopDown == nil {
		t.Errorf("nil slice would serialize to JSON null: categories=%v topUp=%v topDown=%v",
			out.Categories, out.TopUp, out.TopDown)
	}
}

func TestAnalyzeExpenses_FutureMonthsHaveNoUpMovers(t *testing.T) {
	// A range spilling into empty future months makes the last month zero, so every
	// category falls vs its average -> topUp ends empty. Must be [] not nil.
	categories := []*category.Category{{ID: "ali", Name: "Alimentação"}}
	txs := []*transaction.Transaction{expTx(100, "x", "ali", "mp", 2026, time.May)}
	out := analyzeExpenses(txs, categories, nil, nil, []string{"2026-05", "2026-12"})
	if out.TopUp == nil || len(out.TopUp) != 0 {
		t.Errorf("expected empty (non-nil) topUp, got %v", out.TopUp)
	}
}

func TestAnalyzeExpenses_ExcludesNonConsumption(t *testing.T) {
	// Cost of living counts consumption only: transfers (TRANSFER-typed category),
	// loans (Empréstimos) and investments (Investimentos) must be excluded.
	periods := []string{"2026-06"}
	categories := []*category.Category{
		{ID: "ali", Name: "Alimentação", Type: category.TypeExpense},
		{ID: "transf", Name: "Transferências", Type: category.TypeTransfer},
		{ID: "emp", Name: "Empréstimos", Type: category.TypeExpense},
		{ID: "inv", Name: "Investimentos", Type: category.TypeExpense},
	}
	txs := []*transaction.Transaction{
		expTx(100, "mercado", "ali", "mp", 2026, time.June),
		expTx(5000, "Transferência para Nubank", "transf", "mp", 2026, time.June),
		expTx(3000, "empréstimo Dani", "emp", "mp", 2026, time.June),
		expTx(2000, "CDB", "inv", "mp", 2026, time.June),
	}
	out := analyzeExpenses(txs, categories, nil, nil, periods)
	if out.TotalByMonth[0] != 100 {
		t.Errorf("total = %v, want 100 (transfer/loan/investment excluded)", out.TotalByMonth[0])
	}
	if len(out.Categories) != 1 || out.Categories[0].Name != "Alimentação" {
		t.Errorf("categories = %+v, want only Alimentação", out.Categories)
	}
}
