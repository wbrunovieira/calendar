package capitalcontribution

import (
	"testing"
	"time"
)

func TestNewCapitalContribution_Valid(t *testing.T) {
	c, err := NewCapitalContribution(CreateParams{
		ProfileID:   "profile-1",
		Type:        TypeContribution,
		Amount:      5000,
		Date:        time.Now(),
		Description: "Capital inicial WB Digital",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == "" {
		t.Fatal("expected generated ID")
	}
	if c.IsReturned {
		t.Fatal("new contribution should not be marked as returned")
	}
	if c.OutstandingBalance() != 5000 {
		t.Fatalf("expected outstanding balance 5000, got %f", c.OutstandingBalance())
	}
}

func TestNewCapitalContribution_RequiresProfileID(t *testing.T) {
	_, err := NewCapitalContribution(CreateParams{
		Type: TypeContribution, Amount: 1000, Date: time.Now(), Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for missing profileID")
	}
}

func TestNewCapitalContribution_RequiresPositiveAmount(t *testing.T) {
	_, err := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeContribution, Amount: 0, Date: time.Now(), Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestNewCapitalContribution_RequiresDescription(t *testing.T) {
	_, err := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeContribution, Amount: 1000, Date: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestNewCapitalContribution_RequiresDate(t *testing.T) {
	_, err := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeContribution, Amount: 1000, Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for zero date")
	}
}

func TestNewCapitalContribution_InvalidType(t *testing.T) {
	_, err := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: "INVALID", Amount: 1000, Date: time.Now(), Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestMarkAsReturned(t *testing.T) {
	c, _ := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeLoan, Amount: 3000, Date: time.Now(), Description: "Empréstimo",
	})
	if err := c.MarkAsReturned(3000, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.IsReturned {
		t.Fatal("expected IsReturned to be true")
	}
	if c.ReturnedAt == nil {
		t.Fatal("expected ReturnedAt to be set")
	}
}

func TestMarkAsReturned_RequiresPositiveAmount(t *testing.T) {
	c, _ := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeLoan, Amount: 3000, Date: time.Now(), Description: "test",
	})
	if err := c.MarkAsReturned(0, time.Now()); err == nil {
		t.Fatal("expected error for zero returned amount")
	}
}

func TestOutstandingBalance_Withdrawal(t *testing.T) {
	c, _ := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeWithdrawal, Amount: 1000, Date: time.Now(), Description: "Retirada",
	})
	if c.OutstandingBalance() != 0 {
		t.Fatal("withdrawal should have 0 outstanding balance")
	}
}

func TestOutstandingBalance_PartialReturn(t *testing.T) {
	c, _ := NewCapitalContribution(CreateParams{
		ProfileID: "p1", Type: TypeContribution, Amount: 5000, Date: time.Now(), Description: "Aporte",
	})
	_ = c.MarkAsReturned(2000, time.Now())
	if c.OutstandingBalance() != 3000 {
		t.Fatalf("expected 3000 outstanding, got %f", c.OutstandingBalance())
	}
}
