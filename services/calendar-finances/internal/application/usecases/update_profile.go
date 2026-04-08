package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type UpdateProfileInput struct {
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	LegalEntityType *profile.LegalEntityType `json:"legalEntityType,omitempty"`
	CompanyName     *string                  `json:"companyName,omitempty"`
	CNPJ            *string                  `json:"cnpj,omitempty"`
	SimplesNacional *bool                    `json:"simplesNacional,omitempty"`
	TaxRegime       *profile.TaxRegime       `json:"taxRegime,omitempty"`
	DasAliquota     *float64                 `json:"dasAliquota,omitempty"`
	OpeningDate     *time.Time               `json:"openingDate,omitempty"`
}

type UpdateProfileUseCase struct {
	repo profile.Repository
}

func NewUpdateProfileUseCase(repo profile.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

func (uc *UpdateProfileUseCase) Execute(id string, input UpdateProfileInput) (*profile.Profile, error) {
	p, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrProfileNotFound
	}

	if err := p.Update(profile.UpdateParams{
		Name:            input.Name,
		Type:            profile.ProfileType(input.Type),
		LegalEntityType: input.LegalEntityType,
		CompanyName:     input.CompanyName,
		CNPJ:            input.CNPJ,
		SimplesNacional: input.SimplesNacional,
		TaxRegime:       input.TaxRegime,
		DasAliquota:     input.DasAliquota,
		OpeningDate:     input.OpeningDate,
	}); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(p); err != nil {
		return nil, err
	}

	return p, nil
}
