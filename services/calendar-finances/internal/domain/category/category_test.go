package category

import "testing"

func TestNewCategory(t *testing.T) {
	color := "#FFAA00"
	icon := "shopping-cart"
	parentID := "parent-123"

	cat, err := NewCategory(CreateParams{
		ProfileID: "profile-1",
		Name:      "Aluguel",
		Type:      TypeExpense,
		Color:     &color,
		Icon:      &icon,
		ParentID:  &parentID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cat.Type != TypeExpense {
		t.Fatalf("expected type expense, got %s", cat.Type)
	}

	if cat.ID == "" {
		t.Fatal("expected category to have generated ID")
	}
}

func TestNewCategoryValidation(t *testing.T) {
	_, err := NewCategory(CreateParams{})
	if err == nil {
		t.Fatal("expected error when profileID and name are empty")
	}

	color := "123456"
	_, err = NewCategory(CreateParams{
		ProfileID: "profile-1",
		Name:      "Investimentos",
		Type:      "UNKNOWN",
		Color:     &color,
	})
	if err == nil {
		t.Fatal("expected error for invalid type or color")
	}
}

func TestUpdateCategory(t *testing.T) {
	cat, err := NewCategory(CreateParams{
		ProfileID: "profile-1",
		Name:      "Receitas",
		Type:      TypeIncome,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	color := "#00FFAA"
	if err := cat.Update("Receitas Fixas", TypeIncome, &color, nil, nil, nil); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if cat.Name != "Receitas Fixas" {
		t.Fatalf("expected updated name, got %s", cat.Name)
	}
}
