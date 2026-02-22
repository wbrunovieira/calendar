package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type RecalculateBalanceUseCase struct {
	accountRepo     bankaccount.Repository
	transactionRepo transaction.Repository
}

func NewRecalculateBalanceUseCase(
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
) *RecalculateBalanceUseCase {
	return &RecalculateBalanceUseCase{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

type RecalculateBalanceResult struct {
	OldBalance float64 `json:"oldBalance"`
	NewBalance float64 `json:"newBalance"`
}

func (uc *RecalculateBalanceUseCase) Execute(accountID string) (*RecalculateBalanceResult, error) {
	account, err := uc.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	oldBalance := account.CurrentBalance

	newBalance, err := uc.transactionRepo.CalculateBalanceByBankAccountID(accountID)
	if err != nil {
		return nil, err
	}

	account.CurrentBalance = newBalance
	account.UpdatedAt = time.Now()

	if err := uc.accountRepo.Update(account); err != nil {
		return nil, err
	}

	return &RecalculateBalanceResult{
		OldBalance: oldBalance,
		NewBalance: newBalance,
	}, nil
}
