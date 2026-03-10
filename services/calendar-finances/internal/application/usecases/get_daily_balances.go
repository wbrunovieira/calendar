package usecases

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
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
	accountRepo     bankaccount.Repository
}

func NewGetDailyBalancesUseCase(txRepo transaction.Repository, accountRepo bankaccount.Repository) *GetDailyBalancesUseCase {
	return &GetDailyBalancesUseCase{transactionRepo: txRepo, accountRepo: accountRepo}
}

func (uc *GetDailyBalancesUseCase) Execute(input GetDailyBalancesInput) ([]transaction.DailyBalance, error) {
	if strings.TrimSpace(input.ProfileID) == "" || strings.TrimSpace(input.BankAccountID) == "" {
		return nil, ErrInvalidInput
	}

	bankAccountID := strings.TrimSpace(input.BankAccountID)

	// Get current balance for the bank account (sum of confirmed transactions)
	txBalance, err := uc.transactionRepo.CalculateBalanceByBankAccountID(bankAccountID)
	if err != nil {
		return nil, err
	}

	// Add initial_balance from the bank account
	account, err := uc.accountRepo.FindByID(bankAccountID)
	if err != nil {
		return nil, err
	}
	var initialBalance float64
	if account != nil {
		initialBalance = account.InitialBalance
	}
	currentBalance := txBalance + initialBalance

	// List transactions for the bank account (including incoming transfers)
	filter := transaction.ListFilter{
		ProfileID:            input.ProfileID,
		BankAccountID:        &bankAccountID,
		IncludeAsDestination: true,
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

	return transaction.CalculateDailyBalances(txs, currentBalance, bankAccountID), nil
}
