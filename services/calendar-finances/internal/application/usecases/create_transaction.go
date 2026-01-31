package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type CreateTransactionSplitInput struct {
	CategoryID *string `json:"categoryId,omitempty"`
	Amount     float64 `json:"amount"`
	Memo       *string `json:"memo,omitempty"`
}

type CreateTransactionInput struct {
	ProfileID            string                        `json:"profileId"`
	BankAccountID        string                        `json:"bankAccountId"`
	DestinationAccountID *string                       `json:"destinationAccountId,omitempty"`
	CategoryID           *string                       `json:"categoryId,omitempty"`
	Type                 string                        `json:"type"`
	Status               *string                       `json:"status,omitempty"`
	Amount               float64                       `json:"amount"`
	Currency             string                        `json:"currency"`
	Description          string                        `json:"description"`
	Notes                *string                       `json:"notes,omitempty"`
	CostCenter           *string                       `json:"costCenter,omitempty"`
	OccurredOn           string                        `json:"occurredOn"`
	DueOn                *string                       `json:"dueOn,omitempty"`
	ReminderOn           *string                       `json:"reminderOn,omitempty"` // Optional reminder date for alerts (10, 5, 1, 0 days before)
	RecurrenceRule       *string                       `json:"recurrenceRule,omitempty"`
	InstallmentNumber    *int                          `json:"installmentNumber,omitempty"`
	InstallmentTotal     *int                          `json:"installmentTotal,omitempty"`
	ExternalID           *string                       `json:"externalId,omitempty"`
	Tags                 []string                      `json:"tags,omitempty"`
	Splits               []CreateTransactionSplitInput `json:"splits,omitempty"`
}

type CreateTransactionUseCase struct {
	profileRepo     profile.Repository
	accountRepo     bankaccount.Repository
	categoryRepo    category.Repository
	transactionRepo transaction.Repository
	invoiceRepo     invoice.Repository
}

func NewCreateTransactionUseCase(
	profileRepo profile.Repository,
	accountRepo bankaccount.Repository,
	categoryRepo category.Repository,
	transactionRepo transaction.Repository,
	invoiceRepo invoice.Repository,
) *CreateTransactionUseCase {
	return &CreateTransactionUseCase{
		profileRepo:     profileRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
		invoiceRepo:     invoiceRepo,
	}
}

func (uc *CreateTransactionUseCase) Execute(input CreateTransactionInput) (*transaction.Transaction, error) {
	if strings.TrimSpace(input.ProfileID) == "" {
		return nil, ErrInvalidInput
	}

	if _, err := uc.profileRepo.FindByID(input.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}

	account, err := uc.accountRepo.FindByID(input.BankAccountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	if account.ProfileID != input.ProfileID {
		return nil, ErrBankAccountMismatch
	}

	typeValue, err := parseTransactionType(input.Type)
	if err != nil {
		return nil, err
	}

	var categoryEntity *category.Category
	if input.CategoryID != nil {
		categoryEntity, err = uc.categoryRepo.FindByID(*input.CategoryID)
		if err != nil {
			return nil, ErrCategoryNotFound
		}
		if categoryEntity.ProfileID != input.ProfileID {
			return nil, ErrCategoryNotFound
		}
		if !isCategoryCompatible(typeValue, categoryEntity.Type) {
			return nil, ErrInvalidInput
		}
	}

	var destinationAccountID *string
	if typeValue == transaction.TypeTransfer {
		if input.DestinationAccountID == nil {
			return nil, ErrDestinationRequired
		}
		destination, err := uc.accountRepo.FindByID(*input.DestinationAccountID)
		if err != nil {
			return nil, ErrBankAccountNotFound
		}
		if destination.ID == account.ID {
			return nil, ErrInvalidInput
		}
		if destination.ProfileID != input.ProfileID {
			return nil, ErrBankAccountMismatch
		}
		destinationAccountID = &destination.ID
	}

	occurredOn, err := parseDate(input.OccurredOn)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// Only validate balance for CONFIRMED transactions that are not historical
	// Historical transactions (date < today) already happened, so balance validation doesn't apply
	isPlanned := input.Status == nil || strings.ToUpper(*input.Status) == string(transaction.StatusPlanned)
	isHistorical := isHistoricalDate(occurredOn)
	if !isPlanned && !isHistorical {
		if err := validateBalances(account, typeValue, input.Amount); err != nil {
			return nil, err
		}
	}

	var dueOn *time.Time
	if input.DueOn != nil {
		d, err := parseDate(*input.DueOn)
		if err != nil {
			return nil, ErrInvalidInput
		}
		dueOn = &d
	}

	var reminderOn *time.Time
	if input.ReminderOn != nil {
		r, err := parseDate(*input.ReminderOn)
		if err != nil {
			return nil, ErrInvalidInput
		}
		reminderOn = &r
	}

	splits, err := uc.buildSplits(input.ProfileID, input.Splits)
	if err != nil {
		return nil, err
	}

	// Handle credit card invoice assignment for expense transactions
	var invoiceID *string
	var inv *invoice.Invoice
	if account.Type == bankaccount.AccountTypeCreditCard && typeValue == transaction.TypeExpense {
		if account.ClosingDay != nil && account.DueDay != nil {
			inv, err = uc.getOrCreateInvoiceForDate(account, occurredOn)
			if err != nil {
				return nil, err
			}
			if inv != nil {
				invoiceID = &inv.ID
			}
		}
	}

	createParams := transaction.CreateParams{
		ProfileID:            input.ProfileID,
		BankAccountID:        input.BankAccountID,
		DestinationAccountID: destinationAccountID,
		CategoryID:           input.CategoryID,
		InvoiceID:            invoiceID,
		Type:                 typeValue,
		Amount:               input.Amount,
		Currency:             input.Currency,
		Description:          input.Description,
		Notes:                input.Notes,
		CostCenter:           input.CostCenter,
		OccurredOn:           occurredOn,
		DueOn:                dueOn,
		ReminderOn:           reminderOn,
		RecurrenceRule:       input.RecurrenceRule,
		InstallmentNumber:    input.InstallmentNumber,
		InstallmentTotal:     input.InstallmentTotal,
		ExternalID:           input.ExternalID,
		Tags:                 input.Tags,
		Splits:               splits,
	}

	txn, err := transaction.New(createParams)
	if err != nil {
		return nil, err
	}

	// Set status if provided (defaults to PLANNED in transaction.New)
	if input.Status != nil {
		status, err := parseTransactionStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		txn.Status = status
	}

	if err := uc.transactionRepo.Create(txn); err != nil {
		return nil, err
	}

	// Update invoice amount if this is a credit card expense
	if inv != nil {
		if err := inv.AddAmount(input.Amount); err == nil {
			_ = uc.invoiceRepo.Update(inv)
		}
	}

	return txn, nil
}

// getOrCreateInvoiceForDate gets or creates the appropriate invoice for a transaction date
func (uc *CreateTransactionUseCase) getOrCreateInvoiceForDate(account *bankaccount.BankAccount, txDate time.Time) (*invoice.Invoice, error) {
	// Try to find existing invoice for this date
	inv, err := uc.invoiceRepo.FindByBankAccountAndDate(account.ID, txDate)
	if err != nil {
		return nil, err
	}
	if inv != nil {
		return inv, nil
	}

	// No invoice exists, create one
	refDate := calculateReferenceMonth(txDate, *account.ClosingDay)

	params := invoice.CreateParams{
		BankAccountID: account.ID,
		ClosingDay:    *account.ClosingDay,
		DueDay:        *account.DueDay,
		ReferenceDate: refDate,
	}

	inv, err = invoice.New(params)
	if err != nil {
		return nil, err
	}

	if err := uc.invoiceRepo.Create(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

func (uc *CreateTransactionUseCase) buildSplits(profileID string, inputs []CreateTransactionSplitInput) ([]*transaction.Split, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	splits := make([]*transaction.Split, 0, len(inputs))
	for _, splitInput := range inputs {
		if splitInput.Amount <= 0 {
			return nil, ErrInvalidInput
		}

		if splitInput.CategoryID != nil {
			cat, err := uc.categoryRepo.FindByID(*splitInput.CategoryID)
			if err != nil {
				return nil, ErrCategoryNotFound
			}
			if cat.ProfileID != profileID {
				return nil, ErrCategoryNotFound
			}
		}

		split, err := transaction.NewSplit(splitInput.CategoryID, splitInput.Amount, splitInput.Memo)
		if err != nil {
			return nil, err
		}
		splits = append(splits, split)
	}

	return splits, nil
}

func validateBalances(account *bankaccount.BankAccount, txType transaction.Type, amount float64) error {
	if account == nil {
		return ErrBankAccountNotFound
	}
	if txType == transaction.TypeExpense {
		if account.Type == bankaccount.AccountTypeCreditCard {
			if account.CreditLimit != nil && amount > *account.CreditLimit {
				return ErrCreditLimitExceeded
			}
			return nil
		}
		if account.CurrentBalance < amount {
			return ErrInsufficientBalance
		}
	}
	return nil
}

func parseTransactionType(value string) (transaction.Type, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(transaction.TypeIncome):
		return transaction.TypeIncome, nil
	case string(transaction.TypeExpense):
		return transaction.TypeExpense, nil
	case string(transaction.TypeTransfer):
		return transaction.TypeTransfer, nil
	default:
		return transaction.Type(""), ErrInvalidTransactionType
	}
}

func parseTransactionStatus(value string) (transaction.Status, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(transaction.StatusPlanned):
		return transaction.StatusPlanned, nil
	case string(transaction.StatusConfirmed):
		return transaction.StatusConfirmed, nil
	case string(transaction.StatusCancelled):
		return transaction.StatusCancelled, nil
	default:
		return transaction.Status(""), ErrInvalidInput
	}
}

func isCategoryCompatible(txType transaction.Type, catType category.Type) bool {
	switch txType {
	case transaction.TypeTransfer:
		return catType == category.TypeTransfer
	case transaction.TypeIncome:
		return catType == category.TypeIncome
	case transaction.TypeExpense:
		return catType == category.TypeExpense
	default:
		return false
	}
}

func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, ErrInvalidInput
	}

	patterns := []string{time.RFC3339, "2006-01-02"}
	for _, pattern := range patterns {
		if t, err := time.Parse(pattern, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, ErrInvalidInput
}

// isHistoricalDate checks if the given date is before today.
// Historical transactions (past dates) should skip balance validation
// since they already happened.
func isHistoricalDate(date time.Time) bool {
	today := time.Now().Truncate(24 * time.Hour)
	return date.Before(today)
}
