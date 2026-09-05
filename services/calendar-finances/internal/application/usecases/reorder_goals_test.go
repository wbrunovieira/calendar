package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/goal"
)

type fakeGoalRepoForReorder struct {
	goals   map[string]*goal.Goal
	updates []goal.DisplayOrderUpdate
}

func (f *fakeGoalRepoForReorder) Create(g *goal.Goal) error { return nil }
func (f *fakeGoalRepoForReorder) Update(g *goal.Goal) error { return nil }
func (f *fakeGoalRepoForReorder) Delete(id string) error    { return nil }
func (f *fakeGoalRepoForReorder) FindByID(id string) (*goal.Goal, error) {
	g, ok := f.goals[id]
	if !ok {
		return nil, nil
	}
	return g, nil
}
func (f *fakeGoalRepoForReorder) ListByProfile(profileID string) ([]*goal.Goal, error) {
	return nil, nil
}
func (f *fakeGoalRepoForReorder) UpdateDisplayOrders(updates []goal.DisplayOrderUpdate) error {
	f.updates = updates
	return nil
}

func TestReorderGoals_ShouldUpdateDisplayOrders(t *testing.T) {
	repo := &fakeGoalRepoForReorder{goals: map[string]*goal.Goal{}}

	uc := NewReorderGoalsUseCase(repo)

	items := []ReorderItem{
		{ID: "goal-1", DisplayOrder: 2},
		{ID: "goal-2", DisplayOrder: 1},
		{ID: "goal-3", DisplayOrder: 3},
	}

	err := uc.Execute(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.updates) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(repo.updates))
	}

	for i, item := range items {
		if repo.updates[i].ID != item.ID {
			t.Errorf("update[%d] ID = %s, want %s", i, repo.updates[i].ID, item.ID)
		}
		if repo.updates[i].DisplayOrder != item.DisplayOrder {
			t.Errorf("update[%d] DisplayOrder = %d, want %d", i, repo.updates[i].DisplayOrder, item.DisplayOrder)
		}
	}
}

func TestReorderGoals_EmptyItems_ShouldSucceed(t *testing.T) {
	repo := &fakeGoalRepoForReorder{goals: map[string]*goal.Goal{}}
	uc := NewReorderGoalsUseCase(repo)

	err := uc.Execute([]ReorderItem{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.updates) != 0 {
		t.Fatalf("expected 0 updates, got %d", len(repo.updates))
	}
}
