package usecases

import (
	"fmt"
	"log"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// BankAccountLister is what the bank side has to provide: every account it can
// see, keyed by the provider id stored on our own accounts.
type BankAccountLister interface {
	// FiguresByProviderID returns what each account is worth AND what the
	// provider says each account is (BANK or CREDIT), in one call.
	FiguresByProviderID() (amounts map[string]float64, kinds map[string]string, err error)
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

	owed, bankKinds, err := uc.bank.FiguresByProviderID()
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
		// Every active account is CONSIDERED, whatever its type. Skipping a type
		// outright made whole classes of account vanish — neither compared nor
		// listed as unconnected — while the report claimed to show "3 of 13".
		// Anything the bank cannot see falls out as not connected, by its own
		// missing provider id, which is the honest reason.

		s := AccountSnapshot{Name: a.Name, Kind: string(a.Type)}
		if a.ProviderAccountID != nil {
			s.ProviderID = *a.ProviderAccountID
		}

		// The system's side. A card is asked through credit-usage so this speaks
		// about the same number the dashboard shows; a checking account carries
		// its balance directly.
		// A card's two sides do not yet measure the same set — see AccountSnapshot
		// .Advisory. Reported, not alarmed on, until the definition is settled.
		s.Advisory = a.Type == bankaccount.AccountTypeCreditCard

		if a.Type == bankaccount.AccountTypeCreditCard {
			usage, uerr := uc.cardUsage.Execute(a.ID)
			if uerr != nil {
				// One unreadable card used to abort the whole run and come back as
				// "could not reach the bank" — losing every healthy account and
				// blaming a bank that was never contacted for it.
				s.Unreadable = true
				log.Printf("reconciliation: reading what card %s owes failed: %v", a.Name, uerr)
			} else {
				s.System = usage.Outstanding
			}
		} else {
			s.System = a.CurrentBalance
		}

		if s.ProviderID != "" {
			v, found := owed[s.ProviderID]
			if !found {
				s.BankMissing = true
			}
			s.Bank = v
			s.BankKind = bankKinds[s.ProviderID]
		}
		snapshots = append(snapshots, s)
	}

	return Reconcile(snapshots), nil
}
