package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type ListBankAccountsUseCase struct {
	repo bankaccount.Repository
}

func NewListBankAccountsUseCase(repo bankaccount.Repository) *ListBankAccountsUseCase {
	return &ListBankAccountsUseCase{repo: repo}
}

func (uc *ListBankAccountsUseCase) Execute(profileID string) ([]*bankaccount.BankAccount, error) {
	if profileID != "" {
		return uc.repo.FindByProfileID(profileID)
	}
	return uc.repo.FindAll()
}
