package cryptopurchase

type Repository interface {
	Create(purchase *CryptoPurchase) error
	FindByProfileID(profileID string) ([]*CryptoPurchase, error)
	FindByAsset(profileID, asset string) ([]*CryptoPurchase, error)
	FindByBankAccountID(bankAccountID string) ([]*CryptoPurchase, error)
	FindByTransactionID(transactionID string) (*CryptoPurchase, error)
	Delete(id string) error
}
