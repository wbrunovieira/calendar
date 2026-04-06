package goal

type DisplayOrderUpdate struct {
	ID           string
	DisplayOrder int
}

type Repository interface {
	Create(goal *Goal) error
	Update(goal *Goal) error
	FindByID(id string) (*Goal, error)
	ListByProfile(profileID string) ([]*Goal, error)
	Delete(id string) error
	UpdateDisplayOrders(updates []DisplayOrderUpdate) error
}
