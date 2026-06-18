package usecases

import (
	"sort"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// defaultMaturityWindowDays is the look-ahead window used when none is provided.
const defaultMaturityWindowDays = 30

// MaturityAlert is a home-screen alert for an investment that is approaching or
// past its maturity date — mirrors how planned transactions surface as alerts.
type MaturityAlert struct {
	AccountID      string    `json:"accountId"`
	Name           string    `json:"name"`
	MaturityDate   time.Time `json:"maturityDate"`
	DaysToMaturity int       `json:"daysToMaturity"` // negative when already matured
	IsMatured      bool      `json:"isMatured"`
	CurrentBalance float64   `json:"currentBalance"`
	Currency       string    `json:"currency"`
}

// ListUpcomingMaturitiesInput holds the query parameters.
type ListUpcomingMaturitiesInput struct {
	ProfileID  string
	WithinDays int       // alert look-ahead window; defaults to 30 when <= 0
	Now        time.Time // reference time; defaults to time.Now() when zero
}

// ListUpcomingMaturitiesUseCase returns active investments whose maturity date
// has passed or falls within the alert window, so the home screen can warn the
// user to redeem/reinvest.
type ListUpcomingMaturitiesUseCase struct {
	repo bankaccount.Repository
}

func NewListUpcomingMaturitiesUseCase(repo bankaccount.Repository) *ListUpcomingMaturitiesUseCase {
	return &ListUpcomingMaturitiesUseCase{repo: repo}
}

func (uc *ListUpcomingMaturitiesUseCase) Execute(input ListUpcomingMaturitiesInput) ([]MaturityAlert, error) {
	withinDays := input.WithinDays
	if withinDays <= 0 {
		withinDays = defaultMaturityWindowDays
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	today := maturityDayFloor(now)
	threshold := today.AddDate(0, 0, withinDays)

	accounts, err := uc.repo.FindByProfileID(input.ProfileID)
	if err != nil {
		return nil, err
	}

	alerts := make([]MaturityAlert, 0)
	for _, a := range accounts {
		if a == nil || !a.IsActive {
			continue
		}
		if a.Type != bankaccount.AccountTypeInvestment {
			continue
		}
		if a.MaturityDate == nil {
			continue
		}

		maturity := maturityDayFloor(*a.MaturityDate)
		if maturity.After(threshold) {
			continue // matures beyond the alert window
		}

		days := int(maturity.Sub(today).Hours() / 24)
		alerts = append(alerts, MaturityAlert{
			AccountID:      a.ID,
			Name:           a.Name,
			MaturityDate:   *a.MaturityDate,
			DaysToMaturity: days,
			IsMatured:      days < 0,
			CurrentBalance: a.CurrentBalance,
			Currency:       a.Currency,
		})
	}

	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].MaturityDate.Before(alerts[j].MaturityDate)
	})

	return alerts, nil
}

// maturityDayFloor normalizes a timestamp to midnight UTC so day-granularity
// comparisons ignore the time component.
func maturityDayFloor(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
