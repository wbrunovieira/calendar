package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/balancecheckpoint"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type RecalculateBalanceUseCase struct {
	accountRepo     bankaccount.Repository
	transactionRepo transaction.Repository
	checkpointRepo  balancecheckpoint.Repository // nil → full recalc always
}

func NewRecalculateBalanceUseCase(
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
	checkpointRepo balancecheckpoint.Repository,
) *RecalculateBalanceUseCase {
	return &RecalculateBalanceUseCase{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		checkpointRepo:  checkpointRepo,
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

	newBalance, err := uc.computeBalance(accountID, account.InitialBalance)
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

// computeBalance returns initial_balance + all confirmed transaction impact.
// When a checkpoint exists it only sums transactions since the checkpoint,
// keeping the query O(recent) regardless of total history.
func (uc *RecalculateBalanceUseCase) computeBalance(accountID string, initialBalance float64) (float64, error) {
	if uc.checkpointRepo != nil {
		cp, err := uc.checkpointRepo.FindLatestBefore(accountID, time.Now())
		if err == nil && cp != nil {
			// Only sum transactions that arrived after the checkpoint's month
			since := addOneMonth(cp.ReferenceMonth)
			delta, err := uc.transactionRepo.CalculateBalanceSince(accountID, since)
			if err != nil {
				return 0, err
			}
			return round2(cp.ClosingBalance + delta), nil
		}
	}

	// No checkpoint available — full recalculation from all confirmed transactions
	txBalance, err := uc.transactionRepo.CalculateBalanceByBankAccountID(accountID)
	if err != nil {
		return 0, err
	}
	return round2(initialBalance + txBalance), nil
}

