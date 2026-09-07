package persistence

// BalanceAdjustmentLog records every time a stored balance was corrected: what it
// was, what it became, why, and by whom.
//
// Without it a recalculation is the erasure of proof. On 06/09/2026 one "saved"
// R$ 1.522,75 of drift and, in saving it, destroyed the fact that a confirmation bug
// existed, when it started, and what it was worth.
type BalanceAdjustmentLog struct {
	db Querier
}

func NewBalanceAdjustmentLog(db Querier) *BalanceAdjustmentLog {
	return &BalanceAdjustmentLog{db: db}
}

func (l *BalanceAdjustmentLog) Record(accountID string, before, after float64, reason, by string) error {
	_, err := l.db.Exec(`
		INSERT INTO finance.balance_adjustments
			(account_id, balance_before, balance_after, delta, reason, adjusted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, accountID, before, after, round2(after-before), reason, by)
	return err
}
