package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/category"
)

type UpdateCategoryInput struct {
	Name              string                     `json:"name"`
	Type              string                     `json:"type"`
	Color             *string                    `json:"color,omitempty"`
	Icon              *string                    `json:"icon,omitempty"`
	ParentID          *string                    `json:"parentId,omitempty"`
	ClassificationDRE *category.ClassificationDRE `json:"classificationDRE,omitempty"`
}

type UpdateCategoryUseCase struct {
	repo category.Repository
}

func NewUpdateCategoryUseCase(repo category.Repository) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{repo: repo}
}

func (uc *UpdateCategoryUseCase) Execute(id string, input UpdateCategoryInput) (*category.Category, error) {
	cat, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	typeValue := category.Type(strings.ToUpper(strings.TrimSpace(input.Type)))
	if err := cat.Update(input.Name, typeValue, input.Color, input.Icon, input.ParentID, input.ClassificationDRE); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(cat); err != nil {
		return nil, err
	}

	return cat, nil
}
