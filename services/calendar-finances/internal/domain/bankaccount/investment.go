package bankaccount

import (
	"errors"
	"time"
)

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
