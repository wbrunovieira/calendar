package costcenter

import "testing"

// A cost center that mirrors a CRM organization has to remember which one, or
// the link between the two systems is the name — and a name that becomes
// "Gomez Studio LTDA" on one side breaks it silently.
func TestNewCostCenter_KeepsTheExternalReference(t *testing.T) {
	ref := "26912ac0-aab7-433e-9524-d36b31df76f9"

	center, err := NewCostCenter(CreateParams{
		ProfileID:   "profile-wb",
		Name:        "Gomez Studio",
		Type:        TypeClient,
		ExternalID:  &ref,
		ExternalSrc: "wb-crm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if center.ExternalID == nil {
		t.Fatal("the external reference was dropped")
	}
	if *center.ExternalID != ref {
		t.Errorf("ExternalID = %q, want %q", *center.ExternalID, ref)
	}
	if center.ExternalSrc != "wb-crm" {
		t.Errorf("ExternalSrc = %q, want wb-crm", center.ExternalSrc)
	}
}

// The CRM's ids are not all one shape: production carries both uuid and cuid in
// the same table, because imported records bring their own. Anything that
// validates the format here breaks on the older half of the customer base.
func TestNewCostCenter_AcceptsAnyExternalIDShape(t *testing.T) {
	for _, ref := range []string{
		"26912ac0-aab7-433e-9524-d36b31df76f9", // uuid
		"cmjvntn4v0000k07mz0bxkb9w",            // cuid
		"legacy-import-00042",                  // whatever a future source uses
	} {
		center, err := NewCostCenter(CreateParams{
			ProfileID: "profile-wb", Name: "Cliente", Type: TypeClient,
			ExternalID: &ref, ExternalSrc: "wb-crm",
		})
		if err != nil {
			t.Errorf("external id %q was rejected: %v", ref, err)
			continue
		}
		if center.ExternalID == nil || *center.ExternalID != ref {
			t.Errorf("external id %q did not survive", ref)
		}
	}
}

// Most cost centers have no counterpart anywhere: a project, a department, a
// client that never entered the CRM.
func TestNewCostCenter_ExternalReferenceIsOptional(t *testing.T) {
	center, err := NewCostCenter(CreateParams{
		ProfileID: "profile-wb", Name: "Interno", Type: TypeDepartment,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if center.ExternalID != nil {
		t.Errorf("an external reference appeared out of nowhere: %q", *center.ExternalID)
	}
}
