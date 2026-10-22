package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A transaction created without a cost centre could never be given one: the create
// path accepts costCenterId and validates it, while the update path only ever had
// the legacy free-text costCenter, which links to nothing.
//
// The cost of that gap is not cosmetic. On 2026-10-22, four test charges for the
// Treino Aliado project had to be REVERSED and re-created just to attach a cost
// centre — four DUPLICADO reversals in a ledger, standing for no economic fact at
// all. In a ledger a reversal means something did not happen or was recorded wrong;
// spending it on a metadata correction makes the history lie about itself.

func ccUpdateFixture(t *testing.T, seedCenters map[string]*costcenter.CostCenter) (*UpdateTransactionUseCase, *fakeTransactionRepo) {
	t.Helper()

	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: testProfile, BankAccountID: testAccount,
		CategoryID: strPtr(testCat),
		Type:       transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 100, Currency: "BRL", Description: "Compra", OccurredOn: day,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 1000),
	}}
	uc := NewUpdateTransactionUseCase(
		accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{testCat: expenseCat(testProfile, testCat)}},
		txRepo,
		&fakeInvoiceRepo{},
		nil,
	)
	if seedCenters != nil {
		uc.SetCostCenterRepository(&fakeCostCenterRepo{centers: seedCenters})
	}
	return uc, txRepo
}

func ccUpdateInput(costCenterID *string) UpdateTransactionInput {
	status := "CONFIRMED"
	return UpdateTransactionInput{
		BankAccountID: testAccount, CategoryID: strPtr(testCat), Type: "EXPENSE",
		Status: &status, Amount: 100, Currency: "BRL", Description: "Compra",
		OccurredOn: "2026-10-20", CostCenterID: costCenterID,
	}
}

func TestUpdateTransaction_AttachesACostCentre(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, map[string]*costcenter.CostCenter{
		"cc-1": {ID: "cc-1", ProfileID: testProfile, Name: "SaaS Treino", Type: costcenter.TypeProject},
	})

	id := "cc-1"
	if _, err := uc.Execute("tx-1", ccUpdateInput(&id)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := rowFrom(t, txRepo, "tx-1")
	if stored.CostCenterID == nil {
		t.Fatal("the cost centre was not attached — the row still has none")
	}
	if *stored.CostCenterID != "cc-1" {
		t.Fatalf("attached the wrong cost centre: %q", *stored.CostCenterID)
	}
}

// A cost centre belonging to another profile would file one company's spending under
// another's project. The create path refuses it; so must this one.
func TestUpdateTransaction_RefusesACostCentreFromAnotherProfile(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, map[string]*costcenter.CostCenter{
		"cc-other": {ID: "cc-other", ProfileID: "profile-2", Name: "Outra empresa", Type: costcenter.TypeProject},
	})

	id := "cc-other"
	_, err := uc.Execute("tx-1", ccUpdateInput(&id))
	if !errors.Is(err, ErrCostCenterNotFound) {
		t.Fatalf("expected ErrCostCenterNotFound, got %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("the refusal still attached it: %q", *stored.CostCenterID)
	}
}

func TestUpdateTransaction_RefusesACostCentreThatDoesNotExist(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, map[string]*costcenter.CostCenter{})

	id := "cc-ghost"
	if _, err := uc.Execute("tx-1", ccUpdateInput(&id)); !errors.Is(err, ErrCostCenterNotFound) {
		t.Fatalf("expected ErrCostCenterNotFound, got %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("the refusal still attached it: %q", *stored.CostCenterID)
	}
}

// A read that fails is not a missing cost centre. Collapsing the two answers a
// database outage with 400 and tells the caller their input was wrong when it was
// not — the same distinction the create path already draws.
func TestUpdateTransaction_AFailedReadIsNotAMissingCostCentre(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, nil)
	uc.SetCostCenterRepository(&fakeCostCenterRepo{readErr: errors.New("db down")})

	id := "cc-1"
	_, err := uc.Execute("tx-1", ccUpdateInput(&id))
	if err == nil {
		t.Fatal("expected the read failure to surface")
	}
	if errors.Is(err, ErrCostCenterNotFound) {
		t.Fatalf("a database outage was reported as a missing cost centre: %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("the failure still attached it: %q", *stored.CostCenterID)
	}
}

// The repository is attached after construction, which means it can be forgotten.
// An unwired validator that silently accepts whatever it is given is the shape of
// defect this repository keeps producing, so this one FAILS CLOSED: with no
// repository there is no way to check the id, and an unchecked id is refused rather
// than written.
func TestUpdateTransaction_RefusesACostCentreWhenTheRepositoryIsUnwired(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, nil) // no SetCostCenterRepository

	id := "cc-1"
	_, err := uc.Execute("tx-1", ccUpdateInput(&id))
	if err == nil {
		t.Fatal("an unwired validator accepted an unchecked cost centre")
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("it was written despite being unverifiable: %q", *stored.CostCenterID)
	}
}

// Sending no cost centre must still work when the repository is unwired: the whole
// point of failing closed is to refuse the UNCHECKABLE, not to break every edit.
func TestUpdateTransaction_AnEditWithoutACostCentreWorksUnwired(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, nil)

	if _, err := uc.Execute("tx-1", ccUpdateInput(nil)); err != nil {
		t.Fatalf("an ordinary edit was broken by the cost centre guard: %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("expected no cost centre, got %q", *stored.CostCenterID)
	}
}

// THE REGRESSION THIS FIELD NEARLY SHIPPED.
//
// Every other field on this full-replace route is cleared by omission, and the
// first version of this change followed that rule. It should not: the clients echo
// the other fields back, and none of them knows this one exists. The dashboard's
// edit form loads costCenter off the transaction and sends it again; costCenterId
// appears nowhere in the frontend — not in the form, not even in the read type.
//
// So clearing on omission would mean a receivable filed under a client by the CRM
// contract sync comes unfiled the moment someone fixes a typo in its description.
// 200 OK, nothing on screen, and the loss only surfaces when a per-client revenue
// figure comes up short months later.
//
// Measured on origin/main before changing this: that version never touched the
// field at all, so it PRESERVED it. Clearing would have been a behaviour change for
// every existing caller, not a continuation.
func TestUpdateTransaction_OmittingTheCostCentreKeepsIt(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, map[string]*costcenter.CostCenter{
		"cc-1": {ID: "cc-1", ProfileID: testProfile, Name: "SaaS Treino", Type: costcenter.TypeProject},
	})

	id := "cc-1"
	if _, err := uc.Execute("tx-1", ccUpdateInput(&id)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID == nil {
		t.Fatal("setup failed: the cost centre was never attached")
	}

	// An ordinary edit from a client that does not know the field exists.
	if _, err := uc.Execute("tx-1", ccUpdateInput(nil)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := rowFrom(t, txRepo, "tx-1")
	if stored.CostCenterID == nil {
		t.Fatal("an edit that never mentioned the cost centre unfiled the row")
	}
	if *stored.CostCenterID != "cc-1" {
		t.Fatalf("the cost centre changed to %q", *stored.CostCenterID)
	}
}

// Clearing still has to be possible, just deliberate. The empty string says it,
// because a *string cannot tell "absent" from "null" once JSON is decoded.
func TestUpdateTransaction_AnEmptyCostCentreClearsIt(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, map[string]*costcenter.CostCenter{
		"cc-1": {ID: "cc-1", ProfileID: testProfile, Name: "SaaS Treino", Type: costcenter.TypeProject},
	})

	id := "cc-1"
	if _, err := uc.Execute("tx-1", ccUpdateInput(&id)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	empty := ""
	if _, err := uc.Execute("tx-1", ccUpdateInput(&empty)); err != nil {
		t.Fatalf("clearing was refused: %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("the empty string did not clear it: %q", *stored.CostCenterID)
	}
}

// An empty string names no cost centre, so it must not be looked up — otherwise
// clearing is impossible whenever the repository is unwired or the lookup fails.
func TestUpdateTransaction_ClearingNeedsNoRepository(t *testing.T) {
	uc, txRepo := ccUpdateFixture(t, nil)

	empty := ""
	if _, err := uc.Execute("tx-1", ccUpdateInput(&empty)); err != nil {
		t.Fatalf("clearing required a repository it should not need: %v", err)
	}
	if stored := rowFrom(t, txRepo, "tx-1"); stored.CostCenterID != nil {
		t.Fatalf("expected none, got %q", *stored.CostCenterID)
	}
}
