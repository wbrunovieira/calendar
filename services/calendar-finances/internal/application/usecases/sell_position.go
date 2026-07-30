package usecases

import (
	"fmt"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// SellPositionInput describes the sale of shares/quotas from an investment
// position (FII, stock, fund). The proceeds land in the position's linked cash
// account (e.g. the broker's cash balance); moving that cash on to a bank is a
// separate transfer.
type SellPositionInput struct {
	Quantity   float64 `json:"quantity"`
	UnitPrice  float64 `json:"unitPrice"`
	OccurredOn string  `json:"occurredOn"`
}

// SellPositionResult is what the sale produced: the updated position account and
// the transaction that credited the cash account with the proceeds.
type SellPositionResult struct {
	Account     *bankaccount.BankAccount `json:"account"`
	Transaction *transaction.Transaction `json:"transaction"`
}

// SellPositionUseCase records the sale of a quota-based investment position.
//
// It mirrors how buys are stored (a transfer between the cash account and the
// position account), inverted: the position's shares go down and the linked
// cash account is credited with the proceeds. It does NOT compute realized
// profit or tax — the model stores no cost basis (that is a follow-up).
type SellPositionUseCase struct {
	accountRepo     bankaccount.Repository
	transactionRepo transaction.Repository
}

func NewSellPositionUseCase(
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
) *SellPositionUseCase {
	return &SellPositionUseCase{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

func (uc *SellPositionUseCase) Execute(accountID string, input SellPositionInput) (*SellPositionResult, error) {
	if input.Quantity <= 0 || input.UnitPrice <= 0 {
		return nil, ErrInvalidInput
	}

	occurredOn, err := parseDate(input.OccurredOn)
	if err != nil {
		return nil, ErrInvalidInput
	}

	position, err := uc.accountRepo.FindByID(accountID)
	if err != nil || position == nil {
		return nil, ErrBankAccountNotFound
	}

	if !position.HasQuotas() {
		return nil, ErrPositionHasNoQuotas
	}
	if input.Quantity > *position.NumberOfQuotas+1e-9 {
		return nil, ErrCannotSellMoreThanHeld
	}
	if !position.IsLinked() {
		return nil, ErrAccountNotLinked
	}

	cashAccount, err := uc.accountRepo.FindByID(*position.LinkedAccountID)
	if err != nil || cashAccount == nil {
		return nil, ErrAccountNotLinked
	}

	proceeds, err := position.SellQuotas(input.Quantity, input.UnitPrice)
	if err != nil {
		return nil, err
	}

	txn, err := transaction.New(transaction.CreateParams{
		ProfileID:            position.ProfileID,
		BankAccountID:        position.ID,
		DestinationAccountID: &cashAccount.ID,
		Type:                 transaction.TypeTransfer,
		Amount:               proceeds,
		Currency:             position.Currency,
		Description:          formatSellDescription(position.Name, input.Quantity, input.UnitPrice),
		OccurredOn:           occurredOn,
		Tags:                 []string{"venda", position.Name},
	})
	if err != nil {
		return nil, err
	}
	txn.Status = transaction.StatusConfirmed

	// Credit the linked cash account with the proceeds. Its balance is
	// transaction-driven, so this keeps it consistent with the recorded transfer.
	cashAccount.CurrentBalance += txn.Amount
	cashAccount.UpdatedAt = time.Now()

	if err := uc.transactionRepo.Create(txn); err != nil {
		return nil, err
	}
	if err := uc.accountRepo.Update(position); err != nil {
		return nil, err
	}
	if err := uc.accountRepo.Update(cashAccount); err != nil {
		return nil, err
	}

	return &SellPositionResult{Account: position, Transaction: txn}, nil
}

// formatSellDescription mirrors the buy descriptions, e.g.
// "Venda SNAG11 - 120 cotas @ R$9.84".
func formatSellDescription(name string, quantity, unitPrice float64) string {
	return fmt.Sprintf("Venda %s - %s cotas @ R$%.2f", name, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", quantity), "0"), "."), unitPrice)
}
