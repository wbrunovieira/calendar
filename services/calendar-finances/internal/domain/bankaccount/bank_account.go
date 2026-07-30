package bankaccount

import (
	"errors"
	"time"

	"github.com/google/uuid"
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
	DisplayOrder    *int     `json:"displayOrder,omitempty"`

	// Investment-specific fields
	InvestmentType *InvestmentType `json:"investmentType,omitempty"` // Type of investment product
	YieldType      *YieldType      `json:"yieldType,omitempty"`      // How yield is calculated
	YieldRate      *float64        `json:"yieldRate,omitempty"`      // Rate value (e.g., 100 for 100% CDI)
	MaturityDate   *time.Time      `json:"maturityDate,omitempty"`   // Investment end date (optional)
	Broker         *string         `json:"broker,omitempty"`         // Broker/platform (e.g., "Nubank", "XP")
	NumberOfQuotas *float64        `json:"numberOfQuotas,omitempty"` // Number of shares/quotas (for stocks, FII, funds, crypto)
	QuotaPrice     *float64        `json:"quotaPrice,omitempty"`     // Price per share/quota

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
	if !IsValidCurrency(currency) {
		return nil, errors.New("invalid currency: must be BRL, USD, or EUR")
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

// IsValidCurrency checks if the currency is supported (BRL, USD, EUR)
func IsValidCurrency(currency string) bool {
	switch currency {
	case "BRL", "USD", "EUR":
		return true
	default:
		return false
	}
}

func isValidAccountType(accountType AccountType) bool {
	switch accountType {
	case AccountTypeChecking, AccountTypeSavings, AccountTypeInvestment,
		AccountTypeCreditCard, AccountTypeCash, AccountTypeExchange,
		AccountTypeWallet, AccountTypeOther:
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

// IsInvestment returns true if the account is an investment account
func (ba *BankAccount) IsInvestment() bool {
	return ba.Type == AccountTypeInvestment
}

// IsCreditCard returns true if the account is a credit card
func (ba *BankAccount) IsCreditCard() bool {
	return ba.Type == AccountTypeCreditCard
}

// IsExchange returns true if the account is a crypto exchange
func (ba *BankAccount) IsExchange() bool {
	return ba.Type == AccountTypeExchange
}

// IsWallet returns true if the account is a crypto wallet
func (ba *BankAccount) IsWallet() bool {
	return ba.Type == AccountTypeWallet
}
