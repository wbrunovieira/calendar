package recurringtransaction

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status represents the current lifecycle for a recurring template.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusPaused   Status = "PAUSED"
	StatusCanceled Status = "CANCELLED"
)

// RecurringTransaction defines a scheduled financial entry template.
type RecurringTransaction struct {
	ID             string  `json:"id"`
	ProfileID      string  `json:"profileId"`
	BankAccountID  *string `json:"bankAccountId,omitempty"`
	CategoryID     *string `json:"categoryId,omitempty"`
	Type           string  `json:"type"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	Description    string  `json:"description"`
	RecurrenceRule string  `json:"recurrenceRule"`
	StartOn        time.Time
	EndOn          *time.Time
	NextOccurrence time.Time
	Status         Status
	ReviewOn       *time.Time // Date to review paused transactions
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateParams aggregates the payload to instantiate a recurring template.
type CreateParams struct {
	ProfileID      string
	BankAccountID  *string
	CategoryID     *string
	Type           string
	Amount         float64
	Currency       string
	Description    string
	RecurrenceRule string
	StartOn        time.Time
	EndOn          *time.Time
	NextOccurrence time.Time
	Status         Status
	ReviewOn       *time.Time
	Notes          *string
}

// New creates a new RecurringTransaction enforcing validation rules.
func New(params CreateParams) (*RecurringTransaction, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}

	if params.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if params.StartOn.IsZero() {
		return nil, errors.New("startOn is required")
	}

	if params.NextOccurrence.IsZero() {
		return nil, errors.New("nextOccurrence is required")
	}

	if err := validateStatus(params.Status); err != nil {
		return nil, err
	}

	recurrence := strings.TrimSpace(params.RecurrenceRule)
	if recurrence == "" {
		return nil, errors.New("recurrenceRule is required")
	}

	currency := strings.ToUpper(strings.TrimSpace(params.Currency))
	if currency == "" {
		currency = "BRL"
	}

	now := time.Now()

	return &RecurringTransaction{
		ID:             uuid.New().String(),
		ProfileID:      params.ProfileID,
		BankAccountID:  normalizeString(params.BankAccountID),
		CategoryID:     normalizeString(params.CategoryID),
		Type:           strings.ToUpper(strings.TrimSpace(params.Type)),
		Amount:         round2(params.Amount),
		Currency:       currency,
		Description:    strings.TrimSpace(params.Description),
		RecurrenceRule: recurrence,
		StartOn:        params.StartOn,
		EndOn:          normalizeTime(params.EndOn),
		NextOccurrence: params.NextOccurrence,
		Status:         params.Status,
		ReviewOn:       normalizeTime(params.ReviewOn),
		Notes:          normalizeString(params.Notes),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// Update applies mutable fields to an existing template.
func (r *RecurringTransaction) Update(params CreateParams) error {
	if r == nil {
		return errors.New("recurring transaction is nil")
	}

	updated, err := New(params)
	if err != nil {
		return err
	}

	updated.ID = r.ID
	updated.CreatedAt = r.CreatedAt
	updated.UpdatedAt = time.Now()

	*r = *updated
	return nil
}

// SetStatus changes the current status if valid.
func (r *RecurringTransaction) SetStatus(status Status) error {
	if err := validateStatus(status); err != nil {
		return err
	}
	r.Status = status
	r.UpdatedAt = time.Now()
	return nil
}

// SetReviewOn sets the review date for paused transactions.
func (r *RecurringTransaction) SetReviewOn(reviewOn *time.Time) {
	r.ReviewOn = normalizeTime(reviewOn)
	r.UpdatedAt = time.Now()
}

// CalculateNextOccurrence calculates the next occurrence date based on the
// recurrence rule and a reference date (usually today). Returns zero time
// if the transaction is paused, cancelled, or has ended.
func (r *RecurringTransaction) CalculateNextOccurrence(referenceDate time.Time) time.Time {
	// Paused or cancelled transactions have no next occurrence
	if r.Status != StatusActive {
		return time.Time{}
	}

	// Parse the recurrence rule
	rule := parseRecurrenceRule(r.RecurrenceRule)
	freq := rule["FREQ"]

	// Get the day of month from BYMONTHDAY or from StartOn
	dayOfMonth := r.StartOn.Day()
	if byMonthDay, ok := rule["BYMONTHDAY"]; ok {
		if d, err := parseInt(byMonthDay); err == nil {
			dayOfMonth = d
		}
	}

	var next time.Time

	switch freq {
	case "DAILY":
		next = r.calculateNextDaily(referenceDate)
	case "WEEKLY":
		next = r.calculateNextWeekly(referenceDate)
	case "MONTHLY":
		next = r.calculateNextMonthly(referenceDate, dayOfMonth)
	default:
		// Default to monthly if unknown frequency
		next = r.calculateNextMonthly(referenceDate, dayOfMonth)
	}

	// Check if next occurrence is after end date
	if r.EndOn != nil && !r.EndOn.IsZero() && next.After(*r.EndOn) {
		return time.Time{}
	}

	return next
}

func (r *RecurringTransaction) calculateNextDaily(referenceDate time.Time) time.Time {
	// Start from the day after reference date
	next := time.Date(
		referenceDate.Year(), referenceDate.Month(), referenceDate.Day()+1,
		0, 0, 0, 0, time.UTC,
	)
	return next
}

func (r *RecurringTransaction) calculateNextWeekly(referenceDate time.Time) time.Time {
	// Get the weekday from StartOn
	targetWeekday := r.StartOn.Weekday()

	// Find the next occurrence of that weekday after referenceDate
	next := referenceDate
	for {
		next = next.AddDate(0, 0, 1)
		if next.Weekday() == targetWeekday {
			return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
}

func (r *RecurringTransaction) calculateNextMonthly(referenceDate time.Time, dayOfMonth int) time.Time {
	// Start with the target day in the reference month
	year := referenceDate.Year()
	month := referenceDate.Month()

	// Create a date for the target day in the current month
	candidate := time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)

	// If the candidate is after the reference date, that's our next occurrence
	if candidate.After(referenceDate) {
		return candidate
	}

	// Otherwise, move to next month
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	return time.Date(nextYear, nextMonth, dayOfMonth, 0, 0, 0, 0, time.UTC)
}

func parseRecurrenceRule(rule string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(rule, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func validateStatus(status Status) error {
	switch status {
	case StatusActive, StatusPaused, StatusCanceled:
		return nil
	default:
		return errors.New("invalid status")
	}
}

func normalizeString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}

func normalizeTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	copy := *value
	return &copy
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
