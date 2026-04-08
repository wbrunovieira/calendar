package capitalcontribution

// Repository defines persistence operations for CapitalContribution
type Repository interface {
	Create(c *CapitalContribution) error
	FindByID(id string) (*CapitalContribution, error)
	ListByProfile(profileID string) ([]*CapitalContribution, error)
	Update(c *CapitalContribution) error
	Delete(id string) error
}
