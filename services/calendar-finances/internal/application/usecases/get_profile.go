package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type GetProfileUseCase struct {
	repo profile.Repository
}

func NewGetProfileUseCase(repo profile.Repository) *GetProfileUseCase {
	return &GetProfileUseCase{repo: repo}
}

func (uc *GetProfileUseCase) Execute(id string) (*profile.Profile, error) {
	p, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	return p, nil
}
