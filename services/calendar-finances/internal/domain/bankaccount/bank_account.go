package bankaccount

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeChecking    AccountType = "CHECKING"
	AccountTypeSavings     AccountType = "SAVINGS"
	AccountTypeInvestment  AccountType = "INVESTMENT"
	AccountTypeCreditCard  AccountType = "CREDIT_CARD"
	AccountTypeCash        AccountType = "CASH"
	AccountTypeOther       AccountType = "OTHER"
)

type BankAccount struct {
	ID             string      `json:"id"`
	ProfileID      string      `json:"profileId"`
	Name           string      `json:"name"`
	Type           AccountType `json:"type"`
	InitialBalance float64     `json:"initialBalance"`
	CurrentBalance float64     `json:"currentBalance"`
	Currency       string      `json:"currency"`
	IsActive       bool        `json:"isActive"`

	// Optional fields
	BankName        *string  `json:"bankName,omitempty"`
	BankCode        *string  `json:"bankCode,omitempty"`
	Agency          *string  `json:"agency,omitempty"`
	AccountNumber   *string  `json:"accountNumber,omitempty"`
	AccountDigit    *string  `json:"accountDigit,omitempty"`
	Color           *string  `json:"color,omitempty"`
	Icon            *string  `json:"icon,omitempty"`
	Description     *string  `json:"description,omitempty"`
	CreditLimit     *float64 `json:"creditLimit,omitempty"`
	DueDay          *int     `json:"dueDay,omitempty"`
	ClosingDay      *int     `json:"closingDay,omitempty"`
	LinkedAccountID *string  `json:"linkedAccountId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewBankAccount(
	profileID, name string,
	accountType AccountType,
	initialBalance float64,
	currency string,
) (*BankAccount, error) {
	if profileID == "" {
		return nil, errors.New("profileID is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if !isValidAccountType(accountType) {
		return nil, errors.New("invalid account type")
	}
	if currency == "" {
		return nil, errors.New("currency is required")
	}

	now := time.Now()
	return &BankAccount{
		ID:             uuid.New().String(),
		ProfileID:      profileID,
		Name:           name,
		Type:           accountType,
		InitialBalance: initialBalance,
		CurrentBalance: initialBalance,
		Currency:       currency,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func isValidAccountType(accountType AccountType) bool {
	switch accountType {
	case AccountTypeChecking, AccountTypeSavings, AccountTypeInvestment,
		AccountTypeCreditCard, AccountTypeCash, AccountTypeOther:
		return true
	default:
		return false
	}
}

func (ba *BankAccount) UpdateBalance(amount float64) {
	ba.CurrentBalance = amount
	ba.UpdatedAt = time.Now()
}

func (ba *BankAccount) Activate() {
	ba.IsActive = true
	ba.UpdatedAt = time.Now()
}

func (ba *BankAccount) Deactivate() {
	ba.IsActive = false
	ba.UpdatedAt = time.Now()
}
