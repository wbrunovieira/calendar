package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type ReorderItem struct {
	ID           string `json:"id"`
	DisplayOrder int    `json:"displayOrder"`
}

type ReorderBankAccountsUseCase struct {
	repo bankaccount.Repository
}

func NewReorderBankAccountsUseCase(repo bankaccount.Repository) *ReorderBankAccountsUseCase {
	return &ReorderBankAccountsUseCase{repo: repo}
}

func (uc *ReorderBankAccountsUseCase) Execute(items []ReorderItem) error {
	updates := make([]bankaccount.DisplayOrderUpdate, len(items))
	for i, item := range items {
		updates[i] = bankaccount.DisplayOrderUpdate{
			ID:           item.ID,
			DisplayOrder: item.DisplayOrder,
		}
	}
	return uc.repo.UpdateDisplayOrders(updates)
}
