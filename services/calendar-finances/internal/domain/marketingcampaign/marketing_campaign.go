package marketingcampaign

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Platform represents the advertising platform for a campaign
type Platform string

const (
	PlatformMetaAds   Platform = "META_ADS"
	PlatformGoogleAds Platform = "GOOGLE_ADS"
	PlatformInstagram Platform = "INSTAGRAM"
	PlatformLinkedIn  Platform = "LINKEDIN"
	PlatformOther     Platform = "OTHER"
)

// MarketingCampaign tracks advertising campaigns and their performance metrics
type MarketingCampaign struct {
	ID                string     `json:"id"`
	ProfileID         string     `json:"profileId"`
	Name              string     `json:"name"`
	Platform          Platform   `json:"platform"`
	StartDate         time.Time  `json:"startDate"`
	EndDate           *time.Time `json:"endDate,omitempty"`
	Budget            float64    `json:"budget"`
	RevenueAttributed float64    `json:"revenueAttributed"`
	Leads             int        `json:"leads"`
	Conversions       int        `json:"conversions"`
	Notes             *string    `json:"notes,omitempty"`
	IsActive          bool       `json:"isActive"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// CreateParams encapsulates parameters for creating a marketing campaign
type CreateParams struct {
	ProfileID string
	Name      string
	Platform  Platform
	StartDate time.Time
	Budget    float64
	EndDate   *time.Time
	Notes     *string
}

// UpdateParams encapsulates parameters for updating a marketing campaign
type UpdateParams struct {
	Name              string
	Platform          Platform
	StartDate         time.Time
	EndDate           *time.Time
	Budget            float64
	RevenueAttributed float64
	Leads             int
	Conversions       int
	Notes             *string
	IsActive          bool
}

// NewMarketingCampaign validates and creates a new MarketingCampaign
func NewMarketingCampaign(p CreateParams) (*MarketingCampaign, error) {
	if strings.TrimSpace(p.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("name is required")
	}
	if !isValidPlatform(p.Platform) {
		return nil, errors.New("invalid platform")
	}
	if p.StartDate.IsZero() {
		return nil, errors.New("startDate is required")
	}
	if p.Budget < 0 {
		return nil, errors.New("budget must be non-negative")
	}

	now := time.Now()
	return &MarketingCampaign{
		ID:                uuid.New().String(),
		ProfileID:         p.ProfileID,
		Name:              strings.TrimSpace(p.Name),
		Platform:          p.Platform,
		StartDate:         p.StartDate,
		EndDate:           p.EndDate,
		Budget:            p.Budget,
		RevenueAttributed: 0,
		Leads:             0,
		Conversions:       0,
		Notes:             p.Notes,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// Update applies new values to the campaign
func (c *MarketingCampaign) Update(p UpdateParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	if !isValidPlatform(p.Platform) {
		return errors.New("invalid platform")
	}
	if p.StartDate.IsZero() {
		return errors.New("startDate is required")
	}
	c.Name = strings.TrimSpace(p.Name)
	c.Platform = p.Platform
	c.StartDate = p.StartDate
	c.EndDate = p.EndDate
	c.Budget = p.Budget
	c.RevenueAttributed = p.RevenueAttributed
	c.Leads = p.Leads
	c.Conversions = p.Conversions
	c.Notes = p.Notes
	c.IsActive = p.IsActive
	c.UpdatedAt = time.Now()
	return nil
}

// ROI returns the return on investment percentage: (revenue - spent) / spent * 100
// Returns 0 if totalSpent is zero to avoid division by zero
func (c *MarketingCampaign) ROI(totalSpent float64) float64 {
	if totalSpent == 0 {
		return 0
	}
	return (c.RevenueAttributed - totalSpent) / totalSpent * 100
}

// CostPerLead returns the average cost per lead: spent / leads
// Returns 0 if leads is zero to avoid division by zero
func (c *MarketingCampaign) CostPerLead(totalSpent float64) float64 {
	if c.Leads == 0 {
		return 0
	}
	return totalSpent / float64(c.Leads)
}

// CostPerConversion returns the average cost per conversion: spent / conversions
// Returns 0 if conversions is zero to avoid division by zero
func (c *MarketingCampaign) CostPerConversion(totalSpent float64) float64 {
	if c.Conversions == 0 {
		return 0
	}
	return totalSpent / float64(c.Conversions)
}

func isValidPlatform(p Platform) bool {
	switch p {
	case PlatformMetaAds, PlatformGoogleAds, PlatformInstagram, PlatformLinkedIn, PlatformOther:
		return true
	default:
		return false
	}
}
