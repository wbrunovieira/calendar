package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type DeleteBankAccountUseCase struct {
	repo bankaccount.Repository
}

func NewDeleteBankAccountUseCase(repo bankaccount.Repository) *DeleteBankAccountUseCase {
	return &DeleteBankAccountUseCase{repo: repo}
}

func (uc *DeleteBankAccountUseCase) Execute(id string) error {
	// Check if exists
	_, err := uc.repo.FindByID(id)
	if err != nil {
		return ErrBankAccountNotFound
	}

	return uc.repo.Delete(id)
}
