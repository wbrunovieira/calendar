package usecases

// BalanceRecalculator recomputes an account's current_balance from its confirmed
// transactions, replacing any stale incremental value.
type BalanceRecalculator interface {
	Execute(accountID string) (*RecalculateBalanceResult, error)
}

// recalculateAccounts calls the recalculator for each non-empty account ID.
// Errors are silently absorbed: the mutation already succeeded, and the
// manual /recalculate-balance route exists as a fallback.
func recalculateAccounts(r BalanceRecalculator, ids ...string) error {
	if r == nil {
		return nil
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := r.Execute(id); err != nil {
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
