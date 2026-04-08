package companyasset

// Repository defines persistence operations for CompanyAsset
type Repository interface {
	Create(a *CompanyAsset) error
	FindByID(id string) (*CompanyAsset, error)
	ListByProfile(profileID string) ([]*CompanyAsset, error)
	Update(a *CompanyAsset) error
	Delete(id string) error
}
