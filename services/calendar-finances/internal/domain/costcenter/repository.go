package costcenter

// Repository defines persistence operations for CostCenter
type Repository interface {
	Create(c *CostCenter) error
	FindByID(id string) (*CostCenter, error)
	FindByProfile(profileID string) ([]*CostCenter, error)
	Update(c *CostCenter) error
	Delete(id string) error
}
