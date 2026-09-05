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
	// Skipped is true when the account's balance is not derived from its
	// transactions, so nothing was recalculated and nothing was written. A
	// caller that cannot tell this from "recalculated to the same number" will
	// read an unchanged balance as a confirmed one, which during a
	// reconciliation is the worst possible answer.
	Skipped bool `json:"skipped"`
	// Reason explains a skip in words, for whoever is reading the response.
	Reason string `json:"reason,omitempty"`
}

func (uc *RecalculateBalanceUseCase) Execute(accountID string) (*RecalculateBalanceResult, error) {
	account, err := uc.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}

	oldBalance := account.CurrentBalance

	// A position priced by quotas holds market value, not a ledger: the stock
	// and crypto pollers own that number. Summing its transactions instead
	// would replace the position's worth with what was paid for it, and the
	// next poll would write the price back — an invisible oscillation.
	if account.HasQuotas() {
		return &RecalculateBalanceResult{
			OldBalance: oldBalance,
			NewBalance: oldBalance,
			Skipped:    true,
			Reason:     "balance is market value (quotas x quote), maintained by the price sync; nothing to recalculate from transactions",
		}, nil
	}

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

// computeBalance returns initial_balance + net impact of ALL confirmed transactions.
// Always does a full sum — no checkpoint shortcut — so retroactive transactions
// (added after a checkpoint was created) are never missed.
func (uc *RecalculateBalanceUseCase) computeBalance(accountID string, initialBalance float64) (float64, error) {
	txBalance, err := uc.transactionRepo.CalculateBalanceByBankAccountID(accountID)
	if err != nil {
		return 0, err
	}
	return round2(initialBalance + txBalance), nil
}
