package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type DeleteProfileUseCase struct {
	repo profile.Repository
}

func NewDeleteProfileUseCase(repo profile.Repository) *DeleteProfileUseCase {
	return &DeleteProfileUseCase{repo: repo}
}

func (uc *DeleteProfileUseCase) Execute(id string) error {
	// Check if profile exists
	_, err := uc.repo.FindByID(id)
	if err != nil {
		return ErrProfileNotFound
	}

	// Delete profile
	return uc.repo.Delete(id)
}
