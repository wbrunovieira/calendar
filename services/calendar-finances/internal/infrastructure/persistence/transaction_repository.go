package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(txn *transaction.Transaction) (err error) {
	if txn == nil {
		return errors.New("transaction is nil")
	}
	if txn.ID == "" {
		txn.ID = uuid.New().String()
	}
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = time.Now()
	}
	if txn.UpdatedAt.IsZero() {
		txn.UpdatedAt = txn.CreatedAt
	}

	sqlTx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = sqlTx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = sqlTx.Rollback()
			return
		}
		err = sqlTx.Commit()
	}()

	insertQuery := `
		INSERT INTO finance.transactions (
			id, profile_id, bank_account_id, destination_account_id, category_id, invoice_id,
			type, status, amount, currency, description, notes, cost_center,
			occurred_on, due_on, reminder_on, recurrence_rule, installment_number, installment_total,
			external_id, linked_transaction_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23
		)
	`

	description := strings.TrimSpace(txn.Description)
	if description == "" {
		description = ""
	}

	_, err = sqlTx.Exec(insertQuery,
		txn.ID,
		txn.ProfileID,
		txn.BankAccountID,
		nullableString(txn.DestinationAccountID),
		nullableString(txn.CategoryID),
		nullableString(txn.InvoiceID),
		txn.Type,
		txn.Status,
		txn.Amount,
		txn.Currency,
		nullableStringFromValue(description),
		nullableString(txn.Notes),
		nullableString(txn.CostCenter),
		txn.OccurredOn,
		nullableTime(txn.DueOn),
		nullableTime(txn.ReminderOn),
		nullableString(txn.RecurrenceRule),
		nullableInt(txn.InstallmentNumber),
		nullableInt(txn.InstallmentTotal),
		nullableString(txn.ExternalID),
		nullableString(txn.LinkedTransactionID),
		txn.CreatedAt,
		txn.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if len(txn.Tags) > 0 {
		tagQuery := `INSERT INTO finance.transaction_tags (transaction_id, tag) VALUES ($1, $2)`
		for _, tag := range txn.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if _, err = sqlTx.Exec(tagQuery, txn.ID, tag); err != nil {
				return err
			}
		}
	}

	if len(txn.Splits) > 0 {
		splitQuery := `
			INSERT INTO finance.transaction_splits (id, transaction_id, category_id, amount, memo, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		for _, split := range txn.Splits {
			if split.ID == "" {
				split.ID = uuid.New().String()
			}
			if split.CreatedAt.IsZero() {
				split.CreatedAt = time.Now()
			}
			if _, err = sqlTx.Exec(splitQuery,
				split.ID,
				txn.ID,
				nullableString(split.CategoryID),
				split.Amount,
				nullableString(split.Memo),
				split.CreatedAt,
			); err != nil {
				return err
			}
		}
	}

	return err
}

func (r *TransactionRepository) GetByID(id string) (*transaction.Transaction, error) {
	query := `
		SELECT id, profile_id, bank_account_id, destination_account_id, category_id, invoice_id,
			type, status, amount, currency, description, notes, cost_center,
			occurred_on, due_on, reminder_on, recurrence_rule, installment_number, installment_total,
			external_id, linked_transaction_id, created_at, updated_at
		FROM finance.transactions
		WHERE id = $1
	`

	row := r.db.QueryRow(query, id)
	tx, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}

	if err := r.attachSplitsAndTags([]*transaction.Transaction{tx}); err != nil {
		return nil, err
	}

	return tx, nil
}

func (r *TransactionRepository) List(filter transaction.ListFilter) ([]*transaction.Transaction, error) {
	if strings.TrimSpace(filter.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}

	baseQuery := `
        SELECT id, profile_id, bank_account_id, destination_account_id, category_id, invoice_id,
               type, status, amount, currency, description, notes, cost_center,
               occurred_on, due_on, reminder_on, recurrence_rule, installment_number, installment_total,
               external_id, linked_transaction_id, created_at, updated_at
        FROM finance.transactions
        WHERE profile_id = $1`

	conditions := []string{}
	args := []interface{}{filter.ProfileID}

	addCondition := func(column string, operator string, value interface{}) {
		placeholder := fmt.Sprintf("$%d", len(args)+1)
		conditions = append(conditions, fmt.Sprintf("%s %s %s", column, operator, placeholder))
		args = append(args, value)
	}

	if filter.BankAccountID != nil {
		trimmed := strings.TrimSpace(*filter.BankAccountID)
		if trimmed != "" {
			addCondition("bank_account_id", "=", trimmed)
		}
	}

	if filter.Status != nil {
		addCondition("status", "=", *filter.Status)
	}

	if filter.Type != nil {
		addCondition("type", "=", *filter.Type)
	}

	if filter.OccurredFrom != nil {
		addCondition("occurred_on", ">=", *filter.OccurredFrom)
	}

	if filter.OccurredTo != nil {
		addCondition("occurred_on", "<=", *filter.OccurredTo)
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY occurred_on DESC, created_at DESC"

	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*transaction.Transaction
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	if err := r.attachSplitsAndTags(transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

type transactionScanner interface {
	Scan(dest ...any) error
}

func scanTransaction(scanner transactionScanner) (*transaction.Transaction, error) {
	tx := &transaction.Transaction{}
	var (
		destination       sql.NullString
		categoryID        sql.NullString
		invoiceID         sql.NullString
		description       sql.NullString
		notes             sql.NullString
		costCenter        sql.NullString
		dueOn             sql.NullTime
		reminderOn        sql.NullTime
		recurrence          sql.NullString
		installmentNumber   sql.NullInt64
		installmentTotal    sql.NullInt64
		externalID          sql.NullString
		linkedTransactionID sql.NullString
	)

	err := scanner.Scan(
		&tx.ID,
		&tx.ProfileID,
		&tx.BankAccountID,
		&destination,
		&categoryID,
		&invoiceID,
		&tx.Type,
		&tx.Status,
		&tx.Amount,
		&tx.Currency,
		&description,
		&notes,
		&costCenter,
		&tx.OccurredOn,
		&dueOn,
		&reminderOn,
		&recurrence,
		&installmentNumber,
		&installmentTotal,
		&externalID,
		&linkedTransactionID,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}

	if destination.Valid {
		s := destination.String
		tx.DestinationAccountID = &s
	}
	if categoryID.Valid {
		s := categoryID.String
		tx.CategoryID = &s
	}
	if invoiceID.Valid {
		s := invoiceID.String
		tx.InvoiceID = &s
	}
	if description.Valid {
		tx.Description = description.String
	}
	if notes.Valid {
		s := notes.String
		tx.Notes = &s
	}
	if costCenter.Valid {
		s := costCenter.String
		tx.CostCenter = &s
	}
	if dueOn.Valid {
		t := dueOn.Time
		tx.DueOn = &t
	}
	if reminderOn.Valid {
		t := reminderOn.Time
		tx.ReminderOn = &t
	}
	if recurrence.Valid {
		s := recurrence.String
		tx.RecurrenceRule = &s
	}
	if installmentNumber.Valid {
		i := int(installmentNumber.Int64)
		tx.InstallmentNumber = &i
	}
	if installmentTotal.Valid {
		i := int(installmentTotal.Int64)
		tx.InstallmentTotal = &i
	}
	if externalID.Valid {
		s := externalID.String
		tx.ExternalID = &s
	}
	if linkedTransactionID.Valid {
		s := linkedTransactionID.String
		tx.LinkedTransactionID = &s
	}

	tx.Splits = []*transaction.Split{}
	tx.Tags = []string{}

	return tx, nil
}

func (r *TransactionRepository) Update(txn *transaction.Transaction) (err error) {
	if txn == nil {
		return errors.New("transaction is nil")
	}

	sqlTx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = sqlTx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = sqlTx.Rollback()
			return
		}
		err = sqlTx.Commit()
	}()

	updateQuery := `
		UPDATE finance.transactions SET
			bank_account_id = $2,
			destination_account_id = $3,
			category_id = $4,
			type = $5,
			status = $6,
			amount = $7,
			currency = $8,
			description = $9,
			notes = $10,
			cost_center = $11,
			occurred_on = $12,
			due_on = $13,
			reminder_on = $14,
			recurrence_rule = $15,
			installment_number = $16,
			installment_total = $17,
			external_id = $18,
			linked_transaction_id = $19,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := sqlTx.Exec(updateQuery,
		txn.ID,
		txn.BankAccountID,
		nullableString(txn.DestinationAccountID),
		nullableString(txn.CategoryID),
		txn.Type,
		txn.Status,
		txn.Amount,
		txn.Currency,
		nullableStringFromValue(txn.Description),
		nullableString(txn.Notes),
		nullableString(txn.CostCenter),
		txn.OccurredOn,
		nullableTime(txn.DueOn),
		nullableTime(txn.ReminderOn),
		nullableString(txn.RecurrenceRule),
		nullableInt(txn.InstallmentNumber),
		nullableInt(txn.InstallmentTotal),
		nullableString(txn.ExternalID),
		nullableString(txn.LinkedTransactionID),
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("transaction not found")
	}

	// Delete existing tags and re-insert
	if _, err = sqlTx.Exec(`DELETE FROM finance.transaction_tags WHERE transaction_id = $1`, txn.ID); err != nil {
		return err
	}

	if len(txn.Tags) > 0 {
		tagQuery := `INSERT INTO finance.transaction_tags (transaction_id, tag) VALUES ($1, $2)`
		for _, tag := range txn.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if _, err = sqlTx.Exec(tagQuery, txn.ID, tag); err != nil {
				return err
			}
		}
	}

	// Delete existing splits and re-insert
	if _, err = sqlTx.Exec(`DELETE FROM finance.transaction_splits WHERE transaction_id = $1`, txn.ID); err != nil {
		return err
	}

	if len(txn.Splits) > 0 {
		splitQuery := `
			INSERT INTO finance.transaction_splits (id, transaction_id, category_id, amount, memo, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		for _, split := range txn.Splits {
			if split.ID == "" {
				split.ID = uuid.New().String()
			}
			if split.CreatedAt.IsZero() {
				split.CreatedAt = time.Now()
			}
			if _, err = sqlTx.Exec(splitQuery,
				split.ID,
				txn.ID,
				nullableString(split.CategoryID),
				split.Amount,
				nullableString(split.Memo),
				split.CreatedAt,
			); err != nil {
				return err
			}
		}
	}

	return err
}

func (r *TransactionRepository) UpdateStatus(id string, status transaction.Status, occurredOn time.Time, notes *string) error {
	query := `
        UPDATE finance.transactions
        SET status = $2,
            occurred_on = $3,
            notes = $4,
            updated_at = NOW()
        WHERE id = $1
    `

	result, err := r.db.Exec(query, id, status, occurredOn, nullableString(notes))
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("transaction not found")
	}

	return nil
}

func (r *TransactionRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.transactions WHERE id = $1`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("transaction not found")
	}

	return nil
}

func (r *TransactionRepository) SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, errors.New("profileID is required")
	}

	if from.After(to) {
		return nil, errors.New("invalid period range")
	}

	query := `
		SELECT category_id,
			SUM(CASE WHEN type = 'EXPENSE' THEN amount ELSE 0 END) AS total
		FROM finance.transactions
		WHERE profile_id = $1
			AND occurred_on BETWEEN $2 AND $3
	`

	args := []interface{}{profileID, from, to}

	if len(categoryIDs) > 0 {
		query += " AND category_id = ANY($4::uuid[])"
		args = append(args, pq.Array(categoryIDs))
	}

	query += " GROUP BY category_id"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]float64)
	for rows.Next() {
		var categoryID sql.NullString
		var total sql.NullFloat64

		if err := rows.Scan(&categoryID, &total); err != nil {
			return nil, err
		}

		key := ""
		if categoryID.Valid {
			key = categoryID.String
		}

		results[key] = round2(total.Float64)
	}

	return results, nil
}

func (r *TransactionRepository) attachSplitsAndTags(transactions []*transaction.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	ids := make([]string, 0, len(transactions))
	lookup := make(map[string]*transaction.Transaction, len(transactions))
	for _, tx := range transactions {
		ids = append(ids, tx.ID)
		lookup[tx.ID] = tx
	}

	splitQuery := `
		SELECT id, transaction_id, category_id, amount, memo, created_at
		FROM finance.transaction_splits
		WHERE transaction_id = ANY($1::uuid[])
		ORDER BY created_at
	`

	splitRows, err := r.db.Query(splitQuery, pq.Array(ids))
	if err != nil {
		return err
	}
	defer splitRows.Close()

	for splitRows.Next() {
		var (
			id            string
			transactionID string
			categoryID    sql.NullString
			amount        float64
			memo          sql.NullString
			createdAt     time.Time
		)

		if err := splitRows.Scan(&id, &transactionID, &categoryID, &amount, &memo, &createdAt); err != nil {
			return err
		}

		tx := lookup[transactionID]
		if tx == nil {
			continue
		}

		split := &transaction.Split{
			ID:        id,
			Amount:    amount,
			CreatedAt: createdAt,
		}
		if categoryID.Valid {
			s := categoryID.String
			split.CategoryID = &s
		}
		if memo.Valid {
			s := memo.String
			split.Memo = &s
		}

		tx.Splits = append(tx.Splits, split)
	}

	tagQuery := `
		SELECT transaction_id, tag
		FROM finance.transaction_tags
		WHERE transaction_id = ANY($1::uuid[])
	`

	tagRows, err := r.db.Query(tagQuery, pq.Array(ids))
	if err != nil {
		return err
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var (
			transactionID string
			tag           string
		)
		if err := tagRows.Scan(&transactionID, &tag); err != nil {
			return err
		}
		tx := lookup[transactionID]
		if tx == nil {
			continue
		}
		tx.Tags = append(tx.Tags, tag)
	}

	for _, tx := range transactions {
		if len(tx.Tags) > 1 {
			sort.Strings(tx.Tags)
		}
	}

	return nil
}

func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableStringFromValue(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nullableTime(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// SumByInvoiceID returns the sum of all transaction amounts for a given invoice
func (r *TransactionRepository) SumByInvoiceID(invoiceID string) (float64, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return 0, errors.New("invoiceID is required")
	}

	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM finance.transactions
		WHERE invoice_id = $1 AND status != 'CANCELLED'
	`

	var total float64
	err := r.db.QueryRow(query, invoiceID).Scan(&total)
	if err != nil {
		return 0, err
	}

	return round2(total), nil
}
