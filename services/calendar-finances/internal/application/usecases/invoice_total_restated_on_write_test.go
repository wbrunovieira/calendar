package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A bill's stored total is a cache of its own lines, and three write paths can change
// those lines: creating a charge, confirming one, and deleting one. Only the delete
// refreshed the cache, so every new card purchase left the bill claiming the old
// number. Measured in production on 2026-10-18: four charges posted to the Mercado
// Pago card left invoice nov/2026 stored at -850.84 against -714.94 computed, a drift
// of exactly the 135.90 just posted. /health/invariants reported it; nothing fixed it.
//
// These tests wire the REAL recalculation rather than a spy wherever they can. A spy
// proves a call was made; only the real one proves the stored number ends up right,
// and the number is the whole point.

func cardInvoiceFixture(t *testing.T) (*fakeTransactionRepo, *fakeInvoiceRepo, *bankaccount.BankAccount, *CreateTransactionUseCase) {
	t.Helper()

	card := creditCardAccount(testProfile, testCC)
	accounts := map[string]*bankaccount.BankAccount{testCC: card}
	txRepo := &fakeTransactionRepo{}
	invRepo := &fakeInvoiceRepo{}

	uc := NewCreateTransactionUseCase(
		setupProfile(testProfile),
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{categories: map[string]*category.Category{
			testCat:    expenseCat(testProfile, testCat),
			testCatInc: incomeCat(testProfile, testCatInc),
		}},
		txRepo,
		invRepo,
		nil,
		nil,
	)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	return txRepo, invRepo, card, uc
}

func billFor(t *testing.T, invRepo *fakeInvoiceRepo, id string) *invoice.Invoice {
	t.Helper()
	inv, ok := invRepo.invoices[id]
	if !ok {
		t.Fatalf("invoice %s was never written", id)
	}
	return inv
}

// The charge that was just posted must be reflected in its bill's stored total.
func TestCreateTransaction_RestatesTheBillTotalOfANewCharge(t *testing.T) {
	txRepo, invRepo, _, uc := cardInvoiceFixture(t)

	status := "CONFIRMED"
	txn, err := uc.Execute(CreateTransactionInput{
		ProfileID:     testProfile,
		BankAccountID: testCC,
		CategoryID:    strPtr(testCat),
		Type:          "EXPENSE",
		Amount:        135.90,
		Currency:      "BRL",
		Description:   "Padaria Pamequita",
		OccurredOn:    time.Now().Format("2006-01-02"),
		Status:        &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.InvoiceID == nil {
		t.Fatal("the charge was not linked to a bill, so there is nothing to restate")
	}

	bill := billFor(t, invRepo, *txn.InvoiceID)
	want, err := txRepo.SumByInvoiceID(bill.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill.Amount != want {
		t.Fatalf("bill total stayed stale: stored %.2f, its own lines add up to %.2f", bill.Amount, want)
	}
	if bill.Amount != 135.90 {
		t.Fatalf("expected the bill to owe 135.90, got %.2f", bill.Amount)
	}
}

// A credit on a card reduces what is owed, and it is a line of the bill like any
// other. The restatement has to follow the sign rather than assume a charge.
func TestCreateTransaction_RestatesTheBillWhenTheLineIsACredit(t *testing.T) {
	_, invRepo, _, uc := cardInvoiceFixture(t)

	status := "CONFIRMED"
	charge, err := uc.Execute(CreateTransactionInput{
		ProfileID: testProfile, BankAccountID: testCC, CategoryID: strPtr(testCat),
		Type: "EXPENSE", Amount: 200, Currency: "BRL", Description: "Compra",
		OccurredOn: time.Now().Format("2006-01-02"), Status: &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: testProfile, BankAccountID: testCC, CategoryID: strPtr(testCatInc),
		Type: "INCOME", Amount: 50, Currency: "BRL", Description: "Estorno",
		OccurredOn: time.Now().Format("2006-01-02"), Status: &status,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bill := billFor(t, invRepo, *charge.InvoiceID)
	if bill.Amount != 150 {
		t.Fatalf("expected 200 owed less a 50 credit = 150, got %.2f", bill.Amount)
	}
}

// An instalment plan spreads one purchase across consecutive months, so it touches a
// DIFFERENT bill per instalment. Restating only the first one leaves every later bill
// stale — the same bug, just further away from the write that caused it.
func TestCreateTransaction_RestatesEveryBillAnInstalmentPlanTouches(t *testing.T) {
	txRepo, invRepo, _, uc := cardInvoiceFixture(t)

	status := "CONFIRMED"
	total := 3
	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: testProfile, BankAccountID: testCC, CategoryID: strPtr(testCat),
		Type: "EXPENSE", Amount: 300, Currency: "BRL", Description: "Parcelado",
		OccurredOn: time.Now().Format("2006-01-02"), Status: &status,
		InstallmentTotal: &total,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	touched := map[string]bool{}
	for _, tx := range txRepo.created {
		if tx.InvoiceID != nil {
			touched[*tx.InvoiceID] = true
		}
	}
	if len(touched) != 3 {
		t.Fatalf("expected the plan to land on 3 separate bills, got %d", len(touched))
	}

	for id := range touched {
		bill := billFor(t, invRepo, id)
		want, err := txRepo.SumByInvoiceID(id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bill.Amount != want {
			t.Fatalf("bill %s stayed stale: stored %.2f, its lines add up to %.2f", id, bill.Amount, want)
		}
		if bill.Amount != 100 {
			t.Fatalf("expected each bill to carry one 100.00 instalment, got %.2f", bill.Amount)
		}
	}
}

// A charge on a plain account belongs to no bill. Reaching for a recalculation there
// would be a write with no subject.
func TestCreateTransaction_ACheckingChargeRestatesNoBill(t *testing.T) {
	accounts := map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 500),
	}
	spy := &invoiceRecalcSpy{}
	uc := baseCreateUC(accounts, nil)
	uc.SetInvoiceRecalculator(spy)

	status := "CONFIRMED"
	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: testProfile, BankAccountID: testAccount, CategoryID: strPtr(testCat),
		Type: "EXPENSE", Amount: 10, Currency: "BRL", Description: "Pipoca",
		OccurredOn: time.Now().Format("2006-01-02"), Status: &status,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spy.called) != 0 {
		t.Fatalf("expected no bill to be restated, got %v", spy.called)
	}
}

// The charge is already written when the restatement runs. A refusal there must not
// report the creation as failed, or the caller retries and posts the charge twice.
func TestCreateTransaction_ARestatementRefusalDoesNotFailTheCharge(t *testing.T) {
	card := creditCardAccount(testProfile, testCC)
	accounts := map[string]*bankaccount.BankAccount{testCC: card}
	txRepo := &fakeTransactionRepo{}

	uc := NewCreateTransactionUseCase(
		setupProfile(testProfile),
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{categories: map[string]*category.Category{testCat: expenseCat(testProfile, testCat)}},
		txRepo,
		&fakeInvoiceRepo{},
		nil,
		nil,
	)
	uc.SetInvoiceRecalculator(&invoiceRecalcSpy{err: errors.New("bill unavailable")})

	status := "CONFIRMED"
	txn, err := uc.Execute(CreateTransactionInput{
		ProfileID: testProfile, BankAccountID: testCC, CategoryID: strPtr(testCat),
		Type: "EXPENSE", Amount: 42, Currency: "BRL", Description: "Compra",
		OccurredOn: time.Now().Format("2006-01-02"), Status: &status,
	})
	if err != nil {
		t.Fatalf("the charge was written; a failed restatement must not undo that: %v", err)
	}
	if txn == nil || len(txRepo.created) != 1 {
		t.Fatal("expected the charge to remain written")
	}
}

// A bill separates what is confirmed from what is only planned, so moving a charge
// between those two states changes the bill as surely as adding one does.
func TestUpdateTransactionStatus_RestatesTheBillWhenACardChargeIsConfirmed(t *testing.T) {
	invoiceID := "inv-1"
	card := creditCardAccount(testProfile, testCC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 90, Currency: "BRL", Description: "Assinatura", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testCC: card}}

	spy := &invoiceRecalcSpy{}
	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(spy)

	if _, err := uc.Execute("tx-1", UpdateTransactionStatusInput{Status: "CONFIRMED"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spy.called) != 1 || spy.called[0] != invoiceID {
		t.Fatalf("expected the charge's bill to be restated, got %v", spy.called)
	}
}

// Cancelling a planned charge takes it off the bill. The bill has to hear about it.
func TestUpdateTransactionStatus_RestatesTheBillWhenACardChargeIsCancelled(t *testing.T) {
	invoiceID := "inv-1"
	card := creditCardAccount(testProfile, testCC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 90, Currency: "BRL", Description: "Assinatura", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testCC: card}}

	spy := &invoiceRecalcSpy{}
	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(spy)

	reason := "NUNCA_OCORREU"
	actor := "test"
	if _, err := uc.Execute("tx-1", UpdateTransactionStatusInput{
		Status: "CANCELLED", ReasonCode: &reason, Actor: &actor,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spy.called) != 1 || spy.called[0] != invoiceID {
		t.Fatalf("expected the charge's bill to be restated, got %v", spy.called)
	}
}

// A status change on an account that has no bill must not reach for one.
func TestUpdateTransactionStatus_ACheckingChargeRestatesNoBill(t *testing.T) {
	account := checkingAccount(testProfile, testAccount, 500)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testAccount,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 90, Currency: "BRL", Description: "Conta de luz", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testAccount: account}}

	spy := &invoiceRecalcSpy{}
	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(spy)

	if _, err := uc.Execute("tx-1", UpdateTransactionStatusInput{Status: "CONFIRMED"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spy.called) != 0 {
		t.Fatalf("expected no bill to be restated, got %v", spy.called)
	}
}
