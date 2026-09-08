package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func creditCardAccountWith(profileID, id string, closingDay, dueDay int) *bankaccount.BankAccount {
	limit := 5000.0
	return &bankaccount.BankAccount{
		ID:          id,
		ProfileID:   profileID,
		Name:        id,
		Type:        bankaccount.AccountTypeCreditCard,
		CreditLimit: &limit,
		ClosingDay:  &closingDay,
		DueDay:      &dueDay,
		Currency:    "BRL",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func expenseOnCard(profileID, id, cardID string, occurredOn time.Time) *transaction.Transaction {
	return &transaction.Transaction{
		ID:            id,
		ProfileID:     profileID,
		BankAccountID: cardID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        100,
		Currency:      "BRL",
		Description:   "test expense",
		OccurredOn:    occurredOn,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

type noopBalanceRecalculator struct{}

func (n *noopBalanceRecalculator) Refresh(accountID string) (*RecalculateBalanceResult, error) {
	return &RecalculateBalanceResult{}, nil
}

// ---------------------------------------------------------------------------
// Bug: moving expense to CC when no invoice exists yet → must create invoice
// ---------------------------------------------------------------------------

// TestUpdateTransaction_ChangingCardCreatesInvoiceWhenNoneExists reproduces the
// Loovi bug: transaction was on Nubank card, edited to MP card, but the MP
// invoice for that period hadn't been created yet. Expected: invoice is created
// and assigned; InvoiceID must not be nil.
func TestUpdateTransaction_ChangingCardCreatesInvoiceWhenNoneExists(t *testing.T) {
	profileID := "profile-1"
	nubankCardID := "nubank-card"
	mpCardID := "mp-card"
	txID := "tx-loovi"

	// MP card: closes on day 9, due day 14
	mpCard := creditCardAccountWith(profileID, mpCardID, 9, 14)
	nubankCard := creditCardAccountWith(profileID, nubankCardID, 27, 3)

	occurredOn := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)

	// Transaction is currently on Nubank card with a Nubank invoice
	nubankInvID := "nubank-inv-april"
	tx := expenseOnCard(profileID, txID, nubankCardID, occurredOn)
	tx.InvoiceID = &nubankInvID

	// Only Nubank invoice exists — no MP invoice yet
	nubankInv := &invoice.Invoice{
		ID:            nubankInvID,
		BankAccountID: nubankCardID,
		OpeningDate:   time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Status:        invoice.StatusOpen,
	}

	accounts := map[string]*bankaccount.BankAccount{
		nubankCardID: nubankCard,
		mpCardID:     mpCard,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{tx}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{nubankInvID: nubankInv}}

	uc := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{},
		txRepo,
		invRepo,
		&noopBalanceRecalculator{},
	)

	_, err := uc.Execute(txID, UpdateTransactionInput{
		BankAccountID: mpCardID,
		Type:          "EXPENSE",
		Amount:        225.75,
		Currency:      "BRL",
		Description:   "Loovi Seguro Carro",
		OccurredOn:    "2026-04-13",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := txRepo.created[0]
	if updated.InvoiceID == nil {
		t.Fatal("expected InvoiceID to be set after moving to MP card, got nil")
	}
	if updated.BankAccountID != mpCardID {
		t.Errorf("expected bankAccountId to be %s, got %s", mpCardID, updated.BankAccountID)
	}

	// Invoice for MP card must have been created
	var mpInvoiceFound bool
	for _, inv := range invRepo.invoices {
		if inv.BankAccountID == mpCardID {
			mpInvoiceFound = true
			if !inv.ContainsDate(occurredOn) {
				t.Errorf("created MP invoice does not contain transaction date %v", occurredOn)
			}
		}
	}
	if !mpInvoiceFound {
		t.Error("expected a new invoice to be created for MP card, but none was found")
	}
}

// TestUpdateTransaction_ChangingCardUsesExistingInvoice verifies that when an
// invoice already exists for the target card, it is used and no duplicate is created.
func TestUpdateTransaction_ChangingCardUsesExistingInvoice(t *testing.T) {
	profileID := "profile-1"
	nubankCardID := "nubank-card"
	mpCardID := "mp-card"
	txID := "tx-1"

	mpCard := creditCardAccountWith(profileID, mpCardID, 9, 14)
	nubankCard := creditCardAccountWith(profileID, nubankCardID, 27, 3)

	occurredOn := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)

	nubankInvID := "nubank-inv"
	tx := expenseOnCard(profileID, txID, nubankCardID, occurredOn)
	tx.InvoiceID = &nubankInvID

	mpInvID := "mp-inv-april"
	nubankInv := &invoice.Invoice{
		ID:            nubankInvID,
		BankAccountID: nubankCardID,
		OpeningDate:   time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
		ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Status:        invoice.StatusOpen,
	}
	mpInv := &invoice.Invoice{
		ID:            mpInvID,
		BankAccountID: mpCardID,
		OpeningDate:   time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
		ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Status:        invoice.StatusOpen,
	}

	accounts := map[string]*bankaccount.BankAccount{
		nubankCardID: nubankCard,
		mpCardID:     mpCard,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{tx}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{
		nubankInvID: nubankInv,
		mpInvID:     mpInv,
	}}

	uc := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{},
		txRepo,
		invRepo,
		&noopBalanceRecalculator{},
	)

	_, err := uc.Execute(txID, UpdateTransactionInput{
		BankAccountID: mpCardID,
		Type:          "EXPENSE",
		Amount:        59.98,
		Currency:      "BRL",
		Description:   "iFood",
		OccurredOn:    "2026-04-20",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := txRepo.created[0]
	if updated.InvoiceID == nil {
		t.Fatal("expected InvoiceID to be set")
	}
	if *updated.InvoiceID != mpInvID {
		t.Errorf("expected invoice %s, got %s", mpInvID, *updated.InvoiceID)
	}

	// No new invoice should have been created
	var mpCount int
	for _, inv := range invRepo.invoices {
		if inv.BankAccountID == mpCardID {
			mpCount++
		}
	}
	if mpCount != 1 {
		t.Errorf("expected exactly 1 MP invoice (existing), got %d", mpCount)
	}
}

// TestUpdateTransaction_MovingToCheckingClearsInvoice verifies that moving a
// transaction from a credit card to a checking account removes the invoice link.
func TestUpdateTransaction_MovingToCheckingClearsInvoice(t *testing.T) {
	profileID := "profile-1"
	cardID := "cc-1"
	checkingID := "checking-1"
	txID := "tx-1"
	invID := "inv-1"

	card := creditCardAccountWith(profileID, cardID, 9, 14)
	checking := checkingAccount(profileID, checkingID, 1000)

	occurredOn := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tx := expenseOnCard(profileID, txID, cardID, occurredOn)
	tx.InvoiceID = &invID

	inv := &invoice.Invoice{
		ID:            invID,
		BankAccountID: cardID,
		OpeningDate:   time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
		ReferenceDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Status:        invoice.StatusOpen,
	}

	accounts := map[string]*bankaccount.BankAccount{
		cardID:     card,
		checkingID: checking,
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{tx}}
	invRepo := &fakeInvoiceRepo{invoices: map[string]*invoice.Invoice{invID: inv}}

	uc := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: accounts},
		&fakeCategoryRepo{},
		txRepo,
		invRepo,
		&noopBalanceRecalculator{},
	)

	_, err := uc.Execute(txID, UpdateTransactionInput{
		BankAccountID: checkingID,
		Type:          "EXPENSE",
		Amount:        100,
		Currency:      "BRL",
		Description:   "test",
		OccurredOn:    "2026-04-10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := txRepo.created[0]
	if updated.InvoiceID != nil {
		t.Errorf("expected InvoiceID to be nil after moving to checking, got %s", *updated.InvoiceID)
	}
}

// Ending a transaction's life must say why and by whom, at every door.
//
// The requirement lived on the status route, and the full-replace PUT walked around
// it: pausing a recurrence cancelled its planned entry through here with
// reversal_reason and reversed_by left NULL, and the database CHECK never noticed
// because it only covers REVERSED. A control enforced at one door and open at the
// other is not a control.
func TestUpdateTransaction_CannotCancelThroughTheFullReplaceRoute(t *testing.T) {
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "acc-1",
		Type: transaction.TypeExpense, Status: transaction.StatusPlanned,
		Amount: 25, Currency: "BRL", Description: "Contabo - VPS",
		OccurredOn: time.Date(2026, time.September, 9, 0, 0, 0, 0, time.UTC),
	}
	repo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-1": {ID: "acc-1", ProfileID: "p1", Name: "Conta", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
	}}

	cancelled := "CANCELLED"
	_, err := NewUpdateTransactionUseCase(accounts, nil, repo, nil, nil).Execute("tx-1", UpdateTransactionInput{
		BankAccountID: "acc-1", Type: "EXPENSE", Status: &cancelled,
		Amount: 25, Currency: "BRL", Description: "Contabo - VPS", OccurredOn: "2026-09-09",
	})
	if err == nil {
		t.Fatal("cancelling through the editing route must be refused")
	}
	if repo.created[0].Status != transaction.StatusPlanned {
		t.Errorf("the transaction must be untouched, got %s", repo.created[0].Status)
	}
}

// Editing an entry that is already cancelled must still work: the guard is about
// CHANGING the status, not about the status a row happens to have.
func TestUpdateTransaction_EditingAnAlreadyCancelledEntryStillWorks(t *testing.T) {
	txn := &transaction.Transaction{
		ID: "tx-1", ProfileID: "p1", BankAccountID: "acc-1",
		Type: transaction.TypeExpense, Status: transaction.StatusCancelled,
		Amount: 25, Currency: "BRL", Description: "errado",
		OccurredOn: time.Date(2026, time.September, 9, 0, 0, 0, 0, time.UTC),
	}
	repo := &fakeTransactionRepo{created: []*transaction.Transaction{txn}}
	accounts := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"acc-1": {ID: "acc-1", ProfileID: "p1", Name: "Conta", Type: bankaccount.AccountTypeChecking, Currency: "BRL"},
	}}

	cancelled := "CANCELLED"
	if _, err := NewUpdateTransactionUseCase(accounts, nil, repo, nil, nil).Execute("tx-1", UpdateTransactionInput{
		BankAccountID: "acc-1", Type: "EXPENSE", Status: &cancelled,
		Amount: 25, Currency: "BRL", Description: "Contabo - VPS", OccurredOn: "2026-09-09",
	}); err != nil {
		t.Fatalf("correcting the description of a cancelled entry must work: %v", err)
	}
}

// A full-replacement PUT with no currency in the body used to rewrite the row to BRL.
// On a EUR account that silently multiplies every total by the exchange rate — and now
// that reconciliation compares currencies, the row stops answering for its own bank
// line, so a charge that is fully accounted for is reported as missing money.
func TestUpdateTransaction_OmittingTheCurrencyKeepsTheAccountsOwn(t *testing.T) {
	const accountID, txID = "acc-eur", "tx-eur"
	account := &bankaccount.BankAccount{
		ID: accountID, ProfileID: "p1", Name: "Wise (EUR)",
		Type: bankaccount.AccountTypeChecking, Currency: "EUR", IsActive: true,
	}
	tx := &transaction.Transaction{
		ID: txID, ProfileID: "p1", BankAccountID: accountID,
		Type: transaction.TypeExpense, Status: transaction.StatusConfirmed,
		Amount: 40, Currency: "EUR", Description: "Assinatura",
		OccurredOn: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
	}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{tx}}
	uc := NewUpdateTransactionUseCase(
		&fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{accountID: account}},
		&fakeCategoryRepo{}, txRepo, &fakeInvoiceRepo{}, &noopBalanceRecalculator{},
	)

	if _, err := uc.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID,
		Type:          "EXPENSE",
		Amount:        40,
		Description:   "Assinatura mensal",
		OccurredOn:    "2026-09-05",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := txRepo.created[0].Currency; got != "EUR" {
		t.Errorf("the row was rewritten to %s on a EUR account", got)
	}

	// And an explicit disagreement is refused rather than converted: this service does
	// not know the rate the bank used.
	_, err := uc.Execute(txID, UpdateTransactionInput{
		BankAccountID: accountID, Type: "EXPENSE", Amount: 40,
		Currency: "BRL", Description: "Assinatura", OccurredOn: "2026-09-05",
	})
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}
