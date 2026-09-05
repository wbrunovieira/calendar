package bankaccount

import "encoding/json"

// Outstanding is everything still owed on a credit card: the open invoice, any closed
// invoice not yet paid, and whatever an installment plan or revolving credit left
// behind. It is what the issuer blocks against the limit.
//
// A card owing money carries a negative balance; a positive one is a credit — a refund
// that landed after the bill was paid — and commits nothing, so it clamps at zero.
func (ba *BankAccount) Outstanding() float64 {
	if !ba.IsCreditCard() || ba.CurrentBalance >= 0 {
		return 0
	}
	return -ba.CurrentBalance
}

// AvailableCredit is how much of the limit is still usable.
func (ba *BankAccount) AvailableCredit() float64 {
	if !ba.IsCreditCard() {
		return 0
	}
	limit := 0.0
	if ba.CreditLimit != nil {
		limit = *ba.CreditLimit
	}
	return limit - ba.Outstanding()
}

// CreditUsagePercent is how much of the limit is committed, as a percentage. It can
// exceed 100 when the card is over its limit, and is 0 when no limit is configured.
func (ba *BankAccount) CreditUsagePercent() float64 {
	if !ba.IsCreditCard() || ba.CreditLimit == nil || *ba.CreditLimit <= 0 {
		return 0
	}
	return (ba.Outstanding() / *ba.CreditLimit) * 100
}

// MarshalJSON ships the derived credit figures alongside the account so every client
// reads the same numbers instead of re-deriving them — the sign convention above is
// domain knowledge, and a client that guesses it wrong reports a card as emptier than
// it is. Only credit cards carry the fields.
func (ba BankAccount) MarshalJSON() ([]byte, error) {
	type plain BankAccount
	payload := struct {
		plain
		Outstanding        *float64 `json:"outstanding,omitempty"`
		AvailableCredit    *float64 `json:"availableCredit,omitempty"`
		CreditUsagePercent *float64 `json:"creditUsagePercent,omitempty"`
	}{plain: plain(ba)}

	if ba.IsCreditCard() {
		outstanding := ba.Outstanding()
		available := ba.AvailableCredit()
		usage := ba.CreditUsagePercent()
		payload.Outstanding = &outstanding
		payload.AvailableCredit = &available
		payload.CreditUsagePercent = &usage
	}

	return json.Marshal(payload)
}
