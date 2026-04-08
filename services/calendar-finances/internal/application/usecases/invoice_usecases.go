package usecases

import (
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// CreateInvoiceInput contains the parameters to create an invoice
type CreateInvoiceInput struct {
	BankAccountID string `json:"bankAccountId"`
	ReferenceDate string `json:"referenceDate"` // Format: 2006-01
}

// CreateInvoiceUseCase creates a new credit card invoice
type CreateInvoiceUseCase struct {
	invoiceRepo invoice.Repository
	accountRepo bankaccount.Repository
}

func NewCreateInvoiceUseCase(
	invoiceRepo invoice.Repository,
	accountRepo bankaccount.Repository,
) *CreateInvoiceUseCase {
	return &CreateInvoiceUseCase{
		invoiceRepo: invoiceRepo,
		accountRepo: accountRepo,
	}
}

func (uc *CreateInvoiceUseCase) Execute(input CreateInvoiceInput) (*invoice.Invoice, error) {
	account, err := uc.accountRepo.FindByID(input.BankAccountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, ErrNotCreditCard
	}

	if account.ClosingDay == nil || account.DueDay == nil {
		return nil, ErrInvalidInput
	}

	refDate, err := parseYearMonth(input.ReferenceDate)
	if err != nil {
		return nil, ErrInvalidInput
	}

	params := invoice.CreateParams{
		BankAccountID: input.BankAccountID,
		ClosingDay:    *account.ClosingDay,
		DueDay:        *account.DueDay,
		ReferenceDate: refDate,
	}

	inv, err := invoice.New(params)
	if err != nil {
		return nil, err
	}

	if err := uc.invoiceRepo.Create(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// ListInvoicesInput contains the parameters to list invoices
type ListInvoicesInput struct {
	BankAccountID string `json:"bankAccountId"`
}

// ListInvoicesUseCase lists all invoices for a credit card
type ListInvoicesUseCase struct {
	invoiceRepo     invoice.Repository
	accountRepo     bankaccount.Repository
	transactionRepo transactionPkg.Repository
}

func NewListInvoicesUseCase(
	invoiceRepo invoice.Repository,
	accountRepo bankaccount.Repository,
	transactionRepo ...transactionPkg.Repository,
) *ListInvoicesUseCase {
	uc := &ListInvoicesUseCase{
		invoiceRepo: invoiceRepo,
		accountRepo: accountRepo,
	}
	if len(transactionRepo) > 0 {
		uc.transactionRepo = transactionRepo[0]
	}
	return uc
}

func (uc *ListInvoicesUseCase) Execute(bankAccountID string) ([]*invoice.Invoice, error) {
	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, ErrNotCreditCard
	}

	invoices, err := uc.invoiceRepo.FindByBankAccountID(bankAccountID)
	if err != nil {
		return nil, err
	}

	// Always recalculate amounts from transactions (source of truth)
	if uc.transactionRepo != nil {
		for _, inv := range invoices {
			total, err := uc.transactionRepo.SumByInvoiceID(inv.ID)
			if err == nil {
				inv.Amount = total
			}
			confirmed, err := uc.transactionRepo.SumByInvoiceIDByStatus(inv.ID, transactionPkg.StatusConfirmed)
			if err == nil {
				inv.ConfirmedAmount = confirmed
			}
			planned, err := uc.transactionRepo.SumByInvoiceIDByStatus(inv.ID, transactionPkg.StatusPlanned)
			if err == nil {
				inv.PlannedAmount = planned
			}
		}
	}

	return invoices, nil
}

// GetOrCreateInvoiceForDateUseCase gets or creates the appropriate invoice for a transaction date
type GetOrCreateInvoiceForDateUseCase struct {
	invoiceRepo invoice.Repository
	accountRepo bankaccount.Repository
}

func NewGetOrCreateInvoiceForDateUseCase(
	invoiceRepo invoice.Repository,
	accountRepo bankaccount.Repository,
) *GetOrCreateInvoiceForDateUseCase {
	return &GetOrCreateInvoiceForDateUseCase{
		invoiceRepo: invoiceRepo,
		accountRepo: accountRepo,
	}
}

func (uc *GetOrCreateInvoiceForDateUseCase) Execute(bankAccountID string, txDate time.Time) (*invoice.Invoice, error) {
	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, ErrNotCreditCard
	}

	if account.ClosingDay == nil || account.DueDay == nil {
		return nil, ErrInvalidInput
	}

	// Try to find existing invoice for this date
	inv, err := uc.invoiceRepo.FindByBankAccountAndDate(bankAccountID, txDate)
	if err != nil {
		return nil, err
	}
	if inv != nil {
		return inv, nil
	}

	// No invoice exists, create one
	// Determine which month this transaction belongs to
	refDate := calculateReferenceMonth(txDate, *account.ClosingDay)

	params := invoice.CreateParams{
		BankAccountID: bankAccountID,
		ClosingDay:    *account.ClosingDay,
		DueDay:        *account.DueDay,
		ReferenceDate: refDate,
	}

	inv, err = invoice.New(params)
	if err != nil {
		return nil, err
	}

	if err := uc.invoiceRepo.Create(inv); err != nil {
		// If duplicate key (invoice for this reference month already exists but with
		// different dates that don't contain our txDate), try the next month
		if isUniqueViolation(err) {
			nextMonth := refDate.AddDate(0, 1, 0)
			params.ReferenceDate = nextMonth
			inv, err = invoice.New(params)
			if err != nil {
				return nil, err
			}
			if err := uc.invoiceRepo.Create(inv); err != nil {
				return nil, err
			}
			return inv, nil
		}
		return nil, err
	}

	return inv, nil
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint"))
}

// calculateReferenceMonth determines which invoice month a transaction date belongs to
func calculateReferenceMonth(txDate time.Time, closingDay int) time.Time {
	year := txDate.Year()
	month := txDate.Month()
	day := txDate.Day()

	// If transaction day is on or after closing day, it goes to next month's invoice
	if day >= closingDay {
		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}

// CloseInvoiceInput contains the parameters to close an invoice
type CloseInvoiceInput struct {
	InvoiceID string `json:"invoiceId"`
}

// CloseInvoiceUseCase closes an invoice
type CloseInvoiceUseCase struct {
	invoiceRepo invoice.Repository
}

func NewCloseInvoiceUseCase(invoiceRepo invoice.Repository) *CloseInvoiceUseCase {
	return &CloseInvoiceUseCase{invoiceRepo: invoiceRepo}
}

func (uc *CloseInvoiceUseCase) Execute(invoiceID string) (*invoice.Invoice, error) {
	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	if err := inv.Close(); err != nil {
		return nil, ErrInvoiceNotOpen
	}

	if err := uc.invoiceRepo.Update(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// PayInvoiceInput contains the parameters to pay an invoice
type PayInvoiceInput struct {
	InvoiceID  string  `json:"invoiceId"`
	PaidAmount float64 `json:"paidAmount"`
	PaidAt     string  `json:"paidAt"` // Format: 2006-01-02 or RFC3339
}

// PayInvoiceUseCase marks an invoice as paid
type PayInvoiceUseCase struct {
	invoiceRepo invoice.Repository
}

func NewPayInvoiceUseCase(invoiceRepo invoice.Repository) *PayInvoiceUseCase {
	return &PayInvoiceUseCase{invoiceRepo: invoiceRepo}
}

func (uc *PayInvoiceUseCase) Execute(input PayInvoiceInput) (*invoice.Invoice, error) {
	inv, err := uc.invoiceRepo.FindByID(input.InvoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	paidAt, err := parseDate(input.PaidAt)
	if err != nil {
		return nil, ErrInvalidInput
	}

	if err := inv.Pay(input.PaidAmount, paidAt); err != nil {
		return nil, ErrInvoiceAlreadyPaid
	}

	if err := uc.invoiceRepo.Update(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// GetInvoiceUseCase retrieves a specific invoice
type GetInvoiceUseCase struct {
	invoiceRepo     invoice.Repository
	transactionRepo transactionPkg.Repository
}

func NewGetInvoiceUseCase(invoiceRepo invoice.Repository, transactionRepo ...transactionPkg.Repository) *GetInvoiceUseCase {
	uc := &GetInvoiceUseCase{invoiceRepo: invoiceRepo}
	if len(transactionRepo) > 0 {
		uc.transactionRepo = transactionRepo[0]
	}
	return uc
}

func (uc *GetInvoiceUseCase) Execute(invoiceID string) (*invoice.Invoice, error) {
	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	// Always recalculate amounts from transactions (source of truth)
	if uc.transactionRepo != nil {
		total, err := uc.transactionRepo.SumByInvoiceID(invoiceID)
		if err == nil {
			inv.Amount = total
		}
		confirmed, err := uc.transactionRepo.SumByInvoiceIDByStatus(invoiceID, transactionPkg.StatusConfirmed)
		if err == nil {
			inv.ConfirmedAmount = confirmed
		}
		planned, err := uc.transactionRepo.SumByInvoiceIDByStatus(invoiceID, transactionPkg.StatusPlanned)
		if err == nil {
			inv.PlannedAmount = planned
		}
	}

	return inv, nil
}

// GetCurrentInvoiceUseCase retrieves the current open invoice for a credit card
type GetCurrentInvoiceUseCase struct {
	invoiceRepo     invoice.Repository
	accountRepo     bankaccount.Repository
	transactionRepo transactionPkg.Repository
}

func NewGetCurrentInvoiceUseCase(
	invoiceRepo invoice.Repository,
	accountRepo bankaccount.Repository,
	transactionRepo ...transactionPkg.Repository,
) *GetCurrentInvoiceUseCase {
	uc := &GetCurrentInvoiceUseCase{
		invoiceRepo: invoiceRepo,
		accountRepo: accountRepo,
	}
	if len(transactionRepo) > 0 {
		uc.transactionRepo = transactionRepo[0]
	}
	return uc
}

func (uc *GetCurrentInvoiceUseCase) Execute(bankAccountID string) (*invoice.Invoice, error) {
	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, ErrNotCreditCard
	}

	inv, err := uc.invoiceRepo.FindOpenByBankAccountID(bankAccountID)
	if err != nil {
		return nil, err
	}

	// Always recalculate amounts from transactions (source of truth)
	if inv != nil && uc.transactionRepo != nil {
		total, err := uc.transactionRepo.SumByInvoiceID(inv.ID)
		if err == nil {
			inv.Amount = total
		}
		confirmed, err := uc.transactionRepo.SumByInvoiceIDByStatus(inv.ID, transactionPkg.StatusConfirmed)
		if err == nil {
			inv.ConfirmedAmount = confirmed
		}
		planned, err := uc.transactionRepo.SumByInvoiceIDByStatus(inv.ID, transactionPkg.StatusPlanned)
		if err == nil {
			inv.PlannedAmount = planned
		}
	}

	return inv, nil
}

// RecalculateInvoiceAmountUseCase recalculates the invoice amount based on associated transactions
type RecalculateInvoiceAmountUseCase struct {
	invoiceRepo     invoice.Repository
	transactionRepo transactionPkg.Repository
}

func NewRecalculateInvoiceAmountUseCase(
	invoiceRepo invoice.Repository,
	transactionRepo transactionPkg.Repository,
) *RecalculateInvoiceAmountUseCase {
	return &RecalculateInvoiceAmountUseCase{
		invoiceRepo:     invoiceRepo,
		transactionRepo: transactionRepo,
	}
}

func (uc *RecalculateInvoiceAmountUseCase) Execute(invoiceID string) (*invoice.Invoice, error) {
	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	if inv.Status == invoice.StatusPaid {
		return nil, ErrInvoiceAlreadyPaid
	}

	// Sum all transactions associated with this invoice
	total, err := uc.transactionRepo.SumByInvoiceID(invoiceID)
	if err != nil {
		return nil, err
	}

	// Update the invoice amount
	inv.Amount = total
	inv.UpdatedAt = time.Now()

	if err := uc.invoiceRepo.Update(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// UpdateInvoiceInput contains the parameters to update an invoice
type UpdateInvoiceInput struct {
	DueDate     *string `json:"dueDate"`
	ClosingDate *string `json:"closingDate"`
	OpeningDate *string `json:"openingDate"`
}

// UpdateInvoiceUseCase updates an invoice's editable fields
type UpdateInvoiceUseCase struct {
	invoiceRepo invoice.Repository
}

func NewUpdateInvoiceUseCase(invoiceRepo invoice.Repository) *UpdateInvoiceUseCase {
	return &UpdateInvoiceUseCase{invoiceRepo: invoiceRepo}
}

func (uc *UpdateInvoiceUseCase) Execute(invoiceID string, input UpdateInvoiceInput) (*invoice.Invoice, error) {
	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	if inv.Status == invoice.StatusPaid {
		return nil, ErrInvoiceAlreadyPaid
	}

	if input.DueDate != nil {
		d, err := parseDate(*input.DueDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		inv.DueDate = d
	}

	if input.ClosingDate != nil {
		d, err := parseDate(*input.ClosingDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		inv.ClosingDate = d
	}

	if input.OpeningDate != nil {
		d, err := parseDate(*input.OpeningDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		inv.OpeningDate = d
	}

	inv.UpdatedAt = time.Now()

	if err := uc.invoiceRepo.Update(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// AutoCloseInvoicesOutput contains the result of auto-closing invoices
type AutoCloseInvoicesOutput struct {
	Closed int      `json:"closed"`
	Failed int      `json:"failed"`
	IDs    []string `json:"ids"`
}

// AutoCloseInvoicesUseCase automatically closes open invoices past their closing date
type AutoCloseInvoicesUseCase struct {
	invoiceRepo invoice.Repository
}

func NewAutoCloseInvoicesUseCase(invoiceRepo invoice.Repository) *AutoCloseInvoicesUseCase {
	return &AutoCloseInvoicesUseCase{invoiceRepo: invoiceRepo}
}

func (uc *AutoCloseInvoicesUseCase) Execute(now time.Time) (*AutoCloseInvoicesOutput, error) {
	invoices, err := uc.invoiceRepo.FindOpenPastClosingDate(now)
	if err != nil {
		return nil, err
	}

	output := &AutoCloseInvoicesOutput{}

	for _, inv := range invoices {
		if err := inv.Close(); err != nil {
			output.Failed++
			continue
		}
		if err := uc.invoiceRepo.Update(inv); err != nil {
			output.Failed++
			continue
		}
		output.Closed++
		output.IDs = append(output.IDs, inv.ID)
	}

	return output, nil
}

// parseYearMonth parses a date in format "2006-01"
func parseYearMonth(value string) (time.Time, error) {
	t, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// PayInvoiceUseCaseV2 marks an invoice as paid and creates a payment transaction
// on the linked checking account
type PayInvoiceUseCaseV2 struct {
	invoiceRepo     invoice.Repository
	accountRepo     bankaccount.Repository
	transactionRepo transactionPkg.Repository
}

func NewPayInvoiceUseCaseV2(
	invoiceRepo invoice.Repository,
	accountRepo bankaccount.Repository,
	transactionRepo transactionPkg.Repository,
) *PayInvoiceUseCaseV2 {
	return &PayInvoiceUseCaseV2{
		invoiceRepo:     invoiceRepo,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

func (uc *PayInvoiceUseCaseV2) Execute(input PayInvoiceInput) (*invoice.Invoice, error) {
	inv, err := uc.invoiceRepo.FindByID(input.InvoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	paidAt, err := parseDate(input.PaidAt)
	if err != nil {
		return nil, ErrInvalidInput
	}

	if err := inv.Pay(input.PaidAmount, paidAt); err != nil {
		return nil, ErrInvoiceAlreadyPaid
	}

	if err := uc.invoiceRepo.Update(inv); err != nil {
		return nil, err
	}

	// Get the credit card to find the linked checking account
	creditCard, err := uc.accountRepo.FindByID(inv.BankAccountID)
	if err != nil {
		return inv, nil // Invoice is paid, but couldn't find card - return success
	}

	// If the credit card has a linked checking account, create a payment transaction
	if creditCard.LinkedAccountID != nil && *creditCard.LinkedAccountID != "" {
		linkedAccount, err := uc.accountRepo.FindByID(*creditCard.LinkedAccountID)
		if err != nil {
			return inv, nil // Invoice is paid, but couldn't find linked account - return success
		}

		// Create payment transaction on the linked checking account
		paymentTx, err := transactionPkg.New(transactionPkg.CreateParams{
			ProfileID:     linkedAccount.ProfileID,
			BankAccountID: linkedAccount.ID,
			Type:          transactionPkg.TypeExpense,
			Amount:        input.PaidAmount,
			Currency:      linkedAccount.Currency,
			Description:   "Pagamento fatura " + creditCard.Name,
			OccurredOn:    paidAt,
		})
		if err != nil {
			return inv, nil // Invoice is paid, but couldn't create transaction - return success
		}

		paymentTx.Status = transactionPkg.StatusConfirmed

		if err := uc.transactionRepo.Create(paymentTx); err != nil {
			return inv, nil // Invoice is paid, but couldn't save transaction - return success
		}

		// Update the linked checking account balance
		linkedAccount.CurrentBalance -= input.PaidAmount
		linkedAccount.UpdatedAt = time.Now()
		_ = uc.accountRepo.Update(linkedAccount)
	}

	return inv, nil
}
