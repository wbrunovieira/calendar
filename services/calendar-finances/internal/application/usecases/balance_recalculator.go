package usecases

// BalanceRecalculator recomputes an account's current_balance from its confirmed
// transactions, replacing any stale incremental value.
// BalanceRecalculator is the automatic path: it keeps a derived balance in step after
// a write. Refresh, not Execute — Execute reports without writing, which is what the
// manual endpoint does now, and wiring the automatic path to it would leave every
// balance stale behind a successful response.
type BalanceRecalculator interface {
	Refresh(accountID string) (*RecalculateBalanceResult, error)
}

// recalculateAccounts calls the recalculator for each non-empty account ID and
// RETURNS the first failure.
//
// Most callers still discard it with `_ =`, which is where a stale balance goes
// unnoticed — but the absorbing happens at the call sites, not here. The comment used
// to say otherwise and sent a reviewer looking in the wrong file.
func recalculateAccounts(r BalanceRecalculator, ids ...string) error {
	if r == nil {
		return nil
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := r.Refresh(id); err != nil {
			return err
		}
	}
	return nil
}

// deduplicateIDs returns a slice with unique non-empty IDs, preserving order.
func deduplicateIDs(ids ...string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
