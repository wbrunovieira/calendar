package usecases

import (
	"errors"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/balancecheckpoint"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type RecalculateBalanceUseCase struct {
	accountRepo     bankaccount.Repository
	transactionRepo transaction.Repository
	checkpointRepo  balancecheckpoint.Repository // nil → full recalc always
	adjustmentLog   BalanceAdjustmentLog
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
	// Drift is what the stored balance claimed beyond what its transactions justify.
	// Reported even when nothing is written, because the SIZE of a drift is the
	// finding — silently correcting it is what erased that finding before.
	Drift float64 `json:"drift"`
	// Skipped is true when the account's balance is not derived from its
	// transactions, so nothing was recalculated and nothing was written. A
	// caller that cannot tell this from "recalculated to the same number" will
	// read an unchanged balance as a confirmed one, which during a
	// reconciliation is the worst possible answer.
	Skipped bool `json:"skipped"`
	// Reason explains a skip in words, for whoever is reading the response.
	Reason string `json:"reason,omitempty"`
}

// BalanceAdjustmentLog records a correction so it can be reviewed later. A recalculation
// with no trail is the erasure of proof.
type BalanceAdjustmentLog interface {
	Record(accountID string, before, after float64, reason, by string) error
}

// SetAdjustmentLog wires the trail, following how the other collaborators attach.
func (uc *RecalculateBalanceUseCase) SetAdjustmentLog(l BalanceAdjustmentLog) {
	uc.adjustmentLog = l
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

	// READ ONLY. Correcting silently is what made this the biggest risk to trusting
	// the data: while it ran as a routine fix, nobody could tell whether the system
	// was right or the number had merely been stamped — and the correction destroyed
	// the evidence, so the size and the start of the drift went with it.
	return &RecalculateBalanceResult{
		OldBalance: oldBalance,
		NewBalance: newBalance,
		Drift:      round2(oldBalance - newBalance),
	}, nil
}

// Refresh keeps the stored balance in step after a write. This is maintenance, not a
// correction: the balance is derived, and a transaction just changed what it derives
// from.
//
// It still leaves a trail when the drift is bigger than the rounding noise, because at
// that point the recalculation is not keeping up — it is covering for something. That
// is exactly the case the manual endpoint used to erase: a confirmation bug had left
// R$ 1.522,75 of drift, and "fixing" it destroyed the evidence that the bug existed.
func (uc *RecalculateBalanceUseCase) Refresh(accountID string) (*RecalculateBalanceResult, error) {
	result, err := uc.Execute(accountID)
	if err != nil {
		return nil, err
	}
	if result.Skipped {
		return result, nil
	}

	account, err := uc.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}
	account.CurrentBalance = result.NewBalance
	account.UpdatedAt = time.Now()
	if err := uc.accountRepo.Update(account); err != nil {
		return nil, err
	}

	// A write moves the balance by what it wrote. Anything beyond that is drift the
	// recalculation is absorbing, and absorbing it silently is how a bug stays
	// invisible for months.
	if uc.adjustmentLog != nil && absFloat(result.Drift) > driftNoise {
		_ = uc.adjustmentLog.Record(accountID, result.OldBalance, result.NewBalance,
			"recalculo automatico absorveu deriva maior que o esperado", "system")
	}
	return result, nil
}

// Apply writes the derived balance and records why. The reason is required: a
// correction with no motive cannot be reviewed, and this endpoint's whole history is
// of being used to make a number look right.
//
// It does not invent a value — the balance still comes from the transactions. What it
// adds is that the correction leaves a mark instead of erasing one.
func (uc *RecalculateBalanceUseCase) Apply(accountID, reason, by string) (*RecalculateBalanceResult, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("applying a balance correction requires a reason")
	}
	if strings.TrimSpace(by) == "" {
		return nil, errors.New("applying a balance correction requires an actor")
	}

	result, err := uc.Execute(accountID)
	if err != nil {
		return nil, err
	}
	if result.Skipped || result.Drift == 0 {
		return result, nil
	}

	account, err := uc.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, ErrBankAccountNotFound
	}
	account.CurrentBalance = result.NewBalance
	account.UpdatedAt = time.Now()
	if err := uc.accountRepo.Update(account); err != nil {
		return nil, err
	}

	if uc.adjustmentLog != nil {
		if err := uc.adjustmentLog.Record(accountID, result.OldBalance, result.NewBalance, reason, by); err != nil {
			return nil, err
		}
	}
	return result, nil
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

// driftNoise is the rounding tolerance below which a difference is not worth a trail
// entry. Above it, the recalculation is absorbing something a write did not explain.
const driftNoise = 0.01

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
