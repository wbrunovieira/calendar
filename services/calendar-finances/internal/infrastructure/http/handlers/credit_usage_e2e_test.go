//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"net/http"
	"testing"
	"time"
)

// seedCard creates a profile and a credit card, and returns the card's id. Everything
// it writes is removed when the test ends.
func seedCard(t *testing.T, db *sql.DB, limit float64) (profileID, cardID string) {
	t.Helper()

	if err := db.QueryRow(`
		INSERT INTO finance.profiles (name, type, calendar_id) VALUES ('E2E', 'PERSONAL', 'e2e-' || gen_random_uuid()) RETURNING id
	`).Scan(&profileID); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}
	cardID = seedAccountThroughRepository(t, db, cardAccount(profileID, "Cartao E2E", limit))

	t.Cleanup(func() {
		exec(t, db, `DELETE FROM finance.transactions WHERE profile_id = $1`, profileID)
		exec(t, db, `DELETE FROM finance.credit_card_invoices WHERE bank_account_id = $1`, cardID)
		exec(t, db, `DELETE FROM finance.bank_accounts WHERE profile_id = $1`, profileID)
		exec(t, db, `DELETE FROM finance.profiles WHERE id = $1`, profileID)
	})
	return profileID, cardID
}

// The route must answer at all. It did not: the use case queried transactions without
// a profile id, which the real repository rejects, so every call was a 500. Every
// unit test passed because their fake ignored the filter.
func TestCreditUsageRoute_AnswersForACard(t *testing.T) {
	db := testDB(t)

	_, cardID := seedCard(t, db, 400)

	status, body := get(t, router(t, db), "/api/v1/bank-accounts/"+cardID+"/credit-usage")

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — body %v", status, body)
	}
	if body["data"] == nil {
		t.Fatalf("expected a payload, got %v", body)
	}
}

// The real R$400 card: R$253,48 rolled into instalments on a closed bill plus
// R$126,75 in the open cycle. The bank shows R$19,77 available.
//
// The amounts live in the transactions, not in the invoice's `amount` column: in
// production every open and closed invoice carries 0 there, because the read paths
// recompute it and never persist. Reading the column reports an empty card.
func TestCreditUsageRoute_SumsWhatIsOwedAcrossUnpaidInvoices(t *testing.T) {
	db := testDB(t)

	profileID, cardID := seedCard(t, db, 400)
	closed := seedInvoice(t, db, cardID, "CLOSED", "2026-07-27", "2026-08-27", "2026-09-03")
	open := seedInvoice(t, db, cardID, "OPEN", "2026-08-27", "2026-09-27", "2026-10-03")
	seedCharge(t, db, profileID, cardID, closed, 253.48)
	seedCharge(t, db, profileID, cardID, open, 126.75)

	status, body := get(t, router(t, db), "/api/v1/bank-accounts/"+cardID+"/credit-usage")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — %v", status, body)
	}

	data := body["data"].(map[string]any)
	if got := data["outstanding"].(float64); got < 380.22 || got > 380.24 {
		t.Errorf("expected 380.23 owed, got %v", got)
	}
	if got := data["availableCredit"].(float64); got < 19.76 || got > 19.78 {
		t.Errorf("expected 19.77 available, got %v", got)
	}
}

// A bill part-paid and left open is still owed for the remainder. Two invoices in
// production are marked PAID with paid_amount below amount.
func TestCreditUsageRoute_CountsTheRemainderOfAPartiallyPaidBill(t *testing.T) {
	db := testDB(t)

	profileID, cardID := seedCard(t, db, 400)
	bill := seedInvoice(t, db, cardID, "PAID", "2026-07-27", "2026-08-27", "2026-09-03")
	seedCharge(t, db, profileID, cardID, bill, 361.74)
	exec(t, db, `UPDATE finance.credit_card_invoices SET paid_amount = 60 WHERE id = $1`, bill)

	status, body := get(t, router(t, db), "/api/v1/bank-accounts/"+cardID+"/credit-usage")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d — %v", status, body)
	}

	data := body["data"].(map[string]any)
	if got := data["outstanding"].(float64); got < 301.73 || got > 301.75 {
		t.Errorf("expected the R$301,74 remainder to still be owed, got %v", got)
	}
}

func TestCreditUsageRoute_RejectsAnAccountThatIsNotACard(t *testing.T) {
	db := testDB(t)

	profileID, _ := seedCard(t, db, 400)
	var checkingID string
	checkingID = seedAccountThroughRepository(t, db, checkingAccount(profileID, "Conta E2E", 0, 0))

	status, _ := get(t, router(t, db), "/api/v1/bank-accounts/"+checkingID+"/credit-usage")
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for a non-card account, got %d", status)
	}
}

// seedInvoice takes the opening date instead of deriving it.
//
// The SQL this replaced used Postgres's `- interval '1 month'`, and Go's AddDate is
// not the same function: Postgres clamps to the end of the month, Go overflows. The
// 31st of March goes back to 28 February in one and forward to 3 March in the other.
// Every fixture here closes on the 27th, so the difference is invisible today — and on
// a credit card an opening three days late leaves a window no invoice covers, which
// would pass green. Stating the date beats computing it with the wrong arithmetic.
func seedInvoice(t *testing.T, db *sql.DB, cardID, status, opening, closing, due string) string {
	t.Helper()
	return seedInvoiceThroughRepository(t, db, &invoice.Invoice{
		BankAccountID: cardID, Amount: 0, Status: invoice.Status(status),
		ReferenceDate: mustDate(t, closing), OpeningDate: mustDate(t, opening),
		ClosingDate: mustDate(t, closing), DueDate: mustDate(t, due),
	})
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("bad date %q in the test fixture: %v", value, err)
	}
	return parsed
}

func seedCharge(t *testing.T, db *sql.DB, profileID, cardID, invoiceID string, amount float64) {
	t.Helper()
	exec(t, db, `
		INSERT INTO finance.transactions
			(profile_id, bank_account_id, invoice_id, type, status, amount, currency, description, occurred_on)
		VALUES ($1, $2, $3, 'EXPENSE', 'CONFIRMED', $4, 'BRL', 'Compra E2E', now())
	`, profileID, cardID, invoiceID, amount)
}
