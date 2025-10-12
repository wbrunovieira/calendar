package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type CreateProfileInput struct {
	CalendarID string `json:"calendarId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
}

type CreateProfileUseCase struct {
	repo profile.Repository
}

func NewCreateProfileUseCase(repo profile.Repository) *CreateProfileUseCase {
	return &CreateProfileUseCase{repo: repo}
}

func (uc *CreateProfileUseCase) Execute(input CreateProfileInput) (*profile.Profile, error) {
	// Check if profile already exists for this calendar
	existing, _ := uc.repo.FindByCalendarID(input.CalendarID)
	if existing != nil {
		return nil, ErrProfileAlreadyExists
	}

	// Create new profile
	p, err := profile.NewProfile(input.CalendarID, input.Name, profile.ProfileType(input.Type))
	if err != nil {
		return nil, err
	}

	// Save to database
	if err := uc.repo.Create(p); err != nil {
		return nil, err
	}

	return p, nil
}
