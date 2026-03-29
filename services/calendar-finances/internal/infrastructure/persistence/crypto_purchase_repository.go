package persistence

import (
	"database/sql"

	"github.com/brunovieira/calendar-finances/internal/domain/cryptopurchase"
)

type CryptoPurchaseRepository struct {
	db *sql.DB
}

func NewCryptoPurchaseRepository(db *sql.DB) *CryptoPurchaseRepository {
	return &CryptoPurchaseRepository{db: db}
}

func (r *CryptoPurchaseRepository) Create(cp *cryptopurchase.CryptoPurchase) error {
	query := `INSERT INTO finance.crypto_purchases
		(transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at`
	return r.db.QueryRow(query,
		cp.TransactionID, cp.BankAccountID, cp.ProfileID, cp.Asset,
		cp.Quantity, cp.PriceUSD, cp.ExchangeRate, cp.InvestedBRL, cp.InvestedUSD,
		cp.OccurredOn, cp.Notes, cp.SoldQuantity, cp.Strategy,
	).Scan(&cp.ID, &cp.CreatedAt)
}

func (r *CryptoPurchaseRepository) Update(cp *cryptopurchase.CryptoPurchase) error {
	query := `UPDATE finance.crypto_purchases SET sold_quantity = $1 WHERE id = $2`
	_, err := r.db.Exec(query, cp.SoldQuantity, cp.ID)
	return err
}

func (r *CryptoPurchaseRepository) FindByProfileID(profileID string) ([]*cryptopurchase.CryptoPurchase, error) {
	return r.query(`SELECT id, transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy, created_at
		FROM finance.crypto_purchases WHERE profile_id = $1 ORDER BY occurred_on DESC`, profileID)
}

func (r *CryptoPurchaseRepository) FindByAsset(profileID, asset string) ([]*cryptopurchase.CryptoPurchase, error) {
	return r.query(`SELECT id, transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy, created_at
		FROM finance.crypto_purchases WHERE profile_id = $1 AND asset = $2 ORDER BY occurred_on DESC`, profileID, asset)
}

func (r *CryptoPurchaseRepository) FindUnsoldByAsset(profileID, asset string) ([]*cryptopurchase.CryptoPurchase, error) {
	return r.query(`SELECT id, transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy, created_at
		FROM finance.crypto_purchases WHERE profile_id = $1 AND asset = $2 AND sold_quantity < quantity ORDER BY occurred_on ASC`, profileID, asset)
}

func (r *CryptoPurchaseRepository) FindByStrategy(profileID, strategy string) ([]*cryptopurchase.CryptoPurchase, error) {
	return r.query(`SELECT id, transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy, created_at
		FROM finance.crypto_purchases WHERE profile_id = $1 AND strategy = $2 ORDER BY occurred_on DESC`, profileID, strategy)
}

func (r *CryptoPurchaseRepository) FindByBankAccountID(bankAccountID string) ([]*cryptopurchase.CryptoPurchase, error) {
	return r.query(`SELECT id, transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy, created_at
		FROM finance.crypto_purchases WHERE bank_account_id = $1 ORDER BY occurred_on DESC`, bankAccountID)
}

func (r *CryptoPurchaseRepository) FindByTransactionID(transactionID string) (*cryptopurchase.CryptoPurchase, error) {
	results, err := r.query(`SELECT id, transaction_id, bank_account_id, profile_id, asset, quantity, price_usd, exchange_rate, invested_brl, invested_usd, occurred_on, notes, sold_quantity, strategy, created_at
		FROM finance.crypto_purchases WHERE transaction_id = $1`, transactionID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

func (r *CryptoPurchaseRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM finance.crypto_purchases WHERE id = $1`, id)
	return err
}

func (r *CryptoPurchaseRepository) query(q string, args ...interface{}) ([]*cryptopurchase.CryptoPurchase, error) {
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*cryptopurchase.CryptoPurchase
	for rows.Next() {
		cp := &cryptopurchase.CryptoPurchase{}
		if err := rows.Scan(
			&cp.ID, &cp.TransactionID, &cp.BankAccountID, &cp.ProfileID,
			&cp.Asset, &cp.Quantity, &cp.PriceUSD, &cp.ExchangeRate,
			&cp.InvestedBRL, &cp.InvestedUSD, &cp.OccurredOn, &cp.Notes, &cp.SoldQuantity, &cp.Strategy, &cp.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, cp)
	}
	return results, rows.Err()
}
