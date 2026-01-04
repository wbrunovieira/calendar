package persistence

import (
	"database/sql"
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type BankAccountRepository struct {
	db *sql.DB
}

func NewBankAccountRepository(db *sql.DB) *BankAccountRepository {
	return &BankAccountRepository{db: db}
}

func (r *BankAccountRepository) Create(account *bankaccount.BankAccount) error {
	query := `
		INSERT INTO finance.bank_accounts (
			id, profile_id, name, type, initial_balance, current_balance, currency, is_active,
			bank_name, bank_code, agency, account_number, account_digit, color, icon, description,
			credit_limit, due_day, closing_day, linked_account_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`
	_, err := r.db.Exec(query,
		account.ID, account.ProfileID, account.Name, account.Type,
		account.InitialBalance, account.CurrentBalance, account.Currency, account.IsActive,
		account.BankName, account.BankCode, account.Agency, account.AccountNumber,
		account.AccountDigit, account.Color, account.Icon, account.Description,
		account.CreditLimit, account.DueDay, account.ClosingDay, account.LinkedAccountID,
		account.CreatedAt, account.UpdatedAt,
	)
	return err
}

func (r *BankAccountRepository) FindByID(id string) (*bankaccount.BankAccount, error) {
	query := `
		SELECT id, profile_id, name, type, initial_balance, current_balance, currency, is_active,
			bank_name, bank_code, agency, account_number, account_digit, color, icon, description,
			credit_limit, due_day, closing_day, linked_account_id, created_at, updated_at
		FROM finance.bank_accounts
		WHERE id = $1
	`
	account := &bankaccount.BankAccount{}
	err := r.db.QueryRow(query, id).Scan(
		&account.ID, &account.ProfileID, &account.Name, &account.Type,
		&account.InitialBalance, &account.CurrentBalance, &account.Currency, &account.IsActive,
		&account.BankName, &account.BankCode, &account.Agency, &account.AccountNumber,
		&account.AccountDigit, &account.Color, &account.Icon, &account.Description,
		&account.CreditLimit, &account.DueDay, &account.ClosingDay, &account.LinkedAccountID,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("bank account not found")
	}
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (r *BankAccountRepository) FindByProfileID(profileID string) ([]*bankaccount.BankAccount, error) {
	query := `
		SELECT id, profile_id, name, type, initial_balance, current_balance, currency, is_active,
			bank_name, bank_code, agency, account_number, account_digit, color, icon, description,
			credit_limit, due_day, closing_day, linked_account_id, created_at, updated_at
		FROM finance.bank_accounts
		WHERE profile_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*bankaccount.BankAccount
	for rows.Next() {
		account := &bankaccount.BankAccount{}
		err := rows.Scan(
			&account.ID, &account.ProfileID, &account.Name, &account.Type,
			&account.InitialBalance, &account.CurrentBalance, &account.Currency, &account.IsActive,
			&account.BankName, &account.BankCode, &account.Agency, &account.AccountNumber,
			&account.AccountDigit, &account.Color, &account.Icon, &account.Description,
			&account.CreditLimit, &account.DueDay, &account.ClosingDay, &account.LinkedAccountID,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (r *BankAccountRepository) FindAll() ([]*bankaccount.BankAccount, error) {
	query := `
		SELECT id, profile_id, name, type, initial_balance, current_balance, currency, is_active,
			bank_name, bank_code, agency, account_number, account_digit, color, icon, description,
			credit_limit, due_day, closing_day, linked_account_id, created_at, updated_at
		FROM finance.bank_accounts
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*bankaccount.BankAccount
	for rows.Next() {
		account := &bankaccount.BankAccount{}
		err := rows.Scan(
			&account.ID, &account.ProfileID, &account.Name, &account.Type,
			&account.InitialBalance, &account.CurrentBalance, &account.Currency, &account.IsActive,
			&account.BankName, &account.BankCode, &account.Agency, &account.AccountNumber,
			&account.AccountDigit, &account.Color, &account.Icon, &account.Description,
			&account.CreditLimit, &account.DueDay, &account.ClosingDay, &account.LinkedAccountID,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (r *BankAccountRepository) Update(account *bankaccount.BankAccount) error {
	query := `
		UPDATE finance.bank_accounts
		SET profile_id = $2, name = $3, type = $4, current_balance = $5, currency = $6, is_active = $7,
			bank_name = $8, bank_code = $9, agency = $10, account_number = $11, account_digit = $12,
			color = $13, icon = $14, description = $15, credit_limit = $16, due_day = $17,
			closing_day = $18, linked_account_id = $19, updated_at = $20
		WHERE id = $1
	`
	result, err := r.db.Exec(query,
		account.ID, account.ProfileID, account.Name, account.Type, account.CurrentBalance, account.Currency, account.IsActive,
		account.BankName, account.BankCode, account.Agency, account.AccountNumber, account.AccountDigit,
		account.Color, account.Icon, account.Description, account.CreditLimit, account.DueDay,
		account.ClosingDay, account.LinkedAccountID, account.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("bank account not found")
	}

	return nil
}

func (r *BankAccountRepository) Delete(id string) error {
	query := `DELETE FROM finance.bank_accounts WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("bank account not found")
	}

	return nil
}
