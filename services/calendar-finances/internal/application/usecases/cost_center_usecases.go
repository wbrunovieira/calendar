package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
)

// --- Create ---

type CreateCostCenterInput struct {
	ProfileID string                `json:"profileId"`
	Name      string                `json:"name"`
	Type      costcenter.CenterType `json:"type"`
	Color     *string               `json:"color,omitempty"`
	// ExternalID mirrors a record in another system (a CRM organization, say).
	// Opaque string on purpose — see the domain for why the shape is not
	// validated.
	ExternalID  *string `json:"externalId,omitempty"`
	ExternalSrc string  `json:"externalSource,omitempty"`
}

type CreateCostCenterUseCase struct {
	repo costcenter.Repository
}

func NewCreateCostCenterUseCase(repo costcenter.Repository) *CreateCostCenterUseCase {
	return &CreateCostCenterUseCase{repo: repo}
}

func (uc *CreateCostCenterUseCase) Execute(input CreateCostCenterInput) (*costcenter.CostCenter, error) {
	c, err := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID:   input.ProfileID,
		Name:        input.Name,
		Type:        input.Type,
		Color:       input.Color,
		ExternalID:  input.ExternalID,
		ExternalSrc: input.ExternalSrc,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// --- List ---

type ListCostCentersUseCase struct {
	repo costcenter.Repository
}

func NewListCostCentersUseCase(repo costcenter.Repository) *ListCostCentersUseCase {
	return &ListCostCentersUseCase{repo: repo}
}

func (uc *ListCostCentersUseCase) Execute(profileID string) ([]*costcenter.CostCenter, error) {
	return uc.repo.FindByProfile(profileID)
}

// --- Get ---

type GetCostCenterUseCase struct {
	repo costcenter.Repository
}

func NewGetCostCenterUseCase(repo costcenter.Repository) *GetCostCenterUseCase {
	return &GetCostCenterUseCase{repo: repo}
}

func (uc *GetCostCenterUseCase) Execute(id string) (*costcenter.CostCenter, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCostCenterNotFound
	}
	return c, nil
}

// --- Update ---

type UpdateCostCenterInput struct {
	Name     string                `json:"name"`
	Type     costcenter.CenterType `json:"type"`
	Color    *string               `json:"color,omitempty"`
	IsActive bool                  `json:"isActive"`
}

type UpdateCostCenterUseCase struct {
	repo costcenter.Repository
}

func NewUpdateCostCenterUseCase(repo costcenter.Repository) *UpdateCostCenterUseCase {
	return &UpdateCostCenterUseCase{repo: repo}
}

func (uc *UpdateCostCenterUseCase) Execute(id string, input UpdateCostCenterInput) (*costcenter.CostCenter, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCostCenterNotFound
	}
	if err := c.Update(costcenter.UpdateParams{
		Name:     input.Name,
		Type:     input.Type,
		Color:    input.Color,
		IsActive: input.IsActive,
	}); err != nil {
		return nil, err
	}
	if err := uc.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

// --- Delete ---

type DeleteCostCenterUseCase struct {
	repo costcenter.Repository
}

func NewDeleteCostCenterUseCase(repo costcenter.Repository) *DeleteCostCenterUseCase {
	return &DeleteCostCenterUseCase{repo: repo}
}

func (uc *DeleteCostCenterUseCase) Execute(id string) error {
	if err := uc.repo.Delete(id); err != nil {
		return ErrCostCenterNotFound
	}
	return nil
}
