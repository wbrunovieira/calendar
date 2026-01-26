package recurringtransaction

import (
	"testing"
	"time"
)

func TestNewRecurringTransaction(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         24.99,
		Currency:       "gbp",
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rt.Type != "EXPENSE" {
		t.Fatalf("expected type EXPENSE, got %s", rt.Type)
	}

	if rt.Currency != "GBP" {
		t.Fatalf("expected currency GBP, got %s", rt.Currency)
	}

	if rt.Status != StatusActive {
		t.Fatalf("expected status ACTIVE, got %s", rt.Status)
	}

	if rt.ReviewOn != nil {
		t.Fatalf("expected nil ReviewOn, got %v", rt.ReviewOn)
	}
}

func TestNewRecurringTransactionWithReviewOn(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	reviewOn := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         24.99,
		Currency:       "gbp",
		Description:    "Flight Academy",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=1",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusPaused,
		ReviewOn:       &reviewOn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rt.Status != StatusPaused {
		t.Fatalf("expected status PAUSED, got %s", rt.Status)
	}

	if rt.ReviewOn == nil {
		t.Fatal("expected ReviewOn to be set")
	}

	if !rt.ReviewOn.Equal(reviewOn) {
		t.Fatalf("expected ReviewOn %v, got %v", reviewOn, rt.ReviewOn)
	}
}

func TestSetStatus(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         24.99,
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test valid status transitions
	if err := rt.SetStatus(StatusPaused); err != nil {
		t.Fatalf("unexpected error setting PAUSED: %v", err)
	}
	if rt.Status != StatusPaused {
		t.Fatalf("expected PAUSED, got %s", rt.Status)
	}

	if err := rt.SetStatus(StatusCanceled); err != nil {
		t.Fatalf("unexpected error setting CANCELLED: %v", err)
	}
	if rt.Status != StatusCanceled {
		t.Fatalf("expected CANCELLED, got %s", rt.Status)
	}

	// Test invalid status
	if err := rt.SetStatus(Status("INVALID")); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestSetReviewOn(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         24.99,
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rt.ReviewOn != nil {
		t.Fatal("expected nil ReviewOn initially")
	}

	reviewDate := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	rt.SetReviewOn(&reviewDate)

	if rt.ReviewOn == nil {
		t.Fatal("expected ReviewOn to be set")
	}

	if !rt.ReviewOn.Equal(reviewDate) {
		t.Fatalf("expected %v, got %v", reviewDate, rt.ReviewOn)
	}

	// Test clearing ReviewOn
	rt.SetReviewOn(nil)
	if rt.ReviewOn != nil {
		t.Fatal("expected ReviewOn to be nil after clearing")
	}
}

func TestValidationErrors(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		params CreateParams
	}{
		{
			name: "missing profileID",
			params: CreateParams{
				Type:           "expense",
				Amount:         100,
				RecurrenceRule: "FREQ=MONTHLY",
				StartOn:        startOn,
				NextOccurrence: nextOccurrence,
				Status:         StatusActive,
			},
		},
		{
			name: "zero amount",
			params: CreateParams{
				ProfileID:      "profile-1",
				Type:           "expense",
				Amount:         0,
				RecurrenceRule: "FREQ=MONTHLY",
				StartOn:        startOn,
				NextOccurrence: nextOccurrence,
				Status:         StatusActive,
			},
		},
		{
			name: "negative amount",
			params: CreateParams{
				ProfileID:      "profile-1",
				Type:           "expense",
				Amount:         -50,
				RecurrenceRule: "FREQ=MONTHLY",
				StartOn:        startOn,
				NextOccurrence: nextOccurrence,
				Status:         StatusActive,
			},
		},
		{
			name: "missing recurrence rule",
			params: CreateParams{
				ProfileID:      "profile-1",
				Type:           "expense",
				Amount:         100,
				StartOn:        startOn,
				NextOccurrence: nextOccurrence,
				Status:         StatusActive,
			},
		},
		{
			name: "missing startOn",
			params: CreateParams{
				ProfileID:      "profile-1",
				Type:           "expense",
				Amount:         100,
				RecurrenceRule: "FREQ=MONTHLY",
				NextOccurrence: nextOccurrence,
				Status:         StatusActive,
			},
		},
		{
			name: "missing nextOccurrence",
			params: CreateParams{
				ProfileID:      "profile-1",
				Type:           "expense",
				Amount:         100,
				RecurrenceRule: "FREQ=MONTHLY",
				StartOn:        startOn,
				Status:         StatusActive,
			},
		},
		{
			name: "invalid status",
			params: CreateParams{
				ProfileID:      "profile-1",
				Type:           "expense",
				Amount:         100,
				RecurrenceRule: "FREQ=MONTHLY",
				StartOn:        startOn,
				NextOccurrence: nextOccurrence,
				Status:         Status("INVALID"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.params)
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestAmountRounding(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         24.999,
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rt.Amount != 25.00 {
		t.Fatalf("expected amount 25.00, got %.2f", rt.Amount)
	}
}

func TestUpdate(t *testing.T) {
	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         24.99,
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	originalID := rt.ID
	originalCreatedAt := rt.CreatedAt

	reviewOn := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	err = rt.Update(CreateParams{
		ProfileID:      "profile-1",
		Type:           "expense",
		Amount:         29.99,
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusPaused,
		ReviewOn:       &reviewOn,
	})
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if rt.ID != originalID {
		t.Fatal("ID should not change on update")
	}

	if !rt.CreatedAt.Equal(originalCreatedAt) {
		t.Fatal("CreatedAt should not change on update")
	}

	if rt.Amount != 29.99 {
		t.Fatalf("expected amount 29.99, got %.2f", rt.Amount)
	}

	if rt.Status != StatusPaused {
		t.Fatalf("expected PAUSED status, got %s", rt.Status)
	}

	if rt.ReviewOn == nil || !rt.ReviewOn.Equal(reviewOn) {
		t.Fatalf("expected ReviewOn %v, got %v", reviewOn, rt.ReviewOn)
	}
}
