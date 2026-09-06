package costcenter

// Repository defines persistence operations for CostCenter
type Repository interface {
	Create(c *CostCenter) error
	FindByID(id string) (*CostCenter, error)
	FindByProfile(profileID string) ([]*CostCenter, error)
	// FindByExternalRef resolves a cost center from the id it mirrors elsewhere,
	// which is what lets a repeated sync find the same client instead of a twin.
	FindByExternalRef(profileID, source, externalID string) (*CostCenter, error)
	Update(c *CostCenter) error
	Delete(id string) error
}
