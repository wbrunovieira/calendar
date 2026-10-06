package usecases

import (
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/invoice"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// An instalment plan is grouped so its parts can be counted. Grouping by the raw
// DESCRIPTION breaks whenever the description embeds the instalment number, which is
// the usual way people write them: "Gomez Studio - parcela 1/5" and "... 2/5" became
// two separate series of five with one part each, so a healthy five-part plan raised
// five alarms. Three such plans produced eleven false alarms between them, and a
// report that cries wolf buries the real ones -- "The Dark Film - loja virtual" is
// genuinely missing its 2/3, and it was one line among the noise.
func TestInstallmentSeriesName(t *testing.T) {
	cases := []struct {
		desc   string
		number int
		total  int
		want   string
	}{
		// The three real shapes in production.
		{"Parcelamento fatura 02/09 - parcela 1/3", 1, 3, "parcelamento fatura 02/09"},
		{"Parcelamento fatura 02/09 - parcela 2/3", 2, 3, "parcelamento fatura 02/09"},
		{"Gomez Studio - parcela 4/5", 4, 5, "gomez studio"},
		{"Recarga Solar - conclusao loja Nuvemshop (2/2)", 2, 2, "recarga solar - conclusao loja nuvemshop"},
		{"The Dark Film - loja virtual (1/3)", 1, 3, "the dark film - loja virtual"},
		{"Parcelamento de fatura 2/3", 2, 3, "parcelamento de fatura"},
		{"Kvn Imersao 2026 Dolar - Parcela 2/2", 2, 2, "kvn imersao 2026 dolar"},

		// A description with no number keeps its own identity.
		{"Arte e Cor - registro dominio (dentro do contrato)", 1, 2, "arte e cor - registro dominio (dentro do contrato)"},

		// Only the row's OWN N/total is stripped, so an unrelated fraction survives.
		// "02/09" above is the proof it matters: stripping any N/M would have eaten
		// the date that names the plan.
		{"Compra 1/2 litro de oleo", 1, 3, "compra 1/2 litro de oleo"},

		// Trailing noise after the fraction is kept, because it distinguishes legs
		// a human wrote on purpose.
		{"Refrigeração Garrido - site (1/2) sinal", 1, 2, "refrigeração garrido - site sinal"},

		// Case and padding do not make two series out of one.
		{"  GOMEZ STUDIO - PARCELA 3/5  ", 3, 5, "gomez studio"},

		// A row that is not part of a plan has nothing to strip. Attempting it anyway
		// would eat a fraction that happens to match the degenerate numbers.
		{"Oleo 0/1 na promocao", 0, 1, "oleo 0/1 na promocao"},
		{"Parcela unica 1/1", 1, 1, "parcela unica 1/1"},
	}
	for _, c := range cases {
		if got := installmentSeriesName(c.desc, c.number, c.total); got != c.want {
			t.Errorf("%q (%d/%d): got %q, want %q", c.desc, c.number, c.total, got, c.want)
		}
	}
}

// The point of the grouping, end to end: a healthy plan whose description carries the
// instalment number raises nothing, and a plan that really is missing a part is
// reported once -- not once per part.
func TestCheckInvariants_AHealthyPlanWithNumberedDescriptionsIsNotReported(t *testing.T) {
	card := invariantCheckingAccount("card-1", 0, 0)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}

	n := func(i int) *int { return &i }
	txs := &invariantTxRepo{
		balances: map[string]float64{"card-1": 0},
		installments: []*transaction.Transaction{
			{ID: "p1", BankAccountID: "card-1", Description: "Parcelamento fatura 02/09 - parcela 1/3",
				Status: transaction.StatusConfirmed, Amount: 284.48,
				InstallmentNumber: n(1), InstallmentTotal: n(3)},
			{ID: "p2", BankAccountID: "card-1", Description: "Parcelamento fatura 02/09 - parcela 2/3",
				Status: transaction.StatusPlanned, Amount: 284.48,
				InstallmentNumber: n(2), InstallmentTotal: n(3)},
			{ID: "p3", BankAccountID: "card-1", Description: "Parcelamento fatura 02/09 - parcela 3/3",
				Status: transaction.StatusPlanned, Amount: 284.48,
				InstallmentNumber: n(3), InstallmentTotal: n(3)},
		},
	}
	uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{}})

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.InstallmentDrifts) != 0 {
		t.Fatalf("drifts = %+v, want none: all three parts are there", result.InstallmentDrifts)
	}
}

func TestCheckInvariants_APlanMissingAPartIsReportedOnce(t *testing.T) {
	card := invariantCheckingAccount("card-1", 0, 0)
	card.Type = bankaccount.AccountTypeCreditCard
	accounts := &invariantAccountRepo{accounts: []*bankaccount.BankAccount{card}}

	n := func(i int) *int { return &i }
	txs := &invariantTxRepo{
		balances: map[string]float64{"card-1": 0},
		installments: []*transaction.Transaction{
			{ID: "d1", BankAccountID: "card-1", Description: "The Dark Film - loja virtual (1/3)",
				Status: transaction.StatusConfirmed, Amount: 1160,
				InstallmentNumber: n(1), InstallmentTotal: n(3)},
			{ID: "d3", BankAccountID: "card-1", Description: "The Dark Film - loja virtual (3/3)",
				Status: transaction.StatusPlanned, Amount: 1150,
				InstallmentNumber: n(3), InstallmentTotal: n(3)},
		},
	}
	uc := NewCheckInvariantsUseCase(accounts, txs, &invariantInvoiceRepo{byAccount: map[string][]*invoice.Invoice{}})

	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.InstallmentDrifts) != 1 {
		t.Fatalf("drifts = %d, want exactly 1", len(result.InstallmentDrifts))
	}
	got := result.InstallmentDrifts[0]
	if len(got.Missing) != 1 || got.Missing[0] != 2 {
		t.Fatalf("missing = %v, want [2]", got.Missing)
	}
	if got.Found != 2 {
		t.Errorf("found = %d, want 2", got.Found)
	}
}
