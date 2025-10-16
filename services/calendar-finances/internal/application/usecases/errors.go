package usecases

import "errors"

var (
	ErrProfileAlreadyExists   = errors.New("profile already exists for this calendar")
	ErrProfileNotFound        = errors.New("profile not found")
	ErrInvalidInput           = errors.New("invalid input")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrBankAccountMismatch    = errors.New("bank account does not belong to profile")
	ErrDestinationRequired    = errors.New("destination account is required for transfer")
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrInsufficientBalance    = errors.New("insufficient balance to complete transaction")
	ErrCreditLimitExceeded    = errors.New("credit limit exceeded for this transaction")
)
