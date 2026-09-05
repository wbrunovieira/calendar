package usecases

import (
	"fmt"
	"math"
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
	ProfileID               string                        `json:"profileId"`
	BankAccountID           string                        `json:"bankAccountId"`
	DestinationAccountID    *string                       `json:"destinationAccountId,omitempty"`
	CategoryID              *string                       `json:"categoryId,omitempty"`
	DestinationCategoryID   *string                       `json:"destinationCategoryId,omitempty"` // Category for the INCOME side of cross-profile transfers
	Type                    string                        `json:"type"`
	Status                  *string                       `json:"status,omitempty"`
	Amount                  float64                       `json:"amount"`
	Currency                string                        `json:"currency"`
	Description             string                        `json:"description"`
	Notes                   *string                       `json:"notes,omitempty"`
	CostCenter              *string                       `json:"costCenter,omitempty"`
	IsPersonalReimbursement bool                          `json:"isPersonalReimbursement"`
	OccurredOn              string                        `json:"occurredOn"`
	DueOn                   *string                       `json:"dueOn,omitempty"`
	ReminderOn              *string                       `json:"reminderOn,omitempty"` // Optional reminder date for alerts (10, 5, 1, 0 days before)
	RecurrenceRule          *string                       `json:"recurrenceRule,omitempty"`
	InstallmentNumber       *int                          `json:"installmentNumber,omitempty"`
	InstallmentTotal        *int                          `json:"installmentTotal,omitempty"`
	ExternalID              *string                       `json:"externalId,omitempty"`
	Tags                    []string                      `json:"tags,omitempty"`
	Splits                  []CreateTransactionSplitInput `json:"splits,omitempty"`
}

type CreateTransactionUseCase struct {
	profileRepo         profile.Repository
	accountRepo         bankaccount.Repository
	categoryRepo        category.Repository
	transactionRepo     transaction.Repository
	invoiceRepo         invoice.Repository
	balanceRecalculator BalanceRecalculator
}

func NewCreateTransactionUseCase(
	profileRepo profile.Repository,
	accountRepo bankaccount.Repository,
	categoryRepo category.Repository,
	transactionRepo transaction.Repository,
	invoiceRepo invoice.Repository,
	recalculator BalanceRecalculator,
) *CreateTransactionUseCase {
	return &CreateTransactionUseCase{
		profileRepo:         profileRepo,
		accountRepo:         accountRepo,
		categoryRepo:        categoryRepo,
		transactionRepo:     transactionRepo,
		invoiceRepo:         invoiceRepo,
		balanceRecalculator: recalculator,
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

	// Detect cross-profile transfer first (affects category validation)
	var destinationAccountID *string
	var destinationAccount *bankaccount.BankAccount
	isCrossProfile := false
	if typeValue == transaction.TypeTransfer {
		if input.DestinationAccountID == nil {
			return nil, ErrDestinationRequired
		}
		destinationAccount, err = uc.accountRepo.FindByID(*input.DestinationAccountID)
		if err != nil {
			return nil, ErrBankAccountNotFound
		}
		if destinationAccount.ID == account.ID {
			return nil, ErrInvalidInput
		}
		isCrossProfile = destinationAccount.ProfileID != input.ProfileID
		if isCrossProfile {
			// Cross-profile transfer: requires destination category
			if input.DestinationCategoryID == nil {
				return nil, ErrDestinationCategoryRequired
			}
			// Validate destination category belongs to destination profile and is INCOME
			destCat, err := uc.categoryRepo.FindByID(*input.DestinationCategoryID)
			if err != nil {
				return nil, ErrCategoryNotFound
			}
			if destCat.ProfileID != destinationAccount.ProfileID {
				return nil, ErrCategoryNotFound
			}
			if destCat.Type != category.TypeIncome {
				return nil, ErrInvalidInput
			}
		}
		destinationAccountID = &destinationAccount.ID
	}

	// Validate source category
	var categoryEntity *category.Category
	if input.CategoryID != nil {
		categoryEntity, err = uc.categoryRepo.FindByID(*input.CategoryID)
		if err != nil {
			return nil, ErrCategoryNotFound
		}
		if categoryEntity.ProfileID != input.ProfileID {
			return nil, ErrCategoryNotFound
		}
		// For cross-profile transfers, source category should be EXPENSE (not TRANSFER)
		expectedType := typeValue
		if isCrossProfile {
			expectedType = transaction.TypeExpense
		}
		if !isCategoryCompatible(expectedType, categoryEntity.Type) {
			return nil, ErrInvalidInput
		}
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

	// Handle installments: when installmentTotal > 1 and installmentNumber is NOT set,
	// auto-create multiple transactions. When installmentNumber IS set, the caller is
	// creating a specific installment manually — skip auto-creation.
	if input.InstallmentTotal != nil && *input.InstallmentTotal > 1 && input.InstallmentNumber == nil {
		return uc.createInstallments(input, account, typeValue, occurredOn, dueOn, reminderOn, splits)
	}

	// When installmentTotal is 1, auto-set installmentNumber to 1 if not provided
	if input.InstallmentTotal != nil && *input.InstallmentTotal == 1 && input.InstallmentNumber == nil {
		one := 1
		input.InstallmentNumber = &one
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

	// Determine the effective type for the source transaction
	effectiveType := typeValue
	if isCrossProfile {
		effectiveType = transaction.TypeExpense // Cross-profile: source becomes EXPENSE
	}

	createParams := transaction.CreateParams{
		ProfileID:               input.ProfileID,
		BankAccountID:           input.BankAccountID,
		DestinationAccountID:    destinationAccountID,
		CategoryID:              input.CategoryID,
		InvoiceID:               invoiceID,
		Type:                    effectiveType,
		Amount:                  input.Amount,
		Currency:                input.Currency,
		Description:             input.Description,
		Notes:                   input.Notes,
		CostCenter:              input.CostCenter,
		IsPersonalReimbursement: input.IsPersonalReimbursement,
		OccurredOn:              occurredOn,
		DueOn:                   dueOn,
		ReminderOn:              reminderOn,
		RecurrenceRule:          input.RecurrenceRule,
		InstallmentNumber:       input.InstallmentNumber,
		InstallmentTotal:        input.InstallmentTotal,
		ExternalID:              input.ExternalID,
		Tags:                    input.Tags,
		Splits:                  splits,
	}

	txn, err := transaction.New(createParams)
	if err != nil {
		return nil, err
	}

	// Set status if provided (defaults to PLANNED in transaction.New)
	var txnStatus transaction.Status
	if input.Status != nil {
		txnStatus, err = parseTransactionStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		txn.Status = txnStatus
	} else {
		txnStatus = txn.Status
	}

	// Cross-profile transfer: create paired INCOME transaction in destination profile
	if isCrossProfile {
		destParams := transaction.CreateParams{
			ProfileID:     destinationAccount.ProfileID,
			BankAccountID: destinationAccount.ID,
			CategoryID:    input.DestinationCategoryID,
			Type:          transaction.TypeIncome,
			Amount:        input.Amount,
			Currency:      input.Currency,
			Description:   input.Description,
			Notes:         input.Notes,
			OccurredOn:    occurredOn,
		}
		destTxn, err := transaction.New(destParams)
		if err != nil {
			return nil, err
		}
		destTxn.Status = txnStatus

		// Link transactions to each other
		txn.LinkedTransactionID = &destTxn.ID
		destTxn.LinkedTransactionID = &txn.ID

		if err := uc.transactionRepo.Create(txn); err != nil {
			return nil, err
		}
		if err := uc.transactionRepo.Create(destTxn); err != nil {
			return nil, err
		}

		// Update balances for CONFIRMED cross-profile transfers
		if txnStatus == transaction.StatusConfirmed {
			// Debit source
			account.CurrentBalance -= input.Amount
			account.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(account); err != nil {
				return nil, err
			}
			// Credit destination
			destinationAccount.CurrentBalance += input.Amount
			destinationAccount.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(destinationAccount); err != nil {
				return nil, err
			}
			recalculateAccounts(uc.balanceRecalculator, account.ID, destinationAccount.ID)
		}

		return txn, nil
	}

	// Same-profile transfer or non-transfer: existing behavior
	if err := uc.transactionRepo.Create(txn); err != nil {
		return nil, err
	}

	// Update bank account balance for CONFIRMED transactions
	// NOTE: Credit card transactions do NOT update balance - the balance is only
	// affected when the invoice is paid (via PayInvoiceUseCaseV2)
	if txnStatus == transaction.StatusConfirmed && account.Type != bankaccount.AccountTypeCreditCard {
		if err := uc.updateAccountBalance(account, effectiveType, input.Amount); err != nil {
			return nil, err
		}

		// For same-profile TRANSFER: also credit the destination account
		if effectiveType == transaction.TypeTransfer && destinationAccountID != nil {
			destinationAccount.CurrentBalance += input.Amount
			destinationAccount.UpdatedAt = time.Now()
			if err := uc.accountRepo.Update(destinationAccount); err != nil {
				return nil, err
			}
			recalculateAccounts(uc.balanceRecalculator, account.ID, *destinationAccountID)
		} else {
			recalculateAccounts(uc.balanceRecalculator, account.ID)
		}
	}

	return txn, nil
}

// updateAccountBalance updates the bank account balance based on transaction type
func (uc *CreateTransactionUseCase) updateAccountBalance(account *bankaccount.BankAccount, txType transaction.Type, amount float64) error {
	switch txType {
	case transaction.TypeExpense:
		account.CurrentBalance -= amount
	case transaction.TypeIncome:
		account.CurrentBalance += amount
	case transaction.TypeTransfer:
		// For transfers, decrease source account balance
		// (destination account handled separately if needed)
		account.CurrentBalance -= amount
	}
	account.UpdatedAt = time.Now()
	return uc.accountRepo.Update(account)
}

// getOrCreateInvoiceForDate gets or creates the appropriate invoice for a transaction date.
func (uc *CreateTransactionUseCase) getOrCreateInvoiceForDate(account *bankaccount.BankAccount, txDate time.Time) (*invoice.Invoice, error) {
	return getOrCreateInvoiceForDate(uc.invoiceRepo, account, txDate)
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

// createInstallments creates multiple transactions for installment purchases.
// Total amount is divided equally across installments, with any remainder
// added to the first installment. Each installment is placed one month apart.
func (uc *CreateTransactionUseCase) createInstallments(
	input CreateTransactionInput,
	account *bankaccount.BankAccount,
	typeValue transaction.Type,
	occurredOn time.Time,
	dueOn *time.Time,
	reminderOn *time.Time,
	splits []*transaction.Split,
) (*transaction.Transaction, error) {
	total := *input.InstallmentTotal
	installmentAmount := math.Floor(input.Amount/float64(total)*100) / 100
	remainder := math.Round((input.Amount-installmentAmount*float64(total))*100) / 100

	effectiveType := typeValue

	var txnStatus transaction.Status
	if input.Status != nil {
		var err error
		txnStatus, err = parseTransactionStatus(*input.Status)
		if err != nil {
			return nil, err
		}
	} else {
		txnStatus = transaction.StatusPlanned
	}

	var firstTxn *transaction.Transaction

	for i := 1; i <= total; i++ {
		installmentDate := occurredOn.AddDate(0, i-1, 0)
		amount := installmentAmount
		if i == 1 {
			amount += remainder
		}

		installmentNum := i
		installmentTotal := total
		description := fmt.Sprintf("%s - Parcela %d/%d", input.Description, i, total)

		// Handle credit card invoice assignment
		var invoiceID *string
		if account.Type == bankaccount.AccountTypeCreditCard && typeValue == transaction.TypeExpense {
			if account.ClosingDay != nil && account.DueDay != nil {
				inv, err := uc.getOrCreateInvoiceForDate(account, installmentDate)
				if err != nil {
					return nil, err
				}
				if inv != nil {
					invoiceID = &inv.ID
				}
			}
		}

		createParams := transaction.CreateParams{
			ProfileID:               input.ProfileID,
			BankAccountID:           input.BankAccountID,
			CategoryID:              input.CategoryID,
			InvoiceID:               invoiceID,
			Type:                    effectiveType,
			Amount:                  amount,
			Currency:                input.Currency,
			Description:             description,
			Notes:                   input.Notes,
			CostCenter:              input.CostCenter,
			IsPersonalReimbursement: input.IsPersonalReimbursement,
			OccurredOn:              installmentDate,
			DueOn:                   dueOn,
			ReminderOn:              reminderOn,
			InstallmentNumber:       &installmentNum,
			InstallmentTotal:        &installmentTotal,
			ExternalID:              input.ExternalID,
			Tags:                    input.Tags,
			Splits:                  splitsForInstallment(splits, amount, input.Amount),
		}

		txn, err := transaction.New(createParams)
		if err != nil {
			return nil, err
		}
		txn.Status = txnStatus

		if err := uc.transactionRepo.Create(txn); err != nil {
			return nil, err
		}

		if i == 1 {
			firstTxn = txn
		}
	}

	// The rows are written; the balance still has to follow them. A credit card
	// is settled when its invoice is paid, not when the card is used, so it
	// stays out of this.
	if txnStatus == transaction.StatusConfirmed && account.Type != bankaccount.AccountTypeCreditCard {
		recalculateAccounts(uc.balanceRecalculator, account.ID)
	}

	return firstTxn, nil
}

// splitsForInstallment divides a purchase's category breakdown across one
// installment.
//
// Every installment gets freshly built splits: passing the caller's slice
// straight through made all N installments share the same *Split values, so
// they shared one primary key and each carried the whole purchase's amounts —
// which on its own exceeds the installment and is rejected.
//
// Amounts are scaled to the installment's share of the purchase, so the
// breakdown adds up to the installment rather than to the purchase. Splits that
// cover only part of a purchase keep covering the same part of it.
func splitsForInstallment(splits []*transaction.Split, installmentAmount, totalAmount float64) []*transaction.Split {
	if len(splits) == 0 || totalAmount <= 0 {
		return nil
	}

	share := installmentAmount / totalAmount

	var covered float64
	for _, split := range splits {
		covered += split.Amount
	}
	target := round2(covered * share)

	out := make([]*transaction.Split, 0, len(splits))
	var allocated float64
	largest := -1

	for _, split := range splits {
		amount := round2(split.Amount * share)
		if amount <= 0 {
			continue
		}

		categoryID := split.CategoryID
		if categoryID != nil {
			value := *categoryID
			categoryID = &value
		}
		memo := split.Memo
		if memo != nil {
			value := *memo
			memo = &value
		}

		out = append(out, &transaction.Split{
			CategoryID: categoryID,
			Amount:     amount,
			Memo:       memo,
		})
		allocated += amount

		if largest == -1 || amount > out[largest].Amount {
			largest = len(out) - 1
		}
	}

	// Rounding each split on its own can land a cent or two off the share the
	// installment actually covers. Put the difference on the largest split so
	// the breakdown adds up instead of drifting.
	if largest >= 0 {
		if diff := round2(target - allocated); diff != 0 {
			if adjusted := round2(out[largest].Amount + diff); adjusted > 0 {
				out[largest].Amount = adjusted
			}
		}
	}

	return out
}
