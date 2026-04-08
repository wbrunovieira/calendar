package marketingcampaign

import (
	"testing"
	"time"
)

func TestNewMarketingCampaign_Valid(t *testing.T) {
	now := time.Now()
	c, err := NewMarketingCampaign(CreateParams{
		ProfileID: "profile-1",
		Name:      "Black Friday 2024",
		Platform:  PlatformMetaAds,
		StartDate: now,
		Budget:    5000.00,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == "" {
		t.Fatal("expected generated ID")
	}
	if c.Name != "Black Friday 2024" {
		t.Fatalf("expected name 'Black Friday 2024', got %q", c.Name)
	}
	if !c.IsActive {
		t.Fatal("new campaign should be active")
	}
	if c.RevenueAttributed != 0 {
		t.Fatalf("new campaign should have zero revenue attributed, got %f", c.RevenueAttributed)
	}
	if c.Leads != 0 || c.Conversions != 0 {
		t.Fatal("new campaign should have zero leads and conversions")
	}
}

func TestNewMarketingCampaign_RequiresProfileID(t *testing.T) {
	_, err := NewMarketingCampaign(CreateParams{
		Name: "Campaign", Platform: PlatformGoogleAds, StartDate: time.Now(), Budget: 100,
	})
	if err == nil {
		t.Fatal("expected error for missing profileID")
	}
}

func TestNewMarketingCampaign_RequiresName(t *testing.T) {
	_, err := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Platform: PlatformGoogleAds, StartDate: time.Now(), Budget: 100,
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestNewMarketingCampaign_RequiresValidPlatform(t *testing.T) {
	_, err := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: "TIKTOK", StartDate: time.Now(), Budget: 100,
	})
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestNewMarketingCampaign_RequiresStartDate(t *testing.T) {
	_, err := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, Budget: 100,
	})
	if err == nil {
		t.Fatal("expected error for zero start date")
	}
}

func TestNewMarketingCampaign_RequiresPositiveBudget(t *testing.T) {
	_, err := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative budget")
	}
}

func TestNewMarketingCampaign_AllPlatforms(t *testing.T) {
	platforms := []Platform{PlatformMetaAds, PlatformGoogleAds, PlatformInstagram, PlatformLinkedIn, PlatformOther}
	for _, p := range platforms {
		_, err := NewMarketingCampaign(CreateParams{
			ProfileID: "p1", Name: "Campaign", Platform: p, StartDate: time.Now(), Budget: 100,
		})
		if err != nil {
			t.Fatalf("unexpected error for platform %q: %v", p, err)
		}
	}
}

func TestMarketingCampaign_ROI(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})
	c.RevenueAttributed = 3000

	roi := c.ROI(1000)
	// (3000 - 1000) / 1000 * 100 = 200
	if roi != 200 {
		t.Fatalf("expected ROI 200, got %f", roi)
	}
}

func TestMarketingCampaign_ROI_ZeroSpent(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})
	c.RevenueAttributed = 5000

	roi := c.ROI(0)
	if roi != 0 {
		t.Fatalf("expected ROI 0 when totalSpent is 0, got %f", roi)
	}
}

func TestMarketingCampaign_CostPerLead(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})
	c.Leads = 50

	cpl := c.CostPerLead(500)
	// 500 / 50 = 10
	if cpl != 10 {
		t.Fatalf("expected CostPerLead 10, got %f", cpl)
	}
}

func TestMarketingCampaign_CostPerLead_ZeroLeads(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})

	cpl := c.CostPerLead(500)
	if cpl != 0 {
		t.Fatalf("expected CostPerLead 0 when leads is 0, got %f", cpl)
	}
}

func TestMarketingCampaign_CostPerConversion(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})
	c.Conversions = 10

	cpc := c.CostPerConversion(300)
	// 300 / 10 = 30
	if cpc != 30 {
		t.Fatalf("expected CostPerConversion 30, got %f", cpc)
	}
}

func TestMarketingCampaign_CostPerConversion_ZeroConversions(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Campaign", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})

	cpc := c.CostPerConversion(300)
	if cpc != 0 {
		t.Fatalf("expected CostPerConversion 0 when conversions is 0, got %f", cpc)
	}
}

func TestMarketingCampaign_Update(t *testing.T) {
	c, _ := NewMarketingCampaign(CreateParams{
		ProfileID: "p1", Name: "Original", Platform: PlatformMetaAds, StartDate: time.Now(), Budget: 1000,
	})
	newStart := time.Now().AddDate(0, 1, 0)
	end := time.Now().AddDate(0, 2, 0)
	notes := "Updated notes"
	err := c.Update(UpdateParams{
		Name:              "Updated",
		Platform:          PlatformGoogleAds,
		StartDate:         newStart,
		EndDate:           &end,
		Budget:            2000,
		RevenueAttributed: 5000,
		Leads:             100,
		Conversions:       20,
		Notes:             &notes,
		IsActive:          false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %q", c.Name)
	}
	if c.Platform != PlatformGoogleAds {
		t.Fatalf("expected platform GOOGLE_ADS")
	}
	if c.Leads != 100 || c.Conversions != 20 {
		t.Fatalf("expected Leads=100, Conversions=20")
	}
	if c.IsActive {
		t.Fatal("expected IsActive to be false")
	}
}
