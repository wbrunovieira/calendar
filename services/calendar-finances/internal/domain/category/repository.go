package category

// Repository defines the persistence contract for categories.
type Repository interface {
	Create(category *Category) error
	Update(category *Category) error
	FindByID(id string) (*Category, error)
	ListByProfile(profileID string) ([]*Category, error)
	Deactivate(id string) error
	// GetDescendantIDs returns all descendant category IDs for a given category (including itself)
	GetDescendantIDs(id string) ([]string, error)
}
