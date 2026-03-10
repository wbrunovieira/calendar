package goal

type Repository interface {
	Create(goal *Goal) error
	Update(goal *Goal) error
	FindByID(id string) (*Goal, error)
	ListByProfile(profileID string) ([]*Goal, error)
	Delete(id string) error
}
