package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ErrNotACreditCard is returned when credit figures are asked of an account that has
// no limit to report on.
// It is a refusal, not a failure: carrying the marker keeps it a 400 wherever a
// payment route reports it.
var ErrNotACreditCard = refuse("account is not a credit card")

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

	outstanding, err := NewCardOutstanding(uc.invoiceRepo, uc.txRepo).For(account)
	if err != nil {
		return nil, err
	}

	usage := account.CreditUsageFor(outstanding)
	return &usage, nil
}
