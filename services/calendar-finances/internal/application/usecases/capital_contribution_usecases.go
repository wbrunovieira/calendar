package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/capitalcontribution"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

// --- Create ---

type CreateCapitalContributionInput struct {
	ProfileID       string                           `json:"profileId"`
	Type            capitalcontribution.ContributionType `json:"type"`
	Amount          float64                          `json:"amount"`
	Date            time.Time                        `json:"date"`
	Description     string                           `json:"description"`
	SourceAccountID *string                          `json:"sourceAccountId,omitempty"`
	Notes           *string                          `json:"notes,omitempty"`
}

type CreateCapitalContributionUseCase struct {
	repo        capitalcontribution.Repository
	profileRepo profile.Repository
}

func NewCreateCapitalContributionUseCase(repo capitalcontribution.Repository, profileRepo profile.Repository) *CreateCapitalContributionUseCase {
	return &CreateCapitalContributionUseCase{repo: repo, profileRepo: profileRepo}
}

func (uc *CreateCapitalContributionUseCase) Execute(input CreateCapitalContributionInput) (*capitalcontribution.CapitalContribution, error) {
	if _, err := uc.profileRepo.FindByID(input.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}
	c, err := capitalcontribution.NewCapitalContribution(capitalcontribution.CreateParams{
		ProfileID:       input.ProfileID,
		Type:            input.Type,
		Amount:          input.Amount,
		Date:            input.Date,
		Description:     input.Description,
		SourceAccountID: input.SourceAccountID,
		Notes:           input.Notes,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// --- List ---

type ListCapitalContributionsUseCase struct {
	repo capitalcontribution.Repository
}

func NewListCapitalContributionsUseCase(repo capitalcontribution.Repository) *ListCapitalContributionsUseCase {
	return &ListCapitalContributionsUseCase{repo: repo}
}

func (uc *ListCapitalContributionsUseCase) Execute(profileID string) ([]*capitalcontribution.CapitalContribution, error) {
	return uc.repo.ListByProfile(profileID)
}

// --- Get ---

type GetCapitalContributionUseCase struct {
	repo capitalcontribution.Repository
}

func NewGetCapitalContributionUseCase(repo capitalcontribution.Repository) *GetCapitalContributionUseCase {
	return &GetCapitalContributionUseCase{repo: repo}
}

func (uc *GetCapitalContributionUseCase) Execute(id string) (*capitalcontribution.CapitalContribution, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCapitalContributionNotFound
	}
	return c, nil
}

// --- Update ---

type UpdateCapitalContributionInput struct {
	Type            capitalcontribution.ContributionType `json:"type"`
	Amount          float64                              `json:"amount"`
	Date            time.Time                            `json:"date"`
	Description     string                               `json:"description"`
	SourceAccountID *string                              `json:"sourceAccountId,omitempty"`
	Notes           *string                              `json:"notes,omitempty"`
	IsReturned      bool                                 `json:"isReturned"`
	ReturnedAmount  float64                              `json:"returnedAmount"`
	ReturnedAt      *time.Time                           `json:"returnedAt,omitempty"`
}

type UpdateCapitalContributionUseCase struct {
	repo capitalcontribution.Repository
}

func NewUpdateCapitalContributionUseCase(repo capitalcontribution.Repository) *UpdateCapitalContributionUseCase {
	return &UpdateCapitalContributionUseCase{repo: repo}
}

func (uc *UpdateCapitalContributionUseCase) Execute(id string, input UpdateCapitalContributionInput) (*capitalcontribution.CapitalContribution, error) {
	c, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrCapitalContributionNotFound
	}
	c.Type = input.Type
	c.Amount = input.Amount
	c.Date = input.Date
	c.Description = input.Description
	c.SourceAccountID = input.SourceAccountID
	c.Notes = input.Notes
	c.IsReturned = input.IsReturned
	c.ReturnedAmount = input.ReturnedAmount
	c.ReturnedAt = input.ReturnedAt
	if err := uc.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

// --- Delete ---

type DeleteCapitalContributionUseCase struct {
	repo capitalcontribution.Repository
}

func NewDeleteCapitalContributionUseCase(repo capitalcontribution.Repository) *DeleteCapitalContributionUseCase {
	return &DeleteCapitalContributionUseCase{repo: repo}
}

func (uc *DeleteCapitalContributionUseCase) Execute(id string) error {
	if err := uc.repo.Delete(id); err != nil {
		return ErrCapitalContributionNotFound
	}
	return nil
}

// --- Summary ---

type CapitalContributionSummary struct {
	TotalContributed float64 `json:"totalContributed"`
	TotalWithdrawn   float64 `json:"totalWithdrawn"`
	TotalLoaned      float64 `json:"totalLoaned"`
	TotalReturned    float64 `json:"totalReturned"`
	OutstandingDebt  float64 `json:"outstandingDebt"` // quanto a empresa deve ao sócio
}

type GetCapitalContributionSummaryUseCase struct {
	repo capitalcontribution.Repository
}

func NewGetCapitalContributionSummaryUseCase(repo capitalcontribution.Repository) *GetCapitalContributionSummaryUseCase {
	return &GetCapitalContributionSummaryUseCase{repo: repo}
}

func (uc *GetCapitalContributionSummaryUseCase) Execute(profileID string) (*CapitalContributionSummary, error) {
	items, err := uc.repo.ListByProfile(profileID)
	if err != nil {
		return nil, err
	}
	summary := &CapitalContributionSummary{}
	for _, c := range items {
		switch c.Type {
		case capitalcontribution.TypeContribution:
			summary.TotalContributed += c.Amount
			summary.TotalReturned += c.ReturnedAmount
			summary.OutstandingDebt += c.OutstandingBalance()
		case capitalcontribution.TypeWithdrawal:
			summary.TotalWithdrawn += c.Amount
		case capitalcontribution.TypeLoan:
			summary.TotalLoaned += c.Amount
			summary.TotalReturned += c.ReturnedAmount
			summary.OutstandingDebt += c.OutstandingBalance()
		}
	}
	return summary, nil
}
