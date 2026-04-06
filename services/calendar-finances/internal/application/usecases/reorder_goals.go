package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/goal"
)

type ReorderGoalsUseCase struct {
	repo goal.Repository
}

func NewReorderGoalsUseCase(repo goal.Repository) *ReorderGoalsUseCase {
	return &ReorderGoalsUseCase{repo: repo}
}

func (uc *ReorderGoalsUseCase) Execute(items []ReorderItem) error {
	updates := make([]goal.DisplayOrderUpdate, len(items))
	for i, item := range items {
		updates[i] = goal.DisplayOrderUpdate{
			ID:           item.ID,
			DisplayOrder: item.DisplayOrder,
		}
	}
	return uc.repo.UpdateDisplayOrders(updates)
}
