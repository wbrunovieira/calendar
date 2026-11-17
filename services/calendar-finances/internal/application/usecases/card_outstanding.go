package usecases

import (
	"time"

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
		owed, err := c.OwedNow(account, time.Now())
		if err != nil {
			return err
		}
		account.CurrentBalance = -owed
	}
	return nil
}

// For is everything the card has COMMITTED: every unpaid bill, future
// instalment cycles included. It is what the available limit is made of, because
// that is what the issuer reserves — the bank counts a January instalment
// against today's limit.
func (c *CardOutstanding) For(account *bankaccount.BankAccount) (float64, error) {
	return c.sum(account, func(*invoice.Invoice) bool { return true })
}

// OwedNow is what the card owes TODAY: bills already issued — those whose cycle
// has closed, plus the one currently accumulating — and nothing from cycles that
// have not opened yet.
//
// It exists because the two questions have different answers and were sharing a
// name. `SumByInvoiceID` counts PLANNED rows, and an instalment plan writes one
// PLANNED charge per future bill, so a card's balance was reporting the whole
// plan as if it were due. On the Mercado Pago card that is 4.502,45 committed
// against 1.358,65 actually owed — and the planning screens subtract the balance
// from today's cash.
//
// Decided with the owner on 16/11/2026, together with issue #1277.
func (c *CardOutstanding) OwedNow(account *bankaccount.BankAccount, on time.Time) (float64, error) {
	invoices, err := c.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return 0, err
	}
	issued := issuedInvoices(invoices, on)
	return c.sum(account, func(inv *invoice.Invoice) bool { return issued[inv.ID] })
}

// issuedInvoices names the bills that exist for the holder at `on`: every cycle
// already closed, plus the single earliest one still open. Anything later has
// not been billed and cannot be paid.
func issuedInvoices(invoices []*invoice.Invoice, on time.Time) map[string]bool {
	issued := make(map[string]bool, len(invoices))
	var current *invoice.Invoice
	for _, inv := range invoices {
		if !inv.ClosingDate.After(on) {
			issued[inv.ID] = true
			continue
		}
		// The open cycle is the nearest closing date still ahead.
		if current == nil || inv.ClosingDate.Before(current.ClosingDate) {
			current = inv
		}
	}
	if current != nil {
		issued[current.ID] = true
	}
	return issued
}

func (c *CardOutstanding) sum(account *bankaccount.BankAccount, counts func(*invoice.Invoice) bool) (float64, error) {
	invoices, err := c.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return 0, err
	}

	outstanding := 0.0
	invoiced := make(map[string]bool, len(invoices))
	for _, inv := range invoices {
		// Every bill marks its charges as invoiced, even one that does not count
		// towards this total: a future instalment is already on a bill, so it must
		// not be picked up again by the unlinked-charges pass below.
		invoiced[inv.ID] = true
		if !counts(inv) {
			continue
		}

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
