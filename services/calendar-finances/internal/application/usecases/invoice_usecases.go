package usecases

import (
	"fmt"
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
			// Same reasoning as getOrCreateInvoiceForDate: never move a cycle
			// forward to dodge a label collision.
			if raced, findErr := uc.invoiceRepo.FindByBankAccountAndDate(bankAccountID, txDate); findErr == nil && raced != nil {
				return raced, nil
			}
			existing, _ := uc.invoiceRepo.FindByBankAccountID(bankAccountID)
			var conflicting *invoice.Invoice
			for _, ei := range existing {
				if ei.ReferenceDate.Year() == refDate.Year() && ei.ReferenceDate.Month() == refDate.Month() {
					conflicting = ei
					break
				}
			}
			if relabelled := relabelByDueMonth(inv); relabelled != nil {
				createErr := uc.invoiceRepo.Create(relabelled)
				if createErr == nil {
					return relabelled, nil
				}
				if !isUniqueViolation(createErr) {
					return nil, createErr
				}
			}
			return nil, referenceMonthConflictError(refDate, conflicting)
		}
		return nil, err
	}

	return inv, nil
}

// relabelByDueMonth returns a copy of inv labelled by the month it FALLS DUE
// instead of the month it closes, leaving opening/closing/due untouched.
//
// Older invoices on this database were labelled by their due month, so when that
// convention collides with invoice.New's closing-month label, borrowing the older
// convention frees the label without touching the billing period. Returns nil when
// both conventions land on the same month, since retrying would just collide again.
func relabelByDueMonth(inv *invoice.Invoice) *invoice.Invoice {
	dueMonth := time.Date(inv.DueDate.Year(), inv.DueDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	if dueMonth.Equal(inv.ReferenceDate) {
		return nil
	}
	relabelled := *inv
	relabelled.ReferenceDate = dueMonth
	return &relabelled
}

// referenceMonthConflictError describes a collision on the
// (bank_account_id, reference_date) unique constraint where the invoice already
// holding the label covers a different billing cycle than the one we need.
//
// It happens when two labelling conventions meet: older rows are labelled by the
// month the invoice FALLS DUE, while invoice.New labels by the month it CLOSES.
// The fix is to repair the mislabelled row's reference_date, which is why the
// message names it.
func referenceMonthConflictError(refDate time.Time, conflicting *invoice.Invoice) error {
	if conflicting == nil {
		return fmt.Errorf("invoice reference month %s is already taken on this card, but the conflicting invoice could not be read",
			refDate.Format("2006-01"))
	}
	return fmt.Errorf("invoice reference month %s is already used by invoice %s, which covers %s to %s (due %s); refusing to shift the new cycle forward — repair that invoice's reference_date",
		refDate.Format("2006-01"),
		conflicting.ID,
		conflicting.OpeningDate.Format("2006-01-02"),
		conflicting.ClosingDate.Format("2006-01-02"),
		conflicting.DueDate.Format("2006-01-02"))
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

// getOrCreateInvoiceForDate finds the invoice that covers txDate for the given
// credit-card account, creating one if none exists yet. Shared by
// CreateTransactionUseCase and UpdateTransactionUseCase so that switching a
// transaction to a different card always assigns a valid invoice.
func getOrCreateInvoiceForDate(repo invoice.Repository, account *bankaccount.BankAccount, txDate time.Time) (*invoice.Invoice, error) {
	inv, err := repo.FindByBankAccountAndDate(account.ID, txDate)
	if err != nil {
		return nil, err
	}
	if inv != nil {
		return inv, nil
	}

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

	if err := repo.Create(inv); err != nil {
		if !isUniqueViolation(err) {
			return nil, err
		}

		existingInvoices, listErr := repo.FindByBankAccountID(account.ID)
		if listErr != nil {
			return nil, listErr
		}

		var existingInv *invoice.Invoice
		for _, ei := range existingInvoices {
			if ei.ReferenceDate.Year() == refDate.Year() && ei.ReferenceDate.Month() == refDate.Month() {
				existingInv = ei
				break
			}
		}

		if existingInv != nil && existingInv.ContainsDate(txDate) {
			return existingInv, nil
		}

		// A concurrent writer may have created the invoice we need between our
		// lookup and our insert. Look again before treating this as a conflict.
		if raced, findErr := repo.FindByBankAccountAndDate(account.ID, txDate); findErr == nil && raced != nil {
			return raced, nil
		}

		// The reference month is taken by an invoice covering a DIFFERENT cycle.
		// reference_date is only a LABEL; opening/closing/due are the truth. So
		// relabel — never move the cycle.
		//
		// Moving it is what the old code did, and it made the dates lie: closing
		// and due jumped past the period the transaction belongs to, and
		// openingDate was stretched back to cover the gap, merging two cycles into
		// one invoice. That is how the Nubank Juridica card ended up with a single
		// invoice spanning 2026-07-28 to 2026-09-27: the cycle that should have
		// closed on 2026-08-27 never existed, and a real R$ 119,94 payment made on
		// 2026-09-03 had no invoice to land on.
		if relabelled := relabelByDueMonth(inv); relabelled != nil {
			createErr := repo.Create(relabelled)
			if createErr == nil {
				return relabelled, nil
			}
			if !isUniqueViolation(createErr) {
				return nil, createErr
			}
		}

		return nil, referenceMonthConflictError(refDate, existingInv)
	}

	return inv, nil
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

// PayInvoiceUseCaseV2 marks an invoice as paid and records the payment as a
// double-sided movement: an expense on the linked checking account (money out)
// AND a credit on the credit-card account (debt down). Recording the card side
// keeps the card balance a pure consequence of its transactions.
type PayInvoiceUseCaseV2 struct {
	invoiceRepo     invoice.Repository
	accountRepo     bankaccount.Repository
	transactionRepo transactionPkg.Repository
	recalculator    BalanceRecalculator // optional; recomputes card balance from transactions
}

func NewPayInvoiceUseCaseV2(
	invoiceRepo invoice.Repository,
	accountRepo bankaccount.Repository,
	transactionRepo transactionPkg.Repository,
	recalculator ...BalanceRecalculator,
) *PayInvoiceUseCaseV2 {
	uc := &PayInvoiceUseCaseV2{
		invoiceRepo:     invoiceRepo,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
	if len(recalculator) > 0 {
		uc.recalculator = recalculator[0]
	}
	return uc
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

	// Model the invoice payment as a single TRANSFER from the funding (linked) account to
	// the credit card. A transfer is naturally excluded from income/expense by every
	// consumer, while still debiting the funding account and crediting (paying down) the
	// card. Recording it as EXPENSE + INCOME polluted the monthly cashflow everywhere.
	if creditCard.LinkedAccountID != nil && *creditCard.LinkedAccountID != "" {
		if linkedAccount, lerr := uc.accountRepo.FindByID(*creditCard.LinkedAccountID); lerr == nil {
			cardID := creditCard.ID
			transferTx, terr := transactionPkg.New(transactionPkg.CreateParams{
				ProfileID:            linkedAccount.ProfileID,
				BankAccountID:        linkedAccount.ID,
				DestinationAccountID: &cardID,
				Type:                 transactionPkg.TypeTransfer,
				Amount:               input.PaidAmount,
				Currency:             linkedAccount.Currency,
				Description:          "Pagamento fatura " + creditCard.Name,
				OccurredOn:           paidAt,
			})
			if terr == nil {
				transferTx.Status = transactionPkg.StatusConfirmed
				if createErr := uc.transactionRepo.Create(transferTx); createErr == nil {
					// Debit the funding account, credit (pay down) the card.
					linkedAccount.CurrentBalance -= input.PaidAmount
					linkedAccount.UpdatedAt = time.Now()
					_ = uc.accountRepo.Update(linkedAccount)
					creditCard.CurrentBalance += input.PaidAmount
					creditCard.UpdatedAt = time.Now()
					_ = uc.accountRepo.Update(creditCard)
					// When wired, recompute both balances from transactions (authoritative).
					_ = recalculateAccounts(uc.recalculator, linkedAccount.ID, creditCard.ID)
				}
			}
		}
		return inv, nil
	}

	// Fallback: a card without a linked funding account. We don't know the payment source,
	// so we can only credit the card to keep its balance from drifting negative. (No real
	// card hits this path - all have a linked account.)
	cardCredit, err := transactionPkg.New(transactionPkg.CreateParams{
		ProfileID:     creditCard.ProfileID,
		BankAccountID: creditCard.ID,
		Type:          transactionPkg.TypeIncome,
		Amount:        input.PaidAmount,
		Currency:      creditCard.Currency,
		Description:   "Pagamento fatura " + creditCard.Name,
		OccurredOn:    paidAt,
	})
	if err == nil {
		cardCredit.Status = transactionPkg.StatusConfirmed
		if createErr := uc.transactionRepo.Create(cardCredit); createErr == nil {
			_ = recalculateAccounts(uc.recalculator, creditCard.ID)
		}
	}

	return inv, nil
}
