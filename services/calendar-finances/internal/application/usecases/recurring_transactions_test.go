package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
)

type fakeRecurringRepo struct {
	items map[string]*recurringtransaction.RecurringTransaction
}

func newFakeRecurringRepo() *fakeRecurringRepo {
	return &fakeRecurringRepo{items: make(map[string]*recurringtransaction.RecurringTransaction)}
}

func (f *fakeRecurringRepo) Create(entity *recurringtransaction.RecurringTransaction) error {
	f.items[entity.ID] = entity
	return nil
}

func (f *fakeRecurringRepo) Update(entity *recurringtransaction.RecurringTransaction) error {
	if _, ok := f.items[entity.ID]; !ok {
		return errors.New("not found")
	}
	f.items[entity.ID] = entity
	return nil
}

func (f *fakeRecurringRepo) FindByID(id string) (*recurringtransaction.RecurringTransaction, error) {
	if entity, ok := f.items[id]; ok {
		return entity, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeRecurringRepo) ListByProfile(profileID string) ([]*recurringtransaction.RecurringTransaction, error) {
	var list []*recurringtransaction.RecurringTransaction
	for _, entity := range f.items {
		if entity.ProfileID == profileID {
			list = append(list, entity)
		}
	}
	return list, nil
}

func (f *fakeRecurringRepo) Delete(id string) error {
	if _, ok := f.items[id]; !ok {
		return errors.New("not found")
	}
	delete(f.items, id)
	return nil
}

func TestCreateRecurringTransaction(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Currency:       "GBP",
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	}

	result, err := service.Create(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Description != "Flight Academy" {
		t.Fatalf("expected description 'Flight Academy', got %s", result.Description)
	}

	if result.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", result.Status)
	}

	if result.ReviewOn != nil {
		t.Fatalf("expected nil ReviewOn, got %v", result.ReviewOn)
	}

	if len(repo.items) != 1 {
		t.Fatalf("expected 1 item in repo, got %d", len(repo.items))
	}
}

func TestCreateRecurringTransactionWithReviewOn(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	reviewOn := "2026-05-01"
	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Currency:       "GBP",
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "PAUSED",
		ReviewOn:       &reviewOn,
	}

	result, err := service.Create(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "PAUSED" {
		t.Fatalf("expected status PAUSED, got %s", result.Status)
	}

	if result.ReviewOn == nil {
		t.Fatal("expected ReviewOn to be set")
	}

	expectedReviewOn := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	if !result.ReviewOn.Equal(expectedReviewOn) {
		t.Fatalf("expected ReviewOn %v, got %v", expectedReviewOn, result.ReviewOn)
	}
}

func TestUpdateRecurringTransactionStatus(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	// Create initial transaction
	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Currency:       "GBP",
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	}

	created, err := service.Create(input)
	if err != nil {
		t.Fatalf("unexpected error creating: %v", err)
	}

	// Update status to PAUSED
	updated, err := service.UpdateStatus(created.ID, "PAUSED")
	if err != nil {
		t.Fatalf("unexpected error updating status: %v", err)
	}

	if updated.Status != "PAUSED" {
		t.Fatalf("expected status PAUSED, got %s", updated.Status)
	}
}

func TestUpdateRecurringTransactionWithReviewOn(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	// Create initial transaction
	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Currency:       "GBP",
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	}

	created, err := service.Create(input)
	if err != nil {
		t.Fatalf("unexpected error creating: %v", err)
	}

	// Update with ReviewOn
	reviewOn := "2026-05-01"
	updateInput := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Currency:       "GBP",
		Description:    "Flight Academy - Paused",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "PAUSED",
		ReviewOn:       &reviewOn,
	}

	updated, err := service.Update(created.ID, updateInput)
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}

	if updated.Status != "PAUSED" {
		t.Fatalf("expected status PAUSED, got %s", updated.Status)
	}

	if updated.ReviewOn == nil {
		t.Fatal("expected ReviewOn to be set after update")
	}

	expectedReviewOn := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	if !updated.ReviewOn.Equal(expectedReviewOn) {
		t.Fatalf("expected ReviewOn %v, got %v", expectedReviewOn, updated.ReviewOn)
	}

	if updated.Description != "Flight Academy - Paused" {
		t.Fatalf("expected description 'Flight Academy - Paused', got %s", updated.Description)
	}
}

func TestListRecurringTransactions(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	// Create transactions for different profiles
	_, _ = service.Create(CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	})

	reviewOn := "2026-05-01"
	_, _ = service.Create(CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         9.99,
		Description:    "Netflix",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "PAUSED",
		ReviewOn:       &reviewOn,
	})

	_, _ = service.Create(CreateRecurringTransactionInput{
		ProfileID:      "profile-2",
		Type:           "EXPENSE",
		Amount:         50.00,
		Description:    "Other Profile Sub",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	})

	list, err := service.List("profile-1")
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 items for profile-1, got %d", len(list))
	}

	// Check that one has ReviewOn set
	var foundReviewOn bool
	for _, item := range list {
		if item.ReviewOn != nil {
			foundReviewOn = true
			break
		}
	}

	if !foundReviewOn {
		t.Fatal("expected at least one item with ReviewOn set")
	}
}

func TestDeleteRecurringTransaction(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	created, err := service.Create(CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	})
	if err != nil {
		t.Fatalf("unexpected error creating: %v", err)
	}

	err = service.Delete(created.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}

	if len(repo.items) != 0 {
		t.Fatalf("expected 0 items after delete, got %d", len(repo.items))
	}
}

func TestDeleteRecurringTransactionNotFound(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	err := service.Delete("non-existent-id")
	if err == nil {
		t.Fatal("expected error for non-existent id")
	}

	if !errors.Is(err, ErrRecurringNotFound) {
		t.Fatalf("expected ErrRecurringNotFound, got %v", err)
	}
}

func TestCreateRecurringTransactionInvalidInput(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	// Invalid date format
	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Test",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "invalid-date",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	}

	_, err := service.Create(input)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateRecurringTransactionInvalidReviewOnDate(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	invalidReviewOn := "not-a-date"
	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Test",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "PAUSED",
		ReviewOn:       &invalidReviewOn,
	}

	_, err := service.Create(input)
	if err == nil {
		t.Fatal("expected error for invalid reviewOn date")
	}

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateRecurringTransactionNotFound(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	input := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Test",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	}

	_, err := service.Update("non-existent-id", input)
	if err == nil {
		t.Fatal("expected error for non-existent id")
	}

	if !errors.Is(err, ErrRecurringNotFound) {
		t.Fatalf("expected ErrRecurringNotFound, got %v", err)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	_, err := service.UpdateStatus("non-existent-id", "PAUSED")
	if err == nil {
		t.Fatal("expected error for non-existent id")
	}

	if !errors.Is(err, ErrRecurringNotFound) {
		t.Fatalf("expected ErrRecurringNotFound, got %v", err)
	}
}

func TestUpdateStatusInvalid(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	created, err := service.Create(CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Test",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
	})
	if err != nil {
		t.Fatalf("unexpected error creating: %v", err)
	}

	_, err = service.UpdateStatus(created.ID, "INVALID_STATUS")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestClearReviewOnWhenReactivating(t *testing.T) {
	repo := newFakeRecurringRepo()
	service := NewRecurringTransactionsService(repo)

	// Create paused transaction with review date
	reviewOn := "2026-05-01"
	created, err := service.Create(CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "PAUSED",
		ReviewOn:       &reviewOn,
	})
	if err != nil {
		t.Fatalf("unexpected error creating: %v", err)
	}

	if created.ReviewOn == nil {
		t.Fatal("expected ReviewOn to be set")
	}

	// Update back to active without ReviewOn
	updateInput := CreateRecurringTransactionInput{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.99,
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        "2026-01-01",
		NextOccurrence: "2026-02-01",
		Status:         "ACTIVE",
		// ReviewOn not set - should be cleared
	}

	updated, err := service.Update(created.ID, updateInput)
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}

	if updated.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", updated.Status)
	}

	if updated.ReviewOn != nil {
		t.Fatalf("expected ReviewOn to be cleared when reactivating, got %v", updated.ReviewOn)
	}
}
