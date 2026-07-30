package bankaccount

import (
	"errors"
	"fmt"
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
	AccountTypeExchange    AccountType = "EXCHANGE" // Corretoras de cripto (Binance, OKX, Rabbit)
	AccountTypeWallet      AccountType = "WALLET"   // Carteiras de cripto (Ledger, MetaMask)
	AccountTypeOther       AccountType = "OTHER"
)

// InvestmentType represents the type of investment product
type InvestmentType string

const (
	InvestmentTypeSavingsBox InvestmentType = "SAVINGS_BOX" // Caixinha (Nubank, etc.)
	InvestmentTypeCDB        InvestmentType = "CDB"         // Certificado de Depósito Bancário
	InvestmentTypeLCI        InvestmentType = "LCI"         // Letra de Crédito Imobiliário
	InvestmentTypeLCA        InvestmentType = "LCA"         // Letra de Crédito do Agronegócio
	InvestmentTypeStocks     InvestmentType = "STOCKS"      // Ações
	InvestmentTypeFunds      InvestmentType = "FUNDS"       // Fundos de investimento
	InvestmentTypeFII        InvestmentType = "FII"         // Fundos Imobiliários
	InvestmentTypeCrypto     InvestmentType = "CRYPTO"      // Criptomoedas
	InvestmentTypeTreasury   InvestmentType = "TREASURY"    // Tesouro Direto
	InvestmentTypeOther      InvestmentType = "OTHER"       // Outros
)

// YieldType represents how the yield/return is calculated
type YieldType string

const (
	YieldTypeFixed         YieldType = "FIXED"          // Taxa fixa (ex: 12% a.a.)
	YieldTypeCDIPercentage YieldType = "CDI_PERCENTAGE" // Percentual do CDI (ex: 100% CDI)
	YieldTypeIPCAPlus      YieldType = "IPCA_PLUS"      // IPCA + taxa (ex: IPCA + 5%)
	YieldTypeVariable      YieldType = "VARIABLE"       // Taxa variável (ações, fundos, crypto)
)

func IsValidInvestmentType(investmentType InvestmentType) bool {
	switch investmentType {
	case InvestmentTypeSavingsBox, InvestmentTypeCDB, InvestmentTypeLCI,
		InvestmentTypeLCA, InvestmentTypeStocks, InvestmentTypeFunds,
		InvestmentTypeFII, InvestmentTypeCrypto, InvestmentTypeTreasury, InvestmentTypeOther:
		return true
	default:
		return false
	}
}

func IsValidYieldType(yieldType YieldType) bool {
	switch yieldType {
	case YieldTypeFixed, YieldTypeCDIPercentage, YieldTypeIPCAPlus, YieldTypeVariable:
		return true
	default:
		return false
	}
}

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

// HasFixedMaturity returns true if the investment has a maturity date
func (ba *BankAccount) HasFixedMaturity() bool {
	return ba.MaturityDate != nil
}

// IsMatured returns true if the investment has passed its maturity date
func (ba *BankAccount) IsMatured() bool {
	if ba.MaturityDate == nil {
		return false
	}
	return time.Now().After(*ba.MaturityDate)
}

// DaysToMaturity returns the number of days until maturity, or -1 if no maturity date
func (ba *BankAccount) DaysToMaturity() int {
	if ba.MaturityDate == nil {
		return -1
	}
	duration := time.Until(*ba.MaturityDate)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// GetYieldDescription returns a human-readable description of the yield
func (ba *BankAccount) GetYieldDescription() string {
	if ba.YieldType == nil {
		return ""
	}

	rate := float64(0)
	if ba.YieldRate != nil {
		rate = *ba.YieldRate
	}

	switch *ba.YieldType {
	case YieldTypeFixed:
		return formatRate(rate) + "% a.a."
	case YieldTypeCDIPercentage:
		return formatRate(rate) + "% do CDI"
	case YieldTypeIPCAPlus:
		return "IPCA + " + formatRate(rate) + "%"
	case YieldTypeVariable:
		return "Variável"
	default:
		return ""
	}
}

func formatRate(rate float64) string {
	if rate == float64(int(rate)) {
		return fmt.Sprintf("%.0f", rate)
	}
	return fmt.Sprintf("%.2f", rate)
}

// CalculateReturn calculates the absolute return (current balance - initial balance)
func (ba *BankAccount) CalculateReturn() float64 {
	return ba.CurrentBalance - ba.InitialBalance
}

// CalculateReturnPercentage calculates the percentage return
func (ba *BankAccount) CalculateReturnPercentage() float64 {
	if ba.InitialBalance == 0 {
		return 0
	}
	return (ba.CalculateReturn() / ba.InitialBalance) * 100
}

// SetInvestmentDetails sets investment-specific fields
func (ba *BankAccount) SetInvestmentDetails(
	investmentType *InvestmentType,
	yieldType *YieldType,
	yieldRate *float64,
	maturityDate *time.Time,
	broker *string,
) error {
	if ba.Type != AccountTypeInvestment {
		return errors.New("investment details can only be set for investment accounts")
	}

	if investmentType != nil && !IsValidInvestmentType(*investmentType) {
		return errors.New("invalid investment type")
	}

	if yieldType != nil && !IsValidYieldType(*yieldType) {
		return errors.New("invalid yield type")
	}

	ba.InvestmentType = investmentType
	ba.YieldType = yieldType
	ba.YieldRate = yieldRate
	ba.MaturityDate = maturityDate
	ba.Broker = broker
	ba.UpdatedAt = time.Now()

	return nil
}

// RequiresLinking returns true if this account type should be linked to a source account
func (ba *BankAccount) RequiresLinking() bool {
	return ba.Type == AccountTypeInvestment || ba.Type == AccountTypeCreditCard
}

// IsValidLinkTarget returns true if this account can be used as a link target (source of funds)
func (ba *BankAccount) IsValidLinkTarget() bool {
	return ba.Type == AccountTypeChecking || ba.Type == AccountTypeSavings || ba.Type == AccountTypeCash || ba.Type == AccountTypeExchange || ba.Type == AccountTypeWallet
}

// IsLinked returns true if this account is linked to another account
func (ba *BankAccount) IsLinked() bool {
	return ba.LinkedAccountID != nil && *ba.LinkedAccountID != ""
}

// CanLinkToAccount validates if this account can be linked to the target account
func (ba *BankAccount) CanLinkToAccount(target *BankAccount) bool {
	if target == nil {
		return false
	}

	// Only investment and credit card accounts can link
	if !ba.RequiresLinking() {
		return false
	}

	// Target must be a valid link target (checking, savings, or cash)
	if !target.IsValidLinkTarget() {
		return false
	}

	// Must be from the same profile
	if ba.ProfileID != target.ProfileID {
		return false
	}

	return true
}

// SetLinkedAccount links this account to a source account
func (ba *BankAccount) SetLinkedAccount(target *BankAccount) error {
	if !ba.CanLinkToAccount(target) {
		if target == nil {
			return errors.New("target account cannot be nil")
		}
		if !ba.RequiresLinking() {
			return errors.New("only investment and credit card accounts can be linked")
		}
		if !target.IsValidLinkTarget() {
			return errors.New("target must be a checking, savings, cash, exchange, or wallet account")
		}
		if ba.ProfileID != target.ProfileID {
			return errors.New("accounts must belong to the same profile")
		}
		return errors.New("cannot link to the specified account")
	}

	ba.LinkedAccountID = &target.ID
	ba.UpdatedAt = time.Now()
	return nil
}

// ClearLinkedAccount removes the link to the source account
func (ba *BankAccount) ClearLinkedAccount() {
	ba.LinkedAccountID = nil
	ba.UpdatedAt = time.Now()
}

// SupportsQuotas returns true if this investment type uses quotas/shares
func (ba *BankAccount) SupportsQuotas() bool {
	if ba.InvestmentType == nil {
		return false
	}
	switch *ba.InvestmentType {
	case InvestmentTypeStocks, InvestmentTypeFII, InvestmentTypeFunds, InvestmentTypeCrypto:
		return true
	default:
		return false
	}
}

// HasQuotas returns true if this account has quota information set
func (ba *BankAccount) HasQuotas() bool {
	return ba.NumberOfQuotas != nil && *ba.NumberOfQuotas > 0
}

// SetQuotasFromTotal sets the number of quotas and calculates the quota price from total value
// Use this when you know the total invested and number of quotas
func (ba *BankAccount) SetQuotasFromTotal(numberOfQuotas float64, totalValue float64) error {
	if ba.Type != AccountTypeInvestment && ba.Type != AccountTypeExchange && ba.Type != AccountTypeWallet {
		return errors.New("quotas can only be set for investment, exchange, or wallet accounts")
	}
	if numberOfQuotas <= 0 {
		return errors.New("number of quotas must be greater than zero")
	}
	if totalValue < 0 {
		return errors.New("total value cannot be negative")
	}

	quotaPrice := totalValue / numberOfQuotas

	ba.NumberOfQuotas = &numberOfQuotas
	ba.QuotaPrice = &quotaPrice
	ba.InitialBalance = totalValue
	ba.CurrentBalance = totalValue
	ba.UpdatedAt = time.Now()

	return nil
}

// SetQuotasFromPrice sets the number of quotas and quota price, calculating the total value
// Use this when you know the price per quota and number of quotas
func (ba *BankAccount) SetQuotasFromPrice(numberOfQuotas float64, pricePerQuota float64) error {
	if ba.Type != AccountTypeInvestment && ba.Type != AccountTypeExchange && ba.Type != AccountTypeWallet {
		return errors.New("quotas can only be set for investment, exchange, or wallet accounts")
	}
	if numberOfQuotas <= 0 {
		return errors.New("number of quotas must be greater than zero")
	}
	if pricePerQuota < 0 {
		return errors.New("quota price cannot be negative")
	}

	totalValue := numberOfQuotas * pricePerQuota

	ba.NumberOfQuotas = &numberOfQuotas
	ba.QuotaPrice = &pricePerQuota
	ba.InitialBalance = totalValue
	ba.CurrentBalance = totalValue
	ba.UpdatedAt = time.Now()

	return nil
}

// UpdateQuotaPrice updates the current quota price and recalculates the current balance
func (ba *BankAccount) UpdateQuotaPrice(newPrice float64) error {
	if !ba.HasQuotas() {
		return errors.New("account does not have quotas set")
	}
	if newPrice < 0 {
		return errors.New("quota price cannot be negative")
	}

	ba.QuotaPrice = &newPrice
	ba.CurrentBalance = *ba.NumberOfQuotas * newPrice
	ba.UpdatedAt = time.Now()

	return nil
}

// quotaEpsilon absorbs floating-point noise when comparing share counts, so a
// full sell (quantity == held) closes the position instead of leaving a
// sub-cent residue.
const quotaEpsilon = 1e-9

// SellQuotas reduces the position by `quantity` shares sold at `unitPrice`,
// returning the gross proceeds (quantity * unitPrice). The remaining shares are
// re-marked at the current quota price. Selling the whole position closes it:
// the share count drops to zero and the balance becomes zero.
//
// NOTE: on a full sell the share count is set to 0 (not nil) on purpose — the
// price-sync job dereferences NumberOfQuotas for any account whose name is a
// ticker, so a nil here would panic it. Zero shares makes HasQuotas() false and
// keeps the synced balance at 0.
func (ba *BankAccount) SellQuotas(quantity, unitPrice float64) (float64, error) {
	if !ba.HasQuotas() {
		return 0, errors.New("account does not have quotas set")
	}
	if quantity <= 0 {
		return 0, errors.New("quantity must be greater than zero")
	}
	if unitPrice < 0 {
		return 0, errors.New("unit price cannot be negative")
	}
	if quantity > *ba.NumberOfQuotas+quotaEpsilon {
		return 0, errors.New("cannot sell more quotas than held")
	}

	proceeds := quantity * unitPrice
	remaining := *ba.NumberOfQuotas - quantity

	if remaining <= quotaEpsilon {
		zero := 0.0
		ba.NumberOfQuotas = &zero
		ba.CurrentBalance = 0
	} else {
		markPrice := unitPrice
		if ba.QuotaPrice != nil {
			markPrice = *ba.QuotaPrice
		}
		ba.NumberOfQuotas = &remaining
		ba.CurrentBalance = remaining * markPrice
	}
	ba.UpdatedAt = time.Now()

	return proceeds, nil
}
