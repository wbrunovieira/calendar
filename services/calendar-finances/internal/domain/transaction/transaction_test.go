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
