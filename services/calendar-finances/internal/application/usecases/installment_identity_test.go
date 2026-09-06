package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// A purchase carries one external id; its instalments are separate rows, and
// finance.transactions has a UNIQUE index on external_id. Copying the same id
// onto all of them means the first row is written and the second violates the
// key — leaving an orphan "1/5" in the ledger, no rollback, and a 500 for the
// caller. That is precisely the shape the CRM integration is meant to take.
func TestCreateInstallments_GivesEachInstallmentItsOwnExternalID(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeChecking, 5000)

	total := 5
	planned := "PLANNED"
	dealID := "crm-deal-gomez-2026"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		CategoryID:       &f.categoryID,
		Type:             "EXPENSE",
		Status:           &planned,
		Amount:           2500,
		Currency:         "BRL",
		Description:      "Contrato Gomez Studio",
		OccurredOn:       "2026-08-31",
		InstallmentTotal: &total,
		ExternalID:       &dealID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(f.txRepo.created) != 5 {
		t.Fatalf("expected 5 instalments, got %d", len(f.txRepo.created))
	}

	seen := map[string]bool{}
	for i, txn := range f.txRepo.created {
		if txn.ExternalID == nil {
			t.Fatalf("instalment %d lost the external id", i+1)
		}
		if seen[*txn.ExternalID] {
			t.Errorf("external id %q is reused; the unique index would reject it", *txn.ExternalID)
		}
		seen[*txn.ExternalID] = true
	}

	// The deal has to stay recognisable, or the rows cannot be traced back to it.
	for i, txn := range f.txRepo.created {
		if len(*txn.ExternalID) <= len(dealID) || (*txn.ExternalID)[:len(dealID)] != dealID {
			t.Errorf("instalment %d: external id %q does not carry the deal id", i+1, *txn.ExternalID)
		}
	}
}

// A single due date for the whole plan makes an accounts-receivable view wrong
// from the first read: five instalments, all apparently due on the same day.
func TestCreateInstallments_AdvancesTheDueDateWithEachInstallment(t *testing.T) {
	f := newInstallmentFixture(t, bankaccount.AccountTypeChecking, 5000)

	total := 3
	planned := "PLANNED"
	due := "2026-09-14"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:        f.profileID,
		BankAccountID:    f.accountID,
		CategoryID:       &f.categoryID,
		Type:             "EXPENSE",
		Status:           &planned,
		Amount:           300,
		Currency:         "BRL",
		Description:      "Compra parcelada",
		OccurredOn:       "2026-08-31",
		DueOn:            &due,
		InstallmentTotal: &total,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []time.Time{
		time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC),
	}
	for i, txn := range f.txRepo.created {
		if txn.DueOn == nil {
			t.Fatalf("instalment %d has no due date", i+1)
		}
		if !txn.DueOn.Equal(want[i]) {
			t.Errorf("instalment %d due %s, want %s", i+1, txn.DueOn.Format("2006-01-02"), want[i].Format("2006-01-02"))
		}
	}
}
