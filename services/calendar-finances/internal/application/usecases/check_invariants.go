package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Notes explaining a difference that is expected today. An entry carrying a note
// is reported for visibility but does not fail the check, because there is no
// action available that would clear it. Each note is meant to be deleted as the
// gap it describes is closed.
const (
	// Stock, FII and crypto positions hold quotas x quote, written by the price
	// pollers. Their balance is not a ledger and never will be while the two
	// live in the same column.
	noteMarketValue = "stored balance tracks market quotes, not transactions"

	// A card at exactly zero has never been written by any path: every write
	// path guards cards out of the balance, and only PayInvoiceUseCaseV2 ever
	// sets one. A card holding a non-zero balance IS maintained, so its drift
	// is a real defect and is reported without this excuse.
	noteCreditCardUnmaintained = "credit card balance was never written by any path"

	// credit_card_invoices.amount has no owner: no write path maintains it, the
	// read paths recompute it in memory without persisting, and the one route
	// that does persist it refuses PAID invoices. Every stored value is stale by
	// construction, so this can never be brought to zero and must not gate the
	// check. Note also that the computed side counts every non-CANCELLED row,
	// including PLANNED, while the account check counts CONFIRMED only.
	noteInvoiceDerived = "invoice totals are recomputed on read and never persisted"
)

// AccountInvariant is one account whose stored balance differs from the balance
// its own transactions justify.
type AccountInvariant struct {
	AccountID       string  `json:"accountId"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	IsActive        bool    `json:"isActive"`
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
	Status         string  `json:"status"`
	StoredAmount   float64 `json:"storedAmount"`
	ComputedAmount float64 `json:"computedAmount"`
	Drift          float64 `json:"drift"`
	Note           string  `json:"note,omitempty"`
}

// CheckInvariantsResult is a read-only report. Nothing here writes: a drift is a
// transaction to hunt down, never a number to overwrite.
//
// OK is false only for differences that someone can actually act on. A signal
// that cannot be brought to zero stops being read.
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
// `initial_balance + sum(CONFIRMED transactions)`, and for every credit-card
// invoice, the stored total against the sum of its linked transactions.
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
	// round2 puts both sides on cent granularity, so anything left is a real
	// difference rather than the float noise of summing amounts.
	drift := round2(account.CurrentBalance - computed)
	if drift == 0 {
		return nil
	}

	note := expectedDifferenceNote(account)
	result.AccountDrifts = append(result.AccountDrifts, AccountInvariant{
		AccountID:       account.ID,
		Name:            account.Name,
		Type:            string(account.Type),
		IsActive:        account.IsActive,
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
		if drift == 0 {
			continue
		}

		result.InvoiceDrifts = append(result.InvoiceDrifts, InvoiceInvariant{
			InvoiceID:      inv.ID,
			BankAccountID:  inv.BankAccountID,
			ReferenceDate:  inv.ReferenceDate.Format("2006-01"),
			Status:         string(inv.Status),
			StoredAmount:   round2(inv.Amount),
			ComputedAmount: computed,
			Drift:          drift,
			Note:           noteInvoiceDerived,
		})
	}

	return nil
}

// expectedDifferenceNote returns why an account's stored balance is allowed to
// differ from its ledger today, or "" when the difference is a real defect
// someone can act on.
func expectedDifferenceNote(account *bankaccount.BankAccount) string {
	switch account.Type {
	case bankaccount.AccountTypeInvestment,
		bankaccount.AccountTypeExchange,
		bankaccount.AccountTypeWallet:
		return noteMarketValue

	case bankaccount.AccountTypeCreditCard:
		// Only a card nothing has ever written gets the excuse. Once
		// PayInvoiceUseCaseV2 has touched one, its balance is maintained by the
		// same invariant as any other account, and a difference is a real bug.
		if account.CurrentBalance == 0 {
			return noteCreditCardUnmaintained
		}
		return ""

	default:
		return ""
	}
}
