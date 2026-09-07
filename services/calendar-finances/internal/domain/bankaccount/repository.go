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
	// FindByProviderAccountID resolves the provider's account id to the account in
	// this system, or nothing. Nothing is an answer: importing into a guessed account
	// writes spending onto the wrong statement.
	FindByProviderAccountID(providerAccountID string) (*BankAccount, error)
	Update(account *BankAccount) error
	Delete(id string) error
	UpdateDisplayOrders(updates []DisplayOrderUpdate) error
}
