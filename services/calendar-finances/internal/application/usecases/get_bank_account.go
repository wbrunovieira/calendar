package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type GetBankAccountUseCase struct {
	repo        bankaccount.Repository
	outstanding *CardOutstanding
}

func NewGetBankAccountUseCase(repo bankaccount.Repository, outstanding *CardOutstanding) *GetBankAccountUseCase {
	return &GetBankAccountUseCase{repo: repo, outstanding: outstanding}
}

func (uc *GetBankAccountUseCase) Execute(id string) (*bankaccount.BankAccount, error) {
	account, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}
	// A card's balance is derived here too, so reading one account and reading
	// the list cannot disagree.
	if err := uc.outstanding.Apply([]*bankaccount.BankAccount{account}); err != nil {
		return nil, err
	}
	return account, nil
}
