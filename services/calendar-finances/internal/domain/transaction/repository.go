package transaction

import (
	"errors"
	"time"
)

// ErrNotFound is returned by repository lookups when no transaction matches.
var ErrNotFound = errors.New("transaction not found")

// ErrAlreadyReversed distinguishes "this was already undone" from "this does not
// exist". Collapsing both into 404 tells the caller there was nothing there, so a
// balance left crooked by a failed reversal is never investigated.
var ErrAlreadyReversed = errors.New("transaction is already reversed")

// ErrConcurrentModification means the row changed between being read and being
// written. The caller had already loaded it, so "no rows affected" cannot mean
// "does not exist" — it means someone else got there first.
var ErrConcurrentModification = errors.New("transaction changed since it was read")

// ListFilter encapsulates query options for fetching transactions.
type ListFilter struct {
	ProfileID            string
	BankAccountID        *string
	InvoiceID            *string
	CostCenterID         *string
	Status               *Status
	Type                 *Type
	OccurredFrom         *time.Time
	OccurredTo           *time.Time
	IncludeAsDestination bool // Also match transfers where BankAccountID is the destination
	Limit                *int
	Offset               *int
	// IncludeReversed brings reversed rows back into the result. Off by default:
	// they are audit history, not transactions, and a reconciliation that sees them
	// reports phantoms.
	IncludeReversed bool
}

// Repository represents the persistence contract for transactions.
type Repository interface {
	Create(tx *Transaction) error
	GetByID(id string) (*Transaction, error)
	List(filter ListFilter) ([]*Transaction, error)
	Count(filter ListFilter) (int, error)
	Update(tx *Transaction) error
	UpdateStatus(id string, status Status, occurredOn time.Time, notes *string) error
	// CancelStatus persists a cancellation WITH the motive and the actor the domain
	// captured. UpdateStatus writes only status, date and notes, so routing a
	// cancellation through it demanded a reason and an actor at the door and then
	// threw both away — a control that appears to operate while recording nothing.
	CancelStatus(txn *Transaction, occurredOn time.Time) error
	Delete(id string) error
	// DeleteMany removes several transactions as one unit of work. Deleting the legs
	// of a linked pair one by one can leave the ledger half-removed — one profile
	// holding a credit with no row behind it — with no way to tell afterwards.
	DeleteMany(ids []string) error
	// ReverseMany marks several transactions as reversed as ONE unit of work. A
	// linked pair reversed one at a time can leave the other profile holding a
	// credit whose counterpart no longer counts, while the caller is told the whole
	// thing failed and nobody goes looking.
	ReverseMany(txns []*Transaction) error
	SumByCategories(profileID string, categoryIDs []string, from, to time.Time) (map[string]float64, error)
	SumByInvoiceID(invoiceID string) (float64, error)
	SumByInvoiceIDByStatus(invoiceID string, status Status) (float64, error)
	CalculateBalanceByBankAccountID(bankAccountID string) (float64, error)
	// CalculateBalanceSince returns the net balance impact of all CONFIRMED
	// transactions for the account that occurred on or after `since`.
	CalculateBalanceSince(accountID string, since time.Time) (float64, error)
	// CalculateBalanceUpTo returns the net balance impact of all CONFIRMED
	// transactions for the account that occurred on or before `upTo`.
	CalculateBalanceUpTo(accountID string, upTo time.Time) (float64, error)
	FindByExternalID(externalID string) (*Transaction, error)
}
