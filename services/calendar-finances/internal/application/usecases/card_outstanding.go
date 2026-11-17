package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// CardOutstanding answers what a credit card still owes, derived from its bills.
//
// It is the ONE owner of that number, by decision (issue #1277, 16/11/2026). A
// card's stored balance used to mean "the sum of the ledger the last time a bill
// was paid": six write paths skip credit cards on purpose and only the
// pay-invoice path writes, so between bills the stored figure stands still while
// purchases happen. Measured that day on one card: stored -455,70, derived
// 371,50, raw sum of every confirmed row 6.652,93 — and only the derived one
// matched the bank.
//
// Every read of a card's balance goes through here, so there is no second number
// to drift.
type CardOutstanding struct {
	invoiceRepo invoice.Repository
	txRepo      transaction.Repository
}

func NewCardOutstanding(invoiceRepo invoice.Repository, txRepo transaction.Repository) *CardOutstanding {
	return &CardOutstanding{invoiceRepo: invoiceRepo, txRepo: txRepo}
}

// Apply rewrites the balance of every credit card in the slice with the derived
// figure, NEGATIVE for money owed — the convention the stored column already
// used, so every total that adds balances keeps meaning what it meant.
func (c *CardOutstanding) Apply(accounts []*bankaccount.BankAccount) error {
	for _, account := range accounts {
		if account == nil || !account.IsCreditCard() {
			continue
		}
		owed, err := c.For(account)
		if err != nil {
			return err
		}
		account.CurrentBalance = -owed
	}
	return nil
}

func (c *CardOutstanding) For(account *bankaccount.BankAccount) (float64, error) {
	invoices, err := c.invoiceRepo.FindByBankAccountID(account.ID)
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
		billed, err := c.txRepo.SumByInvoiceID(inv.ID)
		if err != nil {
			return 0, err
		}

		// What is owed is always billed minus paid, whatever the status. Branching on
		// status instead put PARTIALLY_PAID in the "owes everything" case and counted
		// money that had already left the checking account.
		//
		// A settled bill with no recorded paidAmount owes nothing: the status is the
		// only evidence there is. Anything else defaults to nothing paid yet, which
		// also covers legacy rows where a partial payment was stored as PAID.
		paid := 0.0
		switch {
		case inv.Status == invoice.StatusRefinanced:
			// Converted: settled, owing nothing. The instalments it became are
			// charges of later cycles and counted there. A charge that lands on the
			// cycle afterwards is surfaced as drift by the invariants check, not as
			// debt here — debt nobody can pay would be worse than debt nobody sees.
			paid = billed
		case inv.PaidAmount != nil:
			paid = *inv.PaidAmount
		case inv.Status == invoice.StatusPaid:
			paid = billed
		}
		if remaining := billed - paid; remaining > 0 {
			outstanding += remaining
		}
	}

	// A charge that never joined an invoice is still owed — the bill it belongs to
	// simply does not know about it yet.
	// The repository requires a profile: without it every call fails and the endpoint
	// answers 500, which is exactly what shipped when the tests used a fake that
	// ignored the filter.
	txs, err := c.txRepo.List(transaction.ListFilter{
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
