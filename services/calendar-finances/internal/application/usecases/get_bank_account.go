package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type GetBankAccountUseCase struct {
	repo bankaccount.Repository
}

func NewGetBankAccountUseCase(repo bankaccount.Repository) *GetBankAccountUseCase {
	return &GetBankAccountUseCase{repo: repo}
}

func (uc *GetBankAccountUseCase) Execute(id string) (*bankaccount.BankAccount, error) {
	account, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}
	return account, nil
}
