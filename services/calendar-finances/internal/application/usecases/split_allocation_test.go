package usecases

import (
	"math"
	"math/rand"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func makeSplits(amounts ...float64) []*transaction.Split {
	out := make([]*transaction.Split, 0, len(amounts))
	for _, amount := range amounts {
		out = append(out, &transaction.Split{Amount: amount})
	}
	return out
}

func sumSplits(splits []*transaction.Split) float64 {
	var total float64
	for _, split := range splits {
		total += split.Amount
	}
	return round2(total)
}

// installmentAmounts mirrors how createInstallments divides a purchase: equal
// parts, with the rounding remainder on the first one.
func installmentAmounts(total float64, count int) []float64 {
	each := math.Floor(total/float64(count)*100) / 100
	remainder := round2(total - each*float64(count))
	out := make([]float64, count)
	for i := range out {
		out[i] = each
		if i == 0 {
			out[i] = round2(each + remainder)
		}
	}
	return out
}

// The rule the domain enforces, verbatim from transaction.setSplits. Anything
// this rejects is a 500 on the API and a purchase the user cannot record.
func exceedsTransaction(splits []*transaction.Split, amount float64) bool {
	return sumSplits(splits) > amount+0.01
}

func TestSplitsForInstallment_NeverExceedsTheInstallment(t *testing.T) {
	cases := []struct {
		name    string
		total   float64
		count   int
		amounts []float64
	}{
		{"exact thirds", 300, 3, []float64{200, 100}},
		{"indivisible total", 100, 3, []float64{70, 30}},
		{"prime-ish total", 1003.92, 2, []float64{134.03, 419.66, 450.23}},
		{"splits one centavo over the purchase", 1003.92, 2, []float64{134.03, 419.66, 450.24}},
		{"partial coverage", 500, 4, []float64{120.55}},
		{"five cents in three", 0.05, 3, []float64{0.03, 0.02}},
		{"one real in twelve, five ways", 1, 12, []float64{0.2, 0.2, 0.2, 0.2, 0.2}},
		{"long installment plan", 4999.99, 24, []float64{2500.01, 1499.98, 1000}},
		{"single split covering everything", 89.9, 7, []float64{89.9}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			splits := makeSplits(tc.amounts...)
			for i, installment := range installmentAmounts(tc.total, tc.count) {
				got := splitsForInstallment(splits, installment, tc.total)
				if exceedsTransaction(got, installment) {
					t.Errorf("installment %d of %.2f: splits sum to %.2f, which the domain rejects",
						i+1, installment, sumSplits(got))
				}
			}
		})
	}
}

// A sweep, because the failure mode is arithmetic and examples chosen by hand
// are exactly the ones that divide nicely. A single rejection here is a 500 the
// user would hit while recording a real purchase.
func TestSplitsForInstallment_SweepNeverProducesARejectedBreakdown(t *testing.T) {
	random := rand.New(rand.NewSource(20260905))

	for iteration := 0; iteration < 200000; iteration++ {
		total := round2(1 + random.Float64()*4999)
		count := 2 + random.Intn(23)
		splitCount := 1 + random.Intn(5)

		// Split the purchase into parts that together cover it, give or take
		// the centavo the domain tolerates.
		remaining := total
		amounts := make([]float64, 0, splitCount)
		for i := 0; i < splitCount-1; i++ {
			part := round2(remaining * random.Float64())
			if part <= 0 {
				part = 0.01
			}
			amounts = append(amounts, part)
			remaining = round2(remaining - part)
			if remaining <= 0 {
				break
			}
		}
		if remaining > 0 {
			amounts = append(amounts, remaining)
		}
		if random.Intn(10) == 0 && len(amounts) > 0 {
			amounts[0] = round2(amounts[0] + 0.01) // the centavo the domain allows
		}

		splits := makeSplits(amounts...)
		for i, installment := range installmentAmounts(total, count) {
			got := splitsForInstallment(splits, installment, total)
			if exceedsTransaction(got, installment) {
				t.Fatalf("total %.2f in %dx, splits %v: installment %d of %.2f got splits summing %.2f",
					total, count, amounts, i+1, installment, sumSplits(got))
			}
		}
	}
}

// The whole purchase must still be accounted for across the plan. Cents lost to
// rounding are unavoidable, but they must stay cents.
func TestSplitsForInstallment_ConservesThePurchaseAcrossInstallments(t *testing.T) {
	const total = 1003.92
	amounts := []float64{134.03, 419.66, 450.23}
	splits := makeSplits(amounts...)

	var allocated float64
	for _, installment := range installmentAmounts(total, 7) {
		allocated += sumSplits(splitsForInstallment(splits, installment, total))
	}

	covered := round2(amounts[0] + amounts[1] + amounts[2])
	if diff := math.Abs(round2(allocated - covered)); diff > 0.07 {
		t.Errorf("allocated %.2f across the plan against %.2f covered; lost %.2f", allocated, covered, diff)
	}
}

// Largest remainder means the cents lost to flooring are handed back, so a
// split is only dropped when the installment genuinely has no cent left for it.
func TestSplitsForInstallment_KeepsEverySplitThatCanEarnACentavo(t *testing.T) {
	splits := makeSplits(70, 30)

	got := splitsForInstallment(splits, 33.34, 100)
	if len(got) != 2 {
		t.Fatalf("got %d splits, want both to survive", len(got))
	}
	if got[0].Amount != 23.34 || got[1].Amount != 10.00 {
		t.Errorf("got %.2f/%.2f, want 23.34/10.00 — the lost centavo goes to the larger remainder",
			got[0].Amount, got[1].Amount)
	}
	if sum := sumSplits(got); sum != 33.34 {
		t.Errorf("splits sum to %.2f, want exactly the installment 33.34", sum)
	}
}

// Below one centavo per split there is nothing to hand out, and inventing one
// would push the breakdown past the installment.
func TestSplitsForInstallment_DropsASplitTooSmallToEarnACentavo(t *testing.T) {
	splits := makeSplits(1.00, 0.001)

	got := splitsForInstallment(splits, 0.02, 1.001)
	if sum := sumSplits(got); sum > 0.02+0.01 {
		t.Errorf("splits sum to %.2f, which the domain rejects for a 0.02 installment", sum)
	}
	for _, split := range got {
		if split.Amount <= 0 {
			t.Error("a zero split would be rejected by the domain")
		}
	}
}

func TestSplitsForInstallment_ClonesSoInstallmentsShareNothing(t *testing.T) {
	categoryID := "cat-1"
	memo := "rateio"
	splits := []*transaction.Split{{ID: "original", CategoryID: &categoryID, Amount: 100, Memo: &memo}}

	first := splitsForInstallment(splits, 50, 100)
	second := splitsForInstallment(splits, 50, 100)

	if first[0] == second[0] {
		t.Fatal("two installments got the same *Split")
	}
	if first[0].ID != "" {
		t.Error("a cloned split must have no id, so each installment gets its own")
	}
	if first[0].CategoryID == second[0].CategoryID {
		t.Error("installments share a CategoryID pointer")
	}
	if first[0].Memo == second[0].Memo {
		t.Error("installments share a Memo pointer")
	}
	if *first[0].CategoryID != categoryID || *first[0].Memo != memo {
		t.Error("the clone lost its category or memo")
	}
}
