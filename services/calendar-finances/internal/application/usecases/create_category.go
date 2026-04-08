package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

type CreateCategoryInput struct {
	ProfileID         string                      `json:"profileId"`
	Name              string                      `json:"name"`
	Type              string                      `json:"type"`
	Color             *string                     `json:"color,omitempty"`
	Icon              *string                     `json:"icon,omitempty"`
	ParentID          *string                     `json:"parentId,omitempty"`
	ClassificationDRE *category.ClassificationDRE `json:"classificationDRE,omitempty"`
}

type CreateCategoryUseCase struct {
	profileRepo  profile.Repository
	categoryRepo category.Repository
}

func NewCreateCategoryUseCase(profileRepo profile.Repository, categoryRepo category.Repository) *CreateCategoryUseCase {
	return &CreateCategoryUseCase{profileRepo: profileRepo, categoryRepo: categoryRepo}
}

func (uc *CreateCategoryUseCase) Execute(input CreateCategoryInput) (*category.Category, error) {
	if strings.TrimSpace(input.ProfileID) == "" {
		return nil, ErrInvalidInput
	}

	if _, err := uc.profileRepo.FindByID(input.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}

	typeValue := category.Type(strings.ToUpper(strings.TrimSpace(input.Type)))
	cat, err := category.NewCategory(category.CreateParams{
		ProfileID:         input.ProfileID,
		Name:              input.Name,
		Type:              typeValue,
		Color:             input.Color,
		Icon:              input.Icon,
		ParentID:          input.ParentID,
		ClassificationDRE: input.ClassificationDRE,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.categoryRepo.Create(cat); err != nil {
		return nil, err
	}

	return cat, nil
}
