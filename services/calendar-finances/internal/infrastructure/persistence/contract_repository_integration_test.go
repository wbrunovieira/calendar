//go:build integration
// +build integration

package persistence

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/brunovieira/calendar-finances/internal/domain/contract"
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
)

// dealID keeps external ids unique per run. uq_contracts_source_external is global,
// not scoped to a profile, so a fixed literal collides with whatever a previous run
// left behind and fails the NEXT test instead of its own.
func dealID(profileID, name string) string { return name + "-" + profileID }

func contractFixture(t *testing.T, db *sql.DB) (profileID, centerID string) {
	t.Helper()
	profileID = uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'Contract Repo', 'BUSINESS')`, profileID, "contract-"+profileID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM finance.contracts WHERE profile_id = $1", profileID)
		db.Exec("DELETE FROM finance.cost_centers WHERE profile_id = $1", profileID)
		db.Exec("DELETE FROM finance.profiles WHERE id = $1", profileID)
	})

	orgID := "org-" + profileID
	center, err := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID: profileID, Name: "Cliente", Type: costcenter.TypeClient,
		ExternalID: &orgID, ExternalSrc: "wb-crm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewCostCenterRepository(db).Create(center); err != nil {
		t.Fatalf("seed cost center: %v", err)
	}
	return profileID, center.ID
}

func newContract(t *testing.T, profileID, centerID, externalID string, stamp time.Time) *contract.Contract {
	t.Helper()
	total := int64(35000)
	c, err := contract.New(contract.CreateParams{
		ProfileID: profileID, CostCenterID: centerID, Source: "wb-crm",
		ExternalID: externalID, Title: "Site institucional",
		TotalMinor: &total, Currency: "BRL", Status: contract.StatusOpen,
		RemoteUpdatedAt: stamp,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// THE ONE THAT MATTERS. The ordering stamp only orders if the instant survives the
// round trip. A plain TIMESTAMP column keeps the wall clock and DISCARDS the
// offset, so 15:00-03:00 stores as 15:00 and reads back as 15:00Z — three hours
// EARLIER than what was sent. An older delivery then beats a newer one, which is
// the precise failure the stamp exists to prevent, and idempotency breaks too:
// the stored value is always behind, so nothing is ever recognised as stale.
func TestContractRepository_PreservesTheInstantAcrossOffsets(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID, centerID := contractFixture(t, db)
	repo := NewContractRepository(db)

	saoPaulo := time.FixedZone("-03", -3*60*60)
	sent := time.Date(2026, 9, 23, 15, 0, 0, 0, saoPaulo) // == 18:00Z

	c := newContract(t, profileID, centerID, dealID(profileID, "deal-offset"), sent)
	closed := sent.Add(30 * time.Minute)
	c.ClosedAt = &closed
	if err := repo.Create(c); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByExternalRef("wb-crm", dealID(profileID, "deal-offset"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !got.RemoteUpdatedAt.Equal(sent) {
		t.Fatalf("remoteUpdatedAt round-tripped as %s, want the same instant as %s (off by %s)",
			got.RemoteUpdatedAt.UTC(), sent.UTC(), got.RemoteUpdatedAt.Sub(sent))
	}
	if got.ClosedAt == nil || !got.ClosedAt.Equal(closed) {
		t.Fatalf("closedAt round-tripped as %v, want %s", got.ClosedAt, closed.UTC())
	}

	// And the consequence, stated as the rule itself: a delivery stamped one hour
	// before the stored one must still be recognised as older after the round trip.
	if !got.IsStale(sent.Add(-time.Hour)) {
		t.Fatal("an older delivery was not recognised as stale after a database round trip")
	}
	if !got.IsStale(sent) {
		t.Fatal("a redelivery of the same write was not recognised as stale after a round trip")
	}
}

func TestContractRepository_RefusesASecondMirrorOfTheSameDeal(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID, centerID := contractFixture(t, db)
	repo := NewContractRepository(db)

	stamp := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	if err := repo.Create(newContract(t, profileID, centerID, dealID(profileID, "deal-dup"), stamp)); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// A different row object, same (source, external_id): the uniqueness has to come
	// from the database, because two concurrent deliveries never see each other.
	err := repo.Create(newContract(t, profileID, centerID, dealID(profileID, "deal-dup"), stamp.Add(time.Hour)))
	if !errors.Is(err, contract.ErrDuplicate) {
		t.Fatalf("err = %v, want contract.ErrDuplicate — without it the caller cannot recover from losing the race", err)
	}
}

// The same deal id under a DIFFERENT source is a different deal. If the lookup
// ignored source, two systems feeding this table would overwrite each other.
func TestContractRepository_ScopesTheLookupBySource(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID, centerID := contractFixture(t, db)
	repo := NewContractRepository(db)

	stamp := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	if err := repo.Create(newContract(t, profileID, centerID, dealID(profileID, "deal-shared-id"), stamp)); err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err := repo.FindByExternalRef("another-crm", dealID(profileID, "deal-shared-id"))
	if !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound: the lookup matched a deal belonging to another source", err)
	}
}

// A write that changed no rows is not a successful write. Reporting nil would tell
// the sender its state was stored when the row is not there.
func TestContractRepository_UpdateOfAMissingRowIsNotSuccess(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID, centerID := contractFixture(t, db)
	repo := NewContractRepository(db)

	ghost := newContract(t, profileID, centerID, dealID(profileID, "deal-never-created"),
		time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC))

	if err := repo.Update(ghost); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// A deal with no value carries no currency either, and an empty string is not a
// currency. Stored as "" it reads back as "" and every later delivery reports a
// phantom currency change; stored as NULL it reads back absent, which is the truth.
func TestContractRepository_StoresAnAbsentCurrencyAsNull(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID, centerID := contractFixture(t, db)
	repo := NewContractRepository(db)

	c, err := contract.New(contract.CreateParams{
		ProfileID: profileID, CostCenterID: centerID, Source: "wb-crm",
		ExternalID: dealID(profileID, "deal-no-currency"), Title: "Proposta em rascunho",
		Status: contract.StatusOpen, RemoteUpdatedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(c); err != nil {
		t.Fatalf("create: %v", err)
	}

	var currency sql.NullString
	if err := db.QueryRow(`SELECT currency FROM finance.contracts
		WHERE source='wb-crm' AND external_id=$1`,
		dealID(profileID, "deal-no-currency")).Scan(&currency); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if currency.Valid {
		t.Fatalf("currency stored as %q, want NULL", currency.String)
	}
}

// uq_cost_centers_external is what settles two concurrent first deliveries for the
// same organization. Without the sentinel the loser answers 500 and the sender
// burns a retry on a delivery that was perfectly good.
func TestCostCenterRepository_RefusesASecondMirrorOfTheSameOrganization(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	profileID, _ := contractFixture(t, db)
	repo := NewCostCenterRepository(db)

	orgID := "org-race-" + profileID
	build := func() *costcenter.CostCenter {
		c, err := costcenter.NewCostCenter(costcenter.CreateParams{
			ProfileID: profileID, Name: "Cliente Novo", Type: costcenter.TypeClient,
			ExternalID: &orgID, ExternalSrc: "wb-crm",
		})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	if err := repo.Create(build()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := repo.Create(build()); !errors.Is(err, costcenter.ErrDuplicate) {
		t.Fatalf("err = %v, want costcenter.ErrDuplicate", err)
	}
}
