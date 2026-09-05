package companyasset

import (
	"testing"
	"time"
)

func TestNewCompanyAsset_Valid(t *testing.T) {
	a, err := NewCompanyAsset(CreateParams{
		ProfileID:        "profile-1",
		Name:             "MacBook Pro M3",
		Category:         AssetCategoryHardware,
		PurchaseDate:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PurchaseAmount:   15000,
		DepreciationRate: 20, // 20% per year
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == "" {
		t.Fatal("expected generated ID")
	}
	if a.CurrentValue != 15000 {
		t.Fatalf("expected currentValue 15000 on creation, got %f", a.CurrentValue)
	}
	if !a.IsActive {
		t.Fatal("new asset should be active")
	}
}

func TestNewCompanyAsset_RequiresProfileID(t *testing.T) {
	_, err := NewCompanyAsset(CreateParams{
		Name: "PC", Category: AssetCategoryHardware,
		PurchaseDate: time.Now(), PurchaseAmount: 1000,
	})
	if err == nil {
		t.Fatal("expected error for missing profileID")
	}
}

func TestNewCompanyAsset_RequiresName(t *testing.T) {
	_, err := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Category: AssetCategoryHardware,
		PurchaseDate: time.Now(), PurchaseAmount: 1000,
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestNewCompanyAsset_InvalidCategory(t *testing.T) {
	_, err := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "thing", Category: "INVALID",
		PurchaseDate: time.Now(), PurchaseAmount: 1000,
	})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestNewCompanyAsset_RequiresPositiveAmount(t *testing.T) {
	_, err := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "PC", Category: AssetCategoryHardware,
		PurchaseDate: time.Now(), PurchaseAmount: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestNewCompanyAsset_InvalidDepreciationRate(t *testing.T) {
	_, err := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "PC", Category: AssetCategoryHardware,
		PurchaseDate: time.Now(), PurchaseAmount: 1000, DepreciationRate: 150,
	})
	if err == nil {
		t.Fatal("expected error for depreciation > 100")
	}
}

func TestEstimateCurrentValue_NoDepreciation(t *testing.T) {
	a, _ := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "Table", Category: AssetCategoryFurniture,
		PurchaseDate: time.Now(), PurchaseAmount: 2000, DepreciationRate: 0,
	})
	if a.EstimateCurrentValue() != 2000 {
		t.Fatalf("expected 2000 with no depreciation, got %f", a.EstimateCurrentValue())
	}
}

func TestEstimateCurrentValue_WithDepreciation(t *testing.T) {
	// Bought 2 years ago at 10000 with 20% annual depreciation
	// After 2 years: 10000 - (10000 * 0.20 * 2) = 10000 - 4000 = 6000
	purchaseDate := time.Now().AddDate(-2, 0, 0)
	a, _ := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "PC", Category: AssetCategoryHardware,
		PurchaseDate: purchaseDate, PurchaseAmount: 10000, DepreciationRate: 20,
	})
	estimated := a.EstimateCurrentValue()
	// Allow ±200 tolerance due to exact days calculation
	if estimated < 5800 || estimated > 6200 {
		t.Fatalf("expected ~6000 estimated value, got %f", estimated)
	}
}

func TestDispose(t *testing.T) {
	a, _ := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "PC", Category: AssetCategoryHardware,
		PurchaseDate: time.Now(), PurchaseAmount: 5000,
	})
	if err := a.Dispose(time.Now(), 3000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.IsActive {
		t.Fatal("disposed asset should not be active")
	}
	if a.DisposalDate == nil {
		t.Fatal("expected DisposalDate to be set")
	}
}

func TestUpdateCurrentValue(t *testing.T) {
	a, _ := NewCompanyAsset(CreateParams{
		ProfileID: "p1", Name: "PC", Category: AssetCategoryHardware,
		PurchaseDate: time.Now(), PurchaseAmount: 5000,
	})
	if err := a.UpdateCurrentValue(4000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.CurrentValue != 4000 {
		t.Fatalf("expected 4000, got %f", a.CurrentValue)
	}
}
