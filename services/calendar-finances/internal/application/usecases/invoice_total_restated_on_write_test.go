package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
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

// rowFrom reads the row back out of the repository. The fakes hand GetByID a COPY —
// the fix that stopped an in-memory mutation from masquerading as a write — so the
// pointer a test seeded is stale the moment a use case touches it, and asserting on
// it asserts on nothing.
func rowFrom(t *testing.T, repo *fakeTransactionRepo, id string) *transaction.Transaction {
	t.Helper()
	for _, tx := range repo.created {
		if tx.ID == id {
			return tx
		}
	}
	t.Fatalf("row %s is not in the repository", id)
	return nil
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

// ---------------------------------------------------------------------------
// The branch production actually runs, and the paths that return an error
// ---------------------------------------------------------------------------

// passthroughUnitOfWork runs the work with the same repositories, committing
// nothing. app.go wires a real unit of work onto the create use case, so the
// instalment loop ALWAYS takes the atomically != nil branch in production. A test
// that leaves it unwired exercises the fallback and proves nothing about the code
// that ships.
type passthroughUnitOfWork struct {
	repos TxRepos
	calls int
}

func (u *passthroughUnitOfWork) Do(fn func(TxRepos) error) error {
	u.calls++
	return fn(u.repos)
}

func TestCreateTransaction_RestatesEveryBillWhenInstalmentsRunInsideAUnitOfWork(t *testing.T) {
	txRepo, invRepo, _, uc := cardInvoiceFixture(t)

	uow := &passthroughUnitOfWork{repos: TxRepos{Transactions: txRepo, Invoices: invRepo}}
	uc.SetUnitOfWork(uow)

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

	if uow.calls != 1 {
		t.Fatalf("expected the series to go through the unit of work, it ran %d times", uow.calls)
	}

	touched := map[string]bool{}
	for _, tx := range txRepo.created {
		if tx.InvoiceID != nil {
			touched[*tx.InvoiceID] = true
		}
	}
	if len(touched) != 3 {
		t.Fatalf("expected 3 bills, got %d", len(touched))
	}
	for id := range touched {
		bill := billFor(t, invRepo, id)
		if bill.Amount != 100 {
			t.Fatalf("bill %s stayed stale inside the unit of work: stored %.2f, expected 100.00", id, bill.Amount)
		}
	}
}

// The charge is committed the moment the row is written. Every step after that can
// still fail, and when it does the caller is told the write failed — while the
// charge sits on a bill whose total was never refreshed. That is this bug reached
// through the error path instead of the happy one.
//
// A CROSS-PROFILE transfer is what gets a charge onto a bill here: its source leg is
// stored as an EXPENSE, and only EXPENSE and INCOME are lines of a bill. A
// same-profile transfer stays a TRANSFER and is linked to nothing, which is why an
// earlier version of this test skipped itself and proved nothing.
func TestCreateTransaction_RestatesTheBillEvenWhenALaterStepFails(t *testing.T) {
	const otherProfile = "profile-2"

	card := creditCardAccount(testProfile, testCC)
	dest := checkingAccount(otherProfile, testDest, 0)
	txRepo := &fakeTransactionRepo{}
	invRepo := &fakeInvoiceRepo{}

	// The card is never written (a card's balance does not move when it is used),
	// so the only Update on this path is the destination's — and that is the step
	// made to fail.
	accountRepo := &fakeAccountRepo{
		accounts:  map[string]*bankaccount.BankAccount{testCC: card, testDest: dest},
		updateErr: errors.New("db down"),
	}

	uc := NewCreateTransactionUseCase(
		&fakeProfileRepo{profiles: map[string]*profile.Profile{
			testProfile:  {ID: testProfile, Name: testProfile},
			otherProfile: {ID: otherProfile, Name: otherProfile},
		}},
		accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{
			testCat:    expenseCat(testProfile, testCat),
			testCatInc: incomeCat(otherProfile, testCatInc),
		}},
		txRepo,
		invRepo,
		nil,
		nil,
	)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	status := "CONFIRMED"
	destID := testDest
	destCat := testCatInc
	_, err := uc.Execute(CreateTransactionInput{
		ProfileID: testProfile, BankAccountID: testCC, CategoryID: strPtr(testCat),
		DestinationAccountID: &destID, DestinationCategoryID: &destCat,
		Type: "TRANSFER", Amount: 500, Currency: "BRL", Description: "Pagamento",
		OccurredOn: time.Now().Format("2006-01-02"), Status: &status,
	})
	if err == nil {
		t.Fatal("this test needs the step after the write to fail; it did not")
	}

	var written *transaction.Transaction
	for _, tx := range txRepo.created {
		if tx.BankAccountID == testCC && tx.InvoiceID != nil {
			written = tx
		}
	}
	if written == nil {
		t.Fatal("expected the card charge to be committed with a bill attached")
	}

	bill := billFor(t, invRepo, *written.InvoiceID)
	want, serr := txRepo.SumByInvoiceID(bill.ID)
	if serr != nil {
		t.Fatalf("unexpected error: %v", serr)
	}
	if bill.Amount != want {
		t.Fatalf("the charge was committed and the call failed, leaving the bill stale: stored %.2f, its lines add up to %.2f", bill.Amount, want)
	}
	if bill.Amount != 500 {
		t.Fatalf("expected the committed charge to be on the bill at 500.00, got %.2f", bill.Amount)
	}
}

// Cancelling a planned card charge takes it OUT of the sum, so the stored total has
// to come down. This is the status transition that actually moves the number, and it
// is asserted on the number rather than on a spy.
func TestUpdateTransactionStatus_ACancelledCardChargeLeavesItsBill(t *testing.T) {
	invoiceID := "inv-1"
	card := creditCardAccount(testProfile, testCC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 500, Currency: "BRL", Description: "Compra", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {ID: invoiceID, BankAccountID: testCC, Amount: 500, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testCC: card}}

	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	reason := "NUNCA_OCORREU"
	actor := "test"
	if _, err := uc.Execute("tx-1", UpdateTransactionStatusInput{
		Status: "CANCELLED", ReasonCode: &reason, Actor: &actor,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bill := billFor(t, invRepo, invoiceID)
	if bill.Amount != 0 {
		t.Fatalf("the cancelled charge is still on the bill: stored %.2f, expected 0.00", bill.Amount)
	}
}

// Confirming does NOT move the total — SumByInvoiceID already counts PLANNED. This
// pins that fact: if the filter ever changes so that planned rows stop counting, the
// refresh on this path becomes load-bearing and this test says so.
func TestUpdateTransactionStatus_ConfirmingLeavesTheBillTotalWhereItWas(t *testing.T) {
	invoiceID := "inv-1"
	card := creditCardAccount(testProfile, testCC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 90, Currency: "BRL", Description: "Assinatura", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {ID: invoiceID, BankAccountID: testCC, Amount: 90, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testCC: card}}

	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	if _, err := uc.Execute("tx-1", UpdateTransactionStatusInput{Status: "CONFIRMED"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bill := billFor(t, invRepo, invoiceID); bill.Amount != 90 {
		t.Fatalf("expected the total to stay at 90.00, got %.2f", bill.Amount)
	}
}

// ---------------------------------------------------------------------------
// Editing a charge
// ---------------------------------------------------------------------------

// Editing the amount of a card charge changes what its bill's lines add up to. This
// use case restated only the bill a row PAYS, never the bill a row is a LINE OF, so
// correcting 500.00 to 200.00 left the bill still claiming 500.00.
//
// The date is seeded WITHOUT a time of day. An edit re-files the charge whenever the
// date differs, and "2006-01-02" cannot carry a clock, so seeding time.Now() here
// would silently turn this into the move-between-cycles case below.
func TestUpdateTransaction_RestatesTheBillOfAnEditedCharge(t *testing.T) {
	invoiceID := "inv-1"
	day := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	card := creditCardAccount(testProfile, testCC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		CategoryID: strPtr(testCat),
		Type:       transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 500, Currency: "BRL", Description: "Compra", OccurredOn: day,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {ID: invoiceID, BankAccountID: testCC, Amount: 500, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testCC: card}}

	uc := NewUpdateTransactionUseCase(
		accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{testCat: expenseCat(testProfile, testCat)}},
		txRepo,
		invRepo,
		nil,
	)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	status := "CONFIRMED"
	if _, err := uc.Execute("tx-1", UpdateTransactionInput{
		BankAccountID: testCC, CategoryID: strPtr(testCat), Type: "EXPENSE",
		Status: &status, Amount: 200, Currency: "BRL", Description: "Compra",
		OccurredOn: day.Format("2006-01-02"),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := rowFrom(t, txRepo, "tx-1")
	if stored.InvoiceID == nil || *stored.InvoiceID != invoiceID {
		t.Fatalf("the charge was re-filed; this test is about an amount edit on the same bill")
	}
	if stored.Amount != 200 {
		t.Fatalf("the edit did not reach the repository: amount is %.2f", stored.Amount)
	}

	bill := billFor(t, invRepo, invoiceID)
	want, err := txRepo.SumByInvoiceID(invoiceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill.Amount != want {
		t.Fatalf("bill stayed stale after the edit: stored %.2f, its lines add up to %.2f", bill.Amount, want)
	}
	if bill.Amount != 200 {
		t.Fatalf("expected the bill to follow the edit down to 200.00, got %.2f", bill.Amount)
	}
}

// Moving a charge to another cycle makes TWO bills wrong: the one it left still
// counts it, and the one it joined does not yet. Restating only one of them trades
// one stale bill for another.
func TestUpdateTransaction_RestatesBothBillsWhenAChargeChangesCycle(t *testing.T) {
	oldInvoiceID := "inv-old"
	card := creditCardAccount(testProfile, testCC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &oldInvoiceID,
		CategoryID: strPtr(testCat),
		Type:       transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 500, Currency: "BRL", Description: "Compra",
		OccurredOn: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		oldInvoiceID: {ID: oldInvoiceID, BankAccountID: testCC, Amount: 500, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{testCC: card}}

	uc := NewUpdateTransactionUseCase(
		accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{testCat: expenseCat(testProfile, testCat)}},
		txRepo,
		invRepo,
		nil,
	)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	status := "CONFIRMED"
	if _, err := uc.Execute("tx-1", UpdateTransactionInput{
		BankAccountID: testCC, CategoryID: strPtr(testCat), Type: "EXPENSE",
		Status: &status, Amount: 500, Currency: "BRL", Description: "Compra",
		OccurredOn: "2026-12-13",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := rowFrom(t, txRepo, "tx-1")
	if stored.InvoiceID == nil || *stored.InvoiceID == oldInvoiceID {
		t.Fatalf("expected the charge to be re-filed onto another bill, it is on %v", stored.InvoiceID)
	}

	if left := billFor(t, invRepo, oldInvoiceID); left.Amount != 0 {
		t.Fatalf("the bill it left still counts the charge: stored %.2f, expected 0.00", left.Amount)
	}
	if joined := billFor(t, invRepo, *stored.InvoiceID); joined.Amount != 500 {
		t.Fatalf("the bill it joined does not count the charge: stored %.2f, expected 500.00", joined.Amount)
	}
}

// A cross-profile pair is two rows written one after the other. The first row's
// status is already persisted when the second write runs, so a failure there returns
// an error over a change that DID happen — and the bill behind the first row is
// stale. Measured before the fix: the bill stayed stored at 500.00 while its own
// lines added up to 0.00, this PR's own defect reached through the path it did not
// cover. The refresh is deferred from the first write for exactly this reason.
func TestUpdateTransactionStatus_RestatesTheBillEvenWhenTheLinkedLegFails(t *testing.T) {
	invoiceID := "inv-1"
	card := creditCardAccount(testProfile, testCC)
	other := checkingAccount("profile-2", testDest, 0)

	linkedID := "tx-2"
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		LinkedTransactionID: &linkedID,
		Type:                transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 500, Currency: "BRL", Description: "Compra", OccurredOn: time.Now(),
	}
	linked := &transaction.Transaction{
		ID: linkedID, ProfileID: "profile-2", BankAccountID: testDest,
		LinkedTransactionID: &txn.ID,
		Type:                transaction.TypeIncome, Status: transaction.StatusPlanned,
		Amount: 500, Currency: "BRL", Description: "Compra", OccurredOn: time.Now(),
	}

	txRepo := &fakeTransactionRepo{
		created:      []*transaction.Transaction{txn, linked},
		updateErrFor: map[string]error{linkedID: errors.New("db down")},
	}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {ID: invoiceID, BankAccountID: testCC, Amount: 500, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		testCC: card, testDest: other,
	}}

	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	reason := "NUNCA_OCORREU"
	actor := "test"
	_, err := uc.Execute("tx-1", UpdateTransactionStatusInput{
		Status: "CANCELLED", ReasonCode: &reason, Actor: &actor,
	})
	if err == nil {
		t.Fatal("this test needs the linked leg's write to fail; it did not")
	}

	stored := rowFrom(t, txRepo, "tx-1")
	if stored.Status != transaction.StatusCancelled {
		t.Fatalf("the first leg was supposed to be cancelled already; it is %s", stored.Status)
	}

	bill := billFor(t, invRepo, invoiceID)
	want, serr := txRepo.SumByInvoiceID(invoiceID)
	if serr != nil {
		t.Fatalf("unexpected error: %v", serr)
	}
	if bill.Amount != want {
		t.Fatalf("the cancellation landed and the call failed, leaving the bill stale: stored %.2f, its lines add up to %.2f", bill.Amount, want)
	}
}

// Moving a charge OFF a card onto a plain account takes it off the bill, and the
// balance write that follows can fail. The edit is already persisted by then, so the
// caller is told nothing happened while the bill it left still counts a charge that
// is no longer on it.
//
// This is the one failure reachable after the write here: while the row stays a card
// line nothing downstream can fail, because a card's balance does not move. The
// refresh is deferred from the write rather than placed at the return for this case.
func TestUpdateTransaction_RestatesTheBillLeftBehindEvenWhenALaterStepFails(t *testing.T) {
	invoiceID := "inv-1"
	day := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	card := creditCardAccount(testProfile, testCC)
	checking := checkingAccount(testProfile, testAccount, 1000)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testCC, InvoiceID: &invoiceID,
		CategoryID: strPtr(testCat),
		Type:       transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 500, Currency: "BRL", Description: "Compra", OccurredOn: day,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {ID: invoiceID, BankAccountID: testCC, Amount: 500, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{
		accounts:  map[string]*bankaccount.BankAccount{testCC: card, testAccount: checking},
		updateErr: errors.New("db down"),
	}

	uc := NewUpdateTransactionUseCase(
		accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{testCat: expenseCat(testProfile, testCat)}},
		txRepo,
		invRepo,
		nil,
	)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	status := "CONFIRMED"
	_, err := uc.Execute("tx-1", UpdateTransactionInput{
		BankAccountID: testAccount, CategoryID: strPtr(testCat), Type: "EXPENSE",
		Status: &status, Amount: 500, Currency: "BRL", Description: "Compra",
		OccurredOn: day.Format("2006-01-02"),
	})
	if err == nil {
		t.Fatal("this test needs the balance write after the edit to fail; it did not")
	}

	// Asserted, not skipped past. A test that quietly excuses itself when its
	// premise stops holding reports success for code it never ran.
	stored := rowFrom(t, txRepo, "tx-1")
	if stored.BankAccountID != testAccount {
		t.Fatalf("the move was supposed to be written before the failure; the row is still on %s", stored.BankAccountID)
	}
	if stored.InvoiceID != nil {
		t.Fatalf("a charge on a plain account belongs to no bill, it still points at %s", *stored.InvoiceID)
	}

	bill := billFor(t, invRepo, invoiceID)
	if bill.Amount != 0 {
		t.Fatalf("the charge left and the call failed, leaving the bill stale: stored %.2f, expected 0.00", bill.Amount)
	}
}
