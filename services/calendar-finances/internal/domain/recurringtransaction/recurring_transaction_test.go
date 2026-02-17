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

// =============================================================================
// CalculateNextOccurrence Tests (TDD)
// =============================================================================

func TestCalculateNextOccurrence_Monthly_WhenDateHasPassed(t *testing.T) {
	// Given: A monthly recurring transaction starting Jan 1st
	// When: Today is Jan 26th (the 1st has passed)
	// Then: Next occurrence should be Feb 1st

	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         191.84,
		Description:    "Fly with Greg",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date: Jan 26, 2026
	referenceDate := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: Feb 1, 2026
	expected := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_Monthly_WhenDateHasNotPassed(t *testing.T) {
	// Given: A monthly recurring transaction starting Jan 28th
	// When: Today is Jan 26th (the 28th has NOT passed)
	// Then: Next occurrence should be Jan 28th (same month)

	startOn := time.Date(2026, time.January, 28, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 28, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         100.00,
		Description:    "Test Subscription",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	expected := time.Date(2026, time.January, 28, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_Monthly_MultipleMonthsPassed(t *testing.T) {
	// Given: A monthly recurring transaction starting Jan 1st
	// When: Today is April 15th (3+ months have passed)
	// Then: Next occurrence should be May 1st

	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         100.00,
		Description:    "Test Subscription",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	expected := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_Weekly(t *testing.T) {
	// Given: A weekly recurring transaction starting Monday Jan 5th
	// When: Today is Jan 26th (Monday — same weekday as start)
	// Then: Next occurrence should be today (Jan 26th) — alert should show

	startOn := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC) // Monday
	nextOccurrence := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         50.00,
		Description:    "Weekly Payment",
		RecurrenceRule: "FREQ=WEEKLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC) // Monday
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Jan 26 is the target weekday (Monday), so next occurrence is today
	expected := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_Daily(t *testing.T) {
	// Given: A daily recurring transaction starting Jan 1st
	// When: Today is Jan 26th
	// Then: Next occurrence should be today (Jan 26th) — daily means every day including today

	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         10.00,
		Description:    "Daily Coffee",
		RecurrenceRule: "FREQ=DAILY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	expected := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_WithEndDate_StopsAtEnd(t *testing.T) {
	// Given: A monthly recurring transaction with endOn Feb 15th
	// When: Today is Feb 20th (after end date)
	// Then: Should return zero time (no more occurrences)

	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	endOn := time.Date(2026, time.February, 15, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         100.00,
		Description:    "Limited Subscription",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		EndOn:          &endOn,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.February, 20, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	if !calculatedNext.IsZero() {
		t.Errorf("expected zero time (no more occurrences), got %v", calculatedNext)
	}
}

func TestCalculateNextOccurrence_Paused_ReturnsZero(t *testing.T) {
	// Given: A paused recurring transaction
	// Then: Should return zero time (paused transactions have no next occurrence)

	startOn := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         100.00,
		Description:    "Paused Subscription",
		RecurrenceRule: "FREQ=MONTHLY",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusPaused,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	if !calculatedNext.IsZero() {
		t.Errorf("expected zero time for paused transaction, got %v", calculatedNext)
	}
}

func TestCalculateNextOccurrence_Monthly_WithBYMONTHDAY(t *testing.T) {
	// Given: A monthly recurring transaction with BYMONTHDAY=15
	// When: Today is Jan 26th
	// Then: Next occurrence should be Feb 15th

	startOn := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         100.00,
		Description:    "Mid-month Payment",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=15",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.January, 26, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	expected := time.Date(2026, time.February, 15, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, calculatedNext)
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

func TestCalculateNextOccurrence_ShouldNotReturnDateBeforeStartOn(t *testing.T) {
	// Given: A recurring transaction that starts on March 10, 2026
	// When: Today is January 31, 2026 (before startOn)
	// Then: Next occurrence should be March 10, 2026 (not February 10)

	startOn := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         87.90,
		Description:    "CapCut",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=10",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is January 31, 2026 (before startOn)
	referenceDate := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// The next occurrence should be March 10, not February 10
	expectedNext := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_WhenReferenceDateAfterStartOn(t *testing.T) {
	// Given: A recurring transaction that starts on January 10, 2026
	// When: Today is January 31, 2026 (after startOn, after day 10)
	// Then: Next occurrence should be February 10, 2026

	startOn := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         100.00,
		Description:    "Test",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=10",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is January 31, 2026 (after startOn and after day 10)
	referenceDate := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// The next occurrence should be February 10
	expectedNext := time.Date(2026, time.February, 10, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

// =============================================================================
// February Day Handling Tests (TDD)
// =============================================================================

func TestCalculateNextOccurrence_February_Day30_NonLeapYear(t *testing.T) {
	// Given: A monthly recurring transaction on day 30
	// When: Next month is February 2026 (non-leap year, 28 days)
	// Then: Next occurrence should be February 28, 2026

	startOn := time.Date(2026, time.January, 30, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 30, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         800.00,
		Description:    "Residencial Mae",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=30",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is January 31, 2026 (day 30 has passed)
	referenceDate := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: February 28, 2026 (last day of Feb in non-leap year)
	expectedNext := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_February_Day31_NonLeapYear(t *testing.T) {
	// Given: A monthly recurring transaction on day 31
	// When: Next month is February 2026 (non-leap year, 28 days)
	// Then: Next occurrence should be February 28, 2026

	startOn := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         500.00,
		Description:    "End of Month Payment",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=31",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is February 1, 2026 (Jan 31 has passed)
	referenceDate := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: February 28, 2026 (last day of Feb in non-leap year)
	expectedNext := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_February_Day29_NonLeapYear(t *testing.T) {
	// Given: A monthly recurring transaction on day 29
	// When: Next month is February 2026 (non-leap year, 28 days)
	// Then: Next occurrence should be February 28, 2026

	startOn := time.Date(2026, time.January, 29, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 29, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         300.00,
		Description:    "Day 29 Payment",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=29",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is January 30, 2026 (day 29 has passed)
	referenceDate := time.Date(2026, time.January, 30, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: February 28, 2026 (last day of Feb in non-leap year)
	expectedNext := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_February_Day30_LeapYear(t *testing.T) {
	// Given: A monthly recurring transaction on day 30
	// When: Next month is February 2028 (leap year, 29 days)
	// Then: Next occurrence should be February 29, 2028

	startOn := time.Date(2028, time.January, 30, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2028, time.January, 30, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         800.00,
		Description:    "Residencial Mae",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=30",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is January 31, 2028 (day 30 has passed)
	referenceDate := time.Date(2028, time.January, 31, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: February 29, 2028 (last day of Feb in leap year)
	expectedNext := time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_February_Day31_LeapYear(t *testing.T) {
	// Given: A monthly recurring transaction on day 31
	// When: Next month is February 2028 (leap year, 29 days)
	// Then: Next occurrence should be February 29, 2028

	startOn := time.Date(2028, time.January, 31, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2028, time.January, 31, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         500.00,
		Description:    "End of Month Payment",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=31",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is February 1, 2028 (Jan 31 has passed)
	referenceDate := time.Date(2028, time.February, 1, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: February 29, 2028 (last day of Feb in leap year)
	expectedNext := time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_February_Day29_LeapYear(t *testing.T) {
	// Given: A monthly recurring transaction on day 29
	// When: Next month is February 2028 (leap year, 29 days)
	// Then: Next occurrence should be February 29, 2028

	startOn := time.Date(2028, time.January, 29, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2028, time.January, 29, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         300.00,
		Description:    "Day 29 Payment",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=29",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is January 30, 2028 (day 29 has passed)
	referenceDate := time.Date(2028, time.January, 30, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: February 29, 2028 (day 29 exists in leap year)
	expectedNext := time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

func TestCalculateNextOccurrence_OtherShortMonths_Day31(t *testing.T) {
	// Given: A monthly recurring transaction on day 31
	// When: Next month is April (30 days)
	// Then: Next occurrence should be April 30

	startOn := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         500.00,
		Description:    "End of Month Payment",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=31",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reference date is April 1, 2026 (March 31 has passed)
	referenceDate := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Expected: April 30, 2026 (last day of April)
	expectedNext := time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expectedNext) {
		t.Fatalf("expected next occurrence %v, got %v", expectedNext, calculatedNext)
	}
}

// =============================================================================
// Day-of Occurrence Tests (bug fix: alerts must show on the day of the recurring)
// =============================================================================

func TestCalculateNextOccurrence_Monthly_OnTheDayOf_WithTimeComponent(t *testing.T) {
	// BUG FIX: On Feb 14 at 10:00 BRT (13:00 UTC), a recurring on day 14 should
	// return Feb 14, NOT March 14. The old code used time-aware comparison
	// (candidate midnight UTC < referenceDate 13:00 UTC → jumped to next month).

	startOn := time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.90,
		Description:    "Amazon Kindle Unlimited",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=14",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate time.Now() returning 10:00 BRT (13:00 UTC) on Feb 14
	referenceDate := time.Date(2026, time.February, 14, 13, 0, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	// Must return Feb 14, NOT March 14
	expected := time.Date(2026, time.February, 14, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Fatalf("expected %v (today), got %v (should not skip to next month)", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_Monthly_DayAfter_ReturnsNextMonth(t *testing.T) {
	// On Feb 15, a recurring on day 14 should return March 14

	startOn := time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.90,
		Description:    "Amazon Kindle Unlimited",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=14",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.February, 15, 10, 30, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	expected := time.Date(2026, time.March, 14, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, calculatedNext)
	}
}

func TestCalculateNextOccurrence_Monthly_DayBefore_ReturnsSameMonth(t *testing.T) {
	// On Feb 13, a recurring on day 14 should return Feb 14

	startOn := time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)
	nextOccurrence := time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)

	rt, err := New(CreateParams{
		ProfileID:      "profile-1",
		Type:           "EXPENSE",
		Amount:         24.90,
		Description:    "Amazon Kindle Unlimited",
		RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=14",
		StartOn:        startOn,
		NextOccurrence: nextOccurrence,
		Status:         StatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	referenceDate := time.Date(2026, time.February, 13, 23, 59, 0, 0, time.UTC)
	calculatedNext := rt.CalculateNextOccurrence(referenceDate)

	expected := time.Date(2026, time.February, 14, 0, 0, 0, 0, time.UTC)

	if !calculatedNext.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, calculatedNext)
	}
}
