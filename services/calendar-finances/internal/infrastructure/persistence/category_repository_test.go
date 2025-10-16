package persistence

import (
	"regexp"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/test/sqlmock"
)

func fixedNow() time.Time {
	return time.Unix(time.Now().Unix(), 0)
}

func TestCategoryRepositoryCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCategoryRepository(db)

	now := fixedNow()
	color := "#FFAA00"
	cat := &category.Category{
		ID:        "cat-123",
		ProfileID: "profile-1",
		Name:      "Transporte",
		Type:      category.TypeExpense,
		Color:     &color,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO finance.categories")).
		WithArgs(cat.ID, cat.ProfileID, cat.Name, cat.Type, cat.Color, cat.Icon, cat.ParentID, cat.IsActive, cat.CreatedAt, cat.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Create(cat); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestCategoryRepositoryFindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCategoryRepository(db)

	now := fixedNow()
	rows := sqlmock.NewRows([]string{"id", "profile_id", "name", "type", "color", "icon", "parent_id", "is_active", "created_at", "updated_at"}).
		AddRow("cat-123", "profile-1", "Transporte", "EXPENSE", "#FFAA00", nil, nil, true, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, name, type, color, icon, parent_id, is_active, created_at, updated_at")).
		WithArgs("cat-123").
		WillReturnRows(rows)

	cat, err := repo.FindByID("cat-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cat.Name != "Transporte" || cat.Type != category.TypeExpense {
		t.Fatalf("unexpected category data: %+v", cat)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
