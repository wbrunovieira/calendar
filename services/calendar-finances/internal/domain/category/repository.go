package category

// Repository defines the persistence contract for categories.
type Repository interface {
	Create(category *Category) error
	Update(category *Category) error
	FindByID(id string) (*Category, error)
	ListByProfile(profileID string) ([]*Category, error)
	Deactivate(id string) error
}
