package strategyallocation

import (
	"math"
	"testing"
	"time"
)

func TestNewStrategyAllocation(t *testing.T) {
	t.Run("valid pending allocation", func(t *testing.T) {
		sa, err := NewStrategyAllocation("profile-1", "MACross1", "tx-transfer-1", 2000.00)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sa.ProfileID != "profile-1" {
			t.Errorf("ProfileID = %s, want profile-1", sa.ProfileID)
		}
		if sa.Strategy != "MACross1" {
			t.Errorf("Strategy = %s, want MACross1", sa.Strategy)
		}
		if sa.TransactionID != "tx-transfer-1" {
			t.Errorf("TransactionID = %s, want tx-transfer-1", sa.TransactionID)
		}
		if sa.Amount != 2000.00 {
			t.Errorf("Amount = %f, want 2000", sa.Amount)
		}
		if sa.Status != StatusPending {
			t.Errorf("Status = %s, want %s", sa.Status, StatusPending)
		}
	})

	t.Run("empty strategy", func(t *testing.T) {
		_, err := NewStrategyAllocation("profile-1", "", "tx-1", 1000)
		if err != ErrInvalidStrategy {
			t.Errorf("err = %v, want ErrInvalidStrategy", err)
		}
	})

	t.Run("zero amount", func(t *testing.T) {
		_, err := NewStrategyAllocation("profile-1", "MACross1", "tx-1", 0)
		if err != ErrInvalidAmount {
			t.Errorf("err = %v, want ErrInvalidAmount", err)
		}
	})

	t.Run("empty profile", func(t *testing.T) {
		_, err := NewStrategyAllocation("", "MACross1", "tx-1", 1000)
		if err != ErrInvalidProfile {
			t.Errorf("err = %v, want ErrInvalidProfile", err)
		}
	})
}

func TestApprove(t *testing.T) {
	sa, _ := NewStrategyAllocation("profile-1", "MACross1", "tx-1", 2000)

	t.Run("approve pending", func(t *testing.T) {
		err := sa.Approve()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sa.Status != StatusApproved {
			t.Errorf("Status = %s, want %s", sa.Status, StatusApproved)
		}
	})

	t.Run("cannot approve already approved", func(t *testing.T) {
		err := sa.Approve()
		if err != ErrNotPending {
			t.Errorf("err = %v, want ErrNotPending", err)
		}
	})
}

func TestDecline(t *testing.T) {
	sa, _ := NewStrategyAllocation("profile-1", "MACross1", "tx-1", 2000)

	t.Run("decline pending", func(t *testing.T) {
		err := sa.Decline()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sa.Status != StatusDeclined {
			t.Errorf("Status = %s, want %s", sa.Status, StatusDeclined)
		}
	})

	t.Run("cannot decline already declined", func(t *testing.T) {
		err := sa.Decline()
		if err != ErrNotPending {
			t.Errorf("err = %v, want ErrNotPending", err)
		}
	})
}

func TestTotalAllocated(t *testing.T) {
	allocations := []*StrategyAllocation{
		{Amount: 1672, Status: StatusApproved},
		{Amount: 2000, Status: StatusApproved},
		{Amount: 500, Status: StatusDeclined},
		{Amount: 300, Status: StatusPending},
	}

	total := TotalAllocated(allocations)
	// Only approved: 1672 + 2000 = 3672
	if math.Abs(total-3672) > 0.01 {
		t.Errorf("TotalAllocated = %f, want 3672", total)
	}
}

func TestStrategySummary(t *testing.T) {
	now := time.Now()
	allocations := []*StrategyAllocation{
		{Amount: 1672, Status: StatusApproved, CreatedAt: now.AddDate(0, 0, -2)},
		{Amount: 2000, Status: StatusApproved, CreatedAt: now.AddDate(0, 0, -1)},
	}

	summary := NewStrategySummary("MACross1", allocations)
	if summary.Strategy != "MACross1" {
		t.Errorf("Strategy = %s, want MACross1", summary.Strategy)
	}
	if math.Abs(summary.TotalAllocatedBRL-3672) > 0.01 {
		t.Errorf("TotalAllocatedBRL = %f, want 3672", summary.TotalAllocatedBRL)
	}
	if summary.AllocationCount != 2 {
		t.Errorf("AllocationCount = %d, want 2", summary.AllocationCount)
	}
}
