package usecases

import "math"

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

// ReconciliationReport is what the daily check produces.
type ReconciliationReport struct {
	// OK is false when anything needs a person: a disagreement, or a mapped
	// account the bank did not answer for.
	OK      bool `json:"ok"`
	Checked int  `json:"checked"`
	// NotConnected is listed rather than hidden, so "we compared 3 of 13" is
	// visible and nobody mistakes silence for coverage.
	NotConnected []string     `json:"notConnected"`
	Unanswered   []string     `json:"unanswered"`
	Diverged     []Divergence `json:"diverged"`
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
		OK:           true,
		NotConnected: []string{},
		Unanswered:   []string{},
		Diverged:     []Divergence{},
	}

	for _, s := range snapshots {
		if s.ProviderID == "" {
			report.NotConnected = append(report.NotConnected, s.Name)
			continue
		}
		if s.BankMissing {
			report.Unanswered = append(report.Unanswered, s.Name)
			report.OK = false
			continue
		}

		report.Checked++
		diff := round2(s.System - s.Bank)
		if math.Abs(diff) < centTolerance {
			continue
		}
		report.Diverged = append(report.Diverged, Divergence{
			Name:       s.Name,
			Kind:       s.Kind,
			System:     round2(s.System),
			Bank:       round2(s.Bank),
			Difference: diff,
		})
		report.OK = false
	}

	return report
}
