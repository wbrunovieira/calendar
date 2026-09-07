//go:build integration
// +build integration

package persistence

import (
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
