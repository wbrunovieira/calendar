package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type GetDailyBalancesInput struct {
	ProfileID     string
	BankAccountID string
	OccurredFrom  *string
	OccurredTo    *string
}

type GetDailyBalancesUseCase struct {
	transactionRepo transaction.Repository
}

func NewGetDailyBalancesUseCase(txRepo transaction.Repository) *GetDailyBalancesUseCase {
	return &GetDailyBalancesUseCase{transactionRepo: txRepo}
}

func (uc *GetDailyBalancesUseCase) Execute(input GetDailyBalancesInput) ([]transaction.DailyBalance, error) {
	if strings.TrimSpace(input.ProfileID) == "" || strings.TrimSpace(input.BankAccountID) == "" {
		return nil, ErrInvalidInput
	}

	bankAccountID := strings.TrimSpace(input.BankAccountID)

	// Get current balance for the bank account
	currentBalance, err := uc.transactionRepo.CalculateBalanceByBankAccountID(bankAccountID)
	if err != nil {
		return nil, err
	}

	// List transactions for the bank account
	filter := transaction.ListFilter{
		ProfileID:     input.ProfileID,
		BankAccountID: &bankAccountID,
	}

	if input.OccurredFrom != nil {
		t, err := parseDate(*input.OccurredFrom)
		if err != nil {
			return nil, err
		}
		filter.OccurredFrom = &t
	}

	if input.OccurredTo != nil {
		t, err := parseDate(*input.OccurredTo)
		if err != nil {
			return nil, err
		}
		filter.OccurredTo = &t
	}

	txs, err := uc.transactionRepo.List(filter)
	if err != nil {
		return nil, err
	}

	return transaction.CalculateDailyBalances(txs, currentBalance), nil
}
