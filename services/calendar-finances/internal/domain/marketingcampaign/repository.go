package marketingcampaign

// Repository defines persistence operations for MarketingCampaign
type Repository interface {
	Create(c *MarketingCampaign) error
	FindByID(id string) (*MarketingCampaign, error)
	FindByProfile(profileID string) ([]*MarketingCampaign, error)
	Update(c *MarketingCampaign) error
	Delete(id string) error
	GetTotalSpent(campaignID string) (float64, error)
}
