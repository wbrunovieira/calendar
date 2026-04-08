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
	ErrRecurringNotFound      = errors.New("recurring transaction not found")
	ErrBudgetTargetNotFound   = errors.New("budget target not found")
	ErrBankAccountNotFound    = errors.New("bank account not found")
	ErrInvoiceNotFound        = errors.New("invoice not found")
	ErrInvoiceNotOpen         = errors.New("invoice is not open")
	ErrInvoiceAlreadyPaid     = errors.New("invoice is already paid")
	ErrNotCreditCard              = errors.New("bank account is not a credit card")
	ErrDestinationCategoryRequired = errors.New("destination category is required for cross-profile transfer")
	ErrGoalNotFound                = errors.New("goal not found")
	ErrCapitalContributionNotFound = errors.New("capital contribution not found")
	ErrCompanyAssetNotFound        = errors.New("company asset not found")
)
