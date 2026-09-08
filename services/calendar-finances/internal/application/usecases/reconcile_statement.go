package usecases

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
	transactionPkg "github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// dateTolerance is how far apart a bank line and an entry may be and still be the same
// movement.
//
// The bank prints the purchase date and an entry often carries the posting date; a day
// or two between them is the norm, not a discrepancy. Reconciling the personal Nubank
// card by hand, every single difference over three months was this and nothing else.
const dateTolerance = 2

// ErrReconcileStorage marks a failure to READ what the reconciler needs, as opposed to
// anything wrong with the request. The morning cron decides whether to retry from the
// status code, and a database that blinked deserves a retry where a bad request does
// not.
var ErrReconcileStorage = errors.New("could not read what the reconciliation needs")

// Matches is what the reconciler needs from the match repository.
type Matches interface {
	Create(m *statement.Match) error
	ByLine(lineID string) ([]*statement.Match, error)
	// ClaimedOnAccount is scoped to one account on purpose: a transfer is a single row
	// that appears on two statements, so it must be claimable once on each.
	ClaimedOnAccount(transactionID, accountID string) (bool, error)
	// Unmatch releases a match whose basis no longer holds. Without it the undo half
	// of an append-only design is decoration: the match row outlives the fact it
	// recorded, and the line it covers is never looked at again.
	Unmatch(matchID, reason string) error
}

// MissingLine is a charge the bank made and the system does not have. It is the point
// of the whole exercise, not an error.
type MissingLine struct {
	LineID      string    `json:"lineId"`
	BookedDate  time.Time `json:"bookedDate"`
	AmountMinor int64     `json:"amountMinor"`
	Description string    `json:"description"`
}

// AmbiguousLine is a line with more than one equally good candidate. Three charges of
// R$ 40 in one month are indistinguishable by value, and choosing one is how a
// legitimate Cloudflare charge was once deleted as a phantom.
type AmbiguousLine struct {
	LineID       string    `json:"lineId"`
	BookedDate   time.Time `json:"bookedDate"`
	AmountMinor  int64     `json:"amountMinor"`
	Description  string    `json:"description"`
	CandidateIDs []string  `json:"candidateIds"`
}

// ReadyToConfirm is a planned entry the bank has now paid.
//
// It is neither matched nor missing: matching would assert that a forecast is a fact,
// and calling it missing would ignore the entry sitting right there. It is the signal
// to confirm — which is how an entry posted ahead of time, by the CRM or by a
// recurrence, learns that the money actually arrived.
type ReadyToConfirm struct {
	LineID        string    `json:"lineId"`
	TransactionID string    `json:"transactionId"`
	BookedDate    time.Time `json:"bookedDate"`
	AmountMinor   int64     `json:"amountMinor"`
	Description   string    `json:"description"`
}

// ReconcileStatementOutput separates the three answers a line can get, because they
// need three different things done about them.
type ReconcileStatementOutput struct {
	Checked   int             `json:"checked"`
	Matched   int             `json:"matched"`
	Missing   []MissingLine   `json:"missing"`
	Ambiguous []AmbiguousLine `json:"ambiguous"`
	// ReadyToConfirm is a fourth answer, not a kind of match: the bank paid something
	// the ledger only forecast.
	ReadyToConfirm []ReadyToConfirm `json:"readyToConfirm"`
	// Pending counts lines the bank has not settled. They are neither matched nor
	// missing: a pending authorisation still changes amount and date when it posts, so
	// matching one asserts a check the next sync invalidates.
	Pending int `json:"pending"`
}

// ReconcileStatementUseCase links what the bank says to what the system recorded, and
// names what it cannot link.
//
// It matches on account, amount and date, and only when exactly one candidate fits.
// That is deliberately narrow: reconciling by hand today found R$ 1.092,97 of charges
// the system never had, and every wrong turn along the way came from resolving an
// ambiguity instead of reporting it.
type ReconcileStatementUseCase struct {
	lines    statement.Repository
	txns     transactionPkg.Repository
	matches  Matches
	accounts bankaccount.Repository
}

func NewReconcileStatementUseCase(
	lines statement.Repository,
	txns transactionPkg.Repository,
	matches Matches,
	accounts bankaccount.Repository,
) *ReconcileStatementUseCase {
	return &ReconcileStatementUseCase{lines: lines, txns: txns, matches: matches, accounts: accounts}
}

func (uc *ReconcileStatementUseCase) Execute(accountID string) (*ReconcileStatementOutput, error) {
	account, err := uc.accounts.FindByID(accountID)
	if errors.Is(err, ErrBankAccountNotFound) {
		return nil, ErrBankAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading account %s: %v", ErrReconcileStorage, accountID, err)
	}
	// Belt and braces: a repository that answers (nil, nil) means the same thing.
	if account == nil {
		// The sentinel, not a bare string: the caller is a cron that decides whether
		// to retry from the status code, and "this account does not exist" and "the
		// database is down" call for opposite decisions.
		return nil, ErrBankAccountNotFound
	}

	// Every line, not only the unmatched ones. A match can stop being true after it
	// was made — the entry behind it reversed, or the bank restated the line — and a
	// line filtered out by its own status is a line nobody ever looks at again.
	lines, err := uc.lines.List(statement.ListFilter{AccountID: accountID})
	if err != nil {
		return nil, fmt.Errorf("%w: listing statement lines: %v", ErrReconcileStorage, err)
	}

	id := accountID
	// IncludeAsDestination, or the payment of a card bill is invisible here: it lives
	// on the CHECKING account and only points at the card. That is the one movement
	// appearing on two statements at once, so it is the one most in need of matching.
	txns, err := uc.txns.List(transactionPkg.ListFilter{
		ProfileID: account.ProfileID, BankAccountID: &id, IncludeAsDestination: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: listing transactions: %v", ErrReconcileStorage, err)
	}

	out := &ReconcileStatementOutput{
		Missing: []MissingLine{}, Ambiguous: []AmbiguousLine{}, ReadyToConfirm: []ReadyToConfirm{},
	}
	// Claimed within this run as well as across runs: two lines of the same value must
	// not both point at one entry, or the second charge silently looks accounted for.
	claimed := map[string]bool{}

	live := map[string]*transactionPkg.Transaction{}
	for _, txn := range txns {
		live[txn.ID] = txn
	}

	for _, line := range lines {
		done, err := uc.stillReconciled(line, live)
		if err != nil {
			return nil, err
		}
		if done {
			continue
		}
		out.Checked++

		if !line.Matchable() {
			out.Pending++
			continue
		}
		// A line worth nothing has no money to account for. Banks print them, and a
		// match must cover a positive amount, so trying to build one fails — and that
		// failure used to abort the run and discard every finding already computed.
		if line.InAccountCurrency() == 0 {
			continue
		}

		candidates, err := uc.candidatesFor(line, account, txns, claimed, transactionPkg.StatusConfirmed)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			// Nothing confirmed fits. A forecast might — and if one does, the answer is
			// not "missing", it is "confirm this".
			planned, err := uc.candidatesFor(line, account, txns, claimed, transactionPkg.StatusPlanned)
			if err != nil {
				return nil, err
			}
			switch {
			case len(planned) == 1:
				out.ReadyToConfirm = append(out.ReadyToConfirm, ReadyToConfirm{
					LineID: line.ID, TransactionID: planned[0].ID, BookedDate: line.BookedDate,
					AmountMinor: line.InAccountCurrency(), Description: line.Description,
				})
				// One forecast settles one charge. Without this, two lines of the same
				// value both get told "just confirm this one", and confirming once
				// makes the other charge vanish from the report — the hole then shows
				// up a day later with nothing pointing at why.
				claimed[planned[0].ID] = true
				continue
			case len(planned) > 1:
				// Several forecasts fit and the bank settled one of them. Reporting
				// this as missing money would be false twice: the entries are right
				// there, and whoever reads it goes hunting a charge that was never
				// missing. Choosing between them is not this service's call.
				out.Ambiguous = append(out.Ambiguous, AmbiguousLine{
					LineID: line.ID, BookedDate: line.BookedDate,
					AmountMinor: line.InAccountCurrency(), Description: line.Description,
					CandidateIDs: idsOf(planned),
				})
				continue
			}
		}
		switch len(candidates) {
		case 1:
			match, err := statement.NewMatch(line.ID, candidates[0].ID, abs64(line.InAccountCurrency()),
				statement.MethodDeterministic, "reconciler")
			if err != nil {
				return nil, err
			}
			if err := uc.matches.Create(match); err != nil {
				// Another run claimed this entry on this account between our check and
				// our write. Its match covers the line, so there is nothing to fix and
				// nothing to report — and reporting missing money would be the one
				// answer a reader would act on.
				if errors.Is(err, statement.ErrAlreadyClaimedOnAccount) {
					claimed[candidates[0].ID] = true
					continue
				}
				return nil, err
			}
			// The line has to say so itself. The match row is the evidence; this is
			// what lets a later run skip the line without re-deriving the whole
			// history, and what anything reading the line alone will believe.
			if err := line.MarkMatched(abs64(line.InAccountCurrency())); err != nil {
				return nil, fmt.Errorf("marking line %s matched: %w", line.ID, err)
			}
			if err := uc.lines.Update(line); err != nil {
				return nil, fmt.Errorf("saving the matched status of line %s: %w", line.ID, err)
			}
			claimed[candidates[0].ID] = true
			out.Matched++
		case 0:
			out.Missing = append(out.Missing, MissingLine{
				LineID: line.ID, BookedDate: line.BookedDate,
				AmountMinor: line.InAccountCurrency(), Description: line.Description,
			})
		default:
			out.Ambiguous = append(out.Ambiguous, AmbiguousLine{
				LineID: line.ID, BookedDate: line.BookedDate,
				AmountMinor: line.InAccountCurrency(), Description: line.Description,
				CandidateIDs: idsOf(candidates),
			})
		}
	}
	return out, nil
}

// stillReconciled answers whether this line is reconciled AND still rightly so — or
// refuses to answer. Not knowing is not the same as "no": treating a database failure
// as "not matched" makes the reconciler match it again.
//
// A match is a claim about two things that can both change after it was made. The
// entry can be reversed, which takes its money back out of the balance; and the bank
// can restate the line. Either way the claim is stale, and a stale claim is worse than
// no claim: the line stops being examined, so the report keeps saying a charge is
// accounted for by something that no longer accounts for anything.
func (uc *ReconcileStatementUseCase) stillReconciled(
	line *statement.Line,
	live map[string]*transactionPkg.Transaction,
) (bool, error) {
	matches, err := uc.matches.ByLine(line.ID)
	if err != nil {
		return false, fmt.Errorf("checking whether line %s is already reconciled: %w", line.ID, err)
	}

	standing := false
	for _, m := range matches {
		if m.UnmatchedAt != nil {
			continue
		}
		reason, err := uc.staleReason(m, line, live)
		if err != nil {
			return false, err
		}
		if reason == "" {
			standing = true
			continue
		}
		if err := uc.matches.Unmatch(m.ID, reason); err != nil {
			return false, fmt.Errorf("%w: releasing match %s on line %s: %v",
				ErrReconcileStorage, m.ID, line.ID, err)
		}
	}

	// The line's own status is a projection of the matches, so bring it back in step
	// either way. It drifts when a match is written and this write is not — and a line
	// stored MATCHED with nothing live behind it is invisible from then on.
	want := statement.StatusUnmatched
	if standing {
		want = statement.StatusMatched
	}
	if line.Status != want && line.Status != statement.StatusIgnored {
		if standing {
			if err := line.MarkMatched(abs64(line.InAccountCurrency())); err != nil {
				return false, fmt.Errorf("marking line %s matched: %w", line.ID, err)
			}
		} else {
			line.MarkUnmatched()
		}
		if err := uc.lines.Update(line); err != nil {
			return false, fmt.Errorf("%w: saving the status of line %s: %v",
				ErrReconcileStorage, line.ID, err)
		}
	}
	return standing, nil
}

func idsOf(txns []*transactionPkg.Transaction) []string {
	ids := make([]string, 0, len(txns))
	for _, t := range txns {
		ids = append(ids, t.ID)
	}
	return ids
}

// staleReason names why a match no longer holds, or returns "" if it still does.
func (uc *ReconcileStatementUseCase) staleReason(
	m *statement.Match,
	line *statement.Line,
	live map[string]*transactionPkg.Transaction,
) (string, error) {
	txn, ok := live[m.TransactionID]
	if !ok {
		// Absent from this account's live entries. That is usually a reversal — the
		// listing excludes reversed rows — but it can also be an entry moved to
		// another account or profile, so ask directly rather than assume. Undoing a
		// match that still stands would report money missing that is not.
		found, err := uc.txns.GetByID(m.TransactionID)
		if err != nil || found == nil {
			return statement.UnmatchTransactionReversed, nil
		}
		txn = found
	}
	if txn.Status != transactionPkg.StatusConfirmed {
		// A reversed or cancelled entry moved no money: balances derive from the
		// confirmed rows, so the charge on this line is unaccounted for again.
		return statement.UnmatchTransactionReversed, nil
	}
	if m.AmountMinor != abs64(line.InAccountCurrency()) {
		// The bank restated what it charged. The match was made against a figure the
		// bank no longer claims.
		return statement.UnmatchBankSideChanged, nil
	}
	return "", nil
}

// candidatesFor returns the entries that could be this line: same account, same value
// in the account's currency, within a couple of days, and not already spoken for.
func (uc *ReconcileStatementUseCase) candidatesFor(
	line *statement.Line,
	account *bankaccount.BankAccount,
	txns []*transactionPkg.Transaction,
	claimed map[string]bool,
	status transactionPkg.Status,
) ([]*transactionPkg.Transaction, error) {
	out := []*transactionPkg.Transaction{}

	for _, txn := range txns {
		if claimed[txn.ID] {
			continue
		}
		if txn.Status != status {
			continue
		}
		// The line side is guarded at construction — a foreign line without its
		// converted figure is refused. The ledger side has no such guard, and an
		// entry booked in another currency compared by face value matches a real
		// charge 5.2 times its size. That mistake was made here by hand once, on
		// five dollar charges, and cost R$ 1.117,03 to unwind.
		if !strings.EqualFold(txn.Currency, account.Currency) {
			continue
		}
		if signedMinorFor(txn, account) != line.InAccountCurrency() {
			continue
		}
		if abs64(int64(daysBetween(txn.OccurredOn, line.BookedDate))) > dateTolerance {
			continue
		}
		taken, err := uc.spokenFor(txn.ID, account.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			continue
		}
		out = append(out, txn)
	}
	return out, nil
}

// spokenFor answers whether another line OF THIS ACCOUNT already claims this entry —
// or refuses to answer.
//
// The unique index covers the PAIR (line, transaction), so it stops one line claiming
// one entry twice and does nothing about two lines claiming the same entry. This check
// is the only thing standing between a database hiccup and a second bank charge that
// silently looks accounted for.
//
// The account scope is not a detail: a transfer between the owner's own accounts is a
// single row that the bank prints on both statements, so a global claim made the two
// sides fight over it.
func (uc *ReconcileStatementUseCase) spokenFor(transactionID, accountID string) (bool, error) {
	taken, err := uc.matches.ClaimedOnAccount(transactionID, accountID)
	if err != nil {
		return false, fmt.Errorf("checking whether entry %s is already claimed on account %s: %w",
			transactionID, accountID, err)
	}
	return taken, nil
}

// signedMinorFor puts an entry in the same convention the statement lines use: money
// leaving is negative, whatever the account. Comparing across conventions would invert
// every card payment — the one movement that appears on two statements at once.
func signedMinorFor(txn *transactionPkg.Transaction, account *bankaccount.BankAccount) int64 {
	value := toMinor(txn.Amount)
	leaving := txn.Type == transactionPkg.TypeExpense ||
		(txn.Type == transactionPkg.TypeTransfer && txn.BankAccountID == account.ID)
	if account.Type == bankaccount.AccountTypeCreditCard {
		// On a card an expense increases the debt, which is money leaving the holder.
		if txn.Type == transactionPkg.TypeIncome ||
			(txn.Type == transactionPkg.TypeTransfer && txn.DestinationAccountID != nil && *txn.DestinationAccountID == account.ID) {
			return value
		}
		return -value
	}
	if leaving {
		return -value
	}
	return value
}

func toMinor(amount float64) int64 {
	if amount < 0 {
		return -int64(-amount*100 + 0.5)
	}
	return int64(amount*100 + 0.5)
}

func daysBetween(a, b time.Time) int {
	diff := a.Sub(b).Hours() / 24
	if diff < 0 {
		diff = -diff
	}
	return int(diff + 0.5)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
