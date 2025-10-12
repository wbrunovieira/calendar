package profile

// Repository defines the interface for profile persistence
type Repository interface {
	Create(profile *Profile) error
	FindByID(id string) (*Profile, error)
	FindByCalendarID(calendarID string) (*Profile, error)
	FindAll() ([]*Profile, error)
	Update(profile *Profile) error
	Delete(id string) error
}
