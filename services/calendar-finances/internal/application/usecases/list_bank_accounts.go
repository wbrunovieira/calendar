package usecases

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

type ListBankAccountsUseCase struct {
	repo bankaccount.Repository
	// outstanding derives a credit card's balance from its bills on every read.
	// A constructor argument rather than a setter: a nil one would serve the
	// stored figure, which is the stale number this replaces — and it would do it
	// silently, which is worse than not having the feature.
	outstanding *CardOutstanding
}

func NewListBankAccountsUseCase(repo bankaccount.Repository, outstanding *CardOutstanding) *ListBankAccountsUseCase {
	return &ListBankAccountsUseCase{repo: repo, outstanding: outstanding}
}

func (uc *ListBankAccountsUseCase) Execute(profileID string) ([]*bankaccount.BankAccount, error) {
	accounts, err := uc.repo.FindByProfileID(profileID)
	if profileID == "" {
		accounts, err = uc.repo.FindAll()
	}
	if err != nil {
		return nil, err
	}
	if err := uc.outstanding.Apply(accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}
