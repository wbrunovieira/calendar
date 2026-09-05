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

	// A credit card's balance is a frozen snapshot of the last invoice payment.
	// create_transaction, update_transaction and delete_transaction all skip
	// cards entirely — balance update and recalculation both — and only
	// PayInvoiceUseCaseV2 (and the manual recalculate route) ever writes one. So
	// every purchase made since the last payment shows up here as drift, with no
	// missing transaction behind it. Chasing that number would be chasing the
	// design, not a bug. Phase 1 makes the balance derived and deletes this note.
	noteCreditCardSnapshot = "credit card balance is a snapshot of the last invoice payment"

	// A PAID invoice's total cannot be brought back in line: POST
	// /invoices/{id}/recalculate refuses PAID. An OPEN or CLOSED one can, by
	// that same route, so it is a real finding and is reported without a note.
	notePaidInvoiceFrozen = "a paid invoice total can no longer be recalculated"
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
// OK is false only for differences someone can actually act on: a signal that
// cannot be brought to zero stops being read within a week. Everything else is
// still reported, with a note saying why no action is available.
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

		// An OPEN or CLOSED invoice can be brought back in line with one call to
		// POST /invoices/{id}/recalculate, which persists exactly this sum. That
		// makes its drift actionable, and a stored total that is double the sum
		// of its transactions is the signature of a duplicated or lost charge.
		// Only PAID is frozen, because that route refuses it.
		note := ""
		if inv.Status == invoice.StatusPaid {
			note = notePaidInvoiceFrozen
		}

		result.InvoiceDrifts = append(result.InvoiceDrifts, InvoiceInvariant{
			InvoiceID:      inv.ID,
			BankAccountID:  inv.BankAccountID,
			ReferenceDate:  inv.ReferenceDate.Format("2006-01"),
			Status:         string(inv.Status),
			StoredAmount:   round2(inv.Amount),
			ComputedAmount: computed,
			Drift:          drift,
			Note:           note,
		})
		if note == "" {
			result.OK = false
		}
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
		// Every card, whatever its balance. A card at zero was never paid; a
		// card holding a figure was written by the last invoice payment and has
		// been going stale by design ever since. Neither difference is a
		// transaction anyone can go and find.
		return noteCreditCardSnapshot

	default:
		return ""
	}
}
