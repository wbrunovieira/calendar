package category

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Type represents the financial direction associated with a category.
type Type string

const (
	TypeIncome   Type = "INCOME"
	TypeExpense  Type = "EXPENSE"
	TypeTransfer Type = "TRANSFER"
)

// ClassificationDRE identifies where a category fits in the simplified P&L statement.
// Only relevant for business profiles; personal profiles leave this nil.
type ClassificationDRE string

const (
	DRERevenue         ClassificationDRE = "REVENUE"          // Receita bruta
	DRETax             ClassificationDRE = "TAX"               // Impostos e deduções (DAS, ISS…)
	DREFixedCost       ClassificationDRE = "FIXED_COST"        // Custo fixo operacional
	DREVariableCost    ClassificationDRE = "VARIABLE_COST"     // Custo variável
	DREProlabore       ClassificationDRE = "PROLABORE"         // Retirada do sócio
	DREMarketing       ClassificationDRE = "MARKETING"         // Investimento em marketing/ads
	DREFinancial       ClassificationDRE = "FINANCIAL"         // Rendimentos, juros, IOF
	DREAsset           ClassificationDRE = "ASSET"             // Compra de ativo permanente
	DRECapital         ClassificationDRE = "CAPITAL"           // Aporte de capital
)

// Category defines a spending or income bucket scoped to a financial profile.
type Category struct {
	ID                string             `json:"id"`
	ProfileID         string             `json:"profileId"`
	Name              string             `json:"name"`
	Type              Type               `json:"type"`
	Color             *string            `json:"color,omitempty"`
	Icon              *string            `json:"icon,omitempty"`
	ParentID          *string            `json:"parentId,omitempty"`
	ClassificationDRE *ClassificationDRE `json:"classificationDRE,omitempty"`
	IsActive          bool               `json:"isActive"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CreateParams encapsulates the data required to instantiate a new category.
type CreateParams struct {
	ProfileID         string
	Name              string
	Type              Type
	Color             *string
	Icon              *string
	ParentID          *string
	ClassificationDRE *ClassificationDRE
}

// NewCategory validates the input and returns a new active Category instance.
func NewCategory(params CreateParams) (*Category, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	if err := validateType(params.Type); err != nil {
		return nil, err
	}

	if err := validateColor(params.Color); err != nil {
		return nil, err
	}

	if err := validateClassificationDRE(params.ClassificationDRE); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Category{
		ID:                uuid.New().String(),
		ProfileID:         params.ProfileID,
		Name:              name,
		Type:              params.Type,
		Color:             clonePtr(params.Color),
		Icon:              clonePtr(params.Icon),
		ParentID:          clonePtr(params.ParentID),
		ClassificationDRE: params.ClassificationDRE,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// Update applies new values to the category while keeping validation consistent.
func (c *Category) Update(name string, categoryType Type, color, icon, parentID *string, classificationDRE *ClassificationDRE) error {
	if c == nil {
		return errors.New("category is nil")
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("name is required")
	}

	if err := validateType(categoryType); err != nil {
		return err
	}

	if err := validateColor(color); err != nil {
		return err
	}

	if err := validateClassificationDRE(classificationDRE); err != nil {
		return err
	}

	c.Name = trimmed
	c.Type = categoryType
	c.Color = clonePtr(color)
	c.Icon = clonePtr(icon)
	c.ParentID = clonePtr(parentID)
	c.ClassificationDRE = classificationDRE
	c.UpdatedAt = time.Now()
	return nil
}

// Deactivate performs a soft-delete style operation.
func (c *Category) Deactivate() {
	if c == nil {
		return
	}
	c.IsActive = false
	c.UpdatedAt = time.Now()
}

// Activate marks the category as available again.
func (c *Category) Activate() {
	if c == nil {
		return
	}
	c.IsActive = true
	c.UpdatedAt = time.Now()
}

func validateType(t Type) error {
	switch t {
	case TypeIncome, TypeExpense, TypeTransfer:
		return nil
	default:
		return fmt.Errorf("invalid category type: %s", t)
	}
}

func validateClassificationDRE(c *ClassificationDRE) error {
	if c == nil {
		return nil
	}
	switch *c {
	case DRERevenue, DRETax, DREFixedCost, DREVariableCost,
		DREProlabore, DREMarketing, DREFinancial, DREAsset, DRECapital:
		return nil
	default:
		return fmt.Errorf("invalid classificationDRE: %s", *c)
	}
}

func validateColor(color *string) error {
	if color == nil || strings.TrimSpace(*color) == "" {
		return nil
	}

	value := strings.TrimSpace(*color)
	if len(value) != 7 || !strings.HasPrefix(value, "#") {
		return errors.New("color must be in hex format, e.g. #12ABCD")
	}
	return nil
}

func clonePtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}
