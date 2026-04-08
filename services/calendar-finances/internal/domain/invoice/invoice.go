package invoice

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Status represents the lifecycle stage of an invoice
type Status string

const (
	StatusOpen   Status = "OPEN"   // Current invoice, accepting transactions
	StatusClosed Status = "CLOSED" // Invoice closed, waiting payment
	StatusPaid   Status = "PAID"   // Invoice paid
)

// Invoice represents a credit card billing cycle
type Invoice struct {
	ID            string    `json:"id"`
	BankAccountID string    `json:"bankAccountId"` // Credit card account
	ReferenceDate time.Time `json:"referenceDate"` // Month/Year reference (e.g., 2026-01-01 for January 2026)
	OpeningDate   time.Time `json:"openingDate"`   // When this invoice started accepting transactions
	ClosingDate   time.Time `json:"closingDate"`   // When this invoice closes
	DueDate       time.Time `json:"dueDate"`       // Payment due date
	Amount          float64   `json:"amount"`          // Total invoice amount (confirmed + planned)
	ConfirmedAmount float64   `json:"confirmedAmount"` // Sum of CONFIRMED transactions only
	PlannedAmount   float64   `json:"plannedAmount"`   // Sum of PLANNED transactions only
	Status          Status    `json:"status"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	PaidAmount    *float64   `json:"paidAmount,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// CreateParams contains the parameters to create a new invoice
type CreateParams struct {
	BankAccountID string
	ClosingDay    int
	DueDay        int
	ReferenceDate time.Time // The month this invoice refers to
}

// New creates a new invoice for a credit card billing cycle
func New(params CreateParams) (*Invoice, error) {
	if params.BankAccountID == "" {
		return nil, errors.New("bankAccountID is required")
	}
	if params.ClosingDay < 1 || params.ClosingDay > 31 {
		return nil, errors.New("closingDay must be between 1 and 31")
	}
	if params.DueDay < 1 || params.DueDay > 31 {
		return nil, errors.New("dueDay must be between 1 and 31")
	}
	if params.ReferenceDate.IsZero() {
		return nil, errors.New("referenceDate is required")
	}

	// Calculate opening, closing and due dates based on the reference month
	year := params.ReferenceDate.Year()
	month := params.ReferenceDate.Month()

	// Closing date is in the reference month
	closingDate := safeDate(year, month, params.ClosingDay)

	// Opening date is the day after previous closing (previous month's closing + 1)
	prevMonth := month - 1
	prevYear := year
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}
	prevClosingDate := safeDate(prevYear, prevMonth, params.ClosingDay)
	openingDate := prevClosingDate

	// Due date is typically in the same month as closing, but after closing
	// If dueDay <= closingDay, due date is in the next month
	var dueDate time.Time
	if params.DueDay > params.ClosingDay {
		dueDate = safeDate(year, month, params.DueDay)
	} else {
		nextMonth := month + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1
			nextYear++
		}
		dueDate = safeDate(nextYear, nextMonth, params.DueDay)
	}

	now := time.Now()
	return &Invoice{
		ID:            uuid.New().String(),
		BankAccountID: params.BankAccountID,
		ReferenceDate: time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
		OpeningDate:   openingDate,
		ClosingDate:   closingDate,
		DueDate:       dueDate,
		Amount:        0,
		Status:        StatusOpen,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// safeDate creates a date handling months with fewer days
func safeDate(year int, month time.Month, day int) time.Time {
	// Get the last day of the month
	firstOfNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstOfNextMonth.AddDate(0, 0, -1).Day()

	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// Close closes the invoice for new transactions
func (i *Invoice) Close() error {
	if i.Status != StatusOpen {
		return errors.New("invoice is not open")
	}
	i.Status = StatusClosed
	i.touch()
	return nil
}

// Pay marks the invoice as paid
func (i *Invoice) Pay(paidAmount float64, paidAt time.Time) error {
	if i.Status == StatusPaid {
		return errors.New("invoice is already paid")
	}
	if paidAmount <= 0 {
		return errors.New("paid amount must be greater than zero")
	}
	i.Status = StatusPaid
	i.PaidAmount = &paidAmount
	i.PaidAt = &paidAt
	i.touch()
	return nil
}

// Reopen reopens a closed invoice (not paid)
func (i *Invoice) Reopen() error {
	if i.Status != StatusClosed {
		return errors.New("can only reopen closed invoices")
	}
	i.Status = StatusOpen
	i.touch()
	return nil
}

// IsOpen returns true if the invoice is open for transactions
func (i *Invoice) IsOpen() bool {
	return i.Status == StatusOpen
}

// IsClosed returns true if the invoice is closed
func (i *Invoice) IsClosed() bool {
	return i.Status == StatusClosed
}

// IsPaid returns true if the invoice is paid
func (i *Invoice) IsPaid() bool {
	return i.Status == StatusPaid
}

// ContainsDate checks if a transaction date falls within this invoice's billing cycle
func (i *Invoice) ContainsDate(txDate time.Time) bool {
	// Normalize to date only (ignore time)
	txDateOnly := time.Date(txDate.Year(), txDate.Month(), txDate.Day(), 0, 0, 0, 0, time.UTC)
	openingOnly := time.Date(i.OpeningDate.Year(), i.OpeningDate.Month(), i.OpeningDate.Day(), 0, 0, 0, 0, time.UTC)
	closingOnly := time.Date(i.ClosingDate.Year(), i.ClosingDate.Month(), i.ClosingDate.Day(), 0, 0, 0, 0, time.UTC)

	return (txDateOnly.Equal(openingOnly) || txDateOnly.After(openingOnly)) &&
		txDateOnly.Before(closingOnly)
}

// GetReferenceName returns a human-readable name for the invoice (e.g., "Janeiro 2026")
func (i *Invoice) GetReferenceName() string {
	months := []string{
		"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
	}
	return months[i.ReferenceDate.Month()-1] + " " + string(rune('0'+i.ReferenceDate.Year()/1000)) +
		string(rune('0'+(i.ReferenceDate.Year()%1000)/100)) +
		string(rune('0'+(i.ReferenceDate.Year()%100)/10)) +
		string(rune('0'+i.ReferenceDate.Year()%10))
}

func (i *Invoice) touch() {
	i.UpdatedAt = time.Now()
}

// Repository defines the interface for invoice persistence
type Repository interface {
	Create(invoice *Invoice) error
	FindByID(id string) (*Invoice, error)
	FindByBankAccountID(bankAccountID string) ([]*Invoice, error)
	FindOpenByBankAccountID(bankAccountID string) (*Invoice, error)
	FindByBankAccountAndDate(bankAccountID string, txDate time.Time) (*Invoice, error)
	Update(invoice *Invoice) error
	Delete(id string) error
	FindOpenPastClosingDate(now time.Time) ([]*Invoice, error)
}
