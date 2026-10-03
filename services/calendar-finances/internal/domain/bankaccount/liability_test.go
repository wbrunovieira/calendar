package bankaccount

import "testing"

// A liability is what you OWE to someone who is not a card issuer: money a sister
// put in for you, a loan, a client advance to return. The ledger could not express
// one — the only debt it knew was a credit card — so a real R$ 875 owed to Bruno's
// sisters had to live in the notes field of a forecast entry.
func TestLiabilityIsAValidAccountType(t *testing.T) {
	if !isValidAccountType(AccountTypeLiability) {
		t.Fatal("a debt that is not a credit card cannot be recorded at all")
	}
}

// Cash is what you can spend. A liability is the opposite, and summing it in with
// the checking accounts understates what you have — or, worse, reads as if a debt
// were money. I made exactly that mistake by hand two days ago, adding account
// balances to answer "how much is in cash".
func TestIsCashSeparatesWhatCanBeSpentFromWhatIsOwed(t *testing.T) {
	cash := []AccountType{AccountTypeChecking, AccountTypeCash}
	naoCash := []AccountType{AccountTypeLiability, AccountTypeCreditCard,
		AccountTypeInvestment, AccountTypeSavings, AccountTypeExchange, AccountTypeWallet}

	for _, tipo := range cash {
		if !tipo.IsCash() {
			t.Fatalf("%s is spendable and was left out of cash", tipo)
		}
	}
	for _, tipo := range naoCash {
		if tipo.IsCash() {
			t.Fatalf("%s counted as cash", tipo)
		}
	}
}

// A liability behaves like a card in one way only: what you owe is carried as a
// negative balance. It is NOT a card — it has no bill, no closing day, no limit.
func TestLiabilityIsNotACreditCard(t *testing.T) {
	if AccountTypeLiability == AccountTypeCreditCard {
		t.Fatal("unreachable")
	}
	if AccountTypeLiability.IsCash() {
		t.Fatal("a liability is not cash")
	}
}

// Both profiles need it: Bruno owes his sisters, and the company will owe its
// partner for the pro-labore it cannot pay yet.
func TestLiabilityWorksForEitherProfile(t *testing.T) {
	for _, profileID := range []string{"bruno-pessoal", "wb-digital"} {
		acc, err := NewBankAccount(profileID, "Divida com as irmas",
			AccountTypeLiability, 0, "BRL")
		if err != nil {
			t.Fatalf("profile %s: %v", profileID, err)
		}
		if acc.Type != AccountTypeLiability {
			t.Fatalf("profile %s: type = %s", profileID, acc.Type)
		}
	}
}
