package usecases

import (
	"errors"
	"strings"
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
	owed  map[string]float64
	kinds map[string]string
	err   error
}

func (s *stubBankLister) FiguresByProviderID() (map[string]float64, map[string]string, error) {
	return s.owed, s.kinds, s.err
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
	// A stub that reports no kinds would make the nature cross-check inert in
	// every test that does not care about it — and a defect that only shows up
	// when a fake stays silent is exactly how three bugs reached production here.
	// So the default is the HONEST shape: the checking account is the provider's
	// BANK, the card is its CREDIT. A test that wants a transposition says so.
	if bank.kinds == nil {
		bank.kinds = map[string]string{p1: "BANK", p2: "CREDIT"}
	}
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
	// A card lands in Advisory, not Diverged — the two sides do not yet measure
	// the same set — but the SYSTEM figure must still come from credit-usage,
	// the number the dashboard shows. Reading the stored balance instead would
	// compare something nothing else uses: for this card that column reads
	// -1.059,22.
	if len(report.Advisory) != 1 {
		t.Fatalf("advisory = %+v, want the card", report.Advisory)
	}
	if report.Advisory[0].System != 4237.87 {
		t.Errorf("system = %v, want credit-usage's 4237.87", report.Advisory[0].System)
	}
	if report.Advisory[0].Difference != 11.90 {
		t.Errorf("difference = %v, want 11.90", report.Advisory[0].Difference)
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

// A run that compared nothing is not a clean run. OK is only ever falsified per
// account, so with nothing comparable it reported perfect health — a permanent
// green light over an empty comparison, which is how a check stops being one.
func TestReconcileRefusesToCallAnEmptyComparisonHealthy(t *testing.T) {
	t.Run("no accounts at all", func(t *testing.T) {
		if Reconcile(nil).OK {
			t.Error("OK = true having compared nothing")
		}
	})

	t.Run("accounts, none connected", func(t *testing.T) {
		report := Reconcile([]AccountSnapshot{
			{Name: "Dinheiro (Caixa)", Kind: "CASH", ProviderID: "", System: 2.50},
			{Name: "Wise (EUR)", Kind: "CHECKING", ProviderID: "", System: 0},
		})

		if report.OK {
			t.Error("OK = true with nothing comparable")
		}
		if report.Checked != 0 || report.Total != 2 {
			t.Errorf("checked=%d total=%d, want 0 of 2", report.Checked, report.Total)
		}
	})
}

// "Compared 3 of 13" has to be a fact in the payload, not a claim in a comment.
func TestReconcileReportsHowManyItConsidered(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "a", Kind: "CHECKING", ProviderID: "p1", System: 1, Bank: 1},
		{Name: "b", Kind: "CASH", ProviderID: ""},
		{Name: "c", Kind: "CHECKING", ProviderID: "p3", BankMissing: true},
	})

	if report.Total != 3 || report.Checked != 1 {
		t.Errorf("checked=%d of total=%d, want 1 of 3", report.Checked, report.Total)
	}
}

// A card difference is REPORTED and does not fail the check, because the two
// sides do not yet measure the same set: measured across the three connected
// cards, including PLANNED matches one bank and excluding it matches another.
// Alarming on a number we cannot explain makes a daily check nobody reads.
func TestReconcileReportsACardDifferenceWithoutFailingTheCheck(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", System: 718.36, Bank: 718.36},
		{Name: "Cartão Mercado Pago", Kind: "CREDIT_CARD", ProviderID: "p2", System: 4279.87, Bank: 4237.87, Advisory: true},
	})

	if len(report.Diverged) != 0 {
		t.Errorf("a card difference failed the check: %+v", report.Diverged)
	}
	if len(report.Advisory) != 1 || report.Advisory[0].Difference != 42.00 {
		t.Fatalf("advisory = %+v, want the card with 42.00", report.Advisory)
	}
	if !report.OK {
		t.Error("OK = false; a card difference must not alarm yet")
	}
	if report.Checked != 2 {
		t.Errorf("checked = %d, want 2 — an advisory account IS compared", report.Checked)
	}
}

// A checking account is not advisory: balance against balance is unambiguous,
// and that is where the arithmetic everyone trusts actually lives.
func TestReconcileStillFailsOnACheckingAccount(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", System: 700, Bank: 718.36},
	})

	if len(report.Diverged) != 1 || report.OK {
		t.Errorf("a checking difference did not fail the check: %+v OK=%v", report, report.OK)
	}
}

// One account that cannot be read must not lose the comparison for the others,
// and must not be blamed on the bank.
func TestReconcileKeepsGoingWhenOneAccountCannotBeRead(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "quebrada", Kind: "CREDIT_CARD", ProviderID: "p1", Unreadable: true},
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p2", System: 718.36, Bank: 718.36},
	})

	if len(report.Unreadable) != 1 || report.Unreadable[0] != "quebrada" {
		t.Errorf("unreadable = %v, want the broken account named", report.Unreadable)
	}
	if report.Checked != 1 {
		t.Errorf("checked = %d, want the healthy account still compared", report.Checked)
	}
	if report.OK {
		t.Error("OK = true with an account that was never compared")
	}
}

// The wiring between "the provider did not return this id" and the flag had no
// coverage: a mutant that never set BankMissing survived. It is the single guard
// between an empty provider answer and a silent clean report.
func TestBankReconciliationMarksAnAccountTheProviderDidNotReturn(t *testing.T) {
	uc, _ := reconciliationFixture(&stubBankLister{owed: map[string]float64{
		"prov-cartao": 4237.87, // the checking account's id is absent
	}})

	report, err := uc.Execute()

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(report.Unanswered) != 1 || report.Unanswered[0] != "Mercado Pago" {
		t.Fatalf("unanswered = %v, want the account the provider skipped", report.Unanswered)
	}
	if report.OK {
		t.Error("OK = true though the provider answered nothing for a mapped account")
	}
}

// Every active account is considered, whatever its type. Skipping types outright
// made whole classes vanish — neither compared nor listed — while the report
// claimed to show "3 of 13".
func TestBankReconciliationConsidersEveryAccountType(t *testing.T) {
	uc, repo := reconciliationFixture(&stubBankLister{owed: map[string]float64{
		"prov-conta": 718.36, "prov-cartao": 4237.87,
	}})
	repo.accounts["cripto"] = &bankaccount.BankAccount{
		ID: "cripto", Name: "Binance", Type: bankaccount.AccountTypeExchange,
		CurrentBalance: 1234.00, IsActive: true,
	}

	report, err := uc.Execute()

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if report.Total != 4 {
		t.Errorf("total = %d, want 4 — every active account counted", report.Total)
	}
	found := false
	for _, n := range report.NotConnected {
		if n == "Binance" {
			found = true
		}
	}
	if !found {
		t.Errorf("notConnected = %v, want Binance listed rather than dropped", report.NotConnected)
	}
}

// A card mapped to a checking account compares DEBT against CASH. The figures
// are both real, so the difference is stable and arithmetically correct, and it
// means nothing — it would be investigated as a R$ 4.237 hole forever. Both
// Nubank rows arrive from one connection, so transposing the two ids is a
// one-character mistake.
func TestBankReconciliationRefusesToCompareAcrossNatures(t *testing.T) {
	uc, _ := reconciliationFixture(&stubBankLister{
		owed: map[string]float64{"prov-conta": 4237.87, "prov-cartao": 718.36},
		// transposed on purpose: the ids were swapped when the mapping was filled in
		kinds: map[string]string{"prov-conta": "CREDIT", "prov-cartao": "BANK"},
	})

	report, err := uc.Execute()

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if report.OK {
		t.Error("a transposed mapping reported a clean bill of health")
	}
	if len(report.Mismatched) != 2 {
		t.Fatalf("mismatched = %+v, want both accounts named", report.Mismatched)
	}
	// Named as a MAPPING problem, not as a difference: a divergence sends the
	// investigation after money that never moved.
	if len(report.Diverged) != 0 || len(report.Advisory) != 0 {
		t.Errorf("a mismatch was reported as a money difference: %+v / %+v", report.Diverged, report.Advisory)
	}
	// And not counted as compared, because it was not compared.
	if report.Checked != 0 {
		t.Errorf("checked = %d, want 0", report.Checked)
	}
	byName := map[string]Mismatch{}
	for _, m := range report.Mismatched {
		byName[m.Name] = m
	}
	if got := byName["Cartão Mercado Pago"]; got.Kind != "CREDIT_CARD" || got.BankKind != "BANK" {
		t.Errorf("card mismatch = %+v, want ours CREDIT_CARD against their BANK", got)
	}
	if got := byName["Mercado Pago"]; got.Kind != "CHECKING" || got.BankKind != "CREDIT" {
		t.Errorf("account mismatch = %+v, want ours CHECKING against their CREDIT", got)
	}
}

// A provider that stops sending the field, or sends a kind we have never seen,
// must not turn every account into a mapping error — that would take the whole
// check down over a label.
func TestReconcileAcceptsAnUnknownProviderKind(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "sem tipo", Kind: "CHECKING", ProviderID: "p1", BankKind: "", System: 10, Bank: 10},
		{Name: "tipo novo", Kind: "INVESTMENT", ProviderID: "p2", BankKind: "INVESTMENT", System: 20, Bank: 20},
	})

	if !report.OK || report.Checked != 2 || len(report.Mismatched) != 0 {
		t.Errorf("an unfamiliar kind broke the check: %+v", report)
	}
}

// Our INVESTMENT accounts are mapped to provider BANK accounts (the Clear CCTVM
// row is one), so the test cannot be "same label": it is about NATURE. Only
// CREDIT means debt.
func TestReconcileAcceptsOurInvestmentMappedToTheirBankAccount(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Clear CCTVM", Kind: "INVESTMENT", ProviderID: "p1", BankKind: "BANK", System: 1, Bank: 1},
	})

	if !report.OK || report.Checked != 1 {
		t.Errorf("an investment account mapped to a bank account was rejected: %+v", report)
	}
}

// The one line the daily WhatsApp message shows has to name EVERY kind of
// finding. The first version of the cron grepped names out of the payload
// instead, and that silently reported the advisory cards while dropping the
// unanswered accounts entirely — a message that hides the broken connection it
// exists to report.
func TestSummaryNamesEveryKindOfFinding(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", BankKind: "BANK", System: 718.37, Bank: 718.36},
		{Name: "Nubank", Kind: "CHECKING", ProviderID: "p2", BankKind: "BANK", BankMissing: true},
		{Name: "Cartão MP", Kind: "CREDIT_CARD", ProviderID: "p3", BankKind: "CREDIT", System: 4279.87, Bank: 4237.87, Advisory: true},
		{Name: "Cartão trocado", Kind: "CREDIT_CARD", ProviderID: "p4", BankKind: "BANK", System: 1, Bank: 2},
		{Name: "Clear", Kind: "INVESTMENT", ProviderID: "p5", BankKind: "BANK", Unreadable: true},
		{Name: "Dinheiro (Caixa)", Kind: "CHECKING"},
	})

	for _, want := range []string{
		"2 de 6 conferidas",
		"Mercado Pago (+0.01)",
		"sem resposta do banco: Nubank",
		"mapeamento errado: Cartão trocado",
		"ilegivel: Clear",
		"informativo (cartoes): Cartão MP (+42.00)",
	} {
		if !strings.Contains(report.Summary, want) {
			t.Errorf("summary = %q\n  missing: %q", report.Summary, want)
		}
	}
}

// A run that compared nothing must SAY so in the line a person reads, not only
// in a boolean nobody prints.
func TestSummarySaysWhenNothingWasCompared(t *testing.T) {
	report := Reconcile([]AccountSnapshot{{Name: "Dinheiro (Caixa)", Kind: "CHECKING"}})

	if !strings.Contains(report.Summary, "nada foi comparado") {
		t.Errorf("summary = %q, want it to say nothing was compared", report.Summary)
	}
}

func TestSummaryOnACleanRunIsJustTheCoverage(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", BankKind: "BANK", System: 718.36, Bank: 718.36},
	})

	if report.Summary != "1 de 1 conferidas" {
		t.Errorf("summary = %q", report.Summary)
	}
}

// A wrong mapping has to fail the check ON ITS OWN, next to accounts that agree.
//
// The first test for this had every account transposed, so nothing was compared
// and the zero-coverage guard was what turned OK false — the assertion passed
// while the mismatch branch did nothing. A mutant that deleted `OK = false`
// there survived.
func TestReconcileFailsOnAMismatchEvenWhenEverythingElseAgrees(t *testing.T) {
	report := Reconcile([]AccountSnapshot{
		{Name: "Mercado Pago", Kind: "CHECKING", ProviderID: "p1", BankKind: "BANK", System: 718.36, Bank: 718.36},
		{Name: "Cartão trocado", Kind: "CREDIT_CARD", ProviderID: "p2", BankKind: "BANK", System: 10, Bank: 10},
	})

	if report.Checked != 1 {
		t.Fatalf("checked = %d, want 1 — the healthy account must still be compared", report.Checked)
	}
	if report.OK {
		t.Error("a wrong mapping passed the check because the other account agreed")
	}
}
