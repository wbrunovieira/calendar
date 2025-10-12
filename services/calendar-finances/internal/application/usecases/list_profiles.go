package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type ListProfilesUseCase struct {
	repo profile.Repository
}

func NewListProfilesUseCase(repo profile.Repository) *ListProfilesUseCase {
	return &ListProfilesUseCase{repo: repo}
}

func (uc *ListProfilesUseCase) Execute() ([]*profile.Profile, error) {
	return uc.repo.FindAll()
}
