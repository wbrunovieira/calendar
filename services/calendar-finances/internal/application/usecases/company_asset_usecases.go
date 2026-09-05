package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/companyasset"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

// --- Create ---

type CreateCompanyAssetInput struct {
	ProfileID           string                     `json:"profileId"`
	Name                string                     `json:"name"`
	Category            companyasset.AssetCategory `json:"category"`
	PurchaseDate        time.Time                  `json:"purchaseDate"`
	PurchaseAmount      float64                    `json:"purchaseAmount"`
	DepreciationRate    float64                    `json:"depreciationRate"`
	LinkedTransactionID *string                    `json:"linkedTransactionId,omitempty"`
	Notes               *string                    `json:"notes,omitempty"`
}

type CreateCompanyAssetUseCase struct {
	repo        companyasset.Repository
	profileRepo profile.Repository
}

func NewCreateCompanyAssetUseCase(repo companyasset.Repository, profileRepo profile.Repository) *CreateCompanyAssetUseCase {
	return &CreateCompanyAssetUseCase{repo: repo, profileRepo: profileRepo}
}

func (uc *CreateCompanyAssetUseCase) Execute(input CreateCompanyAssetInput) (*companyasset.CompanyAsset, error) {
	if _, err := uc.profileRepo.FindByID(input.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}
	a, err := companyasset.NewCompanyAsset(companyasset.CreateParams{
		ProfileID:           input.ProfileID,
		Name:                input.Name,
		Category:            input.Category,
		PurchaseDate:        input.PurchaseDate,
		PurchaseAmount:      input.PurchaseAmount,
		DepreciationRate:    input.DepreciationRate,
		LinkedTransactionID: input.LinkedTransactionID,
		Notes:               input.Notes,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Create(a); err != nil {
		return nil, err
	}
	return a, nil
}

// --- List ---

type ListCompanyAssetsUseCase struct {
	repo companyasset.Repository
}

func NewListCompanyAssetsUseCase(repo companyasset.Repository) *ListCompanyAssetsUseCase {
	return &ListCompanyAssetsUseCase{repo: repo}
}

func (uc *ListCompanyAssetsUseCase) Execute(profileID string) ([]*companyasset.CompanyAsset, error) {
	return uc.repo.ListByProfile(profileID)
}

// --- Get ---

type GetCompanyAssetUseCase struct {
	repo companyasset.Repository
}

func NewGetCompanyAssetUseCase(repo companyasset.Repository) *GetCompanyAssetUseCase {
	return &GetCompanyAssetUseCase{repo: repo}
}

func (uc *GetCompanyAssetUseCase) Execute(id string) (*companyasset.CompanyAsset, error) {
	a, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCompanyAssetNotFound
	}
	return a, nil
}

// --- Update ---

type UpdateCompanyAssetInput struct {
	Name             string                     `json:"name"`
	Category         companyasset.AssetCategory `json:"category"`
	PurchaseDate     time.Time                  `json:"purchaseDate"`
	PurchaseAmount   float64                    `json:"purchaseAmount"`
	CurrentValue     float64                    `json:"currentValue"`
	DepreciationRate float64                    `json:"depreciationRate"`
	Notes            *string                    `json:"notes,omitempty"`
	IsActive         bool                       `json:"isActive"`
	DisposalDate     *time.Time                 `json:"disposalDate,omitempty"`
	DisposalAmount   *float64                   `json:"disposalAmount,omitempty"`
}

type UpdateCompanyAssetUseCase struct {
	repo companyasset.Repository
}

func NewUpdateCompanyAssetUseCase(repo companyasset.Repository) *UpdateCompanyAssetUseCase {
	return &UpdateCompanyAssetUseCase{repo: repo}
}

func (uc *UpdateCompanyAssetUseCase) Execute(id string, input UpdateCompanyAssetInput) (*companyasset.CompanyAsset, error) {
	a, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCompanyAssetNotFound
	}
	a.Name = input.Name
	a.Category = input.Category
	a.PurchaseDate = input.PurchaseDate
	a.PurchaseAmount = input.PurchaseAmount
	a.CurrentValue = input.CurrentValue
	a.DepreciationRate = input.DepreciationRate
	a.Notes = input.Notes
	a.IsActive = input.IsActive
	a.DisposalDate = input.DisposalDate
	a.DisposalAmount = input.DisposalAmount
	if err := uc.repo.Update(a); err != nil {
		return nil, err
	}
	return a, nil
}

// --- Delete ---

type DeleteCompanyAssetUseCase struct {
	repo companyasset.Repository
}

func NewDeleteCompanyAssetUseCase(repo companyasset.Repository) *DeleteCompanyAssetUseCase {
	return &DeleteCompanyAssetUseCase{repo: repo}
}

func (uc *DeleteCompanyAssetUseCase) Execute(id string) error {
	if err := uc.repo.Delete(id); err != nil {
		return ErrCompanyAssetNotFound
	}
	return nil
}
