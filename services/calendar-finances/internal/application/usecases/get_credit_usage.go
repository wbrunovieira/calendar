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
	// outstanding is the shared owner of the derivation. Built once and held, not
	// constructed per call: "one owner" has to be one instance, or the sentence
	// is only true by coincidence.
	outstanding *CardOutstanding
}

func NewGetCreditUsageUseCase(
	accountRepo bankaccount.Repository,
	invoiceRepo invoice.Repository,
	txRepo transaction.Repository,
) *GetCreditUsageUseCase {
	return &GetCreditUsageUseCase{accountRepo: accountRepo, outstanding: NewCardOutstanding(invoiceRepo, txRepo)}
}

func (uc *GetCreditUsageUseCase) Execute(accountID string) (*bankaccount.CreditUsage, error) {
	account, err := uc.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}
	if !account.IsCreditCard() {
		return nil, ErrNotACreditCard
	}

	// The COMMITTED total, not what is owed now: the issuer reserves a future
	// instalment against today's limit, so that is what availability is made of.
	outstanding, err := uc.outstanding.For(account)
	if err != nil {
		return nil, err
	}

	usage := account.CreditUsageFor(outstanding)
	return &usage, nil
}
