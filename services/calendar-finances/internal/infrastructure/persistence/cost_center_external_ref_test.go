//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func externalRefProfile(t *testing.T, db *sql.DB) string {
	t.Helper()
	profileID := uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'External Ref', 'BUSINESS')`, profileID, "extref-"+profileID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM finance.cost_centers WHERE profile_id = $1", profileID)
		db.Exec("DELETE FROM finance.profiles WHERE id = $1", profileID)
	})
	return profileID
}

// The whole point: a cost center mirroring a CRM organization must remember
// which one, so the link is an id and not a name.
func TestCostCenterRepository_RoundTripsTheExternalReference(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID := externalRefProfile(t, db)

	repo := NewCostCenterRepository(db)
	ref := "26912ac0-aab7-433e-9524-d36b31df76f9"
	center, err := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID: profileID, Name: "Gomez Studio", Type: costcenter.TypeClient,
		ExternalID: &ref, ExternalSrc: "wb-crm",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := repo.Create(center); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByID(center.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.ExternalID == nil || *got.ExternalID != ref {
		t.Errorf("external id did not survive: %v", got.ExternalID)
	}
	if got.ExternalSrc != "wb-crm" {
		t.Errorf("ExternalSrc = %q, want wb-crm", got.ExternalSrc)
	}
}

// This is what makes a repeated sync safe: the CRM announces the same
// organization again and finds the same client, instead of a twin.
func TestCostCenterRepository_FindsAClientByItsCRMOrganization(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID := externalRefProfile(t, db)

	repo := NewCostCenterRepository(db)
	ref := "26912ac0-aab7-433e-9524-d36b31df76f9"
	center, _ := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID: profileID, Name: "Gomez Studio", Type: costcenter.TypeClient,
		ExternalID: &ref, ExternalSrc: "wb-crm",
	})
	if err := repo.Create(center); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByExternalRef(profileID, "wb-crm", ref)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if found.ID != center.ID {
		t.Errorf("found %q, want %q", found.ID, center.ID)
	}
}

// A cuid has to work exactly as well as a uuid: the CRM's table holds both,
// because imported records bring their own ids. A uuid column here would have
// rejected the older half of the customer base.
func TestCostCenterRepository_AcceptsACuidExternalReference(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID := externalRefProfile(t, db)

	repo := NewCostCenterRepository(db)
	ref := "cmjvntn4v0000k07mz0bxkb9w"
	center, _ := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID: profileID, Name: "Cliente Antigo", Type: costcenter.TypeClient,
		ExternalID: &ref, ExternalSrc: "wb-crm",
	})
	if err := repo.Create(center); err != nil {
		t.Fatalf("a cuid external id was rejected by the database: %v", err)
	}

	found, err := repo.FindByExternalRef(profileID, "wb-crm", ref)
	if err != nil || found.ID != center.ID {
		t.Errorf("cuid lookup failed: %v", err)
	}
}

// The unique index is the guard rail: two cost centers cannot claim the same
// CRM organization in one profile, whatever a retry does.
func TestCostCenterRepository_RefusesASecondClientForTheSameOrganization(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID := externalRefProfile(t, db)

	repo := NewCostCenterRepository(db)
	ref := "26912ac0-aab7-433e-9524-d36b31df76f9"
	for _, name := range []string{"Gomez Studio", "Gomez Studio LTDA"} {
		center, _ := costcenter.NewCostCenter(costcenter.CreateParams{
			ProfileID: profileID, Name: name, Type: costcenter.TypeClient,
			ExternalID: &ref, ExternalSrc: "wb-crm",
		})
		err := repo.Create(center)
		if name == "Gomez Studio" && err != nil {
			t.Fatalf("first create: %v", err)
		}
		if name == "Gomez Studio LTDA" && err == nil {
			t.Error("a twin was created for the same CRM organization")
		}
	}
}

// Cost centers with no counterpart anywhere must not collide with each other.
func TestCostCenterRepository_ManyWithoutAnExternalReferenceCoexist(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID := externalRefProfile(t, db)

	repo := NewCostCenterRepository(db)
	for _, name := range []string{"Interno", "Marketing", "P&D"} {
		center, _ := costcenter.NewCostCenter(costcenter.CreateParams{
			ProfileID: profileID, Name: name, Type: costcenter.TypeDepartment,
		})
		if err := repo.Create(center); err != nil {
			t.Fatalf("cost centers without an external reference collided: %v", err)
		}
	}
}
