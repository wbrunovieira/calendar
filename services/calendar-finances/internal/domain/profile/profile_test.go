package profile

import (
	"testing"
	"time"
)

func TestNewProfile_PersonalProfile(t *testing.T) {
	p, err := NewProfile("cal-1", "Bruno Pessoal", ProfileTypePersonal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Fatal("expected generated ID")
	}
	if p.Type != ProfileTypePersonal {
		t.Fatalf("expected PERSONAL, got %s", p.Type)
	}
	if p.LegalEntityType != nil {
		t.Fatal("personal profile should not have legalEntityType")
	}
}

func TestNewProfile_RequiresCalendarID(t *testing.T) {
	_, err := NewProfile("", "Bruno", ProfileTypePersonal)
	if err == nil {
		t.Fatal("expected error for empty calendarID")
	}
}

func TestNewProfile_RequiresName(t *testing.T) {
	_, err := NewProfile("cal-1", "", ProfileTypePersonal)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNewProfile_InvalidType(t *testing.T) {
	_, err := NewProfile("cal-1", "Test", "UNKNOWN")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestUpdate_BasicFields(t *testing.T) {
	p, _ := NewProfile("cal-1", "WB Digital", ProfileTypeBusiness)
	err := p.Update(UpdateParams{
		Name: "WB Digital Solutions",
		Type: ProfileTypeBusiness,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "WB Digital Solutions" {
		t.Fatalf("expected updated name, got %s", p.Name)
	}
}

func TestUpdate_PJFields(t *testing.T) {
	p, _ := NewProfile("cal-1", "WB Digital", ProfileTypeBusiness)

	letType := LegalEntityMEI
	regime := TaxRegimeSimples
	simples := true
	aliquota := 6.0
	cnpj := "12.345.678/0001-90"
	company := "WB Digital Solutions LTDA"
	opening := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	err := p.Update(UpdateParams{
		Name:            "WB Digital",
		Type:            ProfileTypeBusiness,
		LegalEntityType: &letType,
		CompanyName:     &company,
		CNPJ:            &cnpj,
		SimplesNacional: &simples,
		TaxRegime:       &regime,
		DasAliquota:     &aliquota,
		OpeningDate:     &opening,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.LegalEntityType == nil || *p.LegalEntityType != LegalEntityMEI {
		t.Fatal("expected legalEntityType MEI")
	}
	if p.TaxRegime == nil || *p.TaxRegime != TaxRegimeSimples {
		t.Fatal("expected taxRegime SIMPLES")
	}
	if !p.IsSimples() {
		t.Fatal("expected IsSimples() to be true")
	}
}

func TestUpdate_InvalidLegalEntityType(t *testing.T) {
	p, _ := NewProfile("cal-1", "WB", ProfileTypeBusiness)
	invalid := LegalEntityType("INVALID")
	err := p.Update(UpdateParams{
		Name:            "WB",
		Type:            ProfileTypeBusiness,
		LegalEntityType: &invalid,
	})
	if err == nil {
		t.Fatal("expected error for invalid legalEntityType")
	}
}

func TestUpdate_InvalidDasAliquota(t *testing.T) {
	p, _ := NewProfile("cal-1", "WB", ProfileTypeBusiness)
	bad := -1.0
	err := p.Update(UpdateParams{
		Name:        "WB",
		Type:        ProfileTypeBusiness,
		DasAliquota: &bad,
	})
	if err == nil {
		t.Fatal("expected error for negative dasAliquota")
	}
}

func TestUpdate_ClearsPJFieldsWhenNil(t *testing.T) {
	p, _ := NewProfile("cal-1", "WB", ProfileTypeBusiness)
	letType := LegalEntityMEI
	_ = p.Update(UpdateParams{Name: "WB", Type: ProfileTypeBusiness, LegalEntityType: &letType})

	// Now clear PJ fields
	_ = p.Update(UpdateParams{Name: "WB", Type: ProfileTypeBusiness})
	if p.LegalEntityType != nil {
		t.Fatal("expected legalEntityType to be cleared")
	}
}

func TestUpdate_RequiresName(t *testing.T) {
	p, _ := NewProfile("cal-1", "WB", ProfileTypeBusiness)
	err := p.Update(UpdateParams{Name: "", Type: ProfileTypeBusiness})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestIsBusiness(t *testing.T) {
	personal, _ := NewProfile("cal-1", "Bruno", ProfileTypePersonal)
	business, _ := NewProfile("cal-2", "WB", ProfileTypeBusiness)
	if personal.IsBusiness() {
		t.Fatal("personal profile should not be business")
	}
	if !business.IsBusiness() {
		t.Fatal("business profile should be business")
	}
}
