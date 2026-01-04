package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type UpdateBankAccountInput struct {
	ProfileID       string   `json:"profileId"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	CurrentBalance  float64  `json:"currentBalance"`
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
	account.Currency = input.Currency
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
	account.LinkedAccountID = input.LinkedAccountID
	account.UpdatedAt = time.Now()

	if err := uc.repo.Update(account); err != nil {
		return nil, err
	}

	return account, nil
}
