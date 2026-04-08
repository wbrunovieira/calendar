package costcenter

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CenterType identifies the category of a cost center
type CenterType string

const (
	TypeClient     CenterType = "CLIENT"
	TypeProject    CenterType = "PROJECT"
	TypeDepartment CenterType = "DEPARTMENT"
)

// CostCenter groups transactions by client, project, or department
type CostCenter struct {
	ID        string     `json:"id"`
	ProfileID string     `json:"profileId"`
	Name      string     `json:"name"`
	Type      CenterType `json:"type"`
	Color     *string    `json:"color,omitempty"`
	IsActive  bool       `json:"isActive"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// CreateParams encapsulates parameters for creating a cost center
type CreateParams struct {
	ProfileID string
	Name      string
	Type      CenterType
	Color     *string
}

// UpdateParams encapsulates parameters for updating a cost center
type UpdateParams struct {
	Name     string
	Type     CenterType
	Color    *string
	IsActive bool
}

// NewCostCenter validates and creates a new CostCenter
func NewCostCenter(p CreateParams) (*CostCenter, error) {
	if strings.TrimSpace(p.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("name is required")
	}
	if !isValidType(p.Type) {
		return nil, errors.New("invalid cost center type")
	}

	now := time.Now()
	return &CostCenter{
		ID:        uuid.New().String(),
		ProfileID: p.ProfileID,
		Name:      strings.TrimSpace(p.Name),
		Type:      p.Type,
		Color:     p.Color,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update applies new values to the cost center
func (c *CostCenter) Update(p UpdateParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	if !isValidType(p.Type) {
		return errors.New("invalid cost center type")
	}
	c.Name = strings.TrimSpace(p.Name)
	c.Type = p.Type
	c.Color = p.Color
	c.IsActive = p.IsActive
	c.UpdatedAt = time.Now()
	return nil
}

func isValidType(t CenterType) bool {
	switch t {
	case TypeClient, TypeProject, TypeDepartment:
		return true
	default:
		return false
	}
}
