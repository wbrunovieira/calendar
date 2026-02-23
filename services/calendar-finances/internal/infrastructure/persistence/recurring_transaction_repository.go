package persistence

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
)

type RecurringTransactionRepository struct {
	db *sql.DB
}

func NewRecurringTransactionRepository(db *sql.DB) *RecurringTransactionRepository {
	return &RecurringTransactionRepository{db: db}
}

func (r *RecurringTransactionRepository) Create(entity *recurringtransaction.RecurringTransaction) error {
	query := `
		INSERT INTO finance.recurring_transactions (
			id, profile_id, bank_account_id, category_id, type, amount, currency, description,
			recurrence_rule, start_on, end_on, next_occurrence, status, review_on, notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	_, err := r.db.Exec(query,
		entity.ID,
		entity.ProfileID,
		nullableStringPtr(entity.BankAccountID),
		nullableStringPtr(entity.CategoryID),
		entity.Type,
		entity.Amount,
		entity.Currency,
		nullableText(entity.Description),
		entity.RecurrenceRule,
		entity.StartOn,
		nullableTimePtr(entity.EndOn),
		entity.NextOccurrence,
		entity.Status,
		nullableTimePtr(entity.ReviewOn),
		nullableString(entity.Notes),
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	return err
}

func (r *RecurringTransactionRepository) Update(entity *recurringtransaction.RecurringTransaction) error {
	query := `
		UPDATE finance.recurring_transactions
		SET bank_account_id = $2,
			category_id = $3,
			type = $4,
			amount = $5,
			currency = $6,
			description = $7,
			recurrence_rule = $8,
			start_on = $9,
			end_on = $10,
			next_occurrence = $11,
			status = $12,
			review_on = $13,
			notes = $14,
			updated_at = $15
		WHERE id = $1
	`

	result, err := r.db.Exec(query,
		entity.ID,
		nullableStringPtr(entity.BankAccountID),
		nullableStringPtr(entity.CategoryID),
		entity.Type,
		entity.Amount,
		entity.Currency,
		nullableText(entity.Description),
		entity.RecurrenceRule,
		entity.StartOn,
		nullableTimePtr(entity.EndOn),
		entity.NextOccurrence,
		entity.Status,
		nullableTimePtr(entity.ReviewOn),
		nullableString(entity.Notes),
		entity.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("recurring transaction not found")
	}
	return nil
}

func (r *RecurringTransactionRepository) FindByID(id string) (*recurringtransaction.RecurringTransaction, error) {
	query := `
		SELECT id, profile_id, bank_account_id, category_id, type, amount, currency, description,
			recurrence_rule, start_on, end_on, next_occurrence, status, review_on, notes, created_at, updated_at
		FROM finance.recurring_transactions
		WHERE id = $1
	`

	row := r.db.QueryRow(query, id)
	return scanRecurring(row)
}

func (r *RecurringTransactionRepository) ListByProfile(profileID string) ([]*recurringtransaction.RecurringTransaction, error) {
	query := `
		SELECT id, profile_id, bank_account_id, category_id, type, amount, currency, description,
			recurrence_rule, start_on, end_on, next_occurrence, status, review_on, notes, created_at, updated_at
		FROM finance.recurring_transactions
		WHERE profile_id = $1
		ORDER BY next_occurrence ASC
	`

	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*recurringtransaction.RecurringTransaction
	for rows.Next() {
		item, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *RecurringTransactionRepository) FindDueRecurring(before time.Time) ([]*recurringtransaction.RecurringTransaction, error) {
	query := `
		SELECT id, profile_id, bank_account_id, category_id, type, amount, currency, description,
			recurrence_rule, start_on, end_on, next_occurrence, status, review_on, notes, created_at, updated_at
		FROM finance.recurring_transactions
		WHERE status = 'ACTIVE' AND next_occurrence <= $1
		ORDER BY next_occurrence ASC
	`

	rows, err := r.db.Query(query, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*recurringtransaction.RecurringTransaction
	for rows.Next() {
		item, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *RecurringTransactionRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.recurring_transactions WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("recurring transaction not found")
	}
	return nil
}

type recurringScanner interface {
	Scan(dest ...any) error
}

func scanRecurring(scanner recurringScanner) (*recurringtransaction.RecurringTransaction, error) {
	entity := &recurringtransaction.RecurringTransaction{}
	var (
		bankAccount sql.NullString
		category    sql.NullString
		description sql.NullString
		endOn       sql.NullTime
		reviewOn    sql.NullTime
		notes       sql.NullString
	)

	err := scanner.Scan(
		&entity.ID,
		&entity.ProfileID,
		&bankAccount,
		&category,
		&entity.Type,
		&entity.Amount,
		&entity.Currency,
		&description,
		&entity.RecurrenceRule,
		&entity.StartOn,
		&endOn,
		&entity.NextOccurrence,
		&entity.Status,
		&reviewOn,
		&notes,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("recurring transaction not found")
		}
		return nil, err
	}

	if bankAccount.Valid {
		value := bankAccount.String
		entity.BankAccountID = &value
	}
	if category.Valid {
		value := category.String
		entity.CategoryID = &value
	}
	if description.Valid {
		entity.Description = description.String
	}
	if endOn.Valid {
		value := endOn.Time
		entity.EndOn = &value
	}
	if reviewOn.Valid {
		value := reviewOn.Time
		entity.ReviewOn = &value
	}
	if notes.Valid {
		value := notes.String
		entity.Notes = &value
	}

	return entity, nil
}

func nullableStringPtr(value *string) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableText(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableTimePtr(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
