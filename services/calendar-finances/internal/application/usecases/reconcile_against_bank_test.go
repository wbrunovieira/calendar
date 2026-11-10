package usecases

import (
	"errors"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// The comparison, with no network in it.
//
// Every hole found in this ledger has been a disagreement with the BANK, not an
// arithmetic error: on 10/11/2026 the three internal figures for one card agreed
// exactly with each other and all three were 517,45 away from reality, because
// two purchases sat in the wrong billing cycle. Internally that is consistent,
// so nothing already in the system could ever report it.
//
// This is the piece that notices. It corrects nothing — the balance is a result,
// never a target — it says which account, by how much, the day after.

func TestReconcileReportsNothingWhenEveryAccountAgrees(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", System: 718.36, Bank: 718.36},
		{Name: "Cartão Mercado Pago", Kind: "CREDIT_CARD", ProviderID: "p2", System: 4237.87, Bank: 4237.87},
	})

	if len(report.Diverged) != 0 {
		t.Errorf("diverged = %+v, want none", report.Diverged)
	}
	if report.Checked != 2 {
		t.Errorf("checked = %d, want 2", report.Checked)
	}
	if !report.OK {
		t.Error("OK = false, want true when everything agrees")
	}
}

// The real case, with its real numbers.
func TestReconcileNamesTheAccountAndTheExactDifference(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Cartão Mercado Pago", Kind: "CREDIT_CARD", ProviderID: "p2", System: 4743.42, Bank: 4225.97},
	})

	if len(report.Diverged) != 1 {
		t.Fatalf("diverged = %+v, want 1", report.Diverged)
	}
	d := report.Diverged[0]
	if d.Name != "Cartão Mercado Pago" {
		t.Errorf("name = %q", d.Name)
	}
	if d.Difference != 517.45 {
		t.Errorf("difference = %.2f, want 517.45", d.Difference)
	}
	if report.OK {
		t.Error("OK = true, want false when an account disagrees with the bank")
	}
}

// The sign carries meaning: the system claiming MORE debt than the bank is a
// different investigation from claiming less.
func TestReconcileKeepsTheDirectionOfTheDifference(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "a mais", Kind: "CHECKING", ProviderID: "p1", System: 100, Bank: 90},
		{Name: "a menos", Kind: "CHECKING", ProviderID: "p2", System: 90, Bank: 100},
	})

	if len(report.Diverged) != 2 {
		t.Fatalf("want both reported, got %+v", report.Diverged)
	}
	if report.Diverged[0].Difference != 10 {
		t.Errorf("difference = %v, want +10 when the system claims more", report.Diverged[0].Difference)
	}
	if report.Diverged[1].Difference != -10 {
		t.Errorf("difference = %v, want -10 when the system claims less", report.Diverged[1].Difference)
	}
}

// An account the bank cannot see is not a disagreement. Cash, Wise and the
// Carrefour card have no Open Finance connection; reporting them as drift would
// make the check cry wolf every single day until it is ignored.
func TestReconcileSkipsAccountsThatAreNotConnected(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Dinheiro (Caixa)", Kind: "CHECKING", ProviderID: "", System: 2.50},
		{Name: "Cartão Carrefour", Kind: "CREDIT_CARD", ProviderID: "", System: 379.93},
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", System: 718.36, Bank: 718.36},
	})

	if len(report.Diverged) != 0 {
		t.Errorf("an unconnected account was reported as drift: %+v", report.Diverged)
	}
	if report.Checked != 1 {
		t.Errorf("checked = %d, want 1 — only the connected account was comparable", report.Checked)
	}
	if len(report.NotConnected) != 2 {
		t.Errorf("notConnected = %+v, want the two unconnected accounts listed", report.NotConnected)
	}
}

// An account mapped to the bank that the bank did not return is NOT agreement.
// Silently treating a missing answer as zero drift is how a broken connection
// looks exactly like a healthy one.
func TestReconcileReportsAMappedAccountTheBankDidNotReturn(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Nubank", Kind: "CHECKING", ProviderID: "p9", System: 0.51, BankMissing: true},
	})

	if len(report.Diverged) != 0 {
		t.Errorf("a missing answer was reported as a money difference: %+v", report.Diverged)
	}
	if len(report.Unanswered) != 1 {
		t.Fatalf("unanswered = %+v, want the account listed", report.Unanswered)
	}
	if report.OK {
		t.Error("OK = true; a connection that answered nothing must not read as healthy")
	}
}

// Money rounds to cents on both sides, so sub-cent noise is not a finding.
func TestReconcileIgnoresSubCentNoise(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", System: 718.364, Bank: 718.36},
	})

	if len(report.Diverged) != 0 {
		t.Errorf("sub-cent noise reported as drift: %+v", report.Diverged)
	}
}

// One cent IS a finding: the holes this exists to catch have started at a cent.
func TestReconcileReportsASingleCent(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", System: 718.37, Bank: 718.36},
	})

	if len(report.Diverged) != 1 {
		t.Fatalf("a one-cent difference was not reported")
	}
	if report.Diverged[0].Difference != 0.01 {
		t.Errorf("difference = %v, want 0.01", report.Diverged[0].Difference)
	}
}

// The gathering side: both numbers come from where the rest of the system gets
// them, so the comparison cannot disagree with the dashboard by construction.
type stubBankLister struct {
	owed map[string]float64
	err  error
}

func (s *stubBankLister) OwedByProviderID() (map[string]float64, error) {
	return s.owed, s.err
}

type stubCardUsage struct{ byID map[string]float64 }

func (s *stubCardUsage) Execute(id string) (*bankaccount.CreditUsage, error) {
	return &bankaccount.CreditUsage{Outstanding: s.byID[id]}, nil
}

func reconciliationFixture(bank *stubBankLister) (*BankReconciliationUseCase, *fakeAccountRepo) {
	p1, p2 := "prov-conta", "prov-cartao"
	accounts := map[string]*bankaccount.BankAccount{
		"conta": {
			ID: "conta", Name: "Mercado Pago", Type: bankaccount.AccountTypeChecking,
			CurrentBalance: 718.36, IsActive: true, ProviderAccountID: &p1,
		},
		"cartao": {
			ID: "cartao", Name: "Cartão Mercado Pago", Type: bankaccount.AccountTypeCreditCard,
			IsActive: true, ProviderAccountID: &p2,
		},
		"caixa": {
			ID: "caixa", Name: "Dinheiro (Caixa)", Type: bankaccount.AccountTypeChecking,
			CurrentBalance: 2.50, IsActive: true,
		},
	}
	repo := &fakeAccountRepo{accounts: accounts}
	uc := NewBankReconciliationUseCase(repo, bank, &stubCardUsage{byID: map[string]float64{"cartao": 4237.87}})
	return uc, repo
}

func TestBankReconciliationComparesBothSidesAndSkipsTheUnconnected(t *testing.T) {
	uc, _ := reconciliationFixture(&stubBankLister{owed: map[string]float64{
		"prov-conta": 718.36, "prov-cartao": 4237.87,
	}})

	report, err := uc.Execute()

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !report.OK || len(report.Diverged) != 0 {
		t.Errorf("expected agreement, got %+v", report)
	}
	if report.Checked != 2 {
		t.Errorf("checked = %d, want 2", report.Checked)
	}
	if len(report.NotConnected) != 1 || report.NotConnected[0] != "Dinheiro (Caixa)" {
		t.Errorf("notConnected = %v, want the cash account", report.NotConnected)
	}
}

// A card is asked through credit-usage, the figure the rest of the system calls
// canonical. Reading its stored balance instead would compare a number nothing
// else uses — for the Mercado Pago card that column reads -1.059,22.
func TestBankReconciliationAsksCreditUsageForACard(t *testing.T) {
	uc, _ := reconciliationFixture(&stubBankLister{owed: map[string]float64{
		"prov-conta": 718.36, "prov-cartao": 4225.97,
	}})

	report, err := uc.Execute()

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(report.Diverged) != 1 {
		t.Fatalf("diverged = %+v, want the card", report.Diverged)
	}
	if report.Diverged[0].System != 4237.87 {
		t.Errorf("system = %v, want credit-usage's 4237.87", report.Diverged[0].System)
	}
	if report.Diverged[0].Difference != 11.90 {
		t.Errorf("difference = %v, want 11.90", report.Diverged[0].Difference)
	}
}

// Reaching the bank failed. Answering "no divergences" would make an outage
// indistinguishable from a clean day — the exact failure this check exists for.
func TestBankReconciliationFailsLoudlyWhenTheBankCannotBeReached(t *testing.T) {
	uc, _ := reconciliationFixture(&stubBankLister{err: errors.New("open finance answered 409: not_connected")})

	_, err := uc.Execute()

	if err == nil {
		t.Fatal("an unreachable bank was reported as a clean run")
	}
}
