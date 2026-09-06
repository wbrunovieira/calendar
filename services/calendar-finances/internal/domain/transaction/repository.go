package transaction

import (
	"errors"
	"time"
)

// ErrNotFound is returned by repository lookups when no transaction matches.
var ErrNotFound = errors.New("transaction not found")

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
}

// Repository represents the persistence contract for transactions.
type Repository interface {
	Create(tx *Transaction) error
	GetByID(id string) (*Transaction, error)
	List(filter ListFilter) ([]*Transaction, error)
	Count(filter ListFilter) (int, error)
	Update(tx *Transaction) error
	UpdateStatus(id string, status Status, occurredOn time.Time, notes *string) error
	Delete(id string) error
	// DeleteMany removes several transactions as one unit of work. Deleting the legs
	// of a linked pair one by one can leave the ledger half-removed — one profile
	// holding a credit with no row behind it — with no way to tell afterwards.
	DeleteMany(ids []string) error
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
