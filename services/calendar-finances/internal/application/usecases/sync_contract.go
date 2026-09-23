package usecases

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/contract"
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
)

// SyncContractDeal is the deal as the sender reports it. State is COMPLETE on every
// delivery, never a delta, so a nil TotalValue means the deal has no value — not
// that the field was left out.
type SyncContractDeal struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	TotalValue *float64   `json:"totalValue"`
	Currency   string     `json:"currency"`
	Status     string     `json:"status"`
	ClosedAt   *time.Time `json:"closedAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// SyncContractOrganization is the client the deal belongs to. The sender does not
// transmit a deal without one, because a contract with no client has no cost center
// to be filed under.
type SyncContractOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SyncContractInput struct {
	Source       string                   `json:"source"`
	Deal         SyncContractDeal         `json:"deal"`
	Organization SyncContractOrganization `json:"organization"`
}

// SyncContractOutcome says what the delivery actually did, which is the only way a
// caller can tell a dropped redelivery from a write.
type SyncContractOutcome string

const (
	SyncCreated   SyncContractOutcome = "created"
	SyncUpdated   SyncContractOutcome = "updated"
	SyncUnchanged SyncContractOutcome = "unchanged"
	SyncStale     SyncContractOutcome = "stale"
)

type SyncContractOutput struct {
	ContractID   string              `json:"contractId"`
	CostCenterID string              `json:"costCenterId"`
	Outcome      SyncContractOutcome `json:"outcome"`
	Changes      []contract.Change   `json:"changes,omitempty"`
}

// SyncContractUseCase mirrors a remote deal into the ledger.
//
// It deliberately stops at the mirror: it does not create receivables. Generating
// forecast entries from this feed would be wrong today, because the sender drops any
// deal with no organization linked, and almost every OPEN deal is in that state — a
// forecast built on it would be confidently incomplete, which is worse than absent.
// What arrives complete is the CLOSED half, and that is what the mirror is for.
type SyncContractUseCase struct {
	contracts   contract.Repository
	costCenters costcenter.Repository
	// profileID is the company the CRM sells for. It is fixed at wiring time rather
	// than taken from the payload: the sender has no idea which ledger profile it is
	// feeding, and letting it name one would let a webhook write into any profile.
	profileID string
}

// ErrInvalidContractSync marks a delivery the SENDER got wrong, as opposed to a
// failure on our side. The distinction is the difference between answering 400 and
// 500, and getting it wrong tells a caller to fix a payload that was fine — or
// hides an outage behind a validation message.
var ErrInvalidContractSync = errors.New("invalid contract sync payload")

func invalidSync(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidContractSync, fmt.Sprintf(format, args...))
}

func NewSyncContractUseCase(contracts contract.Repository, costCenters costcenter.Repository, profileID string) *SyncContractUseCase {
	return &SyncContractUseCase{contracts: contracts, costCenters: costCenters, profileID: profileID}
}

func (uc *SyncContractUseCase) Execute(input SyncContractInput) (*SyncContractOutput, error) {
	source := strings.TrimSpace(input.Source)
	dealID := strings.TrimSpace(input.Deal.ID)
	orgID := strings.TrimSpace(input.Organization.ID)
	orgName := strings.TrimSpace(input.Organization.Name)

	if source == "" {
		return nil, invalidSync("source is required")
	}
	if dealID == "" {
		return nil, invalidSync("deal.id is required: it is the key the sync is idempotent on")
	}
	if orgID == "" || orgName == "" {
		return nil, invalidSync("organization.id and organization.name are required")
	}
	status, err := contract.ParseStatus(input.Deal.Status)
	if err != nil {
		return nil, invalidSync("%s", err)
	}
	if input.Deal.UpdatedAt.IsZero() {
		return nil, invalidSync("deal.updatedAt is required: without it deliveries cannot be ordered")
	}
	total, err := totalMinor(input.Deal.TotalValue)
	if err != nil {
		return nil, err
	}

	remote := contract.RemoteState{
		Title:      input.Deal.Title,
		TotalMinor: total,
		Currency:   input.Deal.Currency,
		Status:     status,
		ClosedAt:   input.Deal.ClosedAt,
		UpdatedAt:  input.Deal.UpdatedAt,
	}
	// Everything the sender is responsible for is checked BEFORE any repository is
	// touched. Validating later would leave a client created for a payload that was
	// then rejected.
	if err := contract.ValidateRemote(remote); err != nil {
		return nil, invalidSync("%s", err)
	}

	existing, err := uc.contracts.FindByExternalRef(source, dealID)
	switch {
	case err == nil:
		// Staleness is decided BEFORE anything is written, the client included. A
		// delivery that is dropped must leave nothing behind, and resolving the
		// client first meant a late redelivery naming another organization created
		// a cost center that the dropped contract then never referenced.
		if existing.IsStale(remote.UpdatedAt) {
			return &SyncContractOutput{
				ContractID:   existing.ID,
				CostCenterID: existing.CostCenterID,
				Outcome:      SyncStale,
			}, nil
		}
		centerID, err := uc.resolveCostCenter(source, orgID, orgName)
		if err != nil {
			return nil, err
		}
		return uc.applyTo(existing, centerID, remote)
	case errors.Is(err, contract.ErrNotFound):
		// fall through to create
	default:
		return nil, fmt.Errorf("reading the mirrored contract: %w", err)
	}

	centerID, err := uc.resolveCostCenter(source, orgID, orgName)
	if err != nil {
		return nil, err
	}

	created, err := contract.New(contract.CreateParams{
		ProfileID:       uc.profileID,
		CostCenterID:    centerID,
		Source:          source,
		ExternalID:      dealID,
		Title:           input.Deal.Title,
		TotalMinor:      total,
		Currency:        input.Deal.Currency,
		Status:          status,
		ClosedAt:        input.Deal.ClosedAt,
		RemoteUpdatedAt: input.Deal.UpdatedAt,
	})
	if err != nil {
		return nil, err
	}

	switch err := uc.contracts.Create(created); {
	case err == nil:
		return &SyncContractOutput{ContractID: created.ID, CostCenterID: centerID, Outcome: SyncCreated}, nil
	case errors.Is(err, contract.ErrDuplicate):
		// Another delivery for the same deal landed between the read and the insert.
		// The winner's row is the one that exists, so apply on top of it instead of
		// failing a caller whose only mistake was arriving second.
		winner, readErr := uc.contracts.FindByExternalRef(source, dealID)
		if readErr != nil {
			return nil, fmt.Errorf("re-reading the contract after losing the insert race: %w", readErr)
		}
		if winner.IsStale(remote.UpdatedAt) {
			return &SyncContractOutput{
				ContractID: winner.ID, CostCenterID: winner.CostCenterID, Outcome: SyncStale,
			}, nil
		}
		return uc.applyTo(winner, centerID, remote)
	default:
		return nil, fmt.Errorf("creating the mirrored contract: %w", err)
	}
}

func (uc *SyncContractUseCase) applyTo(existing *contract.Contract, centerID string, remote contract.RemoteState) (*SyncContractOutput, error) {
	out := &SyncContractOutput{ContractID: existing.ID}

	changes, err := existing.Apply(remote)
	if errors.Is(err, contract.ErrStale) {
		// Expected traffic, not a failure: the sender retries, and retries arrive out
		// of order. Answering an error here would make it retry harder.
		out.CostCenterID = existing.CostCenterID
		out.Outcome = SyncStale
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if moved := existing.Refile(centerID); moved != nil {
		changes = append(changes, *moved)
	}
	out.CostCenterID = existing.CostCenterID

	if err := uc.contracts.Update(existing); err != nil {
		return nil, fmt.Errorf("updating the mirrored contract: %w", err)
	}

	out.Changes = changes
	if len(changes) == 0 {
		// Nothing moved, but the ordering stamp advanced, and storing it is what makes
		// the NEXT redelivery of this same state recognisable as stale.
		out.Outcome = SyncUnchanged
		return out, nil
	}
	out.Outcome = SyncUpdated
	return out, nil
}

// resolveCostCenter finds the client this deal belongs to, creating it the first
// time the organization is seen.
//
// It deliberately does NOT rename an existing one. The cost center's name is a local
// label that appears across the ledger and that Bruno may have adjusted on purpose;
// the link that matters is the external reference, and that never changes.
func (uc *SyncContractUseCase) resolveCostCenter(source, orgID, orgName string) (string, error) {
	existing, err := uc.costCenters.FindByExternalRef(uc.profileID, source, orgID)
	switch {
	case err == nil:
		return existing.ID, nil
	case errors.Is(err, costcenter.ErrNotFound):
		// fall through to create
	default:
		return "", fmt.Errorf("resolving the client: %w", err)
	}

	created, err := costcenter.NewCostCenter(costcenter.CreateParams{
		ProfileID:   uc.profileID,
		Name:        orgName,
		Type:        costcenter.TypeClient,
		ExternalID:  &orgID,
		ExternalSrc: source,
	})
	if err != nil {
		return "", err
	}

	switch err := uc.costCenters.Create(created); {
	case err == nil:
		return created.ID, nil
	case errors.Is(err, costcenter.ErrDuplicate):
		// The first two deliveries for a brand new organization race here just as
		// they do on the contract insert, and the unique index settles it. Without
		// this the loser answered 500 and the sender burned a retry on a delivery
		// that was perfectly good.
		winner, readErr := uc.costCenters.FindByExternalRef(uc.profileID, source, orgID)
		if readErr != nil {
			return "", fmt.Errorf("re-reading the client after losing the insert race: %w", readErr)
		}
		return winner.ID, nil
	default:
		return "", fmt.Errorf("creating the client: %w", err)
	}
}

// maxContractValue bounds what a deal may be worth. It is absurdly generous for
// this company on purpose: the point is not to police pricing but to stop a
// nonsense float from reaching toMinor, which SATURATES rather than overflowing —
// 1e17 comes out as MaxInt64 and stores silently as a number nobody sent.
const maxContractValue = 1e11

func totalMinor(value *float64) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil, invalidSync("deal.totalValue is not a number")
	}
	if *value < 0 {
		return nil, invalidSync("deal.totalValue cannot be negative")
	}
	if *value > maxContractValue {
		return nil, invalidSync("deal.totalValue %v is out of range", *value)
	}
	m := toMinor(*value)
	return &m, nil
}
