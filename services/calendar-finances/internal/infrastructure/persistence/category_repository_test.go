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
		WithArgs(cat.ID, cat.ProfileID, cat.Name, cat.Type, cat.Color, cat.Icon, cat.ParentID, cat.ClassificationDRE, cat.IsActive, cat.CreatedAt, cat.UpdatedAt).
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
	rows := sqlmock.NewRows([]string{"id", "profile_id", "name", "type", "color", "icon", "parent_id", "classification_dre", "is_active", "created_at", "updated_at"}).
		AddRow("cat-123", "profile-1", "Transporte", "EXPENSE", "#FFAA00", nil, nil, nil, true, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, name, type, color, icon, parent_id, classification_dre, is_active, created_at, updated_at")).
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

func TestCategoryRepositoryGetDescendantIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Test with parent category that has children
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow("parent-123").
		AddRow("child-1").
		AddRow("child-2").
		AddRow("grandchild-1")

	mock.ExpectQuery(regexp.QuoteMeta("WITH RECURSIVE descendants AS")).
		WithArgs("parent-123").
		WillReturnRows(rows)

	ids, err := repo.GetDescendantIDs("parent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ids) != 4 {
		t.Fatalf("expected 4 IDs, got %d", len(ids))
	}

	expectedIDs := map[string]bool{"parent-123": true, "child-1": true, "child-2": true, "grandchild-1": true}
	for _, id := range ids {
		if !expectedIDs[id] {
			t.Fatalf("unexpected ID: %s", id)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestCategoryRepositoryGetDescendantIDsNoChildren(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewCategoryRepository(db)

	// Test with leaf category (no children)
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow("leaf-cat")

	mock.ExpectQuery(regexp.QuoteMeta("WITH RECURSIVE descendants AS")).
		WithArgs("leaf-cat").
		WillReturnRows(rows)

	ids, err := repo.GetDescendantIDs("leaf-cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}

	if ids[0] != "leaf-cat" {
		t.Fatalf("expected leaf-cat, got %s", ids[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
