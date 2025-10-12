package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type CreateBankAccountInput struct {
	ProfileID      string   `json:"profileId"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	InitialBalance float64  `json:"initialBalance"`
	Currency       string   `json:"currency"`
	BankName       *string  `json:"bankName,omitempty"`
	BankCode       *string  `json:"bankCode,omitempty"`
	Agency         *string  `json:"agency,omitempty"`
	AccountNumber  *string  `json:"accountNumber,omitempty"`
	AccountDigit   *string  `json:"accountDigit,omitempty"`
	Color          *string  `json:"color,omitempty"`
	Icon           *string  `json:"icon,omitempty"`
	Description    *string  `json:"description,omitempty"`
	CreditLimit    *float64 `json:"creditLimit,omitempty"`
	DueDay         *int     `json:"dueDay,omitempty"`
	ClosingDay     *int     `json:"closingDay,omitempty"`
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

	if err := uc.repo.Create(account); err != nil {
		return nil, err
	}

	return account, nil
}
