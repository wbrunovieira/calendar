package invoice

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("should create invoice with valid params", func(t *testing.T) {
		params := CreateParams{
			BankAccountID: "acc-123",
			ClosingDay:    9,
			DueDay:        14,
			ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		inv, err := New(params)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if inv == nil {
			t.Fatal("Expected invoice, got nil")
		}
		if inv.BankAccountID != "acc-123" {
			t.Errorf("Expected BankAccountID 'acc-123', got %s", inv.BankAccountID)
		}
		if inv.Status != StatusOpen {
			t.Errorf("Expected status OPEN, got %s", inv.Status)
		}
		if inv.Amount != 0 {
			t.Errorf("Expected amount 0, got %f", inv.Amount)
		}
		// Reference date should be first of month
		expectedRef := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		if !inv.ReferenceDate.Equal(expectedRef) {
			t.Errorf("Expected reference date %v, got %v", expectedRef, inv.ReferenceDate)
		}
		// Closing date should be 9th of January
		expectedClosing := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
		if !inv.ClosingDate.Equal(expectedClosing) {
			t.Errorf("Expected closing date %v, got %v", expectedClosing, inv.ClosingDate)
		}
		// Due date should be 14th of January (after closing)
		expectedDue := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
		if !inv.DueDate.Equal(expectedDue) {
			t.Errorf("Expected due date %v, got %v", expectedDue, inv.DueDate)
		}
		// Opening date should be December 9th (previous closing day = opening of next cycle)
		expectedOpening := time.Date(2025, 12, 9, 0, 0, 0, 0, time.UTC)
		if !inv.OpeningDate.Equal(expectedOpening) {
			t.Errorf("Expected opening date %v, got %v", expectedOpening, inv.OpeningDate)
		}
	})

	t.Run("should calculate due date in next month when dueDay <= closingDay", func(t *testing.T) {
		params := CreateParams{
			BankAccountID: "acc-123",
			ClosingDay:    25,
			DueDay:        5, // Due day is less than closing day, so due date is next month
			ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		inv, err := New(params)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		// Due date should be February 5th (next month)
		expectedDue := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
		if !inv.DueDate.Equal(expectedDue) {
			t.Errorf("Expected due date %v, got %v", expectedDue, inv.DueDate)
		}
	})

	t.Run("should handle December to January transition", func(t *testing.T) {
		params := CreateParams{
			BankAccountID: "acc-123",
			ClosingDay:    25,
			DueDay:        5,
			ReferenceDate: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		}

		inv, err := New(params)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		// Due date should be January 5th 2026
		expectedDue := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
		if !inv.DueDate.Equal(expectedDue) {
			t.Errorf("Expected due date %v, got %v", expectedDue, inv.DueDate)
		}
		// Opening date should be November 25th (previous closing day)
		expectedOpening := time.Date(2025, 11, 25, 0, 0, 0, 0, time.UTC)
		if !inv.OpeningDate.Equal(expectedOpening) {
			t.Errorf("Expected opening date %v, got %v", expectedOpening, inv.OpeningDate)
		}
	})

	t.Run("should handle months with fewer days", func(t *testing.T) {
		params := CreateParams{
			BankAccountID: "acc-123",
			ClosingDay:    31, // February doesn't have 31 days
			DueDay:        5,
			ReferenceDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		}

		inv, err := New(params)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		// Closing date should be February 28th (last day of Feb 2026)
		expectedClosing := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
		if !inv.ClosingDate.Equal(expectedClosing) {
			t.Errorf("Expected closing date %v, got %v", expectedClosing, inv.ClosingDate)
		}
	})

	t.Run("should return error when bankAccountID is empty", func(t *testing.T) {
		params := CreateParams{
			BankAccountID: "",
			ClosingDay:    9,
			DueDay:        14,
			ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		_, err := New(params)

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("should return error when closingDay is invalid", func(t *testing.T) {
		testCases := []int{0, 32, -1}
		for _, closingDay := range testCases {
			params := CreateParams{
				BankAccountID: "acc-123",
				ClosingDay:    closingDay,
				DueDay:        14,
				ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			_, err := New(params)

			if err == nil {
				t.Errorf("Expected error for closingDay %d, got nil", closingDay)
			}
		}
	})

	t.Run("should return error when dueDay is invalid", func(t *testing.T) {
		testCases := []int{0, 32, -1}
		for _, dueDay := range testCases {
			params := CreateParams{
				BankAccountID: "acc-123",
				ClosingDay:    9,
				DueDay:        dueDay,
				ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			_, err := New(params)

			if err == nil {
				t.Errorf("Expected error for dueDay %d, got nil", dueDay)
			}
		}
	})

	t.Run("should return error when referenceDate is zero", func(t *testing.T) {
		params := CreateParams{
			BankAccountID: "acc-123",
			ClosingDay:    9,
			DueDay:        14,
			ReferenceDate: time.Time{},
		}

		_, err := New(params)

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("should close open invoice", func(t *testing.T) {
		inv := createTestInvoice(StatusOpen)

		err := inv.Close()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if inv.Status != StatusClosed {
			t.Errorf("Expected status CLOSED, got %s", inv.Status)
		}
	})

	t.Run("should return error when invoice is already closed", func(t *testing.T) {
		inv := createTestInvoice(StatusClosed)

		err := inv.Close()

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("should return error when invoice is paid", func(t *testing.T) {
		inv := createTestInvoice(StatusPaid)

		err := inv.Close()

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestPay(t *testing.T) {
	t.Run("should pay closed invoice", func(t *testing.T) {
		inv := createTestInvoice(StatusClosed)
		inv.Amount = 500.00
		paidAt := time.Now()

		err := inv.Pay(500.00, paidAt)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if inv.Status != StatusPaid {
			t.Errorf("Expected status PAID, got %s", inv.Status)
		}
		if inv.PaidAmount == nil || *inv.PaidAmount != 500.00 {
			t.Errorf("Expected paid amount 500.00, got %v", inv.PaidAmount)
		}
		if inv.PaidAt == nil {
			t.Error("Expected paid at to be set")
		}
	})

	t.Run("should pay open invoice", func(t *testing.T) {
		inv := createTestInvoice(StatusOpen)
		inv.Amount = 500.00
		paidAt := time.Now()

		err := inv.Pay(500.00, paidAt)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if inv.Status != StatusPaid {
			t.Errorf("Expected status PAID, got %s", inv.Status)
		}
	})

	t.Run("should return error when invoice is already paid", func(t *testing.T) {
		inv := createTestInvoice(StatusPaid)

		err := inv.Pay(500.00, time.Now())

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("should return error when paid amount is zero or negative", func(t *testing.T) {
		inv := createTestInvoice(StatusClosed)

		testCases := []float64{0, -10.00}
		for _, amount := range testCases {
			err := inv.Pay(amount, time.Now())

			if err == nil {
				t.Errorf("Expected error for amount %f, got nil", amount)
			}
		}
	})
}

func TestReopen(t *testing.T) {
	t.Run("should reopen closed invoice", func(t *testing.T) {
		inv := createTestInvoice(StatusClosed)

		err := inv.Reopen()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if inv.Status != StatusOpen {
			t.Errorf("Expected status OPEN, got %s", inv.Status)
		}
	})

	t.Run("should return error when invoice is open", func(t *testing.T) {
		inv := createTestInvoice(StatusOpen)

		err := inv.Reopen()

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("should return error when invoice is paid", func(t *testing.T) {
		inv := createTestInvoice(StatusPaid)

		err := inv.Reopen()

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestContainsDate(t *testing.T) {
	t.Run("should return true for date within billing cycle", func(t *testing.T) {
		inv := &Invoice{
			OpeningDate: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
			ClosingDate: time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
		}

		testCases := []time.Time{
			time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),  // Opening date
			time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),  // Middle of cycle
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),    // New year
			time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC),    // Day before closing
			time.Date(2025, 12, 15, 10, 30, 0, 0, time.UTC), // With time component
		}

		for _, txDate := range testCases {
			if !inv.ContainsDate(txDate) {
				t.Errorf("Expected ContainsDate to return true for %v", txDate)
			}
		}
	})

	t.Run("should return false for date outside billing cycle", func(t *testing.T) {
		inv := &Invoice{
			OpeningDate: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
			ClosingDate: time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
		}

		testCases := []time.Time{
			time.Date(2025, 12, 9, 0, 0, 0, 0, time.UTC),   // Before opening
			time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),   // After closing
			time.Date(2025, 11, 15, 0, 0, 0, 0, time.UTC),  // Previous month
		}

		for _, txDate := range testCases {
			if inv.ContainsDate(txDate) {
				t.Errorf("Expected ContainsDate to return false for %v", txDate)
			}
		}
	})
}

func TestStatusHelpers(t *testing.T) {
	t.Run("IsOpen", func(t *testing.T) {
		openInv := createTestInvoice(StatusOpen)
		closedInv := createTestInvoice(StatusClosed)
		paidInv := createTestInvoice(StatusPaid)

		if !openInv.IsOpen() {
			t.Error("Expected IsOpen to return true for open invoice")
		}
		if closedInv.IsOpen() {
			t.Error("Expected IsOpen to return false for closed invoice")
		}
		if paidInv.IsOpen() {
			t.Error("Expected IsOpen to return false for paid invoice")
		}
	})

	t.Run("IsClosed", func(t *testing.T) {
		openInv := createTestInvoice(StatusOpen)
		closedInv := createTestInvoice(StatusClosed)
		paidInv := createTestInvoice(StatusPaid)

		if openInv.IsClosed() {
			t.Error("Expected IsClosed to return false for open invoice")
		}
		if !closedInv.IsClosed() {
			t.Error("Expected IsClosed to return true for closed invoice")
		}
		if paidInv.IsClosed() {
			t.Error("Expected IsClosed to return false for paid invoice")
		}
	})

	t.Run("IsPaid", func(t *testing.T) {
		openInv := createTestInvoice(StatusOpen)
		closedInv := createTestInvoice(StatusClosed)
		paidInv := createTestInvoice(StatusPaid)

		if openInv.IsPaid() {
			t.Error("Expected IsPaid to return false for open invoice")
		}
		if closedInv.IsPaid() {
			t.Error("Expected IsPaid to return false for closed invoice")
		}
		if !paidInv.IsPaid() {
			t.Error("Expected IsPaid to return true for paid invoice")
		}
	})
}

func TestGetReferenceName(t *testing.T) {
	testCases := []struct {
		month    time.Month
		year     int
		expected string
	}{
		{time.January, 2026, "Janeiro 2026"},
		{time.February, 2026, "Fevereiro 2026"},
		{time.March, 2026, "Março 2026"},
		{time.December, 2025, "Dezembro 2025"},
	}

	for _, tc := range testCases {
		inv := &Invoice{
			ReferenceDate: time.Date(tc.year, tc.month, 1, 0, 0, 0, 0, time.UTC),
		}

		result := inv.GetReferenceName()

		if result != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, result)
		}
	}
}

func TestSafeDate(t *testing.T) {
	t.Run("should return correct date for valid day", func(t *testing.T) {
		result := safeDate(2026, time.January, 15)
		expected := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)

		if !result.Equal(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should cap day to last day of month", func(t *testing.T) {
		// February 2026 has 28 days
		result := safeDate(2026, time.February, 31)
		expected := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)

		if !result.Equal(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should handle leap year February", func(t *testing.T) {
		// February 2024 has 29 days (leap year)
		result := safeDate(2024, time.February, 31)
		expected := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC)

		if !result.Equal(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should handle April with 30 days", func(t *testing.T) {
		result := safeDate(2026, time.April, 31)
		expected := time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)

		if !result.Equal(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})
}

// Helper function to create test invoices
func createTestInvoice(status Status) *Invoice {
	now := time.Now()
	return &Invoice{
		ID:            "inv-123",
		BankAccountID: "acc-123",
		ReferenceDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		OpeningDate:   time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
		ClosingDate:   time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		Amount:        0,
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
