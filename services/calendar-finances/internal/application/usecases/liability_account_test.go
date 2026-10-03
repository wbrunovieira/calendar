package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func liabilityFixture(t *testing.T, profileID string) (*CreateTransactionUseCase, *fakeAccountRepo, *fakeInvoiceRepo) {
	t.Helper()
	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		profileID: {ID: profileID, CalendarID: "cal", Name: "perfil", Type: profile.ProfileTypePersonal, IsActive: true},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"conta": {ID: "conta", ProfileID: profileID, Name: "Mercado Pago",
			Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true,
			CurrentBalance: 1000, CreatedAt: now, UpdatedAt: now},
		"divida": {ID: "divida", ProfileID: profileID, Name: "Divida com as irmas",
			Type: bankaccount.AccountTypeLiability, Currency: "BRL", IsActive: true,
			CurrentBalance: 0, CreatedAt: now, UpdatedAt: now},
	}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		"moradia": {ID: "moradia", ProfileID: profileID, Name: "Residencial", Type: category.TypeExpense, IsActive: true},
	}}
	invoiceRepo := &fakeInvoiceRepo{}
	uc := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo,
		&fakeTransactionRepo{}, invoiceRepo, nil, nil)
	return uc, accountRepo, invoiceRepo
}

// Someone else paid a bill for Bruno: the expense happened, and what funded it is a
// debt. Spending against a liability makes the balance MORE negative — he owes more,
// exactly as a card purchase works.
func TestLiability_SpendingAgainstItIncreasesWhatIsOwed(t *testing.T) {
	uc, accounts, _ := liabilityFixture(t, "bruno")
	confirmed := "CONFIRMED"
	cat := "moradia"

	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: "bruno", BankAccountID: "divida", CategoryID: &cat,
		Type: "EXPENSE", Status: &confirmed, Amount: 875, Currency: "BRL",
		Description: "Residencial Mae 09/2026 - pago pelas irmas", OccurredOn: "2026-09-30",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := accounts.accounts["divida"].CurrentBalance; got != -875 {
		t.Fatalf("saldo da divida = %.2f, want -875", got)
	}
	if got := accounts.accounts["conta"].CurrentBalance; got != 1000 {
		t.Fatalf("a conta corrente mudou (%.2f): o dinheiro nao saiu dela", got)
	}
}

// Paying the sisters back moves money out of the account and reduces the debt. Both
// sides have to move, or the debt never clears.
func TestLiability_PayingItBackMovesBothSides(t *testing.T) {
	uc, accounts, _ := liabilityFixture(t, "bruno")
	accounts.accounts["divida"].CurrentBalance = -875
	confirmed := "CONFIRMED"
	dest := "divida"

	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: "bruno", BankAccountID: "conta", DestinationAccountID: &dest,
		Type: "TRANSFER", Status: &confirmed, Amount: 500, Currency: "BRL",
		Description: "Reembolso parcial as irmas", OccurredOn: "2026-10-20",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := accounts.accounts["conta"].CurrentBalance; got != 500 {
		t.Fatalf("conta corrente = %.2f, want 500", got)
	}
	if got := accounts.accounts["divida"].CurrentBalance; got != -375 {
		t.Fatalf("divida = %.2f, want -375 (875 menos os 500 pagos)", got)
	}
}

// A liability has no bill. Creating invoices for one would invent closing dates and
// due days that do not exist, and the repairs built for cards would then try to
// reshape them.
func TestLiability_NeverGetsAnInvoice(t *testing.T) {
	uc, _, invoices := liabilityFixture(t, "bruno")
	confirmed := "CONFIRMED"
	cat := "moradia"

	txn, err := uc.Execute(CreateTransactionInput{
		ProfileID: "bruno", BankAccountID: "divida", CategoryID: &cat,
		Type: "EXPENSE", Status: &confirmed, Amount: 875, Currency: "BRL",
		Description: "Residencial Mae", OccurredOn: "2026-09-30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.InvoiceID != nil {
		t.Fatal("a liability got an invoice: it has no bill, no closing day and no limit")
	}
	if len(invoices.invoices) != 0 {
		t.Fatalf("%d invoices created for an account that has none", len(invoices.invoices))
	}
}

// The company owes too — the pro-labore it cannot pay yet is a liability of the WB,
// not a cost it already settled.
//
// Dated in the FUTURE on purpose: the balance guard only runs for dates that have
// not passed, and it refused this, because a liability starts at zero and every
// first debt would look like spending money that is not there. Going negative is
// the whole point.
func TestLiability_WorksOnTheCompanyProfileToo(t *testing.T) {
	uc, accounts, _ := liabilityFixture(t, "wb-digital")
	confirmed := "CONFIRMED"
	cat := "moradia"

	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: "wb-digital", BankAccountID: "divida", CategoryID: &cat,
		Type: "EXPENSE", Status: &confirmed, Amount: 300, Currency: "BRL",
		Description: "Pro-labore a pagar", OccurredOn: "2026-10-13",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := accounts.accounts["divida"].CurrentBalance; got != -300 {
		t.Fatalf("saldo = %.2f, want -300", got)
	}
}

// A PLANNED entry is a forecast and must not move a debt that has not been incurred.
func TestLiability_PlannedDoesNotMoveTheBalance(t *testing.T) {
	uc, accounts, _ := liabilityFixture(t, "bruno")
	planned := "PLANNED"
	cat := "moradia"

	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: "bruno", BankAccountID: "divida", CategoryID: &cat,
		Type: "EXPENSE", Status: &planned, Amount: 875, Currency: "BRL",
		Description: "Residencial do mes que vem", OccurredOn: "2026-10-30",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := accounts.accounts["divida"].CurrentBalance; got != 0 {
		t.Fatalf("saldo = %.2f: um previsto mexeu na divida", got)
	}
}

var _ = transaction.TypeExpense
