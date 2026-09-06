package usecases

import (
	"errors"
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

	created, err := createInvoiceWithFreeLabel(uc.invoiceRepo, inv, txDate, refDate)
	if err != nil {
		return nil, err
	}
	inv = created

	return inv, nil
}

// ErrInvoiceReferenceConflict means no free reference label could be found for a
// billing cycle. It is an operator-actionable data conflict, not a server fault,
// so handlers map it to 409 rather than 500.
var ErrInvoiceReferenceConflict = errors.New("invoice reference month conflict")

// referenceLabelSearchMonths bounds the forward scan for a free label. Exhausting
// it means dozens of consecutive months are taken on one card, which is corruption
// no retry will fix.
const referenceLabelSearchMonths = 24

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// candidateReferenceLabels lists reference months to try for a cycle, in order of
// preference: the month it closes, then the month it falls due, then forward.
//
// reference_date is only a LABEL; the billing period — opening, closing, due — is
// the truth, and no candidate here touches it. Two conventions exist in this
// database (older rows are labelled by due month, invoice.New by closing month), so
// both are tried before falling back to a merely-free month.
//
// The forward scan is not decoration. Cards with dueDay > closingDay fall due in the
// closing month, so the two conventions collapse into a single candidate; without
// the scan those cards would have no fallback at all and a label collision would
// refuse to record a real purchase.
//
// Note the relabel is not a one-off borrow: once a card takes a later label for one
// cycle, the next cycle collides with it and shifts too, so the card stays on the
// later convention. That is cosmetic — the UI renders the reference month — and is
// strictly better than blocking a purchase.
func candidateReferenceLabels(inv *invoice.Invoice) []time.Time {
	closing := firstOfMonth(inv.ClosingDate)
	due := firstOfMonth(inv.DueDate)

	labels := []time.Time{closing}
	if !due.Equal(closing) {
		labels = append(labels, due)
	}

	last := closing
	if due.After(last) {
		last = due
	}
	for i := 1; i <= referenceLabelSearchMonths; i++ {
		labels = append(labels, last.AddDate(0, i, 0))
	}
	return labels
}

// createInvoiceWithFreeLabel inserts inv, resolving collisions on the
// (bank_account_id, reference_date) unique constraint by relabelling it — never by
// moving the billing period, which is what corrupted the Nubank Juridica card.
//
// After every collision it re-reads by date, because a concurrent writer may have
// created exactly the cycle we need. Without that, the loser of a race reports a
// data conflict that does not exist.
func createInvoiceWithFreeLabel(repo invoice.Repository, inv *invoice.Invoice, txDate time.Time, refDate time.Time) (*invoice.Invoice, error) {
	for _, label := range candidateReferenceLabels(inv) {
		candidate := *inv
		candidate.ReferenceDate = label

		err := repo.Create(&candidate)
		if err == nil {
			return &candidate, nil
		}
		if !isUniqueViolation(err) {
			return nil, err
		}

		raced, findErr := repo.FindByBankAccountAndDate(inv.BankAccountID, txDate)
		if findErr != nil {
			return nil, findErr
		}
		if raced != nil {
			return raced, nil
		}
	}

	// Only on the failing path is it worth a scan to name the blocking row.
	return nil, referenceLabelExhaustedError(inv, conflictingInvoiceForLabel(repo, inv.BankAccountID, refDate))
}

// conflictingInvoiceForLabel finds the invoice holding a reference label on this
// card, for the error message only. A read failure must not become a misleading
// "repair your data" instruction, so it degrades to nil.
func conflictingInvoiceForLabel(repo invoice.Repository, bankAccountID string, refDate time.Time) *invoice.Invoice {
	existing, err := repo.FindByBankAccountID(bankAccountID)
	if err != nil {
		return nil
	}
	for _, ei := range existing {
		if ei.ReferenceDate.Year() == refDate.Year() && ei.ReferenceDate.Month() == refDate.Month() {
			return ei
		}
	}
	return nil
}

func referenceLabelExhaustedError(inv *invoice.Invoice, conflicting *invoice.Invoice) error {
	cycle := fmt.Sprintf("cycle %s to %s (due %s)",
		inv.OpeningDate.Format("2006-01-02"),
		inv.ClosingDate.Format("2006-01-02"),
		inv.DueDate.Format("2006-01-02"))

	if conflicting == nil {
		return fmt.Errorf("%w: no free reference month for %s on card %s after %d months",
			ErrInvoiceReferenceConflict, cycle, inv.BankAccountID, referenceLabelSearchMonths)
	}
	return fmt.Errorf("%w: no free reference month for %s; invoice %s covers %s to %s (due %s) — repair its reference_date",
		ErrInvoiceReferenceConflict, cycle, conflicting.ID,
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

	created, err := createInvoiceWithFreeLabel(repo, inv, txDate, refDate)
	if err != nil {
		return nil, err
	}
	inv = created

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
