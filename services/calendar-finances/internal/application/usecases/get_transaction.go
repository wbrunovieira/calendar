package usecases

import "github.com/brunovieira/calendar-finances/internal/domain/transaction"

type GetTransactionUseCase struct {
	repo transaction.Repository
}

func NewGetTransactionUseCase(repo transaction.Repository) *GetTransactionUseCase {
	return &GetTransactionUseCase{repo: repo}
}

func (uc *GetTransactionUseCase) Execute(id string) (*transaction.Transaction, error) {
	tx, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, ErrTransactionNotFound
	}
	return tx, nil
}
