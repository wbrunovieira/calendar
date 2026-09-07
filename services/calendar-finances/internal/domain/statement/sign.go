package statement

// AccountKind is what the line came from, because the provider's sign convention
// depends on it.
type AccountKind string

const (
	AccountKindChecking AccountKind = "CHECKING"
	AccountKindCard     AccountKind = "CARD"
)

// NormalizeSign converts a provider amount into one canonical convention: the effect
// on the HOLDER's net worth. Money leaving is negative, money arriving is positive,
// whatever the account.
//
// This exists because the same field means opposite things per account, confirmed
// against real payloads: on the card a DEBIT purchase arrives POSITIVE (40.00
// Registrobr) and a CREDIT payment arrives NEGATIVE (-1018.18), because a card
// statement speaks in terms of DEBT. On the checking account a DEBIT arrives NEGATIVE
// (-119.94), speaking in terms of BALANCE.
//
// A matcher comparing signed values across both would invert every card-payment
// reconciliation — and a card payment is exactly the movement that appears on both
// sides at once, so it is the one that most needs to match.
//
// The provider's original value is untouched in Raw: this normalises what is compared,
// it does not rewrite what the bank said.
func NormalizeSign(amountMinor int64, kind AccountKind, providerType string) int64 {
	if kind != AccountKindCard {
		return amountMinor
	}

	// On a card the sign is unreliable — purchases usually arrive unsigned — so the
	// direction comes from the type, and the magnitude from the amount.
	magnitude := amountMinor
	if magnitude < 0 {
		magnitude = -magnitude
	}
	if providerType == "CREDIT" {
		// A credit on a card pays the debt down: money in, from the holder's side.
		return magnitude
	}
	return -magnitude
}
