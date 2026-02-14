package persistence

import (
	"regexp"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/test/sqlmock"
)

func fixedTime() time.Time {
	return time.Unix(time.Now().Unix(), 0)
}

func TestTransactionRepositoryCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTransactionRepository(db)

	now := fixedTime()
	txn := &transaction.Transaction{
		ID:            "tx-123",
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusPlanned,
		Amount:        250.75,
		Currency:      "BRL",
		Description:   "Aluguel",
		OccurredOn:    now,
		CreatedAt:     now,
		UpdatedAt:     now,
		Tags:          []string{"fixo"},
		Splits: []*transaction.Split{
			{ID: "split-1", Amount: 250.75, CreatedAt: now},
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO finance.transactions")).
		WithArgs(
			txn.ID,
			txn.ProfileID,
			txn.BankAccountID,
			nil, // destination_account_id
			nil, // category_id
			nil, // invoice_id
			txn.Type,
			txn.Status,
			txn.Amount,
			txn.Currency,
			txn.Description,
			nil, // notes
			nil, // cost_center
			txn.OccurredOn,
			nil, // due_on
			nil, // reminder_on
			nil, // recurrence_rule
			nil, // installment_number
			nil, // installment_total
			nil, // external_id
			nil, // linked_transaction_id
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO finance.transaction_tags")).
		WithArgs(txn.ID, "fixo").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO finance.transaction_splits")).
		WithArgs("split-1", txn.ID, nil, txn.Amount, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := repo.Create(txn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestTransactionRepositoryGetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTransactionRepository(db)

	now := fixedTime()

	txRows := sqlmock.NewRows([]string{
		"id", "profile_id", "bank_account_id", "destination_account_id", "category_id", "invoice_id",
		"type", "status", "amount", "currency", "description", "notes", "cost_center",
		"occurred_on", "due_on", "reminder_on", "recurrence_rule", "installment_number", "installment_total",
		"external_id", "linked_transaction_id", "created_at", "updated_at",
	}).AddRow(
		"tx-123", "profile-1", "account-1", nil, nil, nil,
		"EXPENSE", "CONFIRMED", 120.0, "BRL", "Conta", nil, nil,
		now, nil, nil, nil, nil, nil,
		nil, nil, now, now,
	)

	splitRows := sqlmock.NewRows([]string{"id", "transaction_id", "category_id", "amount", "memo", "created_at"}).
		AddRow("split-1", "tx-123", nil, 120.0, nil, now)

	tagRows := sqlmock.NewRows([]string{"transaction_id", "tag"}).
		AddRow("tx-123", "fixo")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, bank_account_id, destination_account_id, category_id, invoice_id")).
		WithArgs("tx-123").
		WillReturnRows(txRows)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, transaction_id, category_id, amount, memo, created_at")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(splitRows)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT transaction_id, tag")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(tagRows)

	tx, err := repo.GetByID("tx-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != transaction.StatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", tx.Status)
	}

	if len(tx.Splits) != 1 || len(tx.Tags) != 1 {
		t.Fatalf("expected splits and tags to be loaded, got %+v", tx)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
