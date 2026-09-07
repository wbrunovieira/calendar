package statement

import (
	"testing"
	"time"
)

// A match is N:N, not 1:1. One invoice payment covers many card purchases; one Pix can
// settle two bills; a split spreads one line across several entries. Building the
// matcher on a single matched_transaction_id means migrating mid-way.
//
// Matches are also never deleted. A match undone must say why, for the same reason a
// dropped match needs a discoverable cause: someone reconciling sees a line go back to
// pending and has to know whether a human unmatched it or an entry was reversed.

func TestNewMatch_RequiresBothSidesAndAnAmount(t *testing.T) {
	for _, tc := range []struct {
		name        string
		line, txn   string
		amountMinor int64
	}{
		{"no line", "", "tx-1", 100},
		{"no transaction", "line-1", "", 100},
		{"no amount", "line-1", "tx-1", 0},
	} {
		if _, err := NewMatch(tc.line, tc.txn, tc.amountMinor, MethodExternalID, "teste"); err == nil {
			t.Errorf("%s: must be refused", tc.name)
		}
	}
}

func TestNewMatch_RecordsHowAndByWhom(t *testing.T) {
	// The method matters when reviewing: a match made by the provider's own id is
	// evidence, one made by fuzzy scoring is a suggestion someone accepted.
	m, err := NewMatch("line-1", "tx-1", 5558, MethodFuzzy, "agente")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Method != MethodFuzzy || m.MatchedBy != "agente" {
		t.Error("how and by whom must both survive")
	}
	if m.UnmatchedAt != nil {
		t.Error("a fresh match is not undone")
	}
}

func TestMatch_UnmatchKeepsTheRowAndTheReason(t *testing.T) {
	m, _ := NewMatch("line-1", "tx-1", 5558, MethodExternalID, "teste")

	if err := m.Unmatch("", time.Now()); err == nil {
		t.Error("undoing a match without a reason must be refused: whoever reconciles next has to know whether a human did it or an entry was reversed")
	}
	if err := m.Unmatch(UnmatchTransactionReversed, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.UnmatchedAt == nil || m.UnmatchedReason == nil {
		t.Error("both when and why must be recorded")
	}
	if !m.IsUndone() {
		t.Error("the match must read as undone")
	}
}

func TestMatch_PartialAmountsAddUp(t *testing.T) {
	// One payment covering two purchases: neither match carries the whole line.
	a, _ := NewMatch("line-pay", "tx-1", 4000, MethodDeterministic, "teste")
	b, _ := NewMatch("line-pay", "tx-2", 1558, MethodDeterministic, "teste")

	if total := a.AmountMinor + b.AmountMinor; total != 5558 {
		t.Errorf("total matched = %d, want 5558", total)
	}
}

func TestMatch_UndoneMatchDoesNotCountTowardsCoverage(t *testing.T) {
	a, _ := NewMatch("line-1", "tx-1", 5558, MethodExternalID, "teste")
	_ = a.Unmatch(UnmatchTransactionReversed, time.Now())

	if CoveredMinor([]*Match{a}) != 0 {
		t.Error("an undone match must not count as coverage, or a line reads reconciled against nothing")
	}
}

func TestMatch_CoverageSumsOnlyLiveMatches(t *testing.T) {
	a, _ := NewMatch("line-1", "tx-1", 4000, MethodDeterministic, "teste")
	b, _ := NewMatch("line-1", "tx-2", 1558, MethodDeterministic, "teste")
	c, _ := NewMatch("line-1", "tx-3", 999, MethodFuzzy, "teste")
	_ = c.Unmatch(UnmatchManual, time.Now())

	if got := CoveredMinor([]*Match{a, b, c}); got != 5558 {
		t.Errorf("covered = %d, want 5558", got)
	}
}
