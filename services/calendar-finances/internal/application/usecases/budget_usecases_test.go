package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/budgettarget"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type fakeBudgetRepo struct {
	targets []*budgettarget.BudgetTarget
}

func (f *fakeBudgetRepo) Create(entity *budgettarget.BudgetTarget) error {
	f.targets = append(f.targets, entity)
	return nil
}

func (f *fakeBudgetRepo) Update(entity *budgettarget.BudgetTarget) error {
	for i, t := range f.targets {
		if t.ID == entity.ID {
			f.targets[i] = entity
			return nil
		}
	}
	return nil
}

func (f *fakeBudgetRepo) FindByID(id string) (*budgettarget.BudgetTarget, error) {
	for _, t := range f.targets {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, ErrBudgetTargetNotFound
}

func (f *fakeBudgetRepo) ListByProfile(profileID string) ([]*budgettarget.BudgetTarget, error) {
	var result []*budgettarget.BudgetTarget
	for _, t := range f.targets {
		if t.ProfileID == profileID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (f *fakeBudgetRepo) ListByProfileAndPeriod(profileID string, periodStart time.Time) ([]*budgettarget.BudgetTarget, error) {
	var result []*budgettarget.BudgetTarget
	for _, t := range f.targets {
		if t.ProfileID == profileID && t.PeriodStart.Equal(periodStart) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (f *fakeBudgetRepo) Delete(id string) error {
	for i, t := range f.targets {
		if t.ID == id {
			f.targets = append(f.targets[:i], f.targets[i+1:]...)
			return nil
		}
	}
	return nil
}

type fakeTxRepoForBudget struct {
	sums map[string]float64
}

func (f *fakeTxRepoForBudget) SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error) {
	result := make(map[string]float64)
	for _, id := range categoryIDs {
		if sum, ok := f.sums[id]; ok {
			result[id] = sum
		}
	}
	return result, nil
}

// Implement other methods as no-ops
func (f *fakeTxRepoForBudget) Create(tx *transaction.Transaction) error { return nil }
func (f *fakeTxRepoForBudget) GetByID(id string) (*transaction.Transaction, error) {
	return nil, nil
}
func (f *fakeTxRepoForBudget) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	return nil, nil
}
func (f *fakeTxRepoForBudget) Update(tx *transaction.Transaction) error { return nil }
func (f *fakeTxRepoForBudget) UpdateStatus(id string, status transaction.Status, occurredOn time.Time, notes *string) error {
	return nil
}
func (f *fakeTxRepoForBudget) Delete(id string) error { return nil }
func (f *fakeTxRepoForBudget) SumByInvoiceID(invoiceID string) (float64, error) {
	return 0, nil
}

func (f *fakeTxRepoForBudget) SumByInvoiceIDByStatus(invoiceID string, status transaction.Status) (float64, error) {
	return 0, nil
}

func (f *fakeTxRepoForBudget) CalculateBalanceByBankAccountID(bankAccountID string) (float64, error) {
	return 0, nil
}

func (f *fakeTxRepoForBudget) FindByExternalID(externalID string) (*transaction.Transaction, error) {
	return nil, nil
}
func (f *fakeTxRepoForBudget) CalculateBalanceSince(_ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (f *fakeTxRepoForBudget) CalculateBalanceUpTo(_ string, _ time.Time) (float64, error) {
	return 0, nil
}

type fakeCategoryRepoForBudget struct {
	categories map[string]*category.Category
}

func (f *fakeCategoryRepoForBudget) Create(*category.Category) error { return nil }
func (f *fakeCategoryRepoForBudget) Update(*category.Category) error { return nil }
func (f *fakeCategoryRepoForBudget) FindByID(id string) (*category.Category, error) {
	if cat, ok := f.categories[id]; ok {
		return cat, nil
	}
	return nil, ErrCategoryNotFound
}
func (f *fakeCategoryRepoForBudget) ListByProfile(profileID string) ([]*category.Category, error) {
	var list []*category.Category
	for _, cat := range f.categories {
		if cat.ProfileID == profileID {
			list = append(list, cat)
		}
	}
	return list, nil
}
func (f *fakeCategoryRepoForBudget) Deactivate(string) error { return nil }
func (f *fakeCategoryRepoForBudget) GetDescendantIDs(id string) ([]string, error) {
	ids := []string{id}
	for _, cat := range f.categories {
		if cat.ParentID != nil && *cat.ParentID == id {
			childIDs, _ := f.GetDescendantIDs(cat.ID)
			ids = append(ids, childIDs...)
		}
	}
	return ids, nil
}

func TestBudgetSummaryWithSubcategories(t *testing.T) {
	parentID := "parent-cat"
	child1ID := "child-cat-1"
	child2ID := "child-cat-2"
	grandchildID := "grandchild-cat"
	profileID := "profile-1"

	// Create category hierarchy: parent -> child1, child2 -> grandchild
	categories := map[string]*category.Category{
		parentID: {
			ID:        parentID,
			ProfileID: profileID,
			Name:      "Alimentacao",
			Type:      category.TypeExpense,
		},
		child1ID: {
			ID:        child1ID,
			ProfileID: profileID,
			Name:      "Supermercado",
			Type:      category.TypeExpense,
			ParentID:  &parentID,
		},
		child2ID: {
			ID:        child2ID,
			ProfileID: profileID,
			Name:      "Restaurante",
			Type:      category.TypeExpense,
			ParentID:  &parentID,
		},
		grandchildID: {
			ID:        grandchildID,
			ProfileID: profileID,
			Name:      "iFood",
			Type:      category.TypeExpense,
			ParentID:  &child2ID,
		},
	}

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	budgetRepo := &fakeBudgetRepo{
		targets: []*budgettarget.BudgetTarget{
			{
				ID:          "budget-1",
				ProfileID:   profileID,
				CategoryID:  parentID, // Budget on parent category
				PeriodStart: periodStart,
				Amount:      2000,
			},
		},
	}

	// Transactions are on child categories
	txRepo := &fakeTxRepoForBudget{
		sums: map[string]float64{
			child1ID:     500,  // Supermercado
			child2ID:     200,  // Restaurante
			grandchildID: 100,  // iFood
		},
	}

	catRepo := &fakeCategoryRepoForBudget{categories: categories}

	service := NewBudgetTargetsService(budgetRepo, txRepo, catRepo)

	summary, err := service.Summary(profileID, "2026-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary) != 1 {
		t.Fatalf("expected 1 summary item, got %d", len(summary))
	}

	item := summary[0]

	// Should sum all descendants: 500 + 200 + 100 = 800
	expectedSpent := 800.0
	if item.Spent != expectedSpent {
		t.Errorf("expected spent %.2f, got %.2f", expectedSpent, item.Spent)
	}

	expectedRemaining := 2000 - 800.0
	if item.Remaining != expectedRemaining {
		t.Errorf("expected remaining %.2f, got %.2f", expectedRemaining, item.Remaining)
	}
}

func TestBudgetSummaryNoSubcategories(t *testing.T) {
	categoryID := "cat-1"
	profileID := "profile-1"

	categories := map[string]*category.Category{
		categoryID: {
			ID:        categoryID,
			ProfileID: profileID,
			Name:      "Transporte",
			Type:      category.TypeExpense,
		},
	}

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	budgetRepo := &fakeBudgetRepo{
		targets: []*budgettarget.BudgetTarget{
			{
				ID:          "budget-1",
				ProfileID:   profileID,
				CategoryID:  categoryID,
				PeriodStart: periodStart,
				Amount:      500,
			},
		},
	}

	txRepo := &fakeTxRepoForBudget{
		sums: map[string]float64{
			categoryID: 150,
		},
	}

	catRepo := &fakeCategoryRepoForBudget{categories: categories}

	service := NewBudgetTargetsService(budgetRepo, txRepo, catRepo)

	summary, err := service.Summary(profileID, "2026-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary) != 1 {
		t.Fatalf("expected 1 summary item, got %d", len(summary))
	}

	item := summary[0]

	if item.Spent != 150 {
		t.Errorf("expected spent 150, got %.2f", item.Spent)
	}

	if item.Remaining != 350 {
		t.Errorf("expected remaining 350, got %.2f", item.Remaining)
	}
}

func TestBudgetSummaryEmptyPeriod(t *testing.T) {
	budgetRepo := &fakeBudgetRepo{targets: []*budgettarget.BudgetTarget{}}
	txRepo := &fakeTxRepoForBudget{sums: map[string]float64{}}
	catRepo := &fakeCategoryRepoForBudget{categories: map[string]*category.Category{}}

	service := NewBudgetTargetsService(budgetRepo, txRepo, catRepo)

	summary, err := service.Summary("profile-1", "2026-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary) != 0 {
		t.Fatalf("expected 0 summary items, got %d", len(summary))
	}
}
