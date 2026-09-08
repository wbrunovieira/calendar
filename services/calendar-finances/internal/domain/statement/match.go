package statement

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Method is how a match was established, ordered from strongest to weakest. It is
// recorded because it changes how much a match is worth on review: one made from the
// provider's own id is evidence, one made by fuzzy scoring is a suggestion somebody
// accepted.
type Method string

const (
	// MethodExternalID is the provider's transaction id. Deterministic, one-to-one,
	// and the only one that needs no judgement.
	MethodExternalID Method = "EXTERNAL_ID"
	// MethodEndToEnd is the Pix network identifier, which appears on BOTH sides of an
	// internal transfer. Not available from the current provider — see the fixture
	// test — but the strongest key for transfers when it is.
	MethodEndToEnd Method = "END_TO_END"
	// MethodDeterministic is account + booked date + exact amount + normalised
	// description. Exact in cents; no tolerance on value, because a tolerance on value
	// is how a real difference becomes "acceptable rounding" and disappears.
	MethodDeterministic Method = "DETERMINISTIC"
	// MethodFuzzy scores a candidate. It never matches on its own — it proposes, and
	// a person or an explicit rule accepts.
	MethodFuzzy Method = "FUZZY"
	// MethodManual is a human saying these two are the same movement.
	MethodManual Method = "MANUAL"
)

// Reasons a match was undone. Whoever reconciles next has to tell "a person changed
// their mind" from "the entry behind it stopped counting".
const (
	UnmatchManual              = "MANUAL"
	UnmatchTransactionReversed = "TRANSACTION_REVERSED"
	UnmatchBankSideChanged     = "BANK_SIDE_CHANGED"
	// UnmatchLedgerSideChanged covers the entry moving out from under the match
	// without being reversed: corrected to another account, another amount, another
	// date. The match was a claim about all three, so all three have to be re-asked.
	UnmatchLedgerSideChanged = "LEDGER_SIDE_CHANGED"
	UnmatchWrongMatch        = "WRONG_MATCH"
)

// ErrAlreadyUndone means the match was undone before. The first reason stands.
var ErrAlreadyUndone = errors.New("match is already undone")

// Match links one statement line to one ledger entry, for a given amount.
//
// N:N on purpose: an invoice payment covers many purchases, a Pix can settle two
// bills, a split spreads one line across several entries. AmountMinor is what THIS
// match covers, so partial coverage is expressible and a line is only fully
// reconciled when its matches add up to it.
//
// Matches are never deleted, only undone. A reconciliation that destroys its own
// history cannot answer why a line went back to pending.
type Match struct {
	ID              string     `json:"id"`
	LineID          string     `json:"lineId"`
	TransactionID   string     `json:"transactionId"`
	AmountMinor     int64      `json:"amountMinor"`
	Method          Method     `json:"method"`
	Score           *float64   `json:"score,omitempty"`
	MatchedBy       string     `json:"matchedBy"`
	MatchedAt       time.Time  `json:"matchedAt"`
	UnmatchedAt     *time.Time `json:"unmatchedAt,omitempty"`
	UnmatchedReason *string    `json:"unmatchedReason,omitempty"`
}

// ErrAlreadyClaimedOnAccount is what the database says when another run claimed this
// entry on this account first. It is a race, not corruption: the other run's match
// stands and covers the line, so the loser has nothing to report and nothing to fix.
var ErrAlreadyClaimedOnAccount = errors.New("this entry is already claimed on this account")

// ErrLineAlreadyMatched is the OTHER race, and it means the opposite thing. Here the
// winner matched this same line to this same entry, so the line is covered and there
// is nothing left to do. Collapsing the two into one answer loses the difference
// between "somebody else did your work" and "the entry you wanted is gone".
var ErrLineAlreadyMatched = errors.New("this line is already matched to this entry")

func NewMatch(lineID, transactionID string, amountMinor int64, method Method, by string) (*Match, error) {
	if strings.TrimSpace(lineID) == "" {
		return nil, errors.New("a match needs the statement line")
	}
	if strings.TrimSpace(transactionID) == "" {
		return nil, errors.New("a match needs the transaction")
	}
	// Magnitude only. After sign normalisation a negative match would SUBTRACT in
	// CoveredMinor, so a line would never close — or would close with two wrong
	// matches cancelling each other out.
	if amountMinor <= 0 {
		return nil, errors.New("a match covers a positive amount: the direction belongs to the line, not to the match")
	}
	if strings.TrimSpace(by) == "" {
		return nil, errors.New("a match must record who made it")
	}
	return &Match{
		ID:            uuid.New().String(),
		LineID:        lineID,
		TransactionID: transactionID,
		AmountMinor:   amountMinor,
		Method:        method,
		MatchedBy:     by,
		MatchedAt:     time.Now(),
	}, nil
}

// Unmatch undoes a match without destroying it. The reason is required: seeing a line
// return to pending, whoever reconciles next has to know whether a person changed
// their mind or the entry behind it was reversed.
func (m *Match) Unmatch(reason string, at time.Time) error {
	// Undoing twice would overwrite the first reason with the second, and the first
	// is the one that explains what happened. Same double-undo bug closed on the
	// ledger three rounds ago, reincarnated in a new aggregate.
	if m.IsUndone() {
		return ErrAlreadyUndone
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("undoing a match requires a reason")
	}
	m.UnmatchedAt = &at
	m.UnmatchedReason = &reason
	return nil
}

func (m *Match) IsUndone() bool { return m.UnmatchedAt != nil }

// CoveredMinor is how much of a line its live matches account for. Undone matches are
// excluded: counting them would leave a line reading reconciled against nothing.
func CoveredMinor(matches []*Match) int64 {
	var total int64
	for _, m := range matches {
		if m == nil || m.IsUndone() {
			continue
		}
		total += m.AmountMinor
	}
	return total
}
