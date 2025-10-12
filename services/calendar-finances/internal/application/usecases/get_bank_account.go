package usecases

import (
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

var ErrBankAccountNotFound = errors.New("bank account not found")

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
