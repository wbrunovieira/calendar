package persistence

import (
	"regexp"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/budgettarget"
	"github.com/brunovieira/calendar-finances/internal/test/sqlmock"
)

func fixedTime2() time.Time { return time.Unix(time.Now().Unix(), 0) }

func TestBudgetTargetRepositoryUpdate_PersistsProfileAndCategory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewBudgetTargetRepository(db)

	now := fixedTime2()
	entity := &budgettarget.BudgetTarget{
		ID:             "bt-1",
		ProfileID:      "profile-2",
		CategoryID:     "category-2",
		PeriodStart:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		Amount:         123.45,
		Notes:          nil,
		IsRecurring:    true,
		EffectiveUntil: nil,
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now,
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE finance.budget_targets")).
		WithArgs(entity.ID, entity.ProfileID, entity.CategoryID, entity.Amount, nil, entity.PeriodStart, entity.IsRecurring, nil, entity.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Update(entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestBudgetTargetRepositoryUpdate_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewBudgetTargetRepository(db)

	now := fixedTime2()
	entity := &budgettarget.BudgetTarget{
		ID:             "bt-404",
		ProfileID:      "profile-x",
		CategoryID:     "category-x",
		PeriodStart:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		Amount:         50.0,
		Notes:          nil,
		IsRecurring:    false,
		EffectiveUntil: nil,
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now,
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE finance.budget_targets")).
		WithArgs(entity.ID, entity.ProfileID, entity.CategoryID, entity.Amount, nil, entity.PeriodStart, entity.IsRecurring, nil, entity.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.Update(entity); err == nil {
		t.Fatal("expected error but got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
