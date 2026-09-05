package usecases

import (
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ErrNotACreditCard is returned when credit figures are asked of an account that has
// no limit to report on.
var ErrNotACreditCard = errors.New("account is not a credit card")

// GetCreditUsageUseCase answers how much of a card's limit is committed.
//
// The figure comes from what is still owed on the card's invoices, plus any confirmed
// charge that never joined one. It deliberately does NOT come from the account's
// balance: this system does not move a card's balance as purchases are created — only
// paying an invoice does — so a balance-derived answer reports the card as emptier
// than it is, which is the very bug this replaces.
type GetCreditUsageUseCase struct {
	accountRepo bankaccount.Repository
	invoiceRepo invoice.Repository
	txRepo      transaction.Repository
}

func NewGetCreditUsageUseCase(
	accountRepo bankaccount.Repository,
	invoiceRepo invoice.Repository,
	txRepo transaction.Repository,
) *GetCreditUsageUseCase {
	return &GetCreditUsageUseCase{accountRepo: accountRepo, invoiceRepo: invoiceRepo, txRepo: txRepo}
}

func (uc *GetCreditUsageUseCase) Execute(accountID string) (*bankaccount.CreditUsage, error) {
	account, err := uc.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}
	if !account.IsCreditCard() {
		return nil, ErrNotACreditCard
	}

	outstanding, err := uc.outstandingFor(account)
	if err != nil {
		return nil, err
	}

	usage := account.CreditUsageFor(outstanding)
	return &usage, nil
}

func (uc *GetCreditUsageUseCase) outstandingFor(account *bankaccount.BankAccount) (float64, error) {
	invoices, err := uc.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return 0, err
	}

	outstanding := 0.0
	invoiced := make(map[string]bool, len(invoices))
	for _, inv := range invoices {
		invoiced[inv.ID] = true

		// The invoice's `amount` column is not the source of truth: it is written on
		// creation as zero and only refreshed by a manual endpoint, so in production
		// every open and closed bill still reads 0 there. Every other read path
		// recomputes it from the transactions, and so does this one.
		billed, err := uc.txRepo.SumByInvoiceID(inv.ID)
		if err != nil {
			return 0, err
		}

		switch {
		case inv.Status != invoice.StatusPaid:
			outstanding += billed
		case inv.PaidAmount != nil && *inv.PaidAmount < billed:
			// Paying anything at all marks a bill PAID, so a partial payment would
			// otherwise drop the whole remaining balance from view.
			outstanding += billed - *inv.PaidAmount
		}
	}

	// A charge that never joined an invoice is still owed — the bill it belongs to
	// simply does not know about it yet.
	// The repository requires a profile: without it every call fails and the endpoint
	// answers 500, which is exactly what shipped when the tests used a fake that
	// ignored the filter.
	txs, err := uc.txRepo.List(transaction.ListFilter{
		ProfileID:     account.ProfileID,
		BankAccountID: &account.ID,
	})
	if err != nil {
		return 0, err
	}
	for _, tx := range txs {
		if tx.Type != transaction.TypeExpense || tx.Status != transaction.StatusConfirmed {
			continue
		}
		if tx.InvoiceID == nil || !invoiced[*tx.InvoiceID] {
			outstanding += tx.Amount
		}
	}

	return outstanding, nil
}
