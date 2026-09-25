package usecases

import (
	"errors"
	"fmt"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// PinTransactionInvoiceUseCase files a card charge onto a chosen bill and keeps it
// there.
//
// Deriving the bill from the transaction date is a good default and a wrong rule for
// one real case: an issuer charges a late fee on the bill that GENERATED it, not on
// the bill covering the day it posted. Four such charges on the Mercado Pago card are
// dated 10/08 and were billed in the cycle that closed 09/08, so the system's August
// bill read 2.647,97 against the 2.702,66 actually charged — and paid.
//
// The alternative was moving the dates, which a ledger must not do: the date is what
// the statement says, and it was never the thing that was wrong.
type PinTransactionInvoiceUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transactionPkg.Repository
	invoiceRepo invoice.Repository
}

func NewPinTransactionInvoiceUseCase(
	accountRepo bankaccount.Repository,
	txRepo transactionPkg.Repository,
	invoiceRepo invoice.Repository,
) *PinTransactionInvoiceUseCase {
	return &PinTransactionInvoiceUseCase{accountRepo: accountRepo, txRepo: txRepo, invoiceRepo: invoiceRepo}
}

// Execute pins the transaction to the given invoice.
func (uc *PinTransactionInvoiceUseCase) Execute(transactionID, invoiceID string) error {
	txn, account, err := uc.load(transactionID)
	if err != nil {
		return err
	}
	if txn.PaidInvoiceID != nil {
		// A payment settles a bill; it is not a line of one. Filing it in would turn
		// a settled bill into a smaller bill instead of a paid one.
		return errors.New("this is an invoice payment, not a line of a bill")
	}

	inv, err := uc.invoiceRepo.FindByID(invoiceID)
	if err != nil {
		return fmt.Errorf("reading the bill: %w", err)
	}
	if inv == nil {
		return errors.New("invoice not found")
	}
	if inv.BankAccountID != account.ID {
		// Filing a charge onto another card's bill is not a choice worth honouring.
		return errors.New("that bill belongs to another card")
	}

	txn.InvoiceID = &inv.ID
	txn.InvoicePinned = true
	if err := uc.txRepo.Update(txn); err != nil {
		return fmt.Errorf("pinning the charge: %w", err)
	}
	return nil
}

// Release hands the transaction back to the date rule, and re-derives its bill
// immediately — leaving it pointing at a bill the date does not support would just
// be the old defect under a new name.
func (uc *PinTransactionInvoiceUseCase) Release(transactionID string) error {
	txn, account, err := uc.load(transactionID)
	if err != nil {
		return err
	}
	target, err := uc.invoiceRepo.FindByBankAccountAndDate(account.ID, txn.OccurredOn)
	if err != nil {
		return fmt.Errorf("finding the bill for the date: %w", err)
	}

	txn.InvoicePinned = false
	if target != nil {
		txn.InvoiceID = &target.ID
	} else {
		txn.InvoiceID = nil
	}
	if err := uc.txRepo.Update(txn); err != nil {
		return fmt.Errorf("releasing the charge: %w", err)
	}
	return nil
}

func (uc *PinTransactionInvoiceUseCase) load(transactionID string) (*transactionPkg.Transaction, *bankaccount.BankAccount, error) {
	txn, err := uc.txRepo.GetByID(transactionID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the transaction: %w", err)
	}
	if txn == nil {
		return nil, nil, transactionPkg.ErrNotFound
	}
	account, err := uc.accountRepo.FindByID(txn.BankAccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the account: %w", err)
	}
	if account == nil {
		return nil, nil, bankaccount.ErrNotFound
	}
	if account.Type != bankaccount.AccountTypeCreditCard {
		return nil, nil, ErrNotACreditCard
	}
	return txn, account, nil
}
