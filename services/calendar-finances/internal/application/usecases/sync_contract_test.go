package usecases

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/contract"
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
)

// --- fakes ---

// fakeContractRepo stands for the SQL table, including the constraint that makes
// the sync idempotent. A fake without the uniqueness check would let a duplicate
// pass here and fail against Postgres.
type fakeContractRepo struct {
	contract.Repository
	byRef     map[string]*contract.Contract
	readErr   error
	createErr error
	creates   int
	updates   int
	// raceWinner is another delivery for the same deal landing BETWEEN our read and
	// our insert. It is modelled at the insert, not pre-seeded in byRef, because a
	// row that is already visible to the read never reaches Create at all — which is
	// how the first version of this test passed without exercising the recovery.
	raceWinner *contract.Contract
}

func newFakeContractRepo() *fakeContractRepo {
	return &fakeContractRepo{byRef: map[string]*contract.Contract{}}
}

func refKey(source, externalID string) string { return source + "|" + externalID }

func (f *fakeContractRepo) FindByExternalRef(source, externalID string) (*contract.Contract, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	if c, ok := f.byRef[refKey(source, externalID)]; ok {
		copied := *c
		return &copied, nil
	}
	return nil, contract.ErrNotFound
}

func (f *fakeContractRepo) Create(c *contract.Contract) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.raceWinner != nil {
		winner := f.raceWinner
		f.raceWinner = nil
		f.byRef[refKey(winner.Source, winner.ExternalID)] = winner
		return contract.ErrDuplicate
	}
	key := refKey(c.Source, c.ExternalID)
	if _, exists := f.byRef[key]; exists {
		return contract.ErrDuplicate
	}
	copied := *c
	f.byRef[key] = &copied
	f.creates++
	return nil
}

func (f *fakeContractRepo) Update(c *contract.Contract) error {
	key := refKey(c.Source, c.ExternalID)
	if _, exists := f.byRef[key]; !exists {
		return contract.ErrNotFound
	}
	copied := *c
	f.byRef[key] = &copied
	f.updates++
	return nil
}

func (f *fakeCostCenterRepo) FindByExternalRef(profileID, source, externalID string) (*costcenter.CostCenter, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	for _, c := range f.centers {
		if c.ProfileID == profileID && c.ExternalSrc == source && c.ExternalID != nil && *c.ExternalID == externalID {
			return c, nil
		}
	}
	return nil, costcenter.ErrNotFound
}

func (f *fakeCostCenterRepo) Create(c *costcenter.CostCenter) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.centers == nil {
		f.centers = map[string]*costcenter.CostCenter{}
	}
	// uq_cost_centers_external refuses a second client for the same external
	// reference. The fake used to accept it, which is why losing that race — a real
	// 1-in-24 event under concurrent first deliveries — was invisible here.
	if c.ExternalID != nil {
		for _, existing := range f.centers {
			if existing.ProfileID == c.ProfileID && existing.ExternalSrc == c.ExternalSrc &&
				existing.ExternalID != nil && *existing.ExternalID == *c.ExternalID {
				return costcenter.ErrDuplicate
			}
		}
	}
	// raceWinner is another delivery creating the same client BETWEEN our read and
	// our insert.
	if f.raceWinner != nil {
		winner := f.raceWinner
		f.raceWinner = nil
		f.centers[winner.ID] = winner
		return costcenter.ErrDuplicate
	}
	f.centers[c.ID] = c
	return nil
}

// --- fixture ---

const syncProfile = "profile-wb"

type syncFixture struct {
	uc        *SyncContractUseCase
	contracts *fakeContractRepo
	centers   *fakeCostCenterRepo
}

func newSyncFixture() *syncFixture {
	contracts := newFakeContractRepo()
	centers := &fakeCostCenterRepo{centers: map[string]*costcenter.CostCenter{}}
	return &syncFixture{
		uc:        NewSyncContractUseCase(contracts, centers, syncProfile),
		contracts: contracts,
		centers:   centers,
	}
}

func money(v float64) *float64 { return &v }

func baseInput() SyncContractInput {
	return SyncContractInput{
		Source: "wb-crm",
		Deal: SyncContractDeal{
			ID:         "deal-1",
			Title:      "Site institucional",
			TotalValue: money(350.00),
			Currency:   "BRL",
			Status:     "open",
			UpdatedAt:  time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
		},
		Organization: SyncContractOrganization{ID: "org-1", Name: "Refrigeracao Garrido"},
	}
}

// --- tests ---

func TestSyncCreatesTheClientAndTheContractOnFirstDelivery(t *testing.T) {
	f := newSyncFixture()

	out, err := f.uc.Execute(baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Outcome != SyncCreated {
		t.Fatalf("outcome = %q, want created", out.Outcome)
	}
	if out.CostCenterID == "" || out.ContractID == "" {
		t.Fatal("the sync answered without the ids it just created")
	}
	center := f.centers.centers[out.CostCenterID]
	if center == nil {
		t.Fatal("no cost center was created for the organization")
	}
	if center.Type != costcenter.TypeClient || center.Name != "Refrigeracao Garrido" {
		t.Fatalf("cost center = %+v", center)
	}
	if center.ExternalID == nil || *center.ExternalID != "org-1" || center.ExternalSrc != "wb-crm" {
		t.Fatal("the cost center was created without the reference that lets the next sync find it")
	}
	stored := f.contracts.byRef[refKey("wb-crm", "deal-1")]
	if stored.TotalMinor == nil || *stored.TotalMinor != 35000 {
		t.Fatalf("totalMinor = %v, want 35000", stored.TotalMinor)
	}
	if stored.CostCenterID != center.ID {
		t.Fatal("the contract was not filed under the client")
	}
}

func TestSyncReusesTheClientOnASecondDeal(t *testing.T) {
	f := newSyncFixture()
	if _, err := f.uc.Execute(baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second := baseInput()
	second.Deal.ID = "deal-2"
	second.Deal.Title = "Manutencao mensal"
	out, err := f.uc.Execute(second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.centers.centers) != 1 {
		t.Fatalf("%d cost centers; the same organization produced a twin", len(f.centers.centers))
	}
	if out.Outcome != SyncCreated {
		t.Fatalf("outcome = %q, want created", out.Outcome)
	}
}

// Redelivering the very same payload is expected traffic, not an error, and it
// must not write anything.
func TestSyncIsIdempotentOnARedelivery(t *testing.T) {
	f := newSyncFixture()
	first, err := f.uc.Execute(baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	again, err := f.uc.Execute(baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if again.Outcome != SyncStale {
		t.Fatalf("outcome = %q, want stale", again.Outcome)
	}
	if again.ContractID != first.ContractID {
		t.Fatal("the redelivery produced a second contract")
	}
	if f.contracts.creates != 1 || f.contracts.updates != 0 {
		t.Fatalf("creates=%d updates=%d; a redelivery wrote", f.contracts.creates, f.contracts.updates)
	}
}

func TestSyncDropsADeliveryThatArrivedOutOfOrder(t *testing.T) {
	f := newSyncFixture()
	newer := baseInput()
	newer.Deal.TotalValue = money(420.00)
	newer.Deal.Status = "won"
	newer.Deal.UpdatedAt = time.Date(2026, 9, 23, 14, 0, 0, 0, time.UTC)
	if _, err := f.uc.Execute(newer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	older := baseInput() // 12:00, value 350, still open
	out, err := f.uc.Execute(older)
	if err != nil {
		t.Fatalf("a late redelivery is expected traffic, not a failure: %v", err)
	}
	if out.Outcome != SyncStale {
		t.Fatalf("outcome = %q, want stale", out.Outcome)
	}
	stored := f.contracts.byRef[refKey("wb-crm", "deal-1")]
	if stored.Status != contract.StatusWon || *stored.TotalMinor != 42000 {
		t.Fatalf("an older delivery overwrote newer state: %+v", stored)
	}
}

func TestSyncAppliesAndReportsARealChange(t *testing.T) {
	f := newSyncFixture()
	if _, err := f.uc.Execute(baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	won := baseInput()
	won.Deal.Status = "won"
	won.Deal.TotalValue = money(420.00)
	closed := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	won.Deal.ClosedAt = &closed
	won.Deal.UpdatedAt = closed

	out, err := f.uc.Execute(won)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Outcome != SyncUpdated {
		t.Fatalf("outcome = %q, want updated", out.Outcome)
	}
	fields := map[string]contract.Change{}
	for _, c := range out.Changes {
		fields[c.Field] = c
	}
	if fields["status"].To != "WON" || fields["totalValue"].To != "42000" {
		t.Fatalf("changes = %+v", out.Changes)
	}
	stored := f.contracts.byRef[refKey("wb-crm", "deal-1")]
	if stored.Status != contract.StatusWon || stored.ClosedAt == nil {
		t.Fatalf("stored = %+v", stored)
	}
}

// A sale that falls through after being closed has to arrive, or the ledger keeps
// forecasting cash that will never come — worse than no forecast, because it looks
// normal.
func TestSyncAcceptsALostDeal(t *testing.T) {
	f := newSyncFixture()
	if _, err := f.uc.Execute(baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lost := baseInput()
	lost.Deal.Status = "lost"
	lost.Deal.UpdatedAt = baseInput().Deal.UpdatedAt.Add(time.Hour)
	out, err := f.uc.Execute(lost)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Outcome != SyncUpdated {
		t.Fatalf("outcome = %q, want updated", out.Outcome)
	}
	if f.contracts.byRef[refKey("wb-crm", "deal-1")].Status != contract.StatusLost {
		t.Fatal("the lost status was not stored")
	}
}

// A newer stamp carrying identical state advances the ordering mark, so the NEXT
// redelivery of that state is recognised as stale instead of being applied again.
func TestSyncStoresANewerStampEvenWhenNothingElseMoved(t *testing.T) {
	f := newSyncFixture()
	if _, err := f.uc.Execute(baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	touched := baseInput()
	touched.Deal.UpdatedAt = baseInput().Deal.UpdatedAt.Add(time.Hour)
	out, err := f.uc.Execute(touched)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Outcome != SyncUnchanged {
		t.Fatalf("outcome = %q, want unchanged", out.Outcome)
	}
	if len(out.Changes) != 0 {
		t.Fatalf("changes = %+v, want none", out.Changes)
	}
	stored := f.contracts.byRef[refKey("wb-crm", "deal-1")]
	if !stored.RemoteUpdatedAt.Equal(touched.Deal.UpdatedAt) {
		t.Fatal("the ordering stamp was not advanced")
	}
}

func TestSyncRejectsWhatItCannotFile(t *testing.T) {
	cases := map[string]func(*SyncContractInput){
		"no source":            func(i *SyncContractInput) { i.Source = " " },
		"no deal id":           func(i *SyncContractInput) { i.Deal.ID = "" },
		"no organization id":   func(i *SyncContractInput) { i.Organization.ID = "" },
		"no organization name": func(i *SyncContractInput) { i.Organization.Name = "  " },
		"no updatedAt":         func(i *SyncContractInput) { i.Deal.UpdatedAt = time.Time{} },
		"unknown status":       func(i *SyncContractInput) { i.Deal.Status = "abandoned" },
		"negative value":       func(i *SyncContractInput) { i.Deal.TotalValue = money(-1) },
		"no title":             func(i *SyncContractInput) { i.Deal.Title = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newSyncFixture()
			in := baseInput()
			mutate(&in)
			_, err := f.uc.Execute(in)
			if err == nil {
				t.Fatalf("%s was accepted", name)
			}
			// The caller answers 400 or 500 off this classification. A payload fault
			// reported as a server fault tells the sender to retry forever.
			if !errors.Is(err, ErrInvalidContractSync) {
				t.Fatalf("%s: err = %v, want ErrInvalidContractSync", name, err)
			}
			if f.contracts.creates != 0 || len(f.centers.centers) != 0 {
				t.Fatal("a rejected delivery still wrote")
			}
		})
	}
}

// A database that is down must not look like a client that does not exist yet.
func TestSyncDoesNotInventAClientWhenTheReadFails(t *testing.T) {
	f := newSyncFixture()
	f.centers.readErr = errors.New("dial tcp: connection refused")

	if _, err := f.uc.Execute(baseInput()); err == nil {
		t.Fatal("a failed read was accepted")
	}
	if len(f.centers.centers) != 0 {
		t.Fatal("a failed read created a duplicate client")
	}
	if errors.Is(f.centers.readErr, ErrInvalidContractSync) {
		t.Fatal("fixture is wrong: the simulated outage must not be a payload fault")
	}
	if f.contracts.creates != 0 {
		t.Fatal("a contract was filed under a client that was never resolved")
	}
}

func TestSyncDoesNotWriteWhenTheContractReadFails(t *testing.T) {
	f := newSyncFixture()
	f.contracts.readErr = errors.New("dial tcp: connection refused")

	if _, err := f.uc.Execute(baseInput()); err == nil {
		t.Fatal("a failed read was accepted")
	}
	if f.contracts.creates != 0 || f.contracts.updates != 0 {
		t.Fatal("a failed read still wrote")
	}
}

// Two deliveries for the same deal can race. The loser of the insert must fall
// back to reading what the winner wrote and apply on top, not fail the caller.
func TestSyncRecoversWhenAConcurrentDeliveryWinsTheInsert(t *testing.T) {
	f := newSyncFixture()

	winnerInput := baseInput()
	f.contracts.raceWinner = mustContract(t, winnerInput)

	later := baseInput()
	later.Deal.Status = "won"
	later.Deal.UpdatedAt = winnerInput.Deal.UpdatedAt.Add(time.Hour)

	out, err := f.uc.Execute(later)
	if err != nil {
		t.Fatalf("losing the insert race failed the caller: %v", err)
	}
	if out.Outcome != SyncUpdated {
		t.Fatalf("outcome = %q, want updated", out.Outcome)
	}
	stored := f.contracts.byRef[refKey("wb-crm", "deal-1")]
	if stored.Status != contract.StatusWon {
		t.Fatalf("status = %q; the later state was lost after the race", stored.Status)
	}
	if f.contracts.updates != 1 {
		t.Fatalf("updates = %d; the recovery did not write through the winner's row", f.contracts.updates)
	}
}

func mustContract(t *testing.T, in SyncContractInput) *contract.Contract {
	t.Helper()
	total := toMinor(*in.Deal.TotalValue)
	status, err := contract.ParseStatus(in.Deal.Status)
	if err != nil {
		t.Fatal(err)
	}
	c, err := contract.New(contract.CreateParams{
		ProfileID: syncProfile, CostCenterID: "cc-preexisting",
		Source: in.Source, ExternalID: in.Deal.ID, Title: in.Deal.Title,
		TotalMinor: &total, Currency: in.Deal.Currency, Status: status,
		RemoteUpdatedAt: in.Deal.UpdatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Money crosses the boundary as a float and is stored as an integer. The rounding
// has to be the same one the rest of the ledger uses, or a R$ 350,35 contract
// stores 35034 and never matches the receivable.
func TestSyncStoresMoneyInMinorUnitsWithoutDrift(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want int64
	}{{350.35, 35035}, {0.07, 7}, {1234.56, 123456}, {0, 0}} {
		f := newSyncFixture()
		in := baseInput()
		in.Deal.TotalValue = money(tc.in)
		if _, err := f.uc.Execute(in); err != nil {
			t.Fatalf("%v: %v", tc.in, err)
		}
		got := f.contracts.byRef[refKey("wb-crm", "deal-1")].TotalMinor
		if got == nil || *got != tc.want {
			t.Fatalf("%v stored as %v, want %d", tc.in, got, tc.want)
		}
	}
}

func TestSyncAcceptsADealWithNoValueYet(t *testing.T) {
	f := newSyncFixture()
	in := baseInput()
	in.Deal.TotalValue = nil
	if _, err := f.uc.Execute(in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.contracts.byRef[refKey("wb-crm", "deal-1")].TotalMinor != nil {
		t.Fatal("a valueless deal was stored as zero, which a forecast would read as free work")
	}
}

// A failure on our side must never be classified as the sender's fault, or a
// database outage answers 400 and the CRM stops retrying a delivery that would have
// succeeded a minute later.
func TestSyncClassifiesOurOwnFailuresAsOurs(t *testing.T) {
	for name, breaks := range map[string]func(*syncFixture){
		"cost center read":  func(f *syncFixture) { f.centers.readErr = errors.New("connection refused") },
		"cost center write": func(f *syncFixture) { f.centers.createErr = errors.New("disk full") },
		"contract read":     func(f *syncFixture) { f.contracts.readErr = errors.New("connection refused") },
		"contract write":    func(f *syncFixture) { f.contracts.createErr = errors.New("disk full") },
	} {
		t.Run(name, func(t *testing.T) {
			f := newSyncFixture()
			breaks(f)
			_, err := f.uc.Execute(baseInput())
			if err == nil {
				t.Fatal("a broken repository was accepted")
			}
			if errors.Is(err, ErrInvalidContractSync) {
				t.Fatalf("our failure was blamed on the sender: %v", err)
			}
		})
	}
}

// A deal moved to another organization in the CRM has to be re-filed here, and the
// move has to appear in the changes. Leaving it under the previous client files one
// company's revenue under another's, and nothing says so.
func TestSyncRefilesADealThatChangedClient(t *testing.T) {
	f := newSyncFixture()
	first, err := f.uc.Execute(baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	moved := baseInput()
	moved.Organization = SyncContractOrganization{ID: "org-2", Name: "Outro Cliente"}
	moved.Deal.Status = "won"
	moved.Deal.UpdatedAt = baseInput().Deal.UpdatedAt.Add(time.Hour)

	out, err := f.uc.Execute(moved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CostCenterID == first.CostCenterID {
		t.Fatal("the contract is still filed under the previous client")
	}
	var reported bool
	for _, c := range out.Changes {
		if c.Field == "costCenterId" && c.From == first.CostCenterID && c.To == out.CostCenterID {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the client changed without being reported: %+v", out.Changes)
	}
	if stored := f.contracts.byRef[refKey("wb-crm", "deal-1")]; stored.CostCenterID != out.CostCenterID {
		t.Fatal("the move was reported but not stored")
	}
}

// "Dropped without writing" has to mean the client too. Resolving it before the
// staleness check left a cost center behind for a delivery that was then discarded.
func TestSyncStaleDeliveryCreatesNoClient(t *testing.T) {
	f := newSyncFixture()
	newer := baseInput()
	newer.Deal.UpdatedAt = baseInput().Deal.UpdatedAt.Add(time.Hour)
	if _, err := f.uc.Execute(newer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clientsAfterFirst := len(f.centers.centers)

	late := baseInput() // older stamp, and naming a different organization
	late.Organization = SyncContractOrganization{ID: "org-2", Name: "Cliente que nao deveria nascer"}

	out, err := f.uc.Execute(late)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Outcome != SyncStale {
		t.Fatalf("outcome = %q, want stale", out.Outcome)
	}
	if len(f.centers.centers) != clientsAfterFirst {
		t.Fatalf("%d clients, was %d: a dropped delivery still wrote",
			len(f.centers.centers), clientsAfterFirst)
	}
}

// Two first deliveries for the same brand new organization race on the client
// insert exactly as they do on the contract insert. The loser must read the
// winner's client, not answer 500 and make the sender burn a retry.
func TestSyncRecoversWhenAConcurrentDeliveryCreatesTheClientFirst(t *testing.T) {
	f := newSyncFixture()

	orgID := "org-1"
	winner, err := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID: syncProfile, Name: "Refrigeracao Garrido", Type: costcenter.TypeClient,
		ExternalID: &orgID, ExternalSrc: "wb-crm",
	})
	if err != nil {
		t.Fatal(err)
	}
	f.centers.raceWinner = winner

	out, err := f.uc.Execute(baseInput())
	if err != nil {
		t.Fatalf("losing the client insert race failed the caller: %v", err)
	}
	if out.CostCenterID != winner.ID {
		t.Fatalf("costCenterID = %q, want the winner's %q", out.CostCenterID, winner.ID)
	}
	if len(f.centers.centers) != 1 {
		t.Fatalf("%d clients; the race produced a twin", len(f.centers.centers))
	}
}

// A currency the column cannot hold is the sender's fault, not an outage. Before
// this it reached Postgres, came back as a driver error, and was answered 500 —
// which the documented protocol tells the CRM to retry forever.
func TestSyncRejectsACurrencyThatIsNotACode(t *testing.T) {
	for _, bad := range []string{"DOLLAR", "BR", "R$"} {
		f := newSyncFixture()
		in := baseInput()
		in.Deal.Currency = bad
		in.Deal.TotalValue = nil // the old check only ran when there was a value

		_, err := f.uc.Execute(in)
		if err == nil {
			t.Fatalf("currency %q was accepted", bad)
		}
		if !errors.Is(err, ErrInvalidContractSync) {
			t.Fatalf("currency %q: err = %v, want ErrInvalidContractSync", bad, err)
		}
		if f.contracts.creates != 0 || len(f.centers.centers) != 0 {
			t.Fatalf("currency %q: a rejected delivery still wrote", bad)
		}
	}
}

// toMinor SATURATES rather than overflowing: 1e17 comes back as MaxInt64 and would
// store silently as a number nobody sent. Nothing bounded the value before, so an
// absurd payload became an absurd contract instead of a rejection.
func TestSyncRejectsAValueThatCannotBeRepresented(t *testing.T) {
	for _, bad := range []float64{1e17, 1e12, math.Inf(1), math.NaN()} {
		f := newSyncFixture()
		in := baseInput()
		in.Deal.TotalValue = &bad

		_, err := f.uc.Execute(in)
		if err == nil {
			t.Fatalf("totalValue %v was accepted", bad)
		}
		if !errors.Is(err, ErrInvalidContractSync) {
			t.Fatalf("totalValue %v: err = %v, want ErrInvalidContractSync", bad, err)
		}
		if f.contracts.creates != 0 {
			t.Fatalf("totalValue %v: a rejected delivery still wrote", bad)
		}
	}
	// The bound must not reject real money.
	f := newSyncFixture()
	in := baseInput()
	in.Deal.TotalValue = money(1_000_000.00)
	if _, err := f.uc.Execute(in); err != nil {
		t.Fatalf("a million reais was rejected: %v", err)
	}
}
