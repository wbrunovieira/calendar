package costcenter

import (
	"testing"
)

func TestNewCostCenter_Valid(t *testing.T) {
	color := "#FF5733"
	c, err := NewCostCenter(CreateParams{
		ProfileID: "profile-1",
		Name:      "Projeto Alpha",
		Type:      TypeProject,
		Color:     &color,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == "" {
		t.Fatal("expected generated ID")
	}
	if c.Name != "Projeto Alpha" {
		t.Fatalf("expected name 'Projeto Alpha', got %q", c.Name)
	}
	if !c.IsActive {
		t.Fatal("new cost center should be active")
	}
	if c.Color == nil || *c.Color != "#FF5733" {
		t.Fatal("expected color to be set")
	}
}

func TestNewCostCenter_RequiresProfileID(t *testing.T) {
	_, err := NewCostCenter(CreateParams{
		Name: "Centro", Type: TypeClient,
	})
	if err == nil {
		t.Fatal("expected error for missing profileID")
	}
}

func TestNewCostCenter_RequiresName(t *testing.T) {
	_, err := NewCostCenter(CreateParams{
		ProfileID: "p1", Type: TypeClient,
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestNewCostCenter_RequiresValidType(t *testing.T) {
	_, err := NewCostCenter(CreateParams{
		ProfileID: "p1", Name: "Centro", Type: "INVALID",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestNewCostCenter_AllTypes(t *testing.T) {
	types := []CenterType{TypeClient, TypeProject, TypeDepartment}
	for _, ct := range types {
		_, err := NewCostCenter(CreateParams{
			ProfileID: "p1", Name: "Centro", Type: ct,
		})
		if err != nil {
			t.Fatalf("unexpected error for type %q: %v", ct, err)
		}
	}
}

func TestCostCenter_Update(t *testing.T) {
	c, _ := NewCostCenter(CreateParams{
		ProfileID: "p1", Name: "Original", Type: TypeClient,
	})
	color := "#000000"
	err := c.Update(UpdateParams{
		Name:     "Updated",
		Type:     TypeDepartment,
		Color:    &color,
		IsActive: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %q", c.Name)
	}
	if c.Type != TypeDepartment {
		t.Fatalf("expected type DEPARTMENT, got %q", c.Type)
	}
	if c.IsActive {
		t.Fatal("expected IsActive to be false after update")
	}
}

func TestCostCenter_Update_RequiresName(t *testing.T) {
	c, _ := NewCostCenter(CreateParams{
		ProfileID: "p1", Name: "Centro", Type: TypeClient,
	})
	err := c.Update(UpdateParams{
		Name:     "",
		Type:     TypeClient,
		IsActive: true,
	})
	if err == nil {
		t.Fatal("expected error for missing name in update")
	}
}

func TestCostCenter_Update_RequiresValidType(t *testing.T) {
	c, _ := NewCostCenter(CreateParams{
		ProfileID: "p1", Name: "Centro", Type: TypeClient,
	})
	err := c.Update(UpdateParams{
		Name:     "Centro",
		Type:     "BOGUS",
		IsActive: true,
	})
	if err == nil {
		t.Fatal("expected error for invalid type in update")
	}
}
