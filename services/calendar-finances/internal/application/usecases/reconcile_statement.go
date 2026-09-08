package usecases

import (
	"errors"
	"fmt"
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

	unmatched := statement.StatusUnmatched
	lines, err := uc.lines.List(statement.ListFilter{AccountID: accountID, Status: &unmatched})
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

	for _, line := range lines {
		done, err := uc.alreadyMatched(line)
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
			if len(planned) == 1 {
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
			ids := make([]string, 0, len(candidates))
			for _, c := range candidates {
				ids = append(ids, c.ID)
			}
			out.Ambiguous = append(out.Ambiguous, AmbiguousLine{
				LineID: line.ID, BookedDate: line.BookedDate,
				AmountMinor: line.InAccountCurrency(), Description: line.Description,
				CandidateIDs: ids,
			})
		}
	}
	return out, nil
}

// alreadyMatched answers whether this line is already reconciled — or refuses to
// answer. Not knowing is not the same as "no": treating a database failure as "not
// matched" makes the reconciler match it again.
func (uc *ReconcileStatementUseCase) alreadyMatched(line *statement.Line) (bool, error) {
	live, err := uc.matches.ByLine(line.ID)
	if err != nil {
		return false, fmt.Errorf("checking whether line %s is already reconciled: %w", line.ID, err)
	}
	for _, m := range live {
		if m.UnmatchedAt == nil {
			return true, nil
		}
	}
	return false, nil
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
