package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type UpdateProfileInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type UpdateProfileUseCase struct {
	repo profile.Repository
}

func NewUpdateProfileUseCase(repo profile.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

func (uc *UpdateProfileUseCase) Execute(id string, input UpdateProfileInput) (*profile.Profile, error) {
	// Find existing profile
	p, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrProfileNotFound
	}

	// Update profile
	if err := p.Update(input.Name, profile.ProfileType(input.Type)); err != nil {
		return nil, err
	}

	// Save to database
	if err := uc.repo.Update(p); err != nil {
		return nil, err
	}

	return p, nil
}
