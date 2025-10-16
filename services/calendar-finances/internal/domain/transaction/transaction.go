package transaction

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Type represents the direction of the cash flow.
type Type string

const (
	TypeIncome   Type = "INCOME"
	TypeExpense  Type = "EXPENSE"
	TypeTransfer Type = "TRANSFER"
)

// Status represents the lifecycle stage of a transaction.
type Status string

const (
	StatusPlanned   Status = "PLANNED"
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)

// Transaction encapsulates the financial movement registered in the system.
type Transaction struct {
	ID                   string     `json:"id"`
	ProfileID            string     `json:"profileId"`
	BankAccountID        string     `json:"bankAccountId"`
	DestinationAccountID *string    `json:"destinationAccountId,omitempty"`
	CategoryID           *string    `json:"categoryId,omitempty"`
	Type                 Type       `json:"type"`
	Status               Status     `json:"status"`
	Amount               float64    `json:"amount"`
	Currency             string     `json:"currency"`
	Description          string     `json:"description"`
	Notes                *string    `json:"notes,omitempty"`
	CostCenter           *string    `json:"costCenter,omitempty"`
	OccurredOn           time.Time  `json:"occurredOn"`
	DueOn                *time.Time `json:"dueOn,omitempty"`
	RecurrenceRule       *string    `json:"recurrenceRule,omitempty"`
	InstallmentNumber    *int       `json:"installmentNumber,omitempty"`
	InstallmentTotal     *int       `json:"installmentTotal,omitempty"`
	ExternalID           *string    `json:"externalId,omitempty"`
	Tags                 []string   `json:"tags,omitempty"`
	Splits               []*Split   `json:"splits,omitempty"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CreateParams represents the attributes required to instantiate a transaction.
type CreateParams struct {
	ProfileID            string
	BankAccountID        string
	DestinationAccountID *string
	CategoryID           *string
	Type                 Type
	Amount               float64
	Currency             string
	Description          string
	Notes                *string
	CostCenter           *string
	OccurredOn           time.Time
	DueOn                *time.Time
	RecurrenceRule       *string
	InstallmentNumber    *int
	InstallmentTotal     *int
	ExternalID           *string
	Tags                 []string
	Splits               []*Split
}

// New validates and constructs a Transaction ready to be persisted.
func New(params CreateParams) (*Transaction, error) {
	if strings.TrimSpace(params.ProfileID) == "" {
		return nil, errors.New("profileID is required")
	}
	if strings.TrimSpace(params.BankAccountID) == "" {
		return nil, errors.New("bankAccountID is required")
	}
	if params.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if params.OccurredOn.IsZero() {
		return nil, errors.New("occurredOn is required")
	}

	if err := validateType(params.Type); err != nil {
		return nil, err
	}

	if params.Type == TypeTransfer {
		if params.DestinationAccountID == nil || strings.TrimSpace(*params.DestinationAccountID) == "" {
			return nil, errors.New("destinationAccountID is required for transfers")
		}
		if *params.DestinationAccountID == params.BankAccountID {
			return nil, errors.New("destination account must differ from source account")
		}
	}

	currency := strings.ToUpper(strings.TrimSpace(params.Currency))
	if currency == "" {
		currency = "BRL"
	}

	description := strings.TrimSpace(params.Description)

	now := time.Now()
	transaction := &Transaction{
		ID:                   uuid.New().String(),
		ProfileID:            strings.TrimSpace(params.ProfileID),
		BankAccountID:        strings.TrimSpace(params.BankAccountID),
		DestinationAccountID: cloneString(params.DestinationAccountID),
		CategoryID:           cloneString(params.CategoryID),
		Type:                 params.Type,
		Status:               StatusPlanned,
		Amount:               round2(params.Amount),
		Currency:             currency,
		Description:          description,
		Notes:                cloneString(params.Notes),
		CostCenter:           cloneString(params.CostCenter),
		OccurredOn:           params.OccurredOn,
		DueOn:                cloneTime(params.DueOn),
		RecurrenceRule:       cloneString(params.RecurrenceRule),
		InstallmentNumber:    cloneInt(params.InstallmentNumber),
		InstallmentTotal:     cloneInt(params.InstallmentTotal),
		ExternalID:           cloneString(params.ExternalID),
		Tags:                 sanitizeTags(params.Tags),
		Splits:               []*Split{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := transaction.setSplits(params.Splits); err != nil {
		return nil, err
	}

	if err := transaction.validateInstallments(); err != nil {
		return nil, err
	}

	return transaction, nil
}

// Confirm marks the transaction as executed.
func (t *Transaction) Confirm(occurredOn time.Time) error {
	if t == nil {
		return errors.New("transaction is nil")
	}
	if occurredOn.IsZero() {
		return errors.New("occurredOn is required")
	}
	t.Status = StatusConfirmed
	t.OccurredOn = occurredOn
	t.touch()
	return nil
}

// Cancel voids a transaction without removing historical data.
func (t *Transaction) Cancel(reason string) {
	if t == nil {
		return
	}
	if trimmed := strings.TrimSpace(reason); trimmed != "" {
		t.Notes = &trimmed
	}
	t.Status = StatusCancelled
	t.touch()
}

// UpdateAmount adjusts the total amount and validates the existing splits.
func (t *Transaction) UpdateAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	t.Amount = round2(amount)
	if sum := t.splitsTotal(); sum > t.Amount+0.01 {
		return fmt.Errorf("sum of splits (%.2f) exceeds transaction amount (%.2f)", sum, t.Amount)
	}
	t.touch()
	return nil
}

// ReplaceSplits overrides the current splits collection after validation.
func (t *Transaction) ReplaceSplits(splits []*Split) error {
	return t.setSplits(splits)
}

// AddTag appends a new tag if it does not already exist.
func (t *Transaction) AddTag(tag string) {
	if t == nil {
		return
	}
	tag = strings.TrimSpace(strings.ToLower(tag))
	if tag == "" {
		return
	}
	for _, existing := range t.Tags {
		if existing == tag {
			return
		}
	}
	t.Tags = append(t.Tags, tag)
	t.touch()
}

func (t *Transaction) setSplits(splits []*Split) error {
	if t == nil {
		return errors.New("transaction is nil")
	}
	t.Splits = []*Split{}
	for _, split := range splits {
		if split == nil {
			return errors.New("split cannot be nil")
		}
		if split.Amount <= 0 {
			return errors.New("split amount must be greater than zero")
		}
		if split.ID == "" {
			split.ID = uuid.New().String()
		}
		if split.CreatedAt.IsZero() {
			split.CreatedAt = time.Now()
		}
		t.Splits = append(t.Splits, split)
	}

	if sum := t.splitsTotal(); sum > t.Amount+0.01 {
		return fmt.Errorf("sum of splits (%.2f) exceeds transaction amount (%.2f)", sum, t.Amount)
	}

	t.touch()
	return nil
}

func (t *Transaction) splitsTotal() float64 {
	total := 0.0
	for _, split := range t.Splits {
		total += split.Amount
	}
	return round2(total)
}

func (t *Transaction) touch() {
	t.UpdatedAt = time.Now()
}

func (t *Transaction) validateInstallments() error {
	if t.InstallmentNumber == nil && t.InstallmentTotal == nil {
		return nil
	}

	number := 0
	if t.InstallmentNumber != nil {
		number = *t.InstallmentNumber
	}
	total := 0
	if t.InstallmentTotal != nil {
		total = *t.InstallmentTotal
	}

	if total < 1 {
		return errors.New("installmentTotal must be greater than zero when provided")
	}
	if number < 1 || number > total {
		return errors.New("installmentNumber must be between 1 and installmentTotal")
	}
	return nil
}

func validateType(t Type) error {
	switch t {
	case TypeIncome, TypeExpense, TypeTransfer:
		return nil
	default:
		return fmt.Errorf("invalid transaction type: %s", t)
	}
}

func sanitizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	var cleaned []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

func cloneString(value *string) *string {
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

func cloneTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	copy := *value
	return &copy
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
