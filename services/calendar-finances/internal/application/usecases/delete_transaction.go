package usecases

import "github.com/brunovieira/calendar-finances/internal/domain/transaction"

type DeleteTransactionUseCase struct {
	repo transaction.Repository
}

func NewDeleteTransactionUseCase(repo transaction.Repository) *DeleteTransactionUseCase {
	return &DeleteTransactionUseCase{repo: repo}
}

func (uc *DeleteTransactionUseCase) Execute(id string) error {
	if err := uc.repo.Delete(id); err != nil {
		return ErrTransactionNotFound
	}
	return nil
}
