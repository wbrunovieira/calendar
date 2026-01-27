package transaction

import (
	"testing"
	"time"
)

func TestNewTransaction(t *testing.T) {
	occurred := time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC)
	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeExpense,
		Amount:        150.55,
		Currency:      "brl",
		Description:   "Plano de saúde",
		OccurredOn:    occurred,
		Tags:          []string{"Saúde", "saúde", ""},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != StatusPlanned {
		t.Fatalf("expected planned status, got %s", tx.Status)
	}

	if len(tx.Tags) != 1 || tx.Tags[0] != "saúde" {
		t.Fatalf("expected normalized tag, got %v", tx.Tags)
	}

	if tx.Currency != "BRL" {
		t.Fatalf("expected currency uppercase BRL, got %s", tx.Currency)
	}
}

func TestNewTransactionTransferValidation(t *testing.T) {
	occurred := time.Now()
	_, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeTransfer,
		Amount:        100,
		OccurredOn:    occurred,
	})
	if err == nil {
		t.Fatal("expected error when destination account is missing for transfer")
	}
}

func TestTransactionSplits(t *testing.T) {
	occurred := time.Now()
	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeExpense,
		Amount:        200,
		OccurredOn:    occurred,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	splitOne, err := NewSplit(nil, 120, nil)
	if err != nil {
		t.Fatalf("split error: %v", err)
	}
	splitTwo, err := NewSplit(nil, 90, nil)
	if err != nil {
		t.Fatalf("split error: %v", err)
	}

	err = tx.ReplaceSplits([]*Split{splitOne, splitTwo})
	if err == nil {
		t.Fatal("expected error when splits exceed amount")
	}

	splitTwo.Amount = 80
	if err := tx.ReplaceSplits([]*Split{splitOne, splitTwo}); err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	if len(tx.Splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(tx.Splits))
	}
}

func TestAddTag(t *testing.T) {
	occurred := time.Now()
	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeExpense,
		Amount:        50,
		OccurredOn:    occurred,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tx.AddTag("Transporte")
	tx.AddTag("transporte")
	tx.AddTag("  ")

	if len(tx.Tags) != 1 {
		t.Fatalf("expected 1 unique tag, got %d", len(tx.Tags))
	}
}

func TestNewTransactionWithReminderOn(t *testing.T) {
	occurred := time.Date(2025, time.March, 20, 0, 0, 0, 0, time.UTC)
	reminder := time.Date(2025, time.March, 10, 0, 0, 0, 0, time.UTC)

	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		Currency:      "BRL",
		Description:   "Devolucao de curso",
		OccurredOn:    occurred,
		ReminderOn:    &reminder,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.ReminderOn == nil {
		t.Fatal("expected ReminderOn to be set")
	}

	if !tx.ReminderOn.Equal(reminder) {
		t.Fatalf("expected ReminderOn %v, got %v", reminder, *tx.ReminderOn)
	}
}

func TestNewTransactionWithoutReminderOn(t *testing.T) {
	occurred := time.Date(2025, time.March, 20, 0, 0, 0, 0, time.UTC)

	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		OccurredOn:    occurred,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.ReminderOn != nil {
		t.Fatal("expected ReminderOn to be nil when not provided")
	}
}

func TestSetReminderOn(t *testing.T) {
	occurred := time.Date(2025, time.March, 20, 0, 0, 0, 0, time.UTC)

	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		OccurredOn:    occurred,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reminder := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	tx.SetReminderOn(&reminder)

	if tx.ReminderOn == nil {
		t.Fatal("expected ReminderOn to be set after SetReminderOn")
	}

	if !tx.ReminderOn.Equal(reminder) {
		t.Fatalf("expected ReminderOn %v, got %v", reminder, *tx.ReminderOn)
	}
}

func TestClearReminderOn(t *testing.T) {
	occurred := time.Date(2025, time.March, 20, 0, 0, 0, 0, time.UTC)
	reminder := time.Date(2025, time.March, 10, 0, 0, 0, 0, time.UTC)

	tx, err := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		OccurredOn:    occurred,
		ReminderOn:    &reminder,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tx.SetReminderOn(nil)

	if tx.ReminderOn != nil {
		t.Fatal("expected ReminderOn to be nil after clearing")
	}
}

func TestGetReminderDaysUntil(t *testing.T) {
	today := time.Date(2025, time.March, 10, 0, 0, 0, 0, time.UTC)
	reminder := time.Date(2025, time.March, 20, 0, 0, 0, 0, time.UTC)
	occurred := time.Date(2025, time.March, 25, 0, 0, 0, 0, time.UTC)

	tx, _ := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		OccurredOn:    occurred,
		ReminderOn:    &reminder,
	})

	days := tx.GetReminderDaysUntil(today)
	if days != 10 {
		t.Fatalf("expected 10 days until reminder, got %d", days)
	}
}

func TestGetReminderDaysUntilOverdue(t *testing.T) {
	today := time.Date(2025, time.March, 25, 0, 0, 0, 0, time.UTC)
	reminder := time.Date(2025, time.March, 20, 0, 0, 0, 0, time.UTC)
	occurred := time.Date(2025, time.March, 30, 0, 0, 0, 0, time.UTC)

	tx, _ := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		OccurredOn:    occurred,
		ReminderOn:    &reminder,
	})

	days := tx.GetReminderDaysUntil(today)
	if days != -5 {
		t.Fatalf("expected -5 days (overdue), got %d", days)
	}
}

func TestGetReminderDaysUntilNoReminder(t *testing.T) {
	today := time.Date(2025, time.March, 10, 0, 0, 0, 0, time.UTC)
	occurred := time.Date(2025, time.March, 25, 0, 0, 0, 0, time.UTC)

	tx, _ := New(CreateParams{
		ProfileID:     "profile-1",
		BankAccountID: "account-1",
		Type:          TypeIncome,
		Amount:        700,
		OccurredOn:    occurred,
	})

	days := tx.GetReminderDaysUntil(today)
	if days != 0 {
		t.Fatalf("expected 0 when no reminder set, got %d", days)
	}
}

func TestShouldShowReminderAlert(t *testing.T) {
	tests := []struct {
		name       string
		daysUntil  int
		shouldShow bool
	}{
		{"10 days before", 10, true},
		{"5 days before", 5, true},
		{"1 day before", 1, true},
		{"on the day", 0, true},
		{"overdue 1 day", -1, true},
		{"overdue 5 days", -5, true},
		{"7 days before (not alert day)", 7, false},
		{"3 days before (not alert day)", 3, false},
		{"15 days before (too early)", 15, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldShowReminderAlert(tt.daysUntil)
			if result != tt.shouldShow {
				t.Errorf("ShouldShowReminderAlert(%d) = %v, want %v", tt.daysUntil, result, tt.shouldShow)
			}
		})
	}
}
