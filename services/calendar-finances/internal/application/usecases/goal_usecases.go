package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/goal"
)

type GoalInput struct {
	ProfileID    string  `json:"profileId"`
	CategoryID   *string `json:"categoryId,omitempty"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	TargetAmount float64 `json:"targetAmount"`
	Priority     string  `json:"priority"`
	TargetDate   *string `json:"targetDate,omitempty"`
	Link         string  `json:"link,omitempty"`
	GoalType     string  `json:"goalType,omitempty"`
}

type GoalAddAmountInput struct {
	Amount float64 `json:"amount"`
}

type GoalsService struct {
	repo goal.Repository
}

func NewGoalsService(repo goal.Repository) *GoalsService {
	return &GoalsService{repo: repo}
}

func (s *GoalsService) Create(input GoalInput) (*goal.Goal, error) {
	if strings.TrimSpace(input.ProfileID) == "" || strings.TrimSpace(input.Name) == "" {
		return nil, ErrInvalidInput
	}

	var targetDate *time.Time
	if input.TargetDate != nil && *input.TargetDate != "" {
		t, err := parseDate(*input.TargetDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		targetDate = &t
	}

	priority := goal.Priority(strings.ToUpper(input.Priority))
	if priority == "" {
		priority = goal.PriorityMedium
	}

	goalType := goal.GoalType(strings.ToUpper(input.GoalType))
	if goalType == "" {
		goalType = goal.GoalTypePersonalSavings
	}

	g, err := goal.New(goal.CreateParams{
		ProfileID:    input.ProfileID,
		CategoryID:   input.CategoryID,
		Name:         input.Name,
		Description:  input.Description,
		TargetAmount: input.TargetAmount,
		Priority:     priority,
		TargetDate:   targetDate,
		Link:         input.Link,
		GoalType:     goalType,
	})
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *GoalsService) Update(id string, input GoalInput) (*goal.Goal, error) {
	g, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, ErrGoalNotFound
	}

	g.Name = input.Name
	g.Description = input.Description
	g.TargetAmount = input.TargetAmount
	g.Link = input.Link
	g.CategoryID = input.CategoryID

	if input.Priority != "" {
		g.Priority = goal.Priority(strings.ToUpper(input.Priority))
	}

	if input.GoalType != "" {
		g.GoalType = goal.GoalType(strings.ToUpper(input.GoalType))
	}

	if input.TargetDate != nil && *input.TargetDate != "" {
		t, err := parseDate(*input.TargetDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		g.TargetDate = &t
	} else {
		g.TargetDate = nil
	}

	g.UpdatedAt = time.Now()

	if err := s.repo.Update(g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *GoalsService) List(profileID string) ([]*goal.Goal, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.ListByProfile(profileID)
}

func (s *GoalsService) AddAmount(id string, input GoalAddAmountInput) (*goal.Goal, error) {
	g, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, ErrGoalNotFound
	}

	if err := g.AddAmount(input.Amount); err != nil {
		return nil, err
	}

	if err := s.repo.Update(g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *GoalsService) UpdateStatus(id string, status string) (*goal.Goal, error) {
	g, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, ErrGoalNotFound
	}

	switch goal.Status(strings.ToUpper(status)) {
	case goal.StatusCompleted:
		if err := g.Complete(); err != nil {
			return nil, err
		}
	case goal.StatusCancelled:
		if err := g.Cancel(); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidInput
	}

	if err := s.repo.Update(g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *GoalsService) Delete(id string) error {
	return s.repo.Delete(id)
}
