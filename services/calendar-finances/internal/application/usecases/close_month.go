package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/balancecheckpoint"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/google/uuid"
)

type CloseMonthInput struct {
	ReferenceMonth time.Time // any day in the target month; normalized to the 1st
}

type AccountCheckpointResult struct {
	AccountID      string  `json:"accountId"`
	AccountName    string  `json:"accountName"`
	ClosingBalance float64 `json:"closingBalance"`
}

type CloseMonthResult struct {
	ReferenceMonth     time.Time                 `json:"referenceMonth"`
	CheckpointsCreated int                       `json:"checkpointsCreated"`
	Accounts           []AccountCheckpointResult `json:"accounts"`
}

type CloseMonthUseCase struct {
	accountRepo     bankaccount.Repository
	transactionRepo transaction.Repository
	checkpointRepo  balancecheckpoint.Repository
}

func NewCloseMonthUseCase(
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
	checkpointRepo balancecheckpoint.Repository,
) *CloseMonthUseCase {
	return &CloseMonthUseCase{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		checkpointRepo:  checkpointRepo,
	}
}

func (uc *CloseMonthUseCase) Execute(input CloseMonthInput) (*CloseMonthResult, error) {
	refMonth := firstDayOfMonth(input.ReferenceMonth)
	endOfMonth := lastInstantOfMonth(refMonth)

	accounts, err := uc.accountRepo.FindAll()
	if err != nil {
		return nil, err
	}

	result := &CloseMonthResult{ReferenceMonth: refMonth}

	for _, account := range accounts {
		if account.Type == bankaccount.AccountTypeCreditCard || !account.IsActive {
			continue
		}

		txBalance, err := uc.transactionRepo.CalculateBalanceUpTo(account.ID, endOfMonth)
		if err != nil {
			continue
		}

		closingBalance := round2(account.InitialBalance + txBalance)

		cp := &balancecheckpoint.BalanceCheckpoint{
			ID:             uuid.New().String(),
			AccountID:      account.ID,
			ReferenceMonth: refMonth,
			ClosingBalance: closingBalance,
			CreatedAt:      time.Now(),
		}

		if err := uc.checkpointRepo.Save(cp); err != nil {
			continue
		}

		result.CheckpointsCreated++
		result.Accounts = append(result.Accounts, AccountCheckpointResult{
			AccountID:      account.ID,
			AccountName:    account.Name,
			ClosingBalance: closingBalance,
		})
	}

	return result, nil
}

// firstDayOfMonth returns the first day of the month containing t, at midnight UTC.
func firstDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// lastInstantOfMonth returns the last nanosecond of the month containing t.
func lastInstantOfMonth(firstDay time.Time) time.Time {
	return firstDayOfMonth(firstDay.AddDate(0, 1, 0)).Add(-time.Nanosecond)
}

// addOneMonth returns the first day of the month following firstDay.
func addOneMonth(firstDay time.Time) time.Time {
	return firstDayOfMonth(firstDay.AddDate(0, 1, 0))
}
