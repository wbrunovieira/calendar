package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/marketingcampaign"
)

// --- Create ---

type CreateCampaignInput struct {
	ProfileID string                    `json:"profileId"`
	Name      string                    `json:"name"`
	Platform  marketingcampaign.Platform `json:"platform"`
	StartDate time.Time                 `json:"startDate"`
	Budget    float64                   `json:"budget"`
	EndDate   *time.Time                `json:"endDate,omitempty"`
	Notes     *string                   `json:"notes,omitempty"`
}

type CreateCampaignUseCase struct {
	repo marketingcampaign.Repository
}

func NewCreateCampaignUseCase(repo marketingcampaign.Repository) *CreateCampaignUseCase {
	return &CreateCampaignUseCase{repo: repo}
}

func (uc *CreateCampaignUseCase) Execute(input CreateCampaignInput) (*marketingcampaign.MarketingCampaign, error) {
	c, err := marketingcampaign.NewMarketingCampaign(marketingcampaign.CreateParams{
		ProfileID: input.ProfileID,
		Name:      input.Name,
		Platform:  input.Platform,
		StartDate: input.StartDate,
		Budget:    input.Budget,
		EndDate:   input.EndDate,
		Notes:     input.Notes,
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

type ListCampaignsUseCase struct {
	repo marketingcampaign.Repository
}

func NewListCampaignsUseCase(repo marketingcampaign.Repository) *ListCampaignsUseCase {
	return &ListCampaignsUseCase{repo: repo}
}

func (uc *ListCampaignsUseCase) Execute(profileID string) ([]*marketingcampaign.MarketingCampaign, error) {
	return uc.repo.FindByProfile(profileID)
}

// --- Get ---

type GetCampaignUseCase struct {
	repo marketingcampaign.Repository
}

func NewGetCampaignUseCase(repo marketingcampaign.Repository) *GetCampaignUseCase {
	return &GetCampaignUseCase{repo: repo}
}

func (uc *GetCampaignUseCase) Execute(id string) (*marketingcampaign.MarketingCampaign, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrMarketingCampaignNotFound
	}
	return c, nil
}

// --- Update ---

type UpdateCampaignInput struct {
	Name              string                    `json:"name"`
	Platform          marketingcampaign.Platform `json:"platform"`
	StartDate         time.Time                 `json:"startDate"`
	EndDate           *time.Time                `json:"endDate,omitempty"`
	Budget            float64                   `json:"budget"`
	RevenueAttributed float64                   `json:"revenueAttributed"`
	Leads             int                       `json:"leads"`
	Conversions       int                       `json:"conversions"`
	Notes             *string                   `json:"notes,omitempty"`
	IsActive          bool                      `json:"isActive"`
}

type UpdateCampaignUseCase struct {
	repo marketingcampaign.Repository
}

func NewUpdateCampaignUseCase(repo marketingcampaign.Repository) *UpdateCampaignUseCase {
	return &UpdateCampaignUseCase{repo: repo}
}

func (uc *UpdateCampaignUseCase) Execute(id string, input UpdateCampaignInput) (*marketingcampaign.MarketingCampaign, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrMarketingCampaignNotFound
	}
	if err := c.Update(marketingcampaign.UpdateParams{
		Name:              input.Name,
		Platform:          input.Platform,
		StartDate:         input.StartDate,
		EndDate:           input.EndDate,
		Budget:            input.Budget,
		RevenueAttributed: input.RevenueAttributed,
		Leads:             input.Leads,
		Conversions:       input.Conversions,
		Notes:             input.Notes,
		IsActive:          input.IsActive,
	}); err != nil {
		return nil, err
	}
	if err := uc.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

// --- Delete ---

type DeleteCampaignUseCase struct {
	repo marketingcampaign.Repository
}

func NewDeleteCampaignUseCase(repo marketingcampaign.Repository) *DeleteCampaignUseCase {
	return &DeleteCampaignUseCase{repo: repo}
}

func (uc *DeleteCampaignUseCase) Execute(id string) error {
	if err := uc.repo.Delete(id); err != nil {
		return ErrMarketingCampaignNotFound
	}
	return nil
}

// --- GetWithMetrics ---

// CampaignMetrics enriches a campaign with computed financial metrics
type CampaignMetrics struct {
	*marketingcampaign.MarketingCampaign
	TotalSpent        float64 `json:"totalSpent"`
	ROI               float64 `json:"roi"`
	CostPerLead       float64 `json:"costPerLead"`
	CostPerConversion float64 `json:"costPerConversion"`
}

type GetCampaignWithMetricsUseCase struct {
	repo marketingcampaign.Repository
}

func NewGetCampaignWithMetricsUseCase(repo marketingcampaign.Repository) *GetCampaignWithMetricsUseCase {
	return &GetCampaignWithMetricsUseCase{repo: repo}
}

func (uc *GetCampaignWithMetricsUseCase) Execute(id string) (*CampaignMetrics, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrMarketingCampaignNotFound
	}
	totalSpent, err := uc.repo.GetTotalSpent(id)
	if err != nil {
		return nil, err
	}
	return &CampaignMetrics{
		MarketingCampaign: c,
		TotalSpent:        totalSpent,
		ROI:               c.ROI(totalSpent),
		CostPerLead:       c.CostPerLead(totalSpent),
		CostPerConversion: c.CostPerConversion(totalSpent),
	}, nil
}
