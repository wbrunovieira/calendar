package usecases

import (
	"strings"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func paidByACentMore(paid, lines float64) (*CheckInvariantsUseCase, *invariantTxRepo) {
	card := invariantCheckingAccount("card-1", 0, 0)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}
	ref := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	invoices := &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{
		"card-1": {{ID: "inv", BankAccountID: "card-1", ReferenceDate: ref, Amount: lines, Status: invoice.StatusPaid}},
	}}
	txs := &invariantTxRepo{
		balances:    map[string]float64{"card-1": 0},
		invoiceSums: map[string]float64{"inv": lines},
		installments: []*transaction.Transaction{
			{ID: "perna", BankAccountID: "conta", DestinationAccountID: strPtr("card-1"),
				Type: transaction.TypeTransfer, Status: transaction.StatusConfirmed,
				Amount: paid, Currency: "BRL", Description: "Pagamento fatura",
				OccurredOn: ref, PaidInvoiceID: strPtr("inv")},
		},
	}
	return NewCheckInvariantsUseCase(accounts, txs, invoices), txs
}

// Nubank's "Total a pagar" applies its own rounding, so the raw sum of a bill's lines
// comes out one or two cents off the amount actually charged -- and the amount
// actually charged is what was paid. Three settled bills sit exactly there: 1.017,24
// paid against lines of 1.017,23, and the same shape twice more.
//
// It is reported with the reason, like a balance that tracks market quotes, and does
// not fail the check. An alarm nobody can clear is an alarm everybody learns to
// ignore, which costs more than the cent.
func TestCheckInvariants_ACentOfIssuerRoundingIsExplainedNotFailed(t *testing.T) {
	uc, _ := paidByACentMore(1017.24, 1017.23)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.PaymentDrifts) != 1 {
		t.Fatalf("drifts = %d, want it still reported", len(result.PaymentDrifts))
	}
	if note := result.PaymentDrifts[0].Note; note == "" {
		t.Fatal("reported with no reason, so a reader cannot tell it from a real excess")
	} else if !strings.Contains(strings.ToLower(note), "round") {
		t.Errorf("note = %q, want it to name the rounding", note)
	}
	if !result.OK {
		t.Error("OK = false over one cent of the issuer's own rounding")
	}
}

// Two cents is the documented limit of that rounding. Beyond it, money is missing and
// the check must fail.
func TestCheckInvariants_MoreThanTheRoundingStillFails(t *testing.T) {
	uc, _ := paidByACentMore(1020.23, 1017.23)

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.PaymentDrifts) != 1 {
		t.Fatalf("drifts = %d, want 1", len(result.PaymentDrifts))
	}
	if result.PaymentDrifts[0].Note != "" {
		t.Errorf("a 3.00 excess was excused as rounding: %q", result.PaymentDrifts[0].Note)
	}
	if result.OK {
		t.Error("OK = true while a bill records 3.00 more paid than it is worth")
	}
}

// Exactly at the limit is still rounding; a cent past it is not.
func TestCheckInvariants_TheRoundingLimitIsTwoCents(t *testing.T) {
	uc, _ := paidByACentMore(1017.25, 1017.23)
	result, _ := uc.Execute()
	if !result.OK || result.PaymentDrifts[0].Note == "" {
		t.Error("two cents is within the issuer's rounding and must be excused")
	}

	uc, _ = paidByACentMore(1017.26, 1017.23)
	result, _ = uc.Execute()
	if result.OK || result.PaymentDrifts[0].Note != "" {
		t.Error("three cents is past the rounding and must fail")
	}
}
