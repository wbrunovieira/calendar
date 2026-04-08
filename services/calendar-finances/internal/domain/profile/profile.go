package profile

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProfileType represents the type of financial profile
type ProfileType string

const (
	ProfileTypePersonal ProfileType = "PERSONAL"
	ProfileTypeBusiness ProfileType = "BUSINESS"
)

// LegalEntityType identifies the juridical form of a business profile
type LegalEntityType string

const (
	LegalEntityMEI  LegalEntityType = "MEI"
	LegalEntityME   LegalEntityType = "ME"
	LegalEntityEPP  LegalEntityType = "EPP"
	LegalEntityLTDA LegalEntityType = "LTDA"
	LegalEntitySA   LegalEntityType = "SA"
)

// TaxRegime identifies the taxation regime of a business
type TaxRegime string

const (
	TaxRegimeSimples         TaxRegime = "SIMPLES"
	TaxRegimeLucroPresumido  TaxRegime = "LUCRO_PRESUMIDO"
	TaxRegimeLucroReal       TaxRegime = "LUCRO_REAL"
)

// Profile represents a financial profile entity
type Profile struct {
	ID         string      `json:"id"`
	CalendarID string      `json:"calendarId"`
	Name       string      `json:"name"`
	Type       ProfileType `json:"type"`
	IsActive   bool        `json:"isActive"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`

	// PJ-specific fields (nullable — personal profiles leave these nil)
	LegalEntityType *LegalEntityType `json:"legalEntityType,omitempty"`
	CompanyName     *string          `json:"companyName,omitempty"`
	CNPJ            *string          `json:"cnpj,omitempty"`
	SimplesNacional *bool            `json:"simplesNacional,omitempty"`
	TaxRegime       *TaxRegime       `json:"taxRegime,omitempty"`
	DasAliquota     *float64         `json:"dasAliquota,omitempty"`
	OpeningDate     *time.Time       `json:"openingDate,omitempty"`
}

// UpdateParams encapsulates all mutable profile fields
type UpdateParams struct {
	Name            string
	Type            ProfileType
	LegalEntityType *LegalEntityType
	CompanyName     *string
	CNPJ            *string
	SimplesNacional *bool
	TaxRegime       *TaxRegime
	DasAliquota     *float64
	OpeningDate     *time.Time
}

// NewProfile creates a new Profile with validation
func NewProfile(calendarID, name string, profileType ProfileType) (*Profile, error) {
	if calendarID == "" {
		return nil, errors.New("calendarID is required")
	}

	if name == "" {
		return nil, errors.New("name is required")
	}

	if profileType != ProfileTypePersonal && profileType != ProfileTypeBusiness {
		return nil, errors.New("invalid profile type")
	}

	now := time.Now()
	return &Profile{
		ID:         uuid.New().String(),
		CalendarID: calendarID,
		Name:       name,
		Type:       profileType,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Update updates the profile fields
func (p *Profile) Update(params UpdateParams) error {
	if strings.TrimSpace(params.Name) == "" {
		return errors.New("name is required")
	}

	if params.Type != ProfileTypePersonal && params.Type != ProfileTypeBusiness {
		return errors.New("invalid profile type")
	}

	if params.LegalEntityType != nil {
		if !isValidLegalEntityType(*params.LegalEntityType) {
			return errors.New("invalid legal entity type")
		}
	}

	if params.TaxRegime != nil {
		if !isValidTaxRegime(*params.TaxRegime) {
			return errors.New("invalid tax regime")
		}
	}

	if params.DasAliquota != nil && (*params.DasAliquota < 0 || *params.DasAliquota > 100) {
		return errors.New("dasAliquota must be between 0 and 100")
	}

	p.Name = strings.TrimSpace(params.Name)
	p.Type = params.Type
	p.LegalEntityType = params.LegalEntityType
	p.CompanyName = params.CompanyName
	p.CNPJ = params.CNPJ
	p.SimplesNacional = params.SimplesNacional
	p.TaxRegime = params.TaxRegime
	p.DasAliquota = params.DasAliquota
	p.OpeningDate = params.OpeningDate
	p.UpdatedAt = time.Now()
	return nil
}

// IsBusiness returns true if this is a business profile with PJ details
func (p *Profile) IsBusiness() bool {
	return p.Type == ProfileTypeBusiness
}

// IsSimples returns true if the profile uses Simples Nacional
func (p *Profile) IsSimples() bool {
	return p.SimplesNacional != nil && *p.SimplesNacional
}

// Deactivate marks the profile as inactive
func (p *Profile) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now()
}

// Activate marks the profile as active
func (p *Profile) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now()
}

func isValidLegalEntityType(t LegalEntityType) bool {
	switch t {
	case LegalEntityMEI, LegalEntityME, LegalEntityEPP, LegalEntityLTDA, LegalEntitySA:
		return true
	default:
		return false
	}
}

func isValidTaxRegime(r TaxRegime) bool {
	switch r {
	case TaxRegimeSimples, TaxRegimeLucroPresumido, TaxRegimeLucroReal:
		return true
	default:
		return false
	}
}
