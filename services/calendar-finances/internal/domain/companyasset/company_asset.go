package companyasset

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AssetCategory classifies the type of company asset
type AssetCategory string

const (
	AssetCategoryHardware   AssetCategory = "HARDWARE"    // Computadores, servidores, periféricos
	AssetCategorySoftware   AssetCategory = "SOFTWARE"    // Licenças, sistemas
	AssetCategoryFurniture  AssetCategory = "FURNITURE"   // Móveis e utensílios
	AssetCategoryVehicle    AssetCategory = "VEHICLE"     // Veículos
	AssetCategoryRealEstate AssetCategory = "REAL_ESTATE" // Imóveis
	AssetCategoryOther      AssetCategory = "OTHER"       // Outros
)

// CompanyAsset represents a fixed asset owned by the company
type CompanyAsset struct {
	ID                  string        `json:"id"`
	ProfileID           string        `json:"profileId"`
	Name                string        `json:"name"`
	Category            AssetCategory `json:"category"`
	PurchaseDate        time.Time     `json:"purchaseDate"`
	PurchaseAmount      float64       `json:"purchaseAmount"`
	CurrentValue        float64       `json:"currentValue"`
	DepreciationRate    float64       `json:"depreciationRate"` // % per year (0 = no depreciation)
	LinkedTransactionID *string       `json:"linkedTransactionId,omitempty"`
	Notes               *string       `json:"notes,omitempty"`
	IsActive            bool          `json:"isActive"`
	DisposalDate        *time.Time    `json:"disposalDate,omitempty"`
	DisposalAmount      *float64      `json:"disposalAmount,omitempty"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
}

// CreateParams encapsulates parameters for creating a company asset
type CreateParams struct {
	ProfileID           string
	Name                string
	Category            AssetCategory
	PurchaseDate        time.Time
	PurchaseAmount      float64
	DepreciationRate    float64
	LinkedTransactionID *string
	Notes               *string
}

// NewCompanyAsset validates and creates a new asset record
func NewCompanyAsset(params CreateParams) (*CompanyAsset, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}
	if strings.TrimSpace(params.Name) == "" {
		return nil, errors.New("name is required")
	}
	if !isValidCategory(params.Category) {
		return nil, errors.New("invalid asset category")
	}
	if params.PurchaseAmount <= 0 {
		return nil, errors.New("purchaseAmount must be greater than zero")
	}
	if params.PurchaseDate.IsZero() {
		return nil, errors.New("purchaseDate is required")
	}
	if params.DepreciationRate < 0 || params.DepreciationRate > 100 {
		return nil, errors.New("depreciationRate must be between 0 and 100")
	}

	now := time.Now()
	return &CompanyAsset{
		ID:                  uuid.New().String(),
		ProfileID:           params.ProfileID,
		Name:                strings.TrimSpace(params.Name),
		Category:            params.Category,
		PurchaseDate:        params.PurchaseDate,
		PurchaseAmount:      params.PurchaseAmount,
		CurrentValue:        params.PurchaseAmount, // starts at purchase amount
		DepreciationRate:    params.DepreciationRate,
		LinkedTransactionID: params.LinkedTransactionID,
		Notes:               params.Notes,
		IsActive:            true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// EstimateCurrentValue returns the estimated value based on straight-line depreciation
func (a *CompanyAsset) EstimateCurrentValue() float64 {
	if a.DepreciationRate <= 0 {
		return a.PurchaseAmount
	}
	yearsElapsed := time.Since(a.PurchaseDate).Hours() / (24 * 365.25)
	totalDepreciation := a.PurchaseAmount * (a.DepreciationRate / 100) * yearsElapsed
	estimated := a.PurchaseAmount - totalDepreciation
	if estimated < 0 {
		return 0
	}
	return estimated
}

// Dispose marks the asset as disposed (sold, discarded, donated)
func (a *CompanyAsset) Dispose(disposalDate time.Time, disposalAmount float64) error {
	if disposalDate.IsZero() {
		return errors.New("disposalDate is required")
	}
	a.IsActive = false
	a.DisposalDate = &disposalDate
	a.DisposalAmount = &disposalAmount
	a.UpdatedAt = time.Now()
	return nil
}

// UpdateCurrentValue manually updates the current value
func (a *CompanyAsset) UpdateCurrentValue(value float64) error {
	if value < 0 {
		return errors.New("currentValue cannot be negative")
	}
	a.CurrentValue = value
	a.UpdatedAt = time.Now()
	return nil
}

func isValidCategory(c AssetCategory) bool {
	switch c {
	case AssetCategoryHardware, AssetCategorySoftware, AssetCategoryFurniture,
		AssetCategoryVehicle, AssetCategoryRealEstate, AssetCategoryOther:
		return true
	default:
		return false
	}
}
