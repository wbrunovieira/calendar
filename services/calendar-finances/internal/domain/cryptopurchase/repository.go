package cryptopurchase

type Repository interface {
	Create(purchase *CryptoPurchase) error
	Update(purchase *CryptoPurchase) error
	FindByProfileID(profileID string) ([]*CryptoPurchase, error)
	FindByAsset(profileID, asset string) ([]*CryptoPurchase, error)
	FindUnsoldByAsset(profileID, asset string) ([]*CryptoPurchase, error)
	FindByStrategy(profileID, strategy string) ([]*CryptoPurchase, error)
	FindByBankAccountID(bankAccountID string) ([]*CryptoPurchase, error)
	FindByTransactionID(transactionID string) (*CryptoPurchase, error)
	Delete(id string) error
}
