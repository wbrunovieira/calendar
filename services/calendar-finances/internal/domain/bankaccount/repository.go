package bankaccount

type DisplayOrderUpdate struct {
	ID           string
	DisplayOrder int
}

type Repository interface {
	Create(account *BankAccount) error
	FindByID(id string) (*BankAccount, error)
	FindByProfileID(profileID string) ([]*BankAccount, error)
	FindAll() ([]*BankAccount, error)
	Update(account *BankAccount) error
	Delete(id string) error
	UpdateDisplayOrders(updates []DisplayOrderUpdate) error
}
