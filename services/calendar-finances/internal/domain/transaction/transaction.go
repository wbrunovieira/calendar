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
	// StatusReversed marks a transaction that was undone. The row is KEPT: a ledger
	// does not delete, because a deleted row cannot answer what was undone, by whom
	// or when. Balances are derived from CONFIRMED rows, so a reversed one stops
	// counting without disappearing.
	StatusReversed Status = "REVERSED"
)

// ReversalReason classifies WHY a transaction was undone, because the answer decides
// the accounting effect. Two different economic events hide behind "undo":
//
//   - an ERROR (never happened, duplicated, wrong amount or account) is a
//     rectification: nothing obliges the books to carry a September expense that did
//     not exist, so the effect retroacts to the original date.
//   - a REVERSAL BY THE ISSUER (chargeback, returned payment, cancelled purchase) is
//     a second real fact with its own date. The balance between the two dates really
//     was what it was, so the original keeps counting and the undo is a NEW dated
//     line.
//
// Recording only "reversed" collapses both into one verb and destroys the history in
// the second case.
type ReversalReason string

const (
	ReasonNeverHappened     ReversalReason = "NUNCA_OCORREU"
	ReasonDuplicated        ReversalReason = "DUPLICADO"
	ReasonWrongAmount       ReversalReason = "VALOR_INCORRETO"
	ReasonWrongAccount      ReversalReason = "CONTA_INCORRETA"
	ReasonBankReversed      ReversalReason = "ESTORNADO_PELO_BANCO"
	ReasonPaymentReturned   ReversalReason = "PAGAMENTO_DEVOLVIDO"
	ReasonPurchaseCancelled ReversalReason = "COMPRA_CANCELADA"
)

// IsCorrection reports whether the reason is a bookkeeping error rather than a new
// economic fact. Only corrections may retroact to the original date.
func (r ReversalReason) IsCorrection() bool {
	switch r {
	case ReasonNeverHappened, ReasonDuplicated, ReasonWrongAmount, ReasonWrongAccount:
		return true
	}
	return false
}

// Valid reports whether the reason is one the ledger knows how to treat.
func (r ReversalReason) Valid() bool {
	switch r {
	case ReasonNeverHappened, ReasonDuplicated, ReasonWrongAmount, ReasonWrongAccount,
		ReasonBankReversed, ReasonPaymentReturned, ReasonPurchaseCancelled:
		return true
	}
	return false
}

// Transaction encapsulates the financial movement registered in the system.
type Transaction struct {
	ID                   string  `json:"id"`
	ProfileID            string  `json:"profileId"`
	BankAccountID        string  `json:"bankAccountId"`
	DestinationAccountID *string `json:"destinationAccountId,omitempty"`
	CategoryID           *string `json:"categoryId,omitempty"`
	InvoiceID            *string `json:"invoiceId,omitempty"` // Credit card invoice this transaction belongs to
	Type                 Type    `json:"type"`
	Status               Status  `json:"status"`
	Amount               float64 `json:"amount"`
	Currency             string  `json:"currency"`
	Description          string  `json:"description"`
	Notes                *string `json:"notes,omitempty"`
	CostCenter           *string `json:"costCenter,omitempty"`
	// CostCenterID links the transaction to a client, project or department.
	// CostCenter above is the older free-text field, kept as it was.
	CostCenterID            *string    `json:"costCenterId,omitempty"`
	IsPersonalReimbursement bool       `json:"isPersonalReimbursement"`
	OccurredOn              time.Time  `json:"occurredOn"`
	DueOn                   *time.Time `json:"dueOn,omitempty"`
	ReminderOn              *time.Time `json:"reminderOn,omitempty"` // Optional reminder date for alerts (10, 5, 1, 0 days before)
	RecurrenceRule          *string    `json:"recurrenceRule,omitempty"`
	InstallmentNumber       *int       `json:"installmentNumber,omitempty"`
	InstallmentTotal        *int       `json:"installmentTotal,omitempty"`
	ExternalID              *string    `json:"externalId,omitempty"`
	// PaidInvoiceID names the invoice this transaction PAYS, as opposed to InvoiceID,
	// which names the invoice a card purchase BELONGS TO. Without it, the only way to
	// find a bill's payments is matching amount and date — the fragile matching this
	// system is removing — and the invariant that live payments must not exceed the
	// bill cannot be written at all.
	PaidInvoiceID *string `json:"paidInvoiceId,omitempty"`

	ReversedAt     *time.Time      `json:"reversedAt,omitempty"`
	ReversalReason *ReversalReason `json:"reversalReason,omitempty"`
	ReversalNote   *string         `json:"reversalNote,omitempty"`
	// ReversedBy names WHO undid it: a person, an agent, or an importer. The incident
	// that motivated this was an AI agent removing a legitimate entry, so this is not
	// a detail — it is the central control for that risk.
	ReversedBy          *string  `json:"reversedBy,omitempty"`
	LinkedTransactionID *string  `json:"linkedTransactionId,omitempty"` // Points to paired transaction (cross-profile transfers)
	Tags                []string `json:"tags,omitempty"`
	Splits              []*Split `json:"splits,omitempty"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// CreateParams represents the attributes required to instantiate a transaction.
type CreateParams struct {
	ProfileID               string
	BankAccountID           string
	DestinationAccountID    *string
	CategoryID              *string
	InvoiceID               *string // Credit card invoice this transaction belongs to
	Type                    Type
	Amount                  float64
	Currency                string
	Description             string
	Notes                   *string
	CostCenter              *string
	CostCenterID            *string
	IsPersonalReimbursement bool
	OccurredOn              time.Time
	DueOn                   *time.Time
	ReminderOn              *time.Time // Optional reminder date for alerts
	RecurrenceRule          *string
	InstallmentNumber       *int
	InstallmentTotal        *int
	ExternalID              *string
	PaidInvoiceID           *string
	LinkedTransactionID     *string
	Tags                    []string
	Splits                  []*Split
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
		ID:                      uuid.New().String(),
		ProfileID:               strings.TrimSpace(params.ProfileID),
		BankAccountID:           strings.TrimSpace(params.BankAccountID),
		DestinationAccountID:    cloneString(params.DestinationAccountID),
		CategoryID:              cloneString(params.CategoryID),
		InvoiceID:               cloneString(params.InvoiceID),
		Type:                    params.Type,
		Status:                  StatusPlanned,
		Amount:                  round2(params.Amount),
		Currency:                currency,
		Description:             description,
		Notes:                   cloneString(params.Notes),
		CostCenter:              cloneString(params.CostCenter),
		CostCenterID:            cloneString(params.CostCenterID),
		IsPersonalReimbursement: params.IsPersonalReimbursement,
		OccurredOn:              params.OccurredOn,
		DueOn:                   cloneTime(params.DueOn),
		ReminderOn:              cloneTime(params.ReminderOn),
		RecurrenceRule:          cloneString(params.RecurrenceRule),
		InstallmentNumber:       cloneInt(params.InstallmentNumber),
		InstallmentTotal:        cloneInt(params.InstallmentTotal),
		ExternalID:              cloneString(params.ExternalID),
		PaidInvoiceID:           cloneString(params.PaidInvoiceID),
		LinkedTransactionID:     cloneString(params.LinkedTransactionID),
		Tags:                    sanitizeTags(params.Tags),
		Splits:                  []*Split{},
		CreatedAt:               now,
		UpdatedAt:               now,
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
// CanTransitionTo is the single place that decides which status changes are legal.
// Spreading this across use cases is how "reversing twice is refused" ended up
// implemented in one of three write paths, while PUT /status quietly brought a
// reversed row back to CONFIRMED and applied its balance a second time.
func (t *Transaction) CanTransitionTo(next Status) error {
	// Terminal states are checked BEFORE the same-status shortcut. Returning early on
	// "already there" made reversing an already-reversed row succeed silently, which
	// is exactly the double-undo this guard exists to refuse.
	switch t.Status {
	case StatusReversed:
		// Terminal. Undoing an undo is a new transaction, not a status change.
		return ErrAlreadyReversed
	case StatusCancelled:
		if next == StatusCancelled {
			return nil
		}
		return errors.New("a cancelled transaction cannot change status")
	case StatusConfirmed:
		switch next {
		case StatusConfirmed:
			return nil
		case StatusCancelled:
			// A confirmed movement really happened; it is undone by reversal, which
			// records why and by whom. CANCELLED would be an unaudited way out.
			return errors.New("a confirmed transaction must be reversed, not cancelled")
		case StatusPlanned:
			// Same reason, different door. Moving a confirmed row back to planned
			// undoes the money with no motive, no actor and no reversed_at, and the
			// database CHECK that demands an audit trail never sees it.
			return errors.New("a confirmed transaction must be reversed, not moved back to planned")
		}
	case StatusPlanned:
		if next == StatusReversed {
			// A planned row never moved money: undoing it is a cancellation. Two
			// statuses meaning "does not count" with no rule for which is how a
			// ledger gets a balance that differs depending on who wrote the query.
			return errors.New("a planned transaction must be cancelled, not reversed")
		}
	}
	return nil
}

func (t *Transaction) Confirm(occurredOn time.Time) error {
	if t == nil {
		return errors.New("no transaction")
	}
	if err := t.CanTransitionTo(StatusConfirmed); err != nil {
		return err
	}
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

// Cancel voids a transaction that never moved money. A planned row is cancelled; a
// confirmed one must be reversed instead, which records why and by whom. Allowing
// both would leave two statuses meaning "does not count" with no rule for which, and
// an ambiguous status in a ledger becomes a balance that differs depending on who
// wrote the query.
//
// The audit fields are shared with the reversal path on purpose: whichever way a row
// stops counting, the books can say why and who did it.
func (t *Transaction) Cancel(reason ReversalReason, note, by string, at time.Time) error {
	if t == nil {
		return errors.New("no transaction")
	}
	if err := t.CanTransitionTo(StatusCancelled); err != nil {
		return err
	}
	// Only bookkeeping reasons: a planned row never moved money, so it cannot have
	// been reversed by the issuer. Accepting those here would collapse two events
	// into one enum again.
	if !reason.IsCorrection() {
		return errors.New("cancelling takes a bookkeeping reason, not an issuer reversal")
	}
	if by == "" {
		return errors.New("the actor is required")
	}
	t.Status = StatusCancelled
	t.ReversedAt = &at
	t.ReversalReason = &reason
	t.ReversedBy = &by
	if trimmed := strings.TrimSpace(note); trimmed != "" {
		t.ReversalNote = &trimmed
	}
	t.touch()
	return nil
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

// SetReminderOn sets or clears the reminder date.
func (t *Transaction) SetReminderOn(reminderOn *time.Time) {
	if t == nil {
		return
	}
	t.ReminderOn = cloneTime(reminderOn)
	t.touch()
}

// GetReminderDaysUntil returns the number of days until the reminder date.
// Positive values mean days remaining, negative values mean days overdue.
// Returns 0 if no reminder is set.
func (t *Transaction) GetReminderDaysUntil(referenceDate time.Time) int {
	if t == nil || t.ReminderOn == nil {
		return 0
	}

	// Normalize both dates to start of day for accurate calculation
	reminder := time.Date(t.ReminderOn.Year(), t.ReminderOn.Month(), t.ReminderOn.Day(), 0, 0, 0, 0, time.UTC)
	reference := time.Date(referenceDate.Year(), referenceDate.Month(), referenceDate.Day(), 0, 0, 0, 0, time.UTC)

	diff := reminder.Sub(reference)
	return int(diff.Hours() / 24)
}

// ShouldShowReminderAlert returns true if an alert should be shown for the given days until reminder.
// Alerts are shown at: 10 days, 5 days, 1 day, on the day (0), and when overdue (negative).
func ShouldShowReminderAlert(daysUntil int) bool {
	// Show alert when overdue
	if daysUntil < 0 {
		return true
	}
	// Show alert on specific days: 10, 5, 1, 0
	alertDays := []int{10, 5, 1, 0}
	for _, day := range alertDays {
		if daysUntil == day {
			return true
		}
	}
	return false
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

// Reverse undoes a transaction without destroying it. The row keeps its original
// content and stops counting towards balances; what it recorded remains auditable.
//
// Reversing twice is refused: the second call would look like a new correction and
// move the balance again for something already undone.
func (t *Transaction) Reverse(reason ReversalReason, note, by string, at time.Time) error {
	if err := t.CanTransitionTo(StatusReversed); err != nil {
		return err
	}
	// A reason is REQUIRED. A reason field that exists and is never filled is worse
	// than no field: it reads as a control in the design that does not operate, and
	// the reason cannot be reconstructed six months later — the difference between a
	// phantom entry and a genuine chargeback is exactly what a later audit needs.
	if !reason.Valid() {
		return errors.New("a valid reversal reason is required")
	}
	if by == "" {
		return errors.New("the reversal actor is required")
	}
	t.Status = StatusReversed
	t.ReversedAt = &at
	t.ReversalReason = &reason
	t.ReversedBy = &by
	if note != "" {
		t.ReversalNote = &note
	}
	t.UpdatedAt = time.Now()
	return nil
}

// ReversalRetroacts reports whether undoing this transaction should take effect on
// its original date (a bookkeeping correction) or requires a new line dated when the
// fact happened (a real reversal by the issuer).
func (t *Transaction) ReversalRetroacts() bool {
	return t.ReversalReason != nil && t.ReversalReason.IsCorrection()
}

// SetStatus is the only way a caller outside this package may TRANSITION a status.
//
// Setting the initial status on a freshly built transaction is a different act and is
// still done by assignment right after New() — those writes have no previous state to
// violate. The gate that keeps this honest is mechanical:
//
//	grep -rn '\.Status = ' --include='*.go' internal/application internal/infrastructure
//
// every hit must sit within a few lines of a New(), or it is a transition that skipped
// the rules. That check is what would have caught PUT /transactions/{id} writing
// CONFIRMED over a reversed row. Direct
// assignment is what let PUT /transactions/{id} and PUT /status bypass the transition
// rules entirely: the rule lived in one place and two write paths never consulted it.
//
// Undoing a confirmed movement is refused here on purpose — it goes through Reverse,
// which demands a motive and an actor.
func (t *Transaction) SetStatus(next Status) error {
	if t == nil {
		return errors.New("no transaction")
	}
	if err := t.CanTransitionTo(next); err != nil {
		return err
	}
	t.Status = next
	t.touch()
	return nil
}
