package usecases

import (
	"fmt"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// BankAccountLister is what the bank side has to provide: every account it can
// see, keyed by the provider id stored on our own accounts.
type BankAccountLister interface {
	OwedByProviderID() (map[string]float64, error)
}

// CardUsageReader answers what a card owes on our side. It is the endpoint the
// rest of the system already treats as canonical, so the comparison speaks about
// the same number the dashboard does rather than inventing a third one.
type CardUsageReader interface {
	Execute(accountID string) (*bankaccount.CreditUsage, error)
}

// BankReconciliationUseCase compares the ledger against the bank.
//
// It exists because every hole found in this ledger has been a disagreement with
// the BANK rather than an arithmetic error — on 10/11/2026 the three internal
// figures for one card agreed exactly with each other and all three were R$
// 517,45 away from reality, because two purchases sat in the wrong billing
// cycle. Nothing already in the system could report that: internally it is
// perfectly consistent. /health/invariants checks the ledger against itself.
//
// This corrects nothing. The balance is a result, never a target.
type BankReconciliationUseCase struct {
	accountRepo bankaccount.Repository
	bank        BankAccountLister
	cardUsage   CardUsageReader
}

func NewBankReconciliationUseCase(
	accountRepo bankaccount.Repository,
	bank BankAccountLister,
	cardUsage CardUsageReader,
) *BankReconciliationUseCase {
	return &BankReconciliationUseCase{accountRepo: accountRepo, bank: bank, cardUsage: cardUsage}
}

// Execute gathers both sides and compares them.
func (uc *BankReconciliationUseCase) Execute() (ReconciliationReport, error) {
	accounts, err := uc.accountRepo.FindAll()
	if err != nil {
		return ReconciliationReport{}, fmt.Errorf("listing accounts: %w", err)
	}

	owed, err := uc.bank.OwedByProviderID()
	if err != nil {
		// Reaching the bank failed. Answering "no divergences" here would make an
		// outage indistinguishable from a clean day, which is the failure this
		// whole check exists to stop.
		return ReconciliationReport{}, err
	}

	snapshots := make([]AccountSnapshot, 0, len(accounts))
	for _, a := range accounts {
		if !a.IsActive {
			continue
		}
		if a.Type != bankaccount.AccountTypeChecking && a.Type != bankaccount.AccountTypeCreditCard {
			continue
		}

		s := AccountSnapshot{Name: a.Name, Kind: string(a.Type)}
		if a.ProviderAccountID != nil {
			s.ProviderID = *a.ProviderAccountID
		}

		// The system's side. A card is asked through credit-usage so this speaks
		// about the same number the dashboard shows; a checking account carries
		// its balance directly.
		if a.Type == bankaccount.AccountTypeCreditCard {
			usage, uerr := uc.cardUsage.Execute(a.ID)
			if uerr != nil {
				return ReconciliationReport{}, fmt.Errorf("reading what card %s owes: %w", a.Name, uerr)
			}
			s.System = usage.Outstanding
		} else {
			s.System = a.CurrentBalance
		}

		if s.ProviderID != "" {
			v, found := owed[s.ProviderID]
			if !found {
				s.BankMissing = true
			}
			s.Bank = v
		}
		snapshots = append(snapshots, s)
	}

	return Reconcile(snapshots), nil
}
