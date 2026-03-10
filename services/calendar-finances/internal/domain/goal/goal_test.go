package goal

import (
	"testing"
	"time"
)

func TestNewGoal_ValidInput(t *testing.T) {
	targetDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	g, err := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "MacBook Pro M4",
		Description:  "Para desenvolvimento",
		TargetAmount: 15000.00,
		Priority:     PriorityHigh,
		TargetDate:   &targetDate,
		Link:         "https://apple.com/macbook-pro",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if g.ID == "" {
		t.Error("expected ID to be generated")
	}
	if g.ProfileID != "profile-1" {
		t.Errorf("expected profile-1, got %s", g.ProfileID)
	}
	if g.Name != "MacBook Pro M4" {
		t.Errorf("expected MacBook Pro M4, got %s", g.Name)
	}
	if g.TargetAmount != 15000.00 {
		t.Errorf("expected 15000, got %.2f", g.TargetAmount)
	}
	if g.CurrentAmount != 0 {
		t.Errorf("expected 0, got %.2f", g.CurrentAmount)
	}
	if g.Priority != PriorityHigh {
		t.Errorf("expected HIGH, got %s", g.Priority)
	}
	if g.Status != StatusActive {
		t.Errorf("expected ACTIVE, got %s", g.Status)
	}
	if g.Link != "https://apple.com/macbook-pro" {
		t.Errorf("expected link, got %s", g.Link)
	}
}

func TestNewGoal_MissingProfileID(t *testing.T) {
	_, err := New(CreateParams{
		Name:         "Item",
		TargetAmount: 100,
	})
	if err == nil {
		t.Fatal("expected error for missing profileID")
	}
}

func TestNewGoal_MissingName(t *testing.T) {
	_, err := New(CreateParams{
		ProfileID:    "profile-1",
		TargetAmount: 100,
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestNewGoal_ZeroAmount(t *testing.T) {
	_, err := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "Item",
		TargetAmount: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestGoal_AddAmount(t *testing.T) {
	g, _ := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "Item",
		TargetAmount: 1000,
	})

	err := g.AddAmount(250.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.CurrentAmount != 250.50 {
		t.Errorf("expected 250.50, got %.2f", g.CurrentAmount)
	}
}

func TestGoal_AddAmount_ExceedsTarget(t *testing.T) {
	g, _ := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "Item",
		TargetAmount: 100,
	})

	err := g.AddAmount(150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should allow exceeding target (user may save more)
	if g.CurrentAmount != 150 {
		t.Errorf("expected 150, got %.2f", g.CurrentAmount)
	}
}

func TestGoal_Complete(t *testing.T) {
	g, _ := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "Item",
		TargetAmount: 100,
	})

	err := g.Complete()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Status != StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", g.Status)
	}
}

func TestGoal_Complete_AlreadyCompleted(t *testing.T) {
	g, _ := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "Item",
		TargetAmount: 100,
	})
	g.Complete()

	err := g.Complete()
	if err == nil {
		t.Fatal("expected error completing already completed goal")
	}
}

func TestGoal_Progress(t *testing.T) {
	g, _ := New(CreateParams{
		ProfileID:    "profile-1",
		Name:         "Item",
		TargetAmount: 1000,
	})

	g.AddAmount(250)

	progress := g.Progress()
	if progress != 25.0 {
		t.Errorf("expected 25%%, got %.2f%%", progress)
	}
}
