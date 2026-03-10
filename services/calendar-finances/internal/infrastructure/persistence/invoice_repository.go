package persistence

import (
	"database/sql"
	"errors"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
)

type InvoiceRepository struct {
	db *sql.DB
}

func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

func (r *InvoiceRepository) Create(inv *invoice.Invoice) error {
	query := `
		INSERT INTO finance.credit_card_invoices (
			id, bank_account_id, reference_date, opening_date, closing_date, due_date,
			amount, status, paid_at, paid_amount, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(query,
		inv.ID, inv.BankAccountID, inv.ReferenceDate, inv.OpeningDate, inv.ClosingDate, inv.DueDate,
		inv.Amount, inv.Status, inv.PaidAt, inv.PaidAmount, inv.CreatedAt, inv.UpdatedAt,
	)
	return err
}

func (r *InvoiceRepository) FindByID(id string) (*invoice.Invoice, error) {
	query := `
		SELECT id, bank_account_id, reference_date, opening_date, closing_date, due_date,
			amount, status, paid_at, paid_amount, created_at, updated_at
		FROM finance.credit_card_invoices
		WHERE id = $1
	`
	inv := &invoice.Invoice{}
	err := r.db.QueryRow(query, id).Scan(
		&inv.ID, &inv.BankAccountID, &inv.ReferenceDate, &inv.OpeningDate, &inv.ClosingDate, &inv.DueDate,
		&inv.Amount, &inv.Status, &inv.PaidAt, &inv.PaidAmount, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("invoice not found")
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *InvoiceRepository) FindByBankAccountID(bankAccountID string) ([]*invoice.Invoice, error) {
	query := `
		SELECT id, bank_account_id, reference_date, opening_date, closing_date, due_date,
			amount, status, paid_at, paid_amount, created_at, updated_at
		FROM finance.credit_card_invoices
		WHERE bank_account_id = $1
		ORDER BY reference_date DESC
	`
	rows, err := r.db.Query(query, bankAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*invoice.Invoice
	for rows.Next() {
		inv := &invoice.Invoice{}
		err := rows.Scan(
			&inv.ID, &inv.BankAccountID, &inv.ReferenceDate, &inv.OpeningDate, &inv.ClosingDate, &inv.DueDate,
			&inv.Amount, &inv.Status, &inv.PaidAt, &inv.PaidAmount, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *InvoiceRepository) FindOpenByBankAccountID(bankAccountID string) (*invoice.Invoice, error) {
	query := `
		SELECT id, bank_account_id, reference_date, opening_date, closing_date, due_date,
			amount, status, paid_at, paid_amount, created_at, updated_at
		FROM finance.credit_card_invoices
		WHERE bank_account_id = $1 AND status = 'OPEN'
		ORDER BY reference_date DESC
		LIMIT 1
	`
	inv := &invoice.Invoice{}
	err := r.db.QueryRow(query, bankAccountID).Scan(
		&inv.ID, &inv.BankAccountID, &inv.ReferenceDate, &inv.OpeningDate, &inv.ClosingDate, &inv.DueDate,
		&inv.Amount, &inv.Status, &inv.PaidAt, &inv.PaidAmount, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No open invoice found is not an error
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// FindByBankAccountAndDate finds the invoice that contains a given transaction date
func (r *InvoiceRepository) FindByBankAccountAndDate(bankAccountID string, txDate time.Time) (*invoice.Invoice, error) {
	// Normalize to date only
	txDateOnly := time.Date(txDate.Year(), txDate.Month(), txDate.Day(), 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, bank_account_id, reference_date, opening_date, closing_date, due_date,
			amount, status, paid_at, paid_amount, created_at, updated_at
		FROM finance.credit_card_invoices
		WHERE bank_account_id = $1
			AND opening_date <= $2
			AND closing_date >= $2
		LIMIT 1
	`
	inv := &invoice.Invoice{}
	err := r.db.QueryRow(query, bankAccountID, txDateOnly).Scan(
		&inv.ID, &inv.BankAccountID, &inv.ReferenceDate, &inv.OpeningDate, &inv.ClosingDate, &inv.DueDate,
		&inv.Amount, &inv.Status, &inv.PaidAt, &inv.PaidAmount, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Invoice not found for this date
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *InvoiceRepository) Update(inv *invoice.Invoice) error {
	query := `
		UPDATE finance.credit_card_invoices
		SET bank_account_id = $2, reference_date = $3, opening_date = $4, closing_date = $5, due_date = $6,
			amount = $7, status = $8, paid_at = $9, paid_amount = $10, updated_at = $11
		WHERE id = $1
	`
	result, err := r.db.Exec(query,
		inv.ID, inv.BankAccountID, inv.ReferenceDate, inv.OpeningDate, inv.ClosingDate, inv.DueDate,
		inv.Amount, inv.Status, inv.PaidAt, inv.PaidAmount, inv.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("invoice not found")
	}

	return nil
}

func (r *InvoiceRepository) FindOpenPastClosingDate(now time.Time) ([]*invoice.Invoice, error) {
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	query := `
		SELECT id, bank_account_id, reference_date, opening_date, closing_date, due_date,
			amount, status, paid_at, paid_amount, created_at, updated_at
		FROM finance.credit_card_invoices
		WHERE status = 'OPEN' AND closing_date < $1
		ORDER BY closing_date ASC
	`
	rows, err := r.db.Query(query, nowDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*invoice.Invoice
	for rows.Next() {
		inv := &invoice.Invoice{}
		err := rows.Scan(
			&inv.ID, &inv.BankAccountID, &inv.ReferenceDate, &inv.OpeningDate, &inv.ClosingDate, &inv.DueDate,
			&inv.Amount, &inv.Status, &inv.PaidAt, &inv.PaidAmount, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *InvoiceRepository) Delete(id string) error {
	query := `DELETE FROM finance.credit_card_invoices WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("invoice not found")
	}

	return nil
}
