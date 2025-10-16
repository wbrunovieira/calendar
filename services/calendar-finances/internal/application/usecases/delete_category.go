package usecases

import "github.com/brunovieira/calendar-finances/internal/domain/category"

type DeleteCategoryUseCase struct {
	repo category.Repository
}

func NewDeleteCategoryUseCase(repo category.Repository) *DeleteCategoryUseCase {
	return &DeleteCategoryUseCase{repo: repo}
}

func (uc *DeleteCategoryUseCase) Execute(id string) error {
	if _, err := uc.repo.FindByID(id); err != nil {
		return ErrCategoryNotFound
	}
	return uc.repo.Deactivate(id)
}
