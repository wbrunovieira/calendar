package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// A cross-profile transfer is two linked rows, one per profile. Cancelling it has to
// cancel both, or one profile keeps a movement whose counterpart no longer exists.
//
// It did not. Only the addressed row went through tx.Cancel(), so the linked row's
// struct still held its old status — and CancelStatus writes `SET status = $2` from
// txn.Status, the struct's own field. The write was therefore a silent no-op that
// reported success, leaving the audit columns NULL on that side too.
//
// The damage, measured: the surviving leg goes on counting. A linked leg that is a
// card charge stays on its bill, so the other profile's invoice still claims a debt
// whose counterpart was cancelled.
//
// An earlier version of this comment claimed the other profile's BALANCE was
// reversed while its row stayed live. That was asserted, not measured, and it is
// false: updateBalanceOnStatusChange only acts on a transition to or from CONFIRMED,
// and a CONFIRMED leg is refused by CanTransitionTo first. Every combination of the
// two legs' statuses was then run and no balance moved in any of them.
//
// Measured in production on 2026-10-18 before fixing: no pair was in this state yet,
// so the defect was latent and nothing needs repairing — the next cancelled aporte
// would have been the first.
func crossProfilePair(t *testing.T, status transaction.Status) (*fakeTransactionRepo, *fakeAccountRepo, *transaction.Transaction, *transaction.Transaction) {
	t.Helper()

	source := &transaction.Transaction{
		ID: "tx-source", ProfileID: testProfile, BankAccountID: testAccount,
		Type: transaction.TypeExpense, Status: status,
		Amount: 500, Currency: "BRL", Description: "Aporte WB", OccurredOn: time.Now(),
	}
	dest := &transaction.Transaction{
		ID: "tx-dest", ProfileID: "profile-2", BankAccountID: testDest,
		Type: transaction.TypeIncome, Status: status,
		Amount: 500, Currency: "BRL", Description: "Aporte WB", OccurredOn: time.Now(),
	}
	source.LinkedTransactionID = &dest.ID
	dest.LinkedTransactionID = &source.ID

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{source, dest}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 1000),
		testDest:    checkingAccount("profile-2", testDest, 1000),
	}}
	return txRepo, accountRepo, source, dest
}

// cancelPair takes the use case rather than building one. It used to construct its
// own, so a test that wired a recalculator onto its own instance had that instance
// silently ignored — the call went through an unwired twin and the test measured
// nothing. Passing it in makes that mistake impossible to write.
// crossProfilePairMixed seeds the two legs with DIFFERENT statuses. A uniform pair
// cannot distinguish where the linked leg's validation sits: a CONFIRMED addressed
// row is refused before anything is written either way, so both orderings look
// identical. Only a mixed pair — addressed acceptable, linked refusable — reaches the
// window where one row is already persisted and the other is then rejected.
func crossProfilePairMixed(t *testing.T, addressedStatus, linkedStatus transaction.Status) (*fakeTransactionRepo, *fakeAccountRepo) {
	t.Helper()

	source := &transaction.Transaction{
		ID: "tx-source", ProfileID: testProfile, BankAccountID: testAccount,
		Type: transaction.TypeExpense, Status: addressedStatus,
		Amount: 500, Currency: "BRL", Description: "Aporte WB", OccurredOn: time.Now(),
	}
	dest := &transaction.Transaction{
		ID: "tx-dest", ProfileID: "profile-2", BankAccountID: testDest,
		Type: transaction.TypeIncome, Status: linkedStatus,
		Amount: 500, Currency: "BRL", Description: "Aporte WB", OccurredOn: time.Now(),
	}
	source.LinkedTransactionID = &dest.ID
	dest.LinkedTransactionID = &source.ID

	return &fakeTransactionRepo{created: []*transaction.Transaction{source, dest}},
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
			testAccount: checkingAccount(testProfile, testAccount, 1000),
			testDest:    checkingAccount("profile-2", testDest, 1000),
		}}
}

func cancelPair(t *testing.T, uc *UpdateTransactionStatusUseCase, addressed string) {
	t.Helper()
	reason := "NUNCA_OCORREU"
	actor := "bruno"
	if _, err := uc.Execute(addressed, UpdateTransactionStatusInput{
		Status: "CANCELLED", ReasonCode: &reason, Actor: &actor,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func plainCancelUC(txRepo *fakeTransactionRepo, accountRepo *fakeAccountRepo) *UpdateTransactionStatusUseCase {
	return NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
}

func assertBothCancelled(t *testing.T, txRepo *fakeTransactionRepo) {
	t.Helper()

	rows := map[string]*transaction.Transaction{}
	for _, id := range []string{"tx-source", "tx-dest"} {
		row := rowFrom(t, txRepo, id)
		if row.Status != transaction.StatusCancelled {
			t.Fatalf("%s is %s: one profile keeps a movement whose counterpart is gone", id, row.Status)
		}
		// The audit columns matter as much as the status. A leg cancelled with no
		// motive and no actor is the unaudited way out the domain exists to refuse.
		if row.ReversalReason == nil || row.ReversedBy == nil || row.ReversedAt == nil {
			t.Fatalf("%s was cancelled without a motive or an actor: reason=%v by=%v at=%v",
				id, row.ReversalReason, row.ReversedBy, row.ReversedAt)
		}
		rows[id] = row
	}

	// IDENTICAL, not merely present. One act in two rows: if the halves carry
	// different actors or different instants, nothing downstream can tell that they
	// are the same cancellation, and reconciling the two profiles means guessing.
	a, b := rows["tx-source"], rows["tx-dest"]
	if *a.ReversalReason != *b.ReversalReason {
		t.Fatalf("the two legs record different motives: %q and %q", *a.ReversalReason, *b.ReversalReason)
	}
	if *a.ReversedBy != *b.ReversedBy {
		t.Fatalf("the two legs record different actors: %q and %q", *a.ReversedBy, *b.ReversedBy)
	}
	if !a.ReversedAt.Equal(*b.ReversedAt) {
		t.Fatalf("the two legs record different instants: %s and %s", a.ReversedAt, b.ReversedAt)
	}
	if (a.ReversalNote == nil) != (b.ReversalNote == nil) ||
		(a.ReversalNote != nil && *a.ReversalNote != *b.ReversalNote) {
		t.Fatalf("the two legs record different notes: %v and %v", a.ReversalNote, b.ReversalNote)
	}
}

func TestCancelCrossProfile_AddressingTheSourceCancelsBothLegs(t *testing.T) {
	txRepo, accountRepo, _, _ := crossProfilePair(t, transaction.StatusPlanned)
	cancelPair(t, plainCancelUC(txRepo, accountRepo), "tx-source")
	assertBothCancelled(t, txRepo)
}

// Both directions, because the no-op is in whichever leg is NOT addressed — and a
// test for one direction only would pass on a fix that handles one and not the other.
func TestCancelCrossProfile_AddressingTheDestinationCancelsBothLegs(t *testing.T) {
	txRepo, accountRepo, _, _ := crossProfilePair(t, transaction.StatusPlanned)
	cancelPair(t, plainCancelUC(txRepo, accountRepo), "tx-dest")
	assertBothCancelled(t, txRepo)
}

// A CONFIRMED row must be reversed, not cancelled — the domain refuses it. What this
// pins is that the refusal is ATOMIC: neither leg may have moved by the time it is
// reported.
//
// The pair is MIXED on purpose — addressed PLANNED, linked CONFIRMED — because that
// is the only shape that can fail. With both legs CONFIRMED the addressed row is
// refused before anything is written, so the test passes whether the linked leg is
// validated before or after the addressed row is persisted, and the ordering it
// claims to protect goes unmeasured. Measured with the validation block moved below
// persistStatus: the addressed row comes back CANCELLED and audited while the linked
// one stays CONFIRMED, and a uniform fixture sees none of it.
//
// An earlier version used a uniform pair and ended in a call that was dead code.
func TestCancelCrossProfile_ARefusedConfirmedPairLeavesNeitherLegMoved(t *testing.T) {
	txRepo, accountRepo := crossProfilePairMixed(t, transaction.StatusPlanned, transaction.StatusConfirmed)

	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	reason := "NUNCA_OCORREU"
	actor := "bruno"
	_, err := uc.Execute("tx-source", UpdateTransactionStatusInput{
		Status: "CANCELLED", ReasonCode: &reason, Actor: &actor,
	})
	if err == nil {
		t.Fatal("a confirmed leg must be reversed, not cancelled; the domain let the pair through")
	}

	if src := rowFrom(t, txRepo, "tx-source"); src.Status != transaction.StatusPlanned {
		t.Fatalf("the cancel was refused but the addressed leg is %s: a refusal that has already moved one side is worse than the refusal it reports", src.Status)
	}
	if src := rowFrom(t, txRepo, "tx-source"); src.ReversedAt != nil || src.ReversalReason != nil || src.ReversedBy != nil {
		t.Fatalf("the addressed leg was refused but carries audit columns: reason=%v by=%v", src.ReversalReason, src.ReversedBy)
	}
	if dst := rowFrom(t, txRepo, "tx-dest"); dst.Status != transaction.StatusConfirmed {
		t.Fatalf("the linked leg moved to %s despite the refusal", dst.Status)
	}
}

// A row can be CANCELLED with every audit column NULL: POST /transactions accepts
// status=CANCELLED and hands the cross-profile destination leg the same status
// without writing any reversal column, and the CHECK constraint only enforces the
// trail for REVERSED. A guard keyed off the STATUS skipped those legs too and left
// them unaudited forever — the pair audited on one side only that cancelIntent
// exists to prevent. The guard is keyed off the RECORD instead.
func TestCancelCrossProfile_AnUnauditedCancelledLegGetsItsRecord(t *testing.T) {
	txRepo, accountRepo := crossProfilePairMixed(t, transaction.StatusPlanned, transaction.StatusCancelled)

	cancelPair(t, plainCancelUC(txRepo, accountRepo), "tx-source")

	row := rowFrom(t, txRepo, "tx-dest")
	if row.ReversalReason == nil || row.ReversedBy == nil || row.ReversedAt == nil {
		t.Fatalf("the cancelled leg was left with no record at all: reason=%v by=%v at=%v",
			row.ReversalReason, row.ReversedBy, row.ReversedAt)
	}
	src := rowFrom(t, txRepo, "tx-source")
	if *row.ReversedBy != *src.ReversedBy || *row.ReversalReason != *src.ReversalReason {
		t.Fatalf("the backfilled record does not match the cancellation that wrote it: %q/%q vs %q/%q",
			*row.ReversalReason, *row.ReversedBy, *src.ReversalReason, *src.ReversedBy)
	}
}

// The motive travels to the linked leg, not just the reason code. Without this, a
// cancellation explained in one profile is unexplained in the other — and every
// existing test passes no Reason at all, so both legs end with a nil note and the
// helper's note check holds vacuously.
func TestCancelCrossProfile_TheWrittenMotiveReachesBothLegs(t *testing.T) {
	txRepo, accountRepo, _, _ := crossProfilePair(t, transaction.StatusPlanned)

	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	reason := "NUNCA_OCORREU"
	actor := "bruno"
	note := "aporte combinado por telefone e desfeito no mesmo dia"
	if _, err := uc.Execute("tx-source", UpdateTransactionStatusInput{
		Status: "CANCELLED", ReasonCode: &reason, Actor: &actor, Reason: &note,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, id := range []string{"tx-source", "tx-dest"} {
		row := rowFrom(t, txRepo, id)
		if row.ReversalNote == nil {
			t.Fatalf("%s carries no motive at all", id)
		}
		if *row.ReversalNote != note {
			t.Fatalf("%s records a different motive: %q", id, *row.ReversalNote)
		}
	}
}

// CanTransitionTo permits CANCELLED -> CANCELLED, and Cancel() overwrites
// reversed_at, reason and actor unconditionally while keeping the older note. So
// cancelling a pair whose other leg was ALREADY cancelled rewrote that leg's audit
// record into something that never happened: this cancellation's motive and actor
// welded to the first one's note.
//
// Caught by review as a regression introduced by the very fix above — before it, the
// no-op write at least left the older record intact.
func TestCancelCrossProfile_AnAlreadyCancelledLegKeepsItsOwnAuditRecord(t *testing.T) {
	txRepo, accountRepo, _, dest := crossProfilePair(t, transaction.StatusPlanned)

	firstAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	firstReason := transaction.ReasonDuplicated
	firstBy := "someone-else"
	firstNote := "lancado em duplicidade na importacao"
	dest.Status = transaction.StatusCancelled
	dest.ReversedAt = &firstAt
	dest.ReversalReason = &firstReason
	dest.ReversedBy = &firstBy
	dest.ReversalNote = &firstNote

	cancelPair(t, plainCancelUC(txRepo, accountRepo), "tx-source")

	row := rowFrom(t, txRepo, "tx-dest")
	if *row.ReversalReason != firstReason {
		t.Fatalf("the earlier motive was overwritten: %q", *row.ReversalReason)
	}
	if *row.ReversedBy != firstBy {
		t.Fatalf("the earlier actor was overwritten: %q", *row.ReversedBy)
	}
	if !row.ReversedAt.Equal(firstAt) {
		t.Fatalf("the earlier instant was overwritten: %s", row.ReversedAt)
	}
	if *row.ReversalNote != firstNote {
		t.Fatalf("the earlier note was overwritten: %q", *row.ReversalNote)
	}
}

// The consequence that is real, and that nothing covered: a linked leg which is a
// CARD charge stays on its bill when it fails to cancel. The other profile's invoice
// then claims a debt whose counterpart was cancelled — and the bill is what gets
// reconciled against the bank, so that is where it surfaces, months later.
//
// This is the measurable damage the fix prevents. It also shows the interaction with
// the invoice-total refresh is right: that refresh runs after both legs are
// persisted and reads the repository rather than the in-memory structs, so
// SumByInvoiceID's NOT IN ('CANCELLED','REVERSED') filter actually drops the line.
func TestCancelCrossProfile_TheLinkedLegLeavesItsCardBill(t *testing.T) {
	invoiceID := "inv-1"
	card := creditCardAccount("profile-2", testCC)

	source := &transaction.Transaction{
		ID: "tx-source", ProfileID: testProfile, BankAccountID: testAccount,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 500, Currency: "BRL", Description: "Aporte WB", OccurredOn: time.Now(),
	}
	linked := &transaction.Transaction{
		ID: "tx-dest", ProfileID: "profile-2", BankAccountID: testCC, InvoiceID: &invoiceID,
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 500, Currency: "BRL", Description: "Aporte WB", OccurredOn: time.Now(),
	}
	source.LinkedTransactionID = &linked.ID
	linked.LinkedTransactionID = &source.ID

	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{source, linked}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		invoiceID: {ID: invoiceID, BankAccountID: testCC, Amount: 500, Status: invoice.StatusOpen},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		testAccount: checkingAccount(testProfile, testAccount, 1000),
		testCC:      card,
	}}

	uc := NewUpdateTransactionStatusUseCase(txRepo, accountRepo, nil)
	uc.SetInvoiceRecalculator(NewRecalculateInvoiceAmountUseCase(invRepo, txRepo))

	cancelPair(t, uc, "tx-source")

	if row := rowFrom(t, txRepo, "tx-dest"); row.Status != transaction.StatusCancelled {
		t.Fatalf("the card leg is %s", row.Status)
	}

	bill, ok := invRepo.invoices[invoiceID]
	if !ok {
		t.Fatal("the bill vanished")
	}
	want, err := txRepo.SumByInvoiceID(invoiceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill.Amount != want {
		t.Fatalf("bill stale: stored %.2f, its live lines add up to %.2f", bill.Amount, want)
	}
	if bill.Amount != 0 {
		t.Fatalf("the other profile's bill still claims a cancelled debt: %.2f", bill.Amount)
	}
}
