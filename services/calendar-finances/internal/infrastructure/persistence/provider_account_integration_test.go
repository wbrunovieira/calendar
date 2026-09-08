//go:build integration
// +build integration

package persistence

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// The importer has to turn "the provider's account" into "the account in this system",
// and it must never guess. Every statement line arrives labelled with the bank's own id;
// without a stored mapping the only alternative is matching by name or by balance, and
// matching by value is precisely what once deleted a legitimate entry.
func TestIntegration_AnAccountIsFoundByTheProvidersOwnID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)

	repo := NewBankAccountRepository(db)
	account, err := repo.FindByID(accountID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	providerID := "2c9d2ca0-" + uuid.NewString()[:8]
	account.ProviderAccountID = &providerID
	if err := repo.Update(account); err != nil {
		t.Fatalf("update: %v", err)
	}

	found, err := repo.FindByProviderAccountID(providerID)
	if err != nil {
		t.Fatalf("find by provider id: %v", err)
	}
	if found == nil {
		t.Fatal("the mapping did not survive the round trip")
	}
	if found.ID != accountID {
		t.Errorf("resolved to the wrong account: %s", found.ID)
	}
	if found.ProviderAccountID == nil || *found.ProviderAccountID != providerID {
		t.Error("the provider id must come back on the account it identifies")
	}
}

// An unmapped provider account resolves to nothing, and the caller has to deal with
// that. Answering "not found" is the only honest reply: importing into a guessed
// account writes someone else's spending onto the wrong statement.
func TestIntegration_AnUnmappedProviderAccountResolvesToNothing(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	found, err := NewBankAccountRepository(db).FindByProviderAccountID("nunca-mapeada-" + uuid.NewString())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatalf("expected nothing, got account %s", found.ID)
	}
}

// Two accounts must not claim the same provider account: the second import would
// silently land on whichever the query happened to return first.
func TestIntegration_TwoAccountsCannotClaimTheSameProviderAccount(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID := uuid.NewString()
	first, second := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, first)
	seedProfileAndAccount(t, db, profileID, second)

	repo := NewBankAccountRepository(db)
	providerID := "duplicada-" + uuid.NewString()[:8]

	a, _ := repo.FindByID(first)
	a.ProviderAccountID = &providerID
	if err := repo.Update(a); err != nil {
		t.Fatalf("first mapping: %v", err)
	}

	b, _ := repo.FindByID(second)
	b.ProviderAccountID = &providerID
	if err := repo.Update(b); err == nil {
		t.Fatal("the second account must be refused: one provider account, one system account")
	}
}

var _ bankaccount.Repository = (*BankAccountRepository)(nil)

// Create had no test at all: every suite seeds accounts with raw SQL, so the one
// method that actually writes an account through the repository was never exercised.
// A column duplicated in the INSERT list therefore broke account creation everywhere
// while the whole suite stayed green.
func TestIntegration_AnAccountCanActuallyBeCreated(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID := uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO finance.profiles (id, calendar_id, name, type)
		VALUES ($1, $2, 'Integration', 'PERSONAL') ON CONFLICT (id) DO NOTHING`,
		profileID, "create-"+profileID); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	providerID := "prov-" + uuid.NewString()[:8]
	account := &bankaccount.BankAccount{
		ID: uuid.NewString(), ProfileID: profileID, Name: "Conta criada pela API",
		Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true,
		ProviderAccountID: &providerID,
	}

	repo := NewBankAccountRepository(db)
	if err := repo.Create(account); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByID(account.ID)
	if err != nil || found == nil {
		t.Fatalf("the created account cannot be read back: %v", err)
	}
	if found.Name != "Conta criada pela API" {
		t.Errorf("name came back %q", found.Name)
	}
	if found.ProviderAccountID == nil || *found.ProviderAccountID != providerID {
		t.Errorf("the mapping must survive creation, got %v", found.ProviderAccountID)
	}
}

// A write that changed no rows is not "you asked for an account that does not exist".
// The distinction decides an HTTP status: a caller that named a bad account should be
// told 400 and stop, while a balance update that silently touched nothing is this
// service failing and must surface as 5xx so n8n and the agents retry instead of
// discarding the work. Sharing one error value between the two told the caller its
// request was malformed while a transaction row sat written with a balance that never
// moved.
func TestIntegration_AWriteThatTouchedNoRowsIsNotAMissingAccount(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	profileID, accountID := uuid.NewString(), uuid.NewString()
	seedProfileAndAccount(t, db, profileID, accountID)
	repo := NewBankAccountRepository(db)

	account, err := repo.FindByID(accountID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	// The row is gone by the time the write lands — a concurrent delete, or a bug.
	account.ID = uuid.NewString()

	err = repo.Update(account)
	if err == nil {
		t.Fatal("updating a row that is not there reported success")
	}
	if errors.Is(err, bankaccount.ErrNotFound) {
		t.Errorf("a failed write is reported as a missing account, which the handlers map to 400: %v", err)
	}

	if err := repo.Delete(uuid.NewString()); errors.Is(err, bankaccount.ErrNotFound) {
		t.Errorf("same for delete: %v", err)
	}

	// And the read must keep answering "not found", because that one really is the
	// caller naming something that does not exist.
	if _, err := repo.FindByID(uuid.NewString()); !errors.Is(err, bankaccount.ErrNotFound) {
		t.Errorf("FindByID on an unknown id: got %v, want ErrNotFound", err)
	}
}
