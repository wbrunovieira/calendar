package usecases

import (
	"math"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/budgettarget"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type BudgetTargetInput struct {
	ProfileID  string  `json:"profileId"`
	CategoryID string  `json:"categoryId"`
	Period     string  `json:"period"`
	Amount     float64 `json:"amount"`
	Notes      *string `json:"notes,omitempty"`
}

type BudgetTargetPresenter struct {
	ID          string    `json:"id"`
	ProfileID   string    `json:"profileId"`
	CategoryID  string    `json:"categoryId"`
	PeriodStart time.Time `json:"periodStart"`
	Amount      float64   `json:"amount"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type BudgetSummaryItem struct {
	Target    *BudgetTargetPresenter `json:"target"`
	Spent     float64                `json:"spent"`
	Remaining float64                `json:"remaining"`
}

type BudgetTargetsService struct {
	repo             budgettarget.Repository
	transactionsRepo transaction.Repository
}

func NewBudgetTargetsService(repo budgettarget.Repository, txRepo transaction.Repository) *BudgetTargetsService {
	return &BudgetTargetsService{repo: repo, transactionsRepo: txRepo}
}

func (s *BudgetTargetsService) Create(input BudgetTargetInput) (*BudgetTargetPresenter, error) {
	entity, err := s.buildEntity(input)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(entity); err != nil {
		return nil, err
	}

	return presentBudget(entity), nil
}

func (s *BudgetTargetsService) Update(id string, input BudgetTargetInput) (*BudgetTargetPresenter, error) {
	entity, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrBudgetTargetNotFound
	}

	params, err := s.parseParams(input)
	if err != nil {
		return nil, err
	}

	if err := entity.Update(params); err != nil {
		return nil, err
	}

	if err := s.repo.Update(entity); err != nil {
		return nil, err
	}

	return presentBudget(entity), nil
}

func (s *BudgetTargetsService) List(profileID string) ([]*BudgetTargetPresenter, error) {
	items, err := s.repo.ListByProfile(profileID)
	if err != nil {
		return nil, err
	}

	var result []*BudgetTargetPresenter
	for _, item := range items {
		result = append(result, presentBudget(item))
	}
	return result, nil
}

func (s *BudgetTargetsService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return ErrBudgetTargetNotFound
	}
	return nil
}

func (s *BudgetTargetsService) Summary(profileID, period string) ([]*BudgetSummaryItem, error) {
	periodStart, periodEnd, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}

	targets, err := s.repo.ListByProfileAndPeriod(profileID, periodStart)
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return []*BudgetSummaryItem{}, nil
	}

	categoryIDs := make([]string, 0, len(targets))
	for _, target := range targets {
		categoryIDs = append(categoryIDs, target.CategoryID)
	}

	sums, err := s.transactionsRepo.SumByCategories(profileID, categoryIDs, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	var summary []*BudgetSummaryItem
	for _, target := range targets {
		spent := sums[target.CategoryID]
		summary = append(summary, &BudgetSummaryItem{
			Target:    presentBudget(target),
			Spent:     spent,
			Remaining: round2(target.Amount - spent),
		})
	}

	return summary, nil
}

func (s *BudgetTargetsService) buildEntity(input BudgetTargetInput) (*budgettarget.BudgetTarget, error) {
	params, err := s.parseParams(input)
	if err != nil {
		return nil, err
	}
	return budgettarget.NewBudgetTarget(params)
}

func (s *BudgetTargetsService) parseParams(input BudgetTargetInput) (budgettarget.CreateParams, error) {
	periodStart, _, err := parsePeriod(input.Period)
	if err != nil {
		return budgettarget.CreateParams{}, err
	}

	return budgettarget.CreateParams{
		ProfileID:   strings.TrimSpace(input.ProfileID),
		CategoryID:  strings.TrimSpace(input.CategoryID),
		PeriodStart: periodStart,
		Amount:      input.Amount,
		Notes:       normalizePtr(input.Notes),
	}, nil
}

func presentBudget(entity *budgettarget.BudgetTarget) *BudgetTargetPresenter {
	return &BudgetTargetPresenter{
		ID:          entity.ID,
		ProfileID:   entity.ProfileID,
		CategoryID:  entity.CategoryID,
		PeriodStart: entity.PeriodStart,
		Amount:      entity.Amount,
		Notes:       entity.Notes,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

func parsePeriod(period string) (time.Time, time.Time, error) {
	trimmed := strings.TrimSpace(period)
	if trimmed == "" {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, -1), nil
	}

	parsed, err := time.Parse("2006-01", trimmed)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}
	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return start, end, nil
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
