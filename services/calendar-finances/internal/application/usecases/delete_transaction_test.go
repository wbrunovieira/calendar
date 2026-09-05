package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
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
	// Paying an invoice is the one event that moves a card's balance, so undoing the
	// payment must move it back: the R$400 purchase is owed again. This is the mirror
	// of TestDeleteTransaction_CardPurchaseLeavesTheCardBalanceAlone, where the card
	// is the SOURCE and nothing should move.
	if got := accountRepo.accounts["card"].CurrentBalance; got != -400 {
		t.Errorf("expected the card debt back at -400, got %.2f", got)
	}
}

// The cross-profile pair: two linked rows on two profiles. Both legs must be removed
// and both accounts recomputed — the capability this change claims and did not cover.
func TestDeleteTransaction_CrossProfilePairRemovesBothLegs(t *testing.T) {
	source := &bankaccount.BankAccount{
		ID: "personal", ProfileID: "p1", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 1000, CurrentBalance: 700,
	}
	destination := &bankaccount.BankAccount{
		ID: "company", ProfileID: "p2", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 0, CurrentBalance: 300,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"personal": source, "company": destination,
	}}

	destID, linkedID := "company", "tx-linked"
	src := &transaction.Transaction{
		ID: "tx-src", ProfileID: "p1", BankAccountID: "personal",
		DestinationAccountID: &destID, LinkedTransactionID: &linkedID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Aporte", OccurredOn: time.Now(),
	}
	linked := &transaction.Transaction{
		ID: linkedID, ProfileID: "p2", BankAccountID: "company",
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Aporte recebido", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{src, linked}}
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, recalculator).Execute(src.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.created) != 0 {
		t.Errorf("expected both legs removed, %d left", len(txRepo.created))
	}
	if got := accountRepo.accounts["personal"].CurrentBalance; got != 1000 {
		t.Errorf("expected the source profile restored to 1000, got %.2f", got)
	}
	if got := accountRepo.accounts["company"].CurrentBalance; got != 0 {
		t.Errorf("expected the other profile's credit removed, got %.2f", got)
	}
}

// Deleting a purchase must not move the card's balance — nothing else in the system
// moves it as purchases are created either.
func TestDeleteTransaction_CardPurchaseLeavesTheCardBalanceAlone(t *testing.T) {
	card := &bankaccount.BankAccount{
		ID: "card", ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard,
		InitialBalance: 0, CurrentBalance: 0,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card}}
	txns := []*transaction.Transaction{
		{ID: "a", ProfileID: "p1", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 100, Currency: "BRL",
			Description: "Compra 1", OccurredOn: time.Now()},
		{ID: "b", ProfileID: "p1", BankAccountID: "card", Type: transaction.TypeExpense,
			Status: transaction.StatusConfirmed, Amount: 100, Currency: "BRL",
			Description: "Compra 2", OccurredOn: time.Now()},
	}
	txRepo := &fakeTransactionRepo{created: txns}
	recalculator := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, recalculator).Execute("a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := accountRepo.accounts["card"].CurrentBalance; got != 0 {
		t.Fatalf("expected the card balance untouched at 0, got %.2f — it would now jump by every purchase since the last bill", got)
	}
}

// Half-deleting a linked pair is worse than not deleting it: the caller is told it
// worked while one leg survives.
func TestDeleteTransaction_FailingLinkedDeleteSurfacesTheError(t *testing.T) {
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{}}
	linkedID := "missing-leg"
	src := &transaction.Transaction{
		ID: "tx-src", ProfileID: "p1", BankAccountID: "personal",
		LinkedTransactionID: &linkedID,
		Type:                transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Aporte", OccurredOn: time.Now(),
	}
	linked := &transaction.Transaction{
		ID: linkedID, ProfileID: "p2", BankAccountID: "company",
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Aporte recebido", OccurredOn: time.Now(),
	}
	txRepo := &failingDeleteRepo{fakeTransactionRepo: fakeTransactionRepo{
		created: []*transaction.Transaction{src, linked},
	}, failFor: linkedID}

	err := NewDeleteTransactionUseCase(txRepo, accountRepo, nil).Execute(src.ID)
	if err == nil {
		t.Fatal("expected the failure to surface instead of a silent half-delete")
	}
}

type failingDeleteRepo struct {
	fakeTransactionRepo
	failFor string
}

func (r *failingDeleteRepo) DeleteMany(ids []string) error {
	for _, id := range ids {
		if id == r.failFor {
			return errors.New("boom")
		}
	}
	return r.fakeTransactionRepo.DeleteMany(ids)
}

// Deleting a linked pair must be all-or-nothing. Half-deleting it leaves the other
// profile holding a credit with no ledger row behind it, and the caller is told
// something failed — so nobody goes looking.
func TestDeleteTransaction_LinkedPairIsRemovedAtomically(t *testing.T) {
	source := &bankaccount.BankAccount{
		ID: "personal", ProfileID: "p1", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 1000, CurrentBalance: 700,
	}
	destination := &bankaccount.BankAccount{
		ID: "company", ProfileID: "p2", Type: bankaccount.AccountTypeChecking,
		InitialBalance: 0, CurrentBalance: 300,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"personal": source, "company": destination,
	}}

	destID, linkedID := "company", "tx-linked"
	src := &transaction.Transaction{
		ID: "tx-src", ProfileID: "p1", BankAccountID: "personal",
		DestinationAccountID: &destID, LinkedTransactionID: &linkedID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Aporte", OccurredOn: time.Now(),
	}
	linked := &transaction.Transaction{
		ID: linkedID, ProfileID: "p2", BankAccountID: "company",
		Type: transaction.TypeIncome, Status: transaction.StatusConfirmed,
		Amount: 300, Currency: "BRL", Description: "Aporte recebido", OccurredOn: time.Now(),
	}
	txRepo := &atomicDeleteSpy{fakeTransactionRepo: fakeTransactionRepo{
		created: []*transaction.Transaction{src, linked},
	}}

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, nil).Execute(src.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txRepo.deletedTogether) != 1 {
		t.Fatalf("expected a single atomic delete, got %d calls", len(txRepo.deletedTogether))
	}
	if len(txRepo.deletedTogether[0]) != 2 {
		t.Fatalf("expected both legs in one unit of work, got %v", txRepo.deletedTogether[0])
	}
}

// When the deletion fails, nothing must have been removed and the balances must not
// have been touched.
func TestDeleteTransaction_FailedDeleteLeavesEverythingAlone(t *testing.T) {
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
	txRepo := &atomicDeleteSpy{
		fakeTransactionRepo: fakeTransactionRepo{created: []*transaction.Transaction{txn}},
		fail:                true,
	}

	if err := NewDeleteTransactionUseCase(txRepo, accountRepo, nil).Execute(txn.ID); err == nil {
		t.Fatal("expected the failure to surface")
	}
	if len(txRepo.created) != 1 {
		t.Error("expected the transaction to survive a failed delete")
	}
	if account.CurrentBalance != 700 {
		t.Errorf("expected the balance untouched at 700, got %.2f", account.CurrentBalance)
	}
}

// A recalculation that fails must not be swallowed: the row is already gone, so a
// silent failure leaves the balance permanently stale behind a 204.
func TestDeleteTransaction_RecalculationFailureSurfaces(t *testing.T) {
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
	txRepo := &atomicDeleteSpy{fakeTransactionRepo: fakeTransactionRepo{
		created: []*transaction.Transaction{txn},
	}}

	err := NewDeleteTransactionUseCase(txRepo, accountRepo, failingRecalculator{}).Execute(txn.ID)
	if err == nil {
		t.Fatal("expected a failed recalculation to surface, not a silent stale balance")
	}
}

type atomicDeleteSpy struct {
	fakeTransactionRepo
	deletedTogether [][]string
	fail            bool
}

func (r *atomicDeleteSpy) DeleteMany(ids []string) error {
	if r.fail {
		return errors.New("boom")
	}
	r.deletedTogether = append(r.deletedTogether, ids)
	for _, id := range ids {
		if err := r.fakeTransactionRepo.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

type failingRecalculator struct{}

func (failingRecalculator) Execute(string) (*RecalculateBalanceResult, error) {
	return nil, errors.New("recalculation unavailable")
}

type invoiceRecalcSpy struct {
	called []string
	err    error
}

func (s *invoiceRecalcSpy) Execute(invoiceID string) (*invoice.Invoice, error) {
	s.called = append(s.called, invoiceID)
	return nil, s.err
}

// A card purchase belongs to a bill. Deleting it without recomputing that bill leaves
// the invoice total claiming money that is no longer owed — and the card's balance and
// its invoice then disagree, which is exactly the drift a reconciliation has to chase.
func TestDeleteTransaction_RecomputesTheInvoiceOfADeletedCharge(t *testing.T) {
	invoiceID := "inv-1"
	card := &bankaccount.BankAccount{
		ID: "card", ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card}}
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "card", InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 200, Currency: "BRL", Description: "Supermercado", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}

	spy := &invoiceRecalcSpy{}
	uc := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(spy)

	if err := uc.Execute(txn.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spy.called) != 1 || spy.called[0] != invoiceID {
		t.Fatalf("expected the charge's invoice to be recomputed, got %v", spy.called)
	}
}

// A paid invoice is closed history: the recalculation refuses it, and that refusal
// must not fail the deletion.
func TestDeleteTransaction_APaidInvoiceRefusalDoesNotFailTheDelete(t *testing.T) {
	invoiceID := "inv-paid"
	card := &bankaccount.BankAccount{
		ID: "card", ProfileID: "p1", Type: bankaccount.AccountTypeCreditCard,
	}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{"card": card}}
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "card", InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 200, Currency: "BRL", Description: "Supermercado", OccurredOn: time.Now(),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}

	uc := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(&invoiceRecalcSpy{err: ErrInvoiceAlreadyPaid})

	if err := uc.Execute(txn.ID); err != nil {
		t.Fatalf("a paid invoice must not block the deletion, got %v", err)
	}
	if len(txRepo.created) != 0 {
		t.Fatal("expected the transaction to be deleted anyway")
	}
}

// A transaction that belongs to no invoice must not trigger a recalculation.
func TestDeleteTransaction_NoInvoiceNoRecalculation(t *testing.T) {
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

	spy := &invoiceRecalcSpy{}
	uc := NewDeleteTransactionUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(spy)

	if err := uc.Execute(txn.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.called) != 0 {
		t.Fatalf("expected no invoice recomputation, got %v", spy.called)
	}
}
