package usecases

import (
	"sort"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// Notes explaining a difference that is expected today. An entry carrying a note
// is reported for visibility but does not fail the check, because there is no
// action available that would clear it. Each note is meant to be deleted as the
// gap it describes is closed.
const (
	// Stock, FII and crypto positions hold quotas x quote, written by the price
	// pollers. Their balance is not a ledger and never will be while the two
	// live in the same column.
	noteMarketValue = "stored balance tracks market quotes, not transactions"

	// A credit card's balance is a frozen snapshot of the last invoice payment.
	// create_transaction, update_transaction and delete_transaction all skip
	// cards entirely — balance update and recalculation both — and only
	// PayInvoiceUseCaseV2 (and the manual recalculate route) ever writes one. So
	// every purchase made since the last payment shows up here as drift, with no
	// missing transaction behind it. Chasing that number would be chasing the
	// design, not a bug. Phase 1 makes the balance derived and deletes this note.
	noteCreditCardSnapshot = "credit card balance is a snapshot of the last invoice payment"

	// A PAID invoice's total cannot be brought back in line: POST
	// /invoices/{id}/recalculate refuses PAID. An OPEN or CLOSED one can, by
	// that same route, so it is a real finding and is reported without a note.
	notePaidInvoiceFrozen = "a paid invoice total can no longer be recalculated"
)

// AccountInvariant is one account whose stored balance differs from the balance
// its own transactions justify.
type AccountInvariant struct {
	AccountID       string  `json:"accountId"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	IsActive        bool    `json:"isActive"`
	StoredBalance   float64 `json:"storedBalance"`
	ComputedBalance float64 `json:"computedBalance"`
	Drift           float64 `json:"drift"`
	Note            string  `json:"note,omitempty"`
}

// InvoiceInvariant is one invoice whose stored total differs from the sum of the
// transactions linked to it.
type InvoiceInvariant struct {
	InvoiceID      string  `json:"invoiceId"`
	BankAccountID  string  `json:"bankAccountId"`
	ReferenceDate  string  `json:"referenceDate"`
	Status         string  `json:"status"`
	StoredAmount   float64 `json:"storedAmount"`
	ComputedAmount float64 `json:"computedAmount"`
	Drift          float64 `json:"drift"`
	Note           string  `json:"note,omitempty"`
}

// CycleInvariant is a pair of invoices on one card whose periods overlap, or a gap
// between two consecutive ones. Both are money with nowhere correct to go.
//
// This is the shape of the failure the Nubank Juridica card had: a label collision
// made the code stretch a cycle instead of relabelling it, producing one invoice
// covering 2026-07-28 to 2026-09-27, and a real payment with no invoice to land on.
type CycleInvariant struct {
	BankAccountID string `json:"bankAccountId"`
	Kind          string `json:"kind"` // OVERLAP | GAP
	InvoiceA      string `json:"invoiceA"`
	InvoiceB      string `json:"invoiceB"`
	PeriodA       string `json:"periodA"`
	PeriodB       string `json:"periodB"`
	Detail        string `json:"detail"`
}

// InstallmentInvariant is a series whose parts do not add up: numbers missing from
// 1..total, or duplicated.
//
// The instalment loop writes rows one at a time with no database transaction around
// it, so a failure at part 7 of 12 leaves six committed and the caller told the whole
// thing failed — nobody goes looking.
type InstallmentInvariant struct {
	BankAccountID string `json:"bankAccountId"`
	Description   string `json:"description"`
	Total         int    `json:"total"`
	Found         int    `json:"found"`
	Missing       []int  `json:"missing,omitempty"`
	Duplicated    []int  `json:"duplicated,omitempty"`
}

// PaymentInvariant is a bill whose live payments exceed what it is worth.
//
// This is the failure behind the R$ 8.863,07 phantom, seen from the other side: a bill
// can be marked paid without the money moving, and money can move twice against one
// bill. Unlike a uniqueness constraint it blocks nothing — two legitimate identical
// payments stay possible — it only reports what does not add up.
type PaymentInvariant struct {
	InvoiceID     string  `json:"invoiceId"`
	BankAccountID string  `json:"bankAccountId"`
	InvoiceAmount float64 `json:"invoiceAmount"`
	PaidTotal     float64 `json:"paidTotal"`
	Excess        float64 `json:"excess"`
}

// CheckInvariantsResult is a read-only report. Nothing here writes: a drift is a
// transaction to hunt down, never a number to overwrite.
//
// OK is false only for differences someone can actually act on: a signal that
// cannot be brought to zero stops being read within a week. Everything else is
// still reported, with a note saying why no action is available.
type CheckInvariantsResult struct {
	CheckedAccounts int                `json:"checkedAccounts"`
	CheckedInvoices int                `json:"checkedInvoices"`
	AccountDrifts   []AccountInvariant `json:"accountDrifts"`
	InvoiceDrifts   []InvoiceInvariant `json:"invoiceDrifts"`
	// CycleDrifts and InstallmentDrifts are always actionable: unlike a stored
	// balance that tracks market quotes, there is no design reason for a card to
	// have overlapping cycles or a half-written instalment plan.
	CycleDrifts       []CycleInvariant       `json:"cycleDrifts"`
	InstallmentDrifts []InstallmentInvariant `json:"installmentDrifts"`
	PaymentDrifts     []PaymentInvariant     `json:"paymentDrifts"`
	OK                bool                   `json:"ok"`
}

type CheckInvariantsUseCase struct {
	accountRepo bankaccount.Repository
	txRepo      transaction.Repository
	invoiceRepo invoice.Repository
}

func NewCheckInvariantsUseCase(
	accountRepo bankaccount.Repository,
	txRepo transaction.Repository,
	invoiceRepo invoice.Repository,
) *CheckInvariantsUseCase {
	return &CheckInvariantsUseCase{
		accountRepo: accountRepo,
		txRepo:      txRepo,
		invoiceRepo: invoiceRepo,
	}
}

// Execute compares, for every account, the stored balance against
// `initial_balance + sum(CONFIRMED transactions)`, and for every credit-card
// invoice, the stored total against the sum of its linked transactions.
func (uc *CheckInvariantsUseCase) Execute() (*CheckInvariantsResult, error) {
	accounts, err := uc.accountRepo.FindAll()
	if err != nil {
		return nil, err
	}

	result := &CheckInvariantsResult{
		AccountDrifts:     []AccountInvariant{},
		InvoiceDrifts:     []InvoiceInvariant{},
		CycleDrifts:       []CycleInvariant{},
		InstallmentDrifts: []InstallmentInvariant{},
		PaymentDrifts:     []PaymentInvariant{},
		OK:                true,
	}

	for _, account := range accounts {
		result.CheckedAccounts++

		if err := uc.checkAccountBalance(account, result); err != nil {
			return nil, err
		}

		if !account.IsCreditCard() {
			continue
		}
		if err := uc.checkInvoiceTotals(account, result); err != nil {
			return nil, err
		}
		if err := uc.checkCycleCoverage(account, result); err != nil {
			return nil, err
		}
		if err := uc.checkInstallmentSeries(account, result); err != nil {
			return nil, err
		}
		if err := uc.checkPaymentsAgainstBills(account, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// checkPaymentsAgainstBills reports a bill whose live payments exceed its value.
//
// Reversed and cancelled payments are excluded: they were undone on purpose. The
// tolerance absorbs cent rounding only — anything above it is money that moved twice
// or a bill that shrank after being paid, and both are worth a look.
func (uc *CheckInvariantsUseCase) checkPaymentsAgainstBills(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	invoices, err := uc.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return err
	}
	if len(invoices) == 0 {
		return nil
	}

	accountID := account.ID
	txns, err := uc.txRepo.List(transaction.ListFilter{ProfileID: account.ProfileID, BankAccountID: &accountID, IncludeAsDestination: true})
	if err != nil {
		return err
	}

	paid := map[string]float64{}
	for _, txn := range txns {
		if txn.PaidInvoiceID == nil {
			continue
		}
		if txn.Status == transaction.StatusReversed || txn.Status == transaction.StatusCancelled {
			continue
		}
		paid[*txn.PaidInvoiceID] += txn.Amount
	}

	for _, inv := range invoices {
		total, ok := paid[inv.ID]
		if !ok {
			continue
		}
		// Compare against the derived total, never the stored one. inv.Amount is a
		// cache, and production invoices routinely carry a stored amount of 0 — using
		// it would report the whole payment as excess on every one of them, and a
		// report that cries wolf is worse than no report.
		worth, err := uc.txRepo.SumByInvoiceID(inv.ID)
		if err != nil {
			return err
		}
		worth = round2(worth)
		if total <= worth+0.005 {
			continue
		}
		result.PaymentDrifts = append(result.PaymentDrifts, PaymentInvariant{
			InvoiceID: inv.ID, BankAccountID: account.ID,
			InvoiceAmount: worth, PaidTotal: total,
			Excess: round2(total - worth),
		})
		result.OK = false
	}
	return nil
}

// checkCycleCoverage verifies that a card's invoices tile its timeline: no two cover
// the same day, and no day between two consecutive ones is covered by neither.
//
// Periods are half-open, [opening, closing), so one cycle opening exactly on the
// previous closing date is correct and not a gap.
func (uc *CheckInvariantsUseCase) checkCycleCoverage(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	invoices, err := uc.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return err
	}
	if len(invoices) < 2 {
		return nil
	}

	ordered := make([]*invoice.Invoice, len(invoices))
	copy(ordered, invoices)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].OpeningDate.Before(ordered[j].OpeningDate) })

	period := func(inv *invoice.Invoice) string {
		return inv.OpeningDate.Format("2006-01-02") + " a " + inv.ClosingDate.Format("2006-01-02")
	}

	for i := 1; i < len(ordered); i++ {
		prev, cur := ordered[i-1], ordered[i]
		switch {
		case cur.OpeningDate.Before(prev.ClosingDate):
			result.CycleDrifts = append(result.CycleDrifts, CycleInvariant{
				BankAccountID: account.ID, Kind: "OVERLAP",
				InvoiceA: prev.ID, InvoiceB: cur.ID,
				PeriodA: period(prev), PeriodB: period(cur),
				Detail: "two invoices cover the same days; a purchase in the overlap lands on whichever the query happens to return",
			})
			result.OK = false
		case cur.OpeningDate.After(prev.ClosingDate):
			result.CycleDrifts = append(result.CycleDrifts, CycleInvariant{
				BankAccountID: account.ID, Kind: "GAP",
				InvoiceA: prev.ID, InvoiceB: cur.ID,
				PeriodA: period(prev), PeriodB: period(cur),
				Detail: "days covered by no invoice; a purchase there belongs to nothing",
			})
			result.OK = false
		}
	}
	return nil
}

// checkInstallmentSeries verifies that every instalment plan on the card has all its
// parts, numbered 1..total exactly once.
//
// A part that exists as REVERSED or CANCELLED still EXISTS: it was undone on purpose,
// which is a different thing from never having been written. The bug being hunted here
// is the half-written plan — the loop writes rows one at a time with no database
// transaction around it, so a failure at part 7 of 12 leaves six committed and the
// caller told the whole thing failed. Counting an undone part as missing would raise
// a false alarm and train the reader to ignore the signal.
//
// Undone parts are excluded from the DUPLICATE check for the mirror reason: reversing
// a wrong part and writing the right one is a correction, not a duplicate.
func (uc *CheckInvariantsUseCase) checkInstallmentSeries(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	accountID := account.ID
	// The real repository filters out REVERSED unless asked, so without this the
	// branch below that treats a reversed instalment as present is unreachable and a
	// deliberately reversed instalment is reported as missing — a false alarm, which
	// is the one thing an invariant report cannot afford.
	txns, err := uc.txRepo.List(transaction.ListFilter{
		ProfileID:       account.ProfileID,
		BankAccountID:   &accountID,
		IncludeReversed: true,
	})
	if err != nil {
		return err
	}

	type key struct {
		description string
		total       int
	}
	series := map[key]map[int]int{}
	for _, txn := range txns {
		if txn.InstallmentNumber == nil || txn.InstallmentTotal == nil || *txn.InstallmentTotal < 2 {
			continue
		}
		k := key{txn.Description, *txn.InstallmentTotal}
		if series[k] == nil {
			series[k] = map[int]int{}
		}
		if txn.Status == transaction.StatusReversed || txn.Status == transaction.StatusCancelled {
			// Present, but does not count towards duplication.
			if series[k][*txn.InstallmentNumber] == 0 {
				series[k][*txn.InstallmentNumber] = 1
			}
			continue
		}
		series[k][*txn.InstallmentNumber]++
	}

	for k, parts := range series {
		var missing, duplicated []int
		for n := 1; n <= k.total; n++ {
			switch parts[n] {
			case 0:
				missing = append(missing, n)
			case 1:
			default:
				duplicated = append(duplicated, n)
			}
		}
		if len(missing) == 0 && len(duplicated) == 0 {
			continue
		}
		result.InstallmentDrifts = append(result.InstallmentDrifts, InstallmentInvariant{
			BankAccountID: account.ID, Description: k.description,
			Total: k.total, Found: len(parts),
			Missing: missing, Duplicated: duplicated,
		})
		result.OK = false
	}
	return nil
}

func (uc *CheckInvariantsUseCase) checkAccountBalance(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	ledger, err := uc.txRepo.CalculateBalanceByBankAccountID(account.ID)
	if err != nil {
		return err
	}

	computed := round2(account.InitialBalance + ledger)
	// round2 puts both sides on cent granularity, so anything left is a real
	// difference rather than the float noise of summing amounts.
	drift := round2(account.CurrentBalance - computed)
	if drift == 0 {
		return nil
	}

	note := expectedDifferenceNote(account)
	result.AccountDrifts = append(result.AccountDrifts, AccountInvariant{
		AccountID:       account.ID,
		Name:            account.Name,
		Type:            string(account.Type),
		IsActive:        account.IsActive,
		StoredBalance:   round2(account.CurrentBalance),
		ComputedBalance: computed,
		Drift:           drift,
		Note:            note,
	})
	if note == "" {
		result.OK = false
	}
	return nil
}

func (uc *CheckInvariantsUseCase) checkInvoiceTotals(
	account *bankaccount.BankAccount,
	result *CheckInvariantsResult,
) error {
	invoices, err := uc.invoiceRepo.FindByBankAccountID(account.ID)
	if err != nil {
		return err
	}

	for _, inv := range invoices {
		result.CheckedInvoices++

		computed, err := uc.txRepo.SumByInvoiceID(inv.ID)
		if err != nil {
			return err
		}

		computed = round2(computed)
		drift := round2(inv.Amount - computed)
		if drift == 0 {
			continue
		}

		// An OPEN or CLOSED invoice can be brought back in line with one call to
		// POST /invoices/{id}/recalculate, which persists exactly this sum. That
		// makes its drift actionable, and a stored total that is double the sum
		// of its transactions is the signature of a duplicated or lost charge.
		// Only PAID is frozen, because that route refuses it.
		note := ""
		if inv.Status == invoice.StatusPaid {
			note = notePaidInvoiceFrozen
		}

		result.InvoiceDrifts = append(result.InvoiceDrifts, InvoiceInvariant{
			InvoiceID:      inv.ID,
			BankAccountID:  inv.BankAccountID,
			ReferenceDate:  inv.ReferenceDate.Format("2006-01"),
			Status:         string(inv.Status),
			StoredAmount:   round2(inv.Amount),
			ComputedAmount: computed,
			Drift:          drift,
			Note:           note,
		})
		if note == "" {
			result.OK = false
		}
	}

	return nil
}

// expectedDifferenceNote returns why an account's stored balance is allowed to
// differ from its ledger today, or "" when the difference is a real defect
// someone can act on.
func expectedDifferenceNote(account *bankaccount.BankAccount) string {
	switch account.Type {
	case bankaccount.AccountTypeInvestment,
		bankaccount.AccountTypeExchange,
		bankaccount.AccountTypeWallet:
		return noteMarketValue

	case bankaccount.AccountTypeCreditCard:
		// Every card, whatever its balance. A card at zero was never paid; a
		// card holding a figure was written by the last invoice payment and has
		// been going stale by design ever since. Neither difference is a
		// transaction anyone can go and find.
		return noteCreditCardSnapshot

	default:
		return ""
	}
}
