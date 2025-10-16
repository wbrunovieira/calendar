package transaction

import "testing"

func TestNewSplit(t *testing.T) {
	memo := "Divisão"
	category := "cat-1"
	split, err := NewSplit(&category, 42.5, &memo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if split.Amount != 42.5 {
		t.Fatalf("expected amount 42.5, got %.2f", split.Amount)
	}

	if split.ID == "" {
		t.Fatal("expected ID to be generated")
	}

	_, err = NewSplit(nil, 0, nil)
	if err == nil {
		t.Fatal("expected validation error for zero amount")
	}
}
