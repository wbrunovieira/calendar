package usecases

import (
	"fmt"
	"math"
	"strings"
)

// AccountSnapshot is one account as the system sees it, beside what the bank
// says, ready to be compared. Gathering and comparing are separate on purpose:
// the comparison is where the judgement lives, and it should be testable without
// a network.
type AccountSnapshot struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	// ProviderID is the Open Finance account this one is mapped to. Empty means
	// the bank cannot see this account at all — cash, Wise, a card with no
	// connection — and it is excluded from the comparison rather than failing it.
	ProviderID string `json:"providerId"`
	System     float64
	Bank       float64
	// BankMissing marks an account that IS mapped and that the bank did not
	// answer for. Treating that as agreement makes a broken connection look
	// exactly like a healthy one.
	BankMissing bool
	// BankKind is what the PROVIDER says this account is: BANK or CREDIT. It is
	// cross-checked against Kind so a transposed mapping is named instead of
	// compared — see Mismatch.
	BankKind string `json:"bankKind"`
	// BankName and BankInstitution are what the provider CALLS the account this
	// one is mapped to. Reported alongside a difference so a mapping error can be
	// read rather than inferred from a figure that merely looks wrong.
	BankName        string `json:"bankName,omitempty"`
	BankInstitution string `json:"bankInstitution,omitempty"`
	// Unreadable marks an account whose own figure could not be computed. One
	// such account used to abort the whole run and be reported as a bank outage,
	// losing the comparison for every healthy account and sending the
	// investigation to a bank that was never contacted for it.
	Unreadable bool
	// Advisory means a difference here is REPORTED but does not fail the check.
	//
	// It exists for credit cards, where the two sides do not yet measure the same
	// set. Measured on 10/11/2026 across the three connected cards: on Mercado
	// Pago the instalments are PLANNED on our side and already committed at the
	// bank, so including PLANNED matches (4.279,87 vs 4.237,87) and excluding it
	// is wildly off (1.059,22); on Nubank Jurídica the reverse, because its
	// PLANNED rows are subscriptions the bank has not charged (6.336,48 with,
	// 3.162,00 without, bank 3.269,80). No single rule fits both.
	//
	// Failing the check on a difference we cannot yet explain would make it cry
	// wolf daily, and a check nobody reads is worse than none. So cards are shown
	// with both numbers until the definition is settled against real data.
	Advisory bool
}

// Divergence is one account whose figure disagrees with the bank's.
type Divergence struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	// System and Bank are both reported so the reader does not have to ask the
	// endpoint a second question to know which side moved.
	System float64 `json:"system"`
	Bank   float64 `json:"bank"`
	// Difference is system minus bank, sign included: claiming more debt than
	// the bank is a different investigation from claiming less.
	Difference float64 `json:"difference"`
}

// Mismatch is an account pointed at a provider account of the WRONG NATURE:
// cash mapped to a card, or a card mapped to cash.
//
// It is worth its own category because the figures would otherwise be compared
// anyway, and the difference would be arithmetically real, stable and entirely
// meaningless — cash against debt. Both Nubank rows arrive from a single
// connection (a checking account at 0,51 and a card at 365,50), so swapping the
// two ids is a one-character mistake that would then be investigated as a
// R$ 366 hole, forever.
type Mismatch struct {
	Name string `json:"name"`
	// Kind is what WE call it; BankKind is what the provider calls the account it
	// is mapped to. Both are reported because the fix depends on which is wrong.
	Kind     string `json:"kind"`
	BankKind string `json:"bankKind"`
	// BankName and BankInstitution say WHICH provider account it is pointed at,
	// so the fix does not start with an opaque uuid.
	BankName        string `json:"bankName,omitempty"`
	BankInstitution string `json:"bankInstitution,omitempty"`
}

// ReconciliationReport is what the daily check produces.
type ReconciliationReport struct {
	// OK is false when anything needs a person: a disagreement, or a mapped
	// account the bank did not answer for.
	OK      bool `json:"ok"`
	Checked int  `json:"checked"`
	// AdvisoryOnly is how many of Checked were only REPORTED, never verified —
	// the credit cards. Without it, "2 de 3 conferidas" counts as verified a
	// card whose difference can never fail the check, which overstates by
	// exactly the accounts this system has had the most trouble with.
	AdvisoryOnly int `json:"advisoryOnly"`
	// Total is every account considered, so "compared 3 of 13" is a fact in the
	// payload rather than a claim in a comment. Without it, a report that checked
	// nothing looks exactly like one that checked everything and found nothing.
	Total int `json:"total"`
	// NotConnected is listed rather than hidden, so "we compared 3 of 13" is
	// visible and nobody mistakes silence for coverage.
	NotConnected []string     `json:"notConnected"`
	Unanswered   []string     `json:"unanswered"`
	Diverged     []Divergence `json:"diverged"`
	// Advisory carries differences that are reported without failing the check.
	Advisory []Divergence `json:"advisory"`
	// UnmappedAtBank names accounts the BANK reports that no active account of
	// ours claims. It does not fail the check — some are genuinely outside the
	// ledger — but it has to be SAID, because this is the only number that moves
	// when coverage shrinks: dropping a mapping, or deactivating an account,
	// lowers "checked" and "total" together and leaves full coverage showing.
	UnmappedAtBank []string `json:"unmappedAtBank"`
	// Mismatched names accounts mapped to a provider account of the wrong nature.
	// They fail the check and are NOT compared: the comparison would be
	// meaningless, and reporting it as a difference would send the investigation
	// after money rather than after the mapping.
	Mismatched []Mismatch `json:"mismatched"`
	// Summary is ONE line naming what needs a person, built here rather than by
	// each reader.
	//
	// The daily cron has no jq (see scripts/backup.sh), and the first version had
	// it grep every "name" field out of the payload — which quietly reported the
	// advisory cards and silently dropped the unanswered accounts, whose names
	// live in a plain string array. A reader that has to re-derive the verdict
	// will eventually derive a different one.
	Summary string `json:"summary"`
	// Unreadable names accounts whose own figure could not be computed. They fail
	// the check: an account that could not be read was not compared, and a run
	// that quietly drops one is a run that overstates what it verified.
	Unreadable []string `json:"unreadable"`
}

// centTolerance is one cent. Money is rounded to cents on both sides, so
// anything smaller is arithmetic noise — and anything at a cent is not: the
// differences this exists to catch have started there.
const centTolerance = 0.005

// Reconcile compares what the system believes against what the bank reports.
//
// It corrects nothing. The balance is a result, never a target, so the job here
// is to NAME the account and the amount — the day after, while the cause is
// still findable — rather than to quietly make the two numbers match.
func Reconcile(snapshots []AccountSnapshot) ReconciliationReport {
	report := ReconciliationReport{
		OK:             true,
		NotConnected:   []string{},
		UnmappedAtBank: []string{},
		Unreadable:     []string{},
		Unanswered:     []string{},
		Diverged:       []Divergence{},
		Advisory:       []Divergence{},
		Mismatched:     []Mismatch{},
	}

	report.Total = len(snapshots)

	for _, s := range snapshots {
		if s.Unreadable {
			report.Unreadable = append(report.Unreadable, s.Name)
			report.OK = false
			continue
		}
		if s.ProviderID == "" {
			report.NotConnected = append(report.NotConnected, s.Name)
			continue
		}
		if s.BankMissing {
			report.Unanswered = append(report.Unanswered, s.Name)
			report.OK = false
			continue
		}
		if !kindsAgree(s.Kind, s.BankKind) {
			report.Mismatched = append(report.Mismatched, Mismatch{
				Name: s.Name, Kind: s.Kind, BankKind: s.BankKind,
				BankName: s.BankName, BankInstitution: s.BankInstitution,
			})
			report.OK = false
			continue
		}

		report.Checked++
		if s.Advisory {
			report.AdvisoryOnly++
		}
		diff := round2(s.System - s.Bank)
		if math.Abs(diff) < centTolerance {
			continue
		}
		d := Divergence{
			Name:       s.Name,
			Kind:       s.Kind,
			System:     round2(s.System),
			Bank:       round2(s.Bank),
			Difference: diff,
		}
		if s.Advisory {
			report.Advisory = append(report.Advisory, d)
			continue
		}
		report.Diverged = append(report.Diverged, d)
		report.OK = false
	}

	// A run that compared NOTHING is not a clean run. OK starts true and is only
	// ever falsified per account, so with no comparable account it would report
	// perfect health — a permanent green light over an empty comparison, which is
	// how this check would quietly stop being a check.
	if report.Checked == 0 {
		report.OK = false
	}

	report.Summary = summarize(report)
	return report
}

// kindsAgree reports whether our account and the provider account it is mapped
// to are of the same nature.
//
// The test is deliberately one-sided: a credit card must map to the provider's
// CREDIT, and anything else must NOT. Only CREDIT is asserted positively because
// it is the only provider kind whose figure means DEBT; a connector that starts
// labelling cash accounts something new (an investment, a prepaid wallet) is
// still cash and must not be failed for a label we have not seen before.
//
// An unknown (empty) provider kind agrees with everything: a provider that stops
// reporting the field must not turn every account into a mapping error.
func kindsAgree(ours, theirs string) bool {
	if theirs == "" {
		return true
	}
	isCardHere := ours == "CREDIT_CARD"
	isCardThere := theirs == "CREDIT"
	return isCardHere == isCardThere
}

// summarize states what the report found, in one line, in the language the
// person reading the WhatsApp message speaks.
func summarize(r ReconciliationReport) string {
	coverage := fmt.Sprintf("%d de %d conferidas", r.Checked, r.Total)
	if r.AdvisoryOnly > 0 {
		// A card counted in `checked` was reported, not verified — saying so in
		// the same breath keeps the coverage number from overstating itself.
		coverage += fmt.Sprintf(" (%d cartao(oes) apenas reportado(s))", r.AdvisoryOnly)
	}
	parts := []string{coverage}
	if len(r.Diverged) > 0 {
		parts = append(parts, "divergencia: "+namesOf(r.Diverged))
	}
	if len(r.Mismatched) > 0 {
		names := make([]string, 0, len(r.Mismatched))
		for _, m := range r.Mismatched {
			names = append(names, m.Name)
		}
		parts = append(parts, "mapeamento errado: "+strings.Join(names, ", "))
	}
	if len(r.Unanswered) > 0 {
		parts = append(parts, "sem resposta do banco: "+strings.Join(r.Unanswered, ", "))
	}
	if len(r.Unreadable) > 0 {
		parts = append(parts, "ilegivel: "+strings.Join(r.Unreadable, ", "))
	}
	if len(r.Advisory) > 0 {
		// Flagged as informational in the text too: a card difference is not yet
		// a defect, and a message that reads like one trains the reader to ignore
		// the whole line.
		parts = append(parts, "informativo (cartoes): "+namesOf(r.Advisory))
	}
	if len(r.UnmappedAtBank) > 0 {
		parts = append(parts, fmt.Sprintf("no banco sem conta no sistema: %s", strings.Join(r.UnmappedAtBank, ", ")))
	}
	if r.Checked == 0 {
		parts = append(parts, "nada foi comparado")
	}
	return strings.Join(parts, "; ")
}

func namesOf(ds []Divergence) string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, fmt.Sprintf("%s (%+.2f)", d.Name, d.Difference))
	}
	return strings.Join(out, ", ")
}
