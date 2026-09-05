package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// deleteFixture builds a same-profile transfer that has already moved money: 300 left
// the checking account and landed in the investment account.
func deleteFixture() (*fakeTransactionRepo, *fakeAccountRepo, *transaction.Transaction) {
	source := &bankaccount.BankAccount{
		ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 1000, CurrentBalance: 700,
	}
	destination := &bankaccount.BankAccount{
		ID: "investment", ProfileID: "p1", Type: bankaccount.AccountTypeInvestment,
		InitialBalance: 0, CurrentBalance: 300,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"checking": source, "investment": destination,
	}}

	destID := "investment"
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "checking",
		DestinationAccountID: &destID,
		Type:                 transaction.TypeTransfer,
		Status:               transaction.StatusConfirmed,
		Amount:               300, Currency: "BRL",
		Description: "Aporte", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}

	return txRepo, accountRepo, txn
}

// Deleting a confirmed transfer must leave BOTH accounts as if it never happened.
//
// This is wired the way production is — with a real balance recalculator — because
// that is where the bug lived: the use case reversed the balances by hand and then
// recalculated from the database while the row was still there, so the recalculation
// put the old numbers straight back. The destination kept a credit for money that no
// longer moved, which is a phantom balance nobody notices until a reconciliation.
func TestDeleteTransaction_TransferReversesBothLegs(t *testing.T) {
	txRepo, accountRepo, txn := deleteFixture()
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, recalculator).Execute(txn.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := accountRepo.accounts["checking"].CurrentBalance; got != 1000 {
		t.Errorf("expected the source restored to 1000, got %.2f", got)
	}
	if got := accountRepo.accounts["investment"].CurrentBalance; got != 0 {
		t.Errorf("expected the destination credit removed (0), got %.2f — phantom balance", got)
	}
}

// The same ordering trap applies to an ordinary expense: reversing and then
// recalculating over a row that still exists is a no-op.
func TestDeleteTransaction_ExpenseRestoresTheBalance(t *testing.T) {
	account := &bankaccount.BankAccount{
		ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 1000, CurrentBalance: 700,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"checking": account}}
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "checking",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Mercado", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, recalculator).Execute(txn.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := accountRepo.accounts["checking"].CurrentBalance; got != 1000 {
		t.Errorf("expected 1000 after deleting the expense, got %.2f", got)
	}
}

// A planned transfer never moved money, so deleting it must not move any either.
func TestDeleteTransaction_PlannedTransferLeavesBalancesAlone(t *testing.T) {
	txRepo, accountRepo, txn := deleteFixture()
	txn.Status = transaction.StatusPlanned
	accountRepo.accounts["checking"].CurrentBalance = 1000
	accountRepo.accounts["investment"].CurrentBalance = 0
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, recalculator).Execute(txn.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := accountRepo.accounts["checking"].CurrentBalance; got != 1000 {
		t.Errorf("expected the source untouched at 1000, got %.2f", got)
	}
	if got := accountRepo.accounts["investment"].CurrentBalance; got != 0 {
		t.Errorf("expected the destination untouched at 0, got %.2f", got)
	}
}

// Money moved into a credit card reduces its debt; deleting that payment must put the
// debt back, on both sides.
func TestDeleteTransaction_TransferIntoACardRestoresTheDebt(t *testing.T) {
	checking := &bankaccount.BankAccount{
		ID: "checking", ProfileID: "p1", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 1000, CurrentBalance: 600,
	}
	card := &bankaccount.BankAccount{
		ID: "card", ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard,
		InitialBalance: 0, CurrentBalance: 0, // a 400 purchase paid off by the transfer below
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"checking": checking, "card": card,
	}}
	cardID := "card"
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "checking",
		DestinationAccountID: &cardID,
		Type:                 transaction.TypeTransfer, Status: transaction.StatusConfirmed,
		Amount: 400, Currency: "BRL", Description: "Pagamento fatura", OccurredOn: time.Now(),
	}
	purchase := &transaction.Transaction{
		ID: "tx-0", ProfileID: "p1", BankAccountID: "card",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 400, Currency: "BRL", Description: "Supermercado", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{purchase, txn}}
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, recalculator).Execute(txn.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := accountRepo.accounts["checking"].CurrentBalance; got != 1000 {
		t.Errorf("expected the funding account restored to 1000, got %.2f", got)
	}
	if got := accountRepo.accounts["card"].CurrentBalance; got != -400 {
		t.Errorf("expected the card debt back at -400, got %.2f", got)
	}
}
