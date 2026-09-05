package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type UpdateBankAccountInput struct {
	ProfileID      string  `json:"profileId"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	CurrentBalance float64 `json:"currentBalance"`
	// InitialBalance is the seed balance the account was created with. It is a
	// pointer because omitting it must preserve the stored value: only send it to
	// correct a seed that was cadastrado wrong, since every later balance derives
	// from it.
	InitialBalance  *float64 `json:"initialBalance,omitempty"`
	Currency        string   `json:"currency"`
	IsActive        bool     `json:"isActive"`
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
	InvestmentType *string    `json:"investmentType,omitempty"`
	YieldType      *string    `json:"yieldType,omitempty"`
	YieldRate      *float64   `json:"yieldRate,omitempty"`
	MaturityDate   *time.Time `json:"maturityDate,omitempty"`
	Broker         *string    `json:"broker,omitempty"`
	NumberOfQuotas *float64   `json:"numberOfQuotas,omitempty"`
	QuotaPrice     *float64   `json:"quotaPrice,omitempty"`
}

type UpdateBankAccountUseCase struct {
	repo bankaccount.Repository
}

func NewUpdateBankAccountUseCase(repo bankaccount.Repository) *UpdateBankAccountUseCase {
	return &UpdateBankAccountUseCase{repo: repo}
}

func (uc *UpdateBankAccountUseCase) Execute(id string, input UpdateBankAccountInput) (*bankaccount.BankAccount, error) {
	account, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	// Update fields
	account.ProfileID = input.ProfileID
	account.Name = input.Name
	account.Type = bankaccount.AccountType(input.Type)
	account.CurrentBalance = input.CurrentBalance
	// A balance is always initialBalance + the sum of its transactions. Correcting
	// the seed must therefore shift the current balance by the same delta —
	// the transactions themselves did not change.
	if input.InitialBalance != nil {
		account.CurrentBalance += *input.InitialBalance - account.InitialBalance
		account.InitialBalance = *input.InitialBalance
	}
	if input.Currency == "" {
		// keep existing currency
	} else if !bankaccount.IsValidCurrency(input.Currency) {
		return nil, ErrInvalidCurrency
	} else {
		account.Currency = input.Currency
	}
	account.IsActive = input.IsActive
	account.BankName = input.BankName
	account.BankCode = input.BankCode
	account.Agency = input.Agency
	account.AccountNumber = input.AccountNumber
	account.AccountDigit = input.AccountDigit
	account.Color = input.Color
	account.Icon = input.Icon
	account.Description = input.Description
	account.CreditLimit = input.CreditLimit
	account.DueDay = input.DueDay
	account.ClosingDay = input.ClosingDay
	if input.LinkedAccountID != nil {
		account.LinkedAccountID = input.LinkedAccountID
	}
	if input.DisplayOrder != nil {
		account.DisplayOrder = input.DisplayOrder
	}

	// Update investment-specific fields — nil means "not sent", preserve existing
	if input.InvestmentType != nil {
		invType := bankaccount.InvestmentType(*input.InvestmentType)
		account.InvestmentType = &invType
	}
	if input.YieldType != nil {
		yieldType := bankaccount.YieldType(*input.YieldType)
		account.YieldType = &yieldType
	}
	if input.YieldRate != nil {
		account.YieldRate = input.YieldRate
	}
	if input.MaturityDate != nil {
		account.MaturityDate = input.MaturityDate
	}
	if input.Broker != nil {
		account.Broker = input.Broker
	}
	if input.NumberOfQuotas != nil {
		account.NumberOfQuotas = input.NumberOfQuotas
	}
	if input.QuotaPrice != nil {
		account.QuotaPrice = input.QuotaPrice
	}

	account.UpdatedAt = time.Now()

	if err := uc.repo.Update(account); err != nil {
		return nil, err
	}

	return account, nil
}
