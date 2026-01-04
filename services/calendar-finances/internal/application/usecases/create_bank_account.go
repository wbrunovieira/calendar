package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type CreateBankAccountInput struct {
	ProfileID       string   `json:"profileId"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	InitialBalance  float64  `json:"initialBalance"`
	Currency        string   `json:"currency"`
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

	// Investment-specific fields
	InvestmentType *string    `json:"investmentType,omitempty"` // SAVINGS_BOX, CDB, LCI, LCA, STOCKS, FUNDS, FII, CRYPTO, TREASURY, OTHER
	YieldType      *string    `json:"yieldType,omitempty"`      // FIXED, CDI_PERCENTAGE, IPCA_PLUS, VARIABLE
	YieldRate      *float64   `json:"yieldRate,omitempty"`      // Rate value (e.g., 100 for 100% CDI)
	MaturityDate   *time.Time `json:"maturityDate,omitempty"`   // Investment end date (optional)
	Broker         *string    `json:"broker,omitempty"`         // Broker/platform (e.g., "Nubank", "XP")
	NumberOfQuotas *float64   `json:"numberOfQuotas,omitempty"` // Number of shares/quotas
	QuotaPrice     *float64   `json:"quotaPrice,omitempty"`     // Price per share/quota
}

type CreateBankAccountUseCase struct {
	repo bankaccount.Repository
}

func NewCreateBankAccountUseCase(repo bankaccount.Repository) *CreateBankAccountUseCase {
	return &CreateBankAccountUseCase{repo: repo}
}

func (uc *CreateBankAccountUseCase) Execute(input CreateBankAccountInput) (*bankaccount.BankAccount, error) {
	account, err := bankaccount.NewBankAccount(
		input.ProfileID,
		input.Name,
		bankaccount.AccountType(input.Type),
		input.InitialBalance,
		input.Currency,
	)
	if err != nil {
		return nil, err
	}

	// Set optional fields
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
	account.LinkedAccountID = input.LinkedAccountID

	// Set investment-specific fields
	if input.InvestmentType != nil {
		invType := bankaccount.InvestmentType(*input.InvestmentType)
		account.InvestmentType = &invType
	}
	if input.YieldType != nil {
		yieldType := bankaccount.YieldType(*input.YieldType)
		account.YieldType = &yieldType
	}
	account.YieldRate = input.YieldRate
	account.MaturityDate = input.MaturityDate
	account.Broker = input.Broker
	account.NumberOfQuotas = input.NumberOfQuotas
	account.QuotaPrice = input.QuotaPrice

	if err := uc.repo.Create(account); err != nil {
		return nil, err
	}

	return account, nil
}
