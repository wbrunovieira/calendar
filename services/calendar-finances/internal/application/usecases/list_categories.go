package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/category"
)

type ListCategoriesInput struct {
	ProfileID string
	Type      *string
}

type ListCategoriesUseCase struct {
	repo category.Repository
}

func NewListCategoriesUseCase(repo category.Repository) *ListCategoriesUseCase {
	return &ListCategoriesUseCase{repo: repo}
}

func (uc *ListCategoriesUseCase) Execute(input ListCategoriesInput) ([]*category.Category, error) {
	if strings.TrimSpace(input.ProfileID) == "" {
		return nil, ErrInvalidInput
	}

	categories, err := uc.repo.ListByProfile(input.ProfileID)
	if err != nil {
		return nil, err
	}

	if input.Type == nil {
		return categories, nil
	}

	desired := strings.ToUpper(strings.TrimSpace(*input.Type))
	if desired == "" {
		return categories, nil
	}

	filtered := make([]*category.Category, 0, len(categories))
	for _, cat := range categories {
		if string(cat.Type) == desired {
			filtered = append(filtered, cat)
		}
	}
	return filtered, nil
}
