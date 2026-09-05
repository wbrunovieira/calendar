package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// makeQuotaAccount builds a position whose CurrentBalance is quotas × price —
// the value the stock and crypto pollers write every couple of hours.
func makeQuotaAccount(id string, accType bankaccount.AccountType, quotas, price float64) *bankaccount.BankAccount {
	account := makeAccount(id, 0, quotas*price)
	account.Type = accType
	quotasCopy, priceCopy := quotas, price
	account.NumberOfQuotas = &quotasCopy
	account.QuotaPrice = &priceCopy
	return account
}

// A position's balance is its market value, not the sum of its transactions.
// Recalculating it from the ledger replaces R$ 5.000 of HGLG11 with the money
// that was paid for it — and two hours later the poller writes the price back,
// so the position silently oscillates.
func TestRecalculateBalance_LeavesAMarketPricedPositionAlone(t *testing.T) {
	for _, accType := range []bankaccount.AccountType{
		bankaccount.AccountTypeInvestment,
		bankaccount.AccountTypeExchange,
		bankaccount.AccountTypeWallet,
	} {
		t.Run(string(accType), func(t *testing.T) {
			const fiiID = "hglg11"
			account := makeQuotaAccount(fiiID, accType, 50, 100) // R$ 5.000 of market value
			accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
				fiiID: account,
			}}
			txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
				confirmedIncome(fiiID, 12.40), // a dividend: real, but not the position's value
			}}

			uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
			result, err := uc.Execute(fiiID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if accountRepo.accounts[fiiID].CurrentBalance != 5000 {
				t.Errorf("market value overwritten: balance = %.2f, want 5000",
					accountRepo.accounts[fiiID].CurrentBalance)
			}
			if accountRepo.updateCalled {
				t.Error("a market-priced position must not be written by the recalculation")
			}
			if result.NewBalance != 5000 {
				t.Errorf("NewBalance = %.2f, want the untouched market value 5000", result.NewBalance)
			}
			if !result.Skipped {
				t.Error("Skipped = false: the caller has to be able to tell nothing was recalculated")
			}
		})
	}
}

// A caixinha or a CDB is an INVESTMENT account too, but it has no quotas: its
// balance really is deposits plus posted yield. Guarding by account type instead
// of by quotas would freeze these accounts forever.
func TestRecalculateBalance_StillRecalculatesAnInvestmentAccountWithoutQuotas(t *testing.T) {
	const caixinhaID = "caixinha-mercado-pago"
	account := makeAccount(caixinhaID, 1000, 999999) // stored value is stale
	account.Type = bankaccount.AccountTypeInvestment

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		caixinhaID: account,
	}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		confirmedIncome(caixinhaID, 1.96), // rendimento do dia
	}}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	result, err := uc.Execute(caixinhaID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NewBalance != 1001.96 {
		t.Errorf("NewBalance = %.2f, want 1001.96", result.NewBalance)
	}
	if result.Skipped {
		t.Error("Skipped = true: an account without quotas is an ordinary ledger")
	}
	if accountRepo.accounts[caixinhaID].CurrentBalance != 1001.96 {
		t.Errorf("balance not persisted: %.2f", accountRepo.accounts[caixinhaID].CurrentBalance)
	}
}

// A sold-out position has zero quotas, so it stops being market-priced and goes
// back to being an ordinary ledger. That is what this PR decides.
//
// What the resulting figure should be is a separate question this test
// deliberately does not answer: SetQuotasFromTotal writes InitialBalance =
// the amount invested while the purchase is usually also recorded as a transfer
// into the account, so recalculating a closed position can conjure the purchase
// twice. That double count is tracked on its own and is not settled here —
// asserting a number now would certify it.
func TestRecalculateBalance_RecalculatesAPositionThatWasSoldOut(t *testing.T) {
	const soldID = "clear-vendida"
	account := makeQuotaAccount(soldID, bankaccount.AccountTypeInvestment, 0, 0)
	// The realistic shape: the position was opened with SetQuotasFromTotal, so
	// InitialBalance carries the amount invested even after the sale.
	account.InitialBalance = 5000

	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		soldID: account,
	}}
	txRepo := &fakeTransactionRepo{created: []*transaction.Transaction{
		confirmedExpense(soldID, 250),
	}}

	uc := NewRecalculateBalanceUseCase(accountRepo, txRepo, nil)
	result, err := uc.Execute(soldID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Skipped {
		t.Error("Skipped = true for a position with no quotas left; it is an ordinary ledger again")
	}
	if !accountRepo.updateCalled {
		t.Error("a position with no quotas left must be written by the recalculation")
	}
}

// A skipped recalculation has to say so in words. An agent reconciling a FII
// position calls this route, sees success and an unchanged balance, and would
// otherwise conclude the balance is confirmed correct.
func TestRecalculateBalance_ExplainsWhyItSkipped(t *testing.T) {
	const fiiID = "hglg11"
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		fiiID: makeQuotaAccount(fiiID, bankaccount.AccountTypeInvestment, 50, 100),
	}}

	uc := NewRecalculateBalanceUseCase(accountRepo, &fakeTransactionRepo{}, nil)
	result, err := uc.Execute(fiiID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Reason == "" {
		t.Error("a skipped recalculation must explain itself; an unexplained no-op reads as a confirmation")
	}
}
