package bankaccount

// ProviderFigure is one account as the BANK reports it, over Open Finance.
//
// It lives in the domain rather than in the Open Finance client so the
// comparison can state what it needs without the application layer depending on
// an infrastructure package — and so what the bank says travels as ONE value
// instead of several parallel maps that can disagree about which account they
// are describing.
type ProviderFigure struct {
	// Amount is what the account is worth: cash for a checking account, debt for
	// a card. Meaningless unless HasAmount is true.
	Amount float64
	// HasAmount separates "the bank says zero" from "the bank could not put a
	// number on it". Collapsing the two reads as "this account owes nothing",
	// which is a lie that reconciles against an empty ledger.
	HasAmount bool
	// Kind is the nature the provider assigns: BANK or CREDIT. Cross-checked
	// against our own type so a card mapped to a checking account is named rather
	// than compared.
	Kind string
	// Name and Institution are carried so a mapping error can be READ instead of
	// inferred from a figure that looks wrong.
	Name        string
	Institution string
}
