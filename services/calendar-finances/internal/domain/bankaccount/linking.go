package bankaccount

import (
	"errors"
	"time"
)

// RequiresLinking returns true if this account type should be linked to a source account
func (ba *BankAccount) RequiresLinking() bool {
	return ba.Type == AccountTypeInvestment || ba.Type == AccountTypeCreditCard
}

// IsValidLinkTarget returns true if this account can be used as a link target (source of funds)
func (ba *BankAccount) IsValidLinkTarget() bool {
	return ba.Type == AccountTypeChecking || ba.Type == AccountTypeSavings || ba.Type == AccountTypeCash || ba.Type == AccountTypeExchange || ba.Type == AccountTypeWallet
}

// IsLinked returns true if this account is linked to another account
func (ba *BankAccount) IsLinked() bool {
	return ba.LinkedAccountID != nil && *ba.LinkedAccountID != ""
}

// CanLinkToAccount validates if this account can be linked to the target account
func (ba *BankAccount) CanLinkToAccount(target *BankAccount) bool {
	if target == nil {
		return false
	}

	// Only investment and credit card accounts can link
	if !ba.RequiresLinking() {
		return false
	}

	// Target must be a valid link target (checking, savings, or cash)
	if !target.IsValidLinkTarget() {
		return false
	}

	// Must be from the same profile
	if ba.ProfileID != target.ProfileID {
		return false
	}

	return true
}

// SetLinkedAccount links this account to a source account
func (ba *BankAccount) SetLinkedAccount(target *BankAccount) error {
	if !ba.CanLinkToAccount(target) {
		if target == nil {
			return errors.New("target account cannot be nil")
		}
		if !ba.RequiresLinking() {
			return errors.New("only investment and credit card accounts can be linked")
		}
		if !target.IsValidLinkTarget() {
			return errors.New("target must be a checking, savings, cash, exchange, or wallet account")
		}
		if ba.ProfileID != target.ProfileID {
			return errors.New("accounts must belong to the same profile")
		}
		return errors.New("cannot link to the specified account")
	}

	ba.LinkedAccountID = &target.ID
	ba.UpdatedAt = time.Now()
	return nil
}

// ClearLinkedAccount removes the link to the source account
func (ba *BankAccount) ClearLinkedAccount() {
	ba.LinkedAccountID = nil
	ba.UpdatedAt = time.Now()
}
