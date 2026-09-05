package bankaccount

import "math"

// CreditUsage is how much of a card's limit is committed, and how much is left.
type CreditUsage struct {
	Outstanding  float64 `json:"outstanding"`
	Available    float64 `json:"availableCredit"`
	UsagePercent float64 `json:"creditUsagePercent"`
}

// CreditUsageFor answers what a card's limit looks like given everything still owed
// on it.
//
// The outstanding figure is supplied by the caller rather than read off
// CurrentBalance, because this system does not move a card's balance as purchases are
// created — every write path skips credit cards, and the balance only changes when an
// invoice is paid. Deriving availability from it would report a card as emptier than
// it is, which is the same mistake as computing it from the open invoice alone.
//
// A negative outstanding is a credit — a refund that landed after the bill was paid —
// and commits nothing, so it clamps at zero.
func (ba *BankAccount) CreditUsageFor(outstanding float64) CreditUsage {
	if !ba.IsCreditCard() {
		return CreditUsage{}
	}

	limit := 0.0
	if ba.CreditLimit != nil {
		limit = *ba.CreditLimit
	}
	owed := math.Max(0, outstanding)

	usage := CreditUsage{
		Outstanding: round2cents(owed),
		Available:   round2cents(limit - owed),
	}
	if limit > 0 {
		usage.UsagePercent = round2cents((owed / limit) * 100)
	}
	return usage
}

func round2cents(v float64) float64 { return math.Round(v*100) / 100 }
