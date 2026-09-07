package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// DELETE used to remove the row from the database. During the reconciliation of
// 06/09/2026 that destroyed a legitimate entry — a real R$ 55,58 Cloudflare charge
// mistaken for a phantom because the reconciliation matched by amount and the amount
// had been recorded wrong. Nothing in the system can say what was deleted, by whom,
// or when.
//
// A ledger does not delete. It reverses: the row stays, stops counting towards the
// balance, and carries the reason it was reversed.

func reverseFixture(t *testing.T) (*DeleteTransactionUseCase, *fakeTransactionRepo, *fakeAccountRepo, string) {
	t.Helper()
	accountID := "acc-1"
	accRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {ID: accountID, ProfileID: "p1", Type: bankaccount.AccountTypeChecking, CurrentBalance: 1000, Currency: "BRL"},
	}}
	txRepo := &fakeTransactionRepo{}
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: accountID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 55.58, Currency: "BRL", Description: "Cloudflare - dominio",
		OccurredOn: time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC),
	}
	txRepo.created = append(txRepo.created, txn)
	return NewDeleteTransactionUseCase(txRepo, accRepo, nil), txRepo, accRepo, "tx-1"
}

func TestDelete_KeepsTheRowAndMarksItReversed(t *testing.T) {
	uc, txRepo, _, id := reverseFixture(t)

	if err := uc.Execute(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.created) != 1 {
		t.Fatalf("the row must survive: have %d transactions, want 1", len(txRepo.created))
	}
	got := txRepo.created[0]
	if got.Status != transaction.StatusReversed {
		t.Errorf("status = %s, want %s", got.Status, transaction.StatusReversed)
	}
	if got.Amount != 55.58 || got.Description != "Cloudflare - dominio" {
		t.Error("the reversed row must keep its original content, so what was undone stays auditable")
	}
	if got.ReversedAt == nil {
		t.Error("reversedAt must record WHEN it was reversed")
	}
}

func TestDelete_ReversedTransactionLeavesTheBalance(t *testing.T) {
	// The balance is derived from CONFIRMED rows, so reversing must move the balance
	// exactly as deleting did — without losing the row.
	uc, _, accRepo, id := reverseFixture(t)

	if err := uc.Execute(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1000 + 55.58 (an expense was undone)
	if got, want := accRepo.accounts["acc-1"].CurrentBalance, 1055.58; got < want-0.005 || got > want+0.005 {
		t.Errorf("balance = %.2f, want %.2f", got, want)
	}
}

func TestDelete_ReversingTwiceIsRejected(t *testing.T) {
	uc, _, accRepo, id := reverseFixture(t)

	if err := uc.Execute(id); err != nil {
		t.Fatalf("first reversal: %v", err)
	}
	before := accRepo.accounts["acc-1"].CurrentBalance

	if err := uc.Execute(id); err == nil {
		t.Error("reversing an already reversed transaction must be rejected")
	}
	if accRepo.accounts["acc-1"].CurrentBalance != before {
		t.Error("a rejected reversal must not move the balance a second time")
	}
}

func TestDelete_ReversesBothLegsOfALinkedTransfer(t *testing.T) {
	// A cross-profile transfer is two linked rows. Reversing one without the other
	// leaves money that arrived from nowhere.
	accRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-a": {ID: "acc-a", ProfileID: "p1", Type: bankaccount.AccountTypeChecking, CurrentBalance: 500, Currency: "BRL"},
		"acc-b": {ID: "acc-b", ProfileID: "p2", Type: bankaccount.AccountTypeChecking, CurrentBalance: 300, Currency: "BRL"},
	}}
	txRepo := &fakeTransactionRepo{}
	other := "tx-b"
	a := &transaction.Transaction{ID: "tx-a", ProfileID: "p1", BankAccountID: "acc-a",
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed, Amount: 100, Currency: "BRL",
		Description: "Aporte", OccurredOn: time.Now(), LinkedTransactionID: &other}
	first := "tx-a"
	b := &transaction.Transaction{ID: "tx-b", ProfileID: "p2", BankAccountID: "acc-b",
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed, Amount: 100, Currency: "BRL",
		Description: "Aporte", OccurredOn: time.Now(), LinkedTransactionID: &first}
	txRepo.created = append(txRepo.created, a, b)

	uc := NewDeleteTransactionUseCase(txRepo, accRepo, nil)
	if err := uc.Execute("tx-a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.created) != 2 {
		t.Fatalf("both rows must survive: have %d, want 2", len(txRepo.created))
	}
	for _, tx := range txRepo.created {
		if tx.Status != transaction.StatusReversed {
			t.Errorf("%s status = %s, want both legs reversed", tx.ID, tx.Status)
		}
	}
}
