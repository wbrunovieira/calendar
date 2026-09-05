package usecases

import (
	"math"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Notes explaining a difference that is expected today. An entry carrying a note
// is reported for visibility but does not fail the check.
const (
	noteMarketValue = "stored balance tracks market quotes, not transactions"
	noteCreditCard  = "credit card balances are not maintained by the write paths"
)

// driftTolerance is half a cent: below it the difference is float noise from
// summing amounts, not a missing or duplicated transaction.
const driftTolerance = 0.005

// AccountInvariant is one account whose stored balance differs from the balance
// its own transactions justify.
type AccountInvariant struct {
	AccountID       string  `json:"accountId"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	StoredBalance   float64 `json:"storedBalance"`
	ComputedBalance float64 `json:"computedBalance"`
	Drift           float64 `json:"drift"`
	Note            string  `json:"note,omitempty"`
}

// InvoiceInvariant is one invoice whose stored total differs from the sum of the
// transactions linked to it.
type InvoiceInvariant struct {
	InvoiceID      string  `json:"invoiceId"`
	BankAccountID  string  `json:"bankAccountId"`
	ReferenceDate  string  `json:"referenceDate"`
	StoredAmount   float64 `json:"storedAmount"`
	ComputedAmount float64 `json:"computedAmount"`
	Drift          float64 `json:"drift"`
}

// CheckInvariantsResult is a read-only report. Nothing here writes: a drift is a
// bug to hunt down in the transactions, never a number to overwrite.
type CheckInvariantsResult struct {
	CheckedAccounts int                `json:"checkedAccounts"`
	CheckedInvoices int                `json:"checkedInvoices"`
	AccountDrifts   []AccountInvariant `json:"accountDrifts"`
	InvoiceDrifts   []InvoiceInvariant `json:"invoiceDrifts"`
	OK              bool               `json:"ok"`
}

type CheckInvariantsUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transaction.Repository
	invoiceRepo invoice.Repository
}

func NewCheckInvariantsUseCase(
	accountRepo bankaccount.Repository,
	txRepo transaction.Repository,
	invoiceRepo invoice.Repository,
) *CheckInvariantsUseCase {
	return &CheckInvariantsUseCase{
		accountRepo: accountRepo,
		txRepo:      txRepo,
		invoiceRepo: invoiceRepo,
	}
}

// Execute compares, for every account, the stored balance against
// `initial_balance + Σ(CONFIRMED transactions)`, and for every credit card
// invoice, the stored total against the sum of its transactions.
func (uc *CheckInvariantsUseCase) Execute() (*CheckInvariantsResult, error) {
	accounts, err := uc.accountRepo.FindAll()
	if err != nil {
		return nil, err
	}

	result := &CheckInvariantsResult{
		AccountDrifts: []AccountInvariant{},
		InvoiceDrifts: []InvoiceInvariant{},
		OK:            true,
	}

	for _, account := range accounts {
		result.CheckedAccounts++

		if err := uc.checkAccountBalance(account, result); err != nil {
			return nil, err
		}

		if !account.IsCreditCard() {
			continue
		}
		if err := uc.checkInvoiceTotals(account, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (uc *CheckInvariantsUseCase) checkAccountBalance(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	ledger, err := uc.txRepo.CalculateBalanceByBankAccountID(account.ID)
	if err != nil {
		return err
	}

	computed := round2(account.InitialBalance + ledger)
	drift := round2(account.CurrentBalance - computed)
	if math.Abs(drift) < driftTolerance {
		return nil
	}

	note := expectedDifferenceNote(account)
	result.AccountDrifts = append(result.AccountDrifts, AccountInvariant{
		AccountID:       account.ID,
		Name:            account.Name,
		Type:            string(account.Type),
		StoredBalance:   round2(account.CurrentBalance),
		ComputedBalance: computed,
		Drift:           drift,
		Note:            note,
	})
	if note == "" {
		result.OK = false
	}
	return nil
}

func (uc *CheckInvariantsUseCase) checkInvoiceTotals(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	invoices, err := uc.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return err
	}

	for _, inv := range invoices {
		result.CheckedInvoices++

		computed, err := uc.txRepo.SumByInvoiceID(inv.ID)
		if err != nil {
			return err
		}

		computed = round2(computed)
		drift := round2(inv.Amount - computed)
		if math.Abs(drift) < driftTolerance {
			continue
		}

		result.InvoiceDrifts = append(result.InvoiceDrifts, InvoiceInvariant{
			InvoiceID:      inv.ID,
			BankAccountID:  inv.BankAccountID,
			ReferenceDate:  inv.ReferenceDate.Format("2006-01"),
			StoredAmount:   round2(inv.Amount),
			ComputedAmount: computed,
			Drift:          drift,
		})
		result.OK = false
	}

	return nil
}

// expectedDifferenceNote returns why an account's stored balance is allowed to
// differ from its ledger today, or "" when the difference is a real defect.
func expectedDifferenceNote(account *bankaccount.BankAccount) string {
	switch account.Type {
	case bankaccount.AccountTypeInvestment,
		bankaccount.AccountTypeExchange,
		bankaccount.AccountTypeWallet:
		return noteMarketValue
	case bankaccount.AccountTypeCreditCard:
		return noteCreditCard
	default:
		return ""
	}
}
