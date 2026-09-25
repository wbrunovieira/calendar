package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

func cardAccount(id, profileID string) *bankaccount.BankAccount {
	now := time.Now()
	closingDay, dueDay := 9, 14
	return &bankaccount.BankAccount{
		ID: id, ProfileID: profileID, Name: "Cartao MP",
		Type: bankaccount.AccountTypeCreditCard, Currency: "BRL", IsActive: true,
		ClosingDay: &closingDay, DueDay: &dueDay, CreatedAt: now, UpdatedAt: now,
	}
}

// A cross-profile transfer paid with a credit card is stored as an EXPENSE on that
// card — the money is owed to the issuer like any other purchase — so it must land in
// the card's invoice.
//
// It did not: the invoice assignment tested the REQUESTED type (TRANSFER) while the
// row was written with the EFFECTIVE type (EXPENSE), so the charge stayed outside
// every invoice. Two of them, R$ 1.039,07, were missing from the real card's bills.
func TestCreateTransaction_CrossProfileTransferFromCardJoinsTheInvoice(t *testing.T) {
	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		"personal": {ID: "personal", Name: "Bruno", Type: profile.ProfileTypePersonal},
		"company":  {ID: "company", Name: "WB", Type: profile.ProfileTypeBusiness},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": cardAccount("card", "personal"),
		"company-account": {
			ID: "company-account", ProfileID: "company", Name: "Nubank PJ",
			Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true,
			CreatedAt: now, UpdatedAt: now,
		},
	}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		"loan":   {ID: "loan", ProfileID: "personal", Name: "Emprestimos", Type: category.TypeExpense},
		"income": {ID: "income", ProfileID: "company", Name: "Aporte Socio", Type: category.TypeIncome},
	}}
	invoiceRepo := &fakeInvoiceRepo{}

	uc := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, &fakeTransactionRepo{}, invoiceRepo, nil, nil)

	confirmed := "CONFIRMED"
	sourceCat, destCat, destAcc := "loan", "income", "company-account"
	txn, err := uc.Execute(CreateTransactionInput{
		ProfileID:             "personal",
		BankAccountID:         "card",
		DestinationAccountID:  &destAcc,
		CategoryID:            &sourceCat,
		DestinationCategoryID: &destCat,
		Type:                  "TRANSFER",
		Status:                &confirmed,
		Amount:                923.04,
		Currency:              "BRL",
		Description:           "Aporte WB Digital (via cartao MP)",
		OccurredOn:            "2026-07-30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if txn.InvoiceID == nil {
		t.Fatal("the charge was written to the card but joined no invoice — invisible on the bill")
	}
	if len(invoiceRepo.invoices) == 0 {
		t.Fatal("expected the card's invoice for that cycle to exist")
	}
}

// An ordinary purchase on the card must keep joining its invoice, unchanged.
func TestCreateTransaction_CardExpenseStillJoinsTheInvoice(t *testing.T) {
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		"personal": {ID: "personal", Name: "Bruno", Type: profile.ProfileTypePersonal},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": cardAccount("card", "personal"),
	}}

	uc := NewCreateTransactionUseCase(profileRepo, accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{}},
		&fakeTransactionRepo{}, &fakeInvoiceRepo{}, nil, nil)

	confirmed := "CONFIRMED"
	txn, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", Type: "EXPENSE",
		Status: &confirmed, Amount: 200, Currency: "BRL",
		Description: "Supermercado", OccurredOn: "2026-07-30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.InvoiceID == nil {
		t.Fatal("a card purchase must belong to an invoice")
	}
}

// A transfer between two accounts of the SAME profile leaving a card is a bill
// payment, not a purchase: it must not be charged onto the invoice again.
func TestCreateTransaction_SameProfileTransferFromCardJoinsNoInvoice(t *testing.T) {
	now := time.Now()
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		"personal": {ID: "personal", Name: "Bruno", Type: profile.ProfileTypePersonal},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": cardAccount("card", "personal"),
		"checking": {
			ID: "checking", ProfileID: "personal", Name: "Conta",
			Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true,
			CreatedAt: now, UpdatedAt: now,
		},
	}}

	uc := NewCreateTransactionUseCase(profileRepo, accountRepo,
		&fakeCategoryRepo{categories: map[string]*category.Category{}},
		&fakeTransactionRepo{}, &fakeInvoiceRepo{}, nil, nil)

	confirmed := "CONFIRMED"
	dest := "checking"
	txn, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", DestinationAccountID: &dest,
		Type: "TRANSFER", Status: &confirmed, Amount: 100, Currency: "BRL",
		Description: "Ajuste", OccurredOn: "2026-07-30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.InvoiceID != nil {
		t.Fatal("a same-profile transfer out of a card is not a purchase and must not join an invoice")
	}
}

// A credit on a card — an estorno, a refund, a credit the issuer granted — belongs
// INSIDE the bill of its cycle, exactly like a purchase. At the bank it reduces what
// is owed; here it joined no invoice at all, because the assignment tested only for
// EXPENSE.
//
// The cost was silent and permanent: EVERY bill that had a refund stayed inflated by
// the refund. Measured on the real cards, 13 orphan credits worth R$ 5.191,43 — and
// the one that exposed it, a R$ 139,93 credit from 27/02/2026, made the March bill
// read 2.023,46 against the bank's 1.883,53 for six months.
func TestCreateTransaction_CardCreditJoinsTheInvoiceAndReducesIt(t *testing.T) {
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		"personal": {ID: "personal", Name: "Bruno", Type: profile.ProfileTypePersonal},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": cardAccount("card", "personal"),
	}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		"food":   {ID: "food", ProfileID: "personal", Name: "Restaurante", Type: category.TypeExpense},
		"refund": {ID: "refund", ProfileID: "personal", Name: "Reembolso", Type: category.TypeIncome},
	}}
	invoiceRepo := &fakeInvoiceRepo{}
	txRepo := &fakeTransactionRepo{}
	uc := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmed := "CONFIRMED"
	foodCat, refundCat := "food", "refund"

	// Both fall in the same cycle: closing on the 9th, so 20/02 and 27/02 are the
	// bill that closes 09/03.
	purchase, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", CategoryID: &foodCat,
		Type: "EXPENSE", Status: &confirmed, Amount: 199.90, Currency: "BRL",
		Description: "Total Pass", OccurredOn: "2026-02-20",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	credit, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", CategoryID: &refundCat,
		Type: "INCOME", Status: &confirmed, Amount: 139.93, Currency: "BRL",
		Description: "Credito concedido Mercado Pago", OccurredOn: "2026-02-27",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if credit.InvoiceID == nil {
		t.Fatal("the credit joined no invoice: it will never reduce the bill it belongs to")
	}
	if purchase.InvoiceID == nil || *credit.InvoiceID != *purchase.InvoiceID {
		t.Fatalf("credit landed on invoice %v, purchase on %v — same cycle, must be the same bill",
			credit.InvoiceID, purchase.InvoiceID)
	}

	// And the bill has to read as the bank reads it: purchases MINUS credits.
	var total float64
	for _, txn := range txRepo.created {
		if txn.InvoiceID == nil || *txn.InvoiceID != *credit.InvoiceID {
			continue
		}
		if txn.Type == "INCOME" {
			total -= txn.Amount
		} else {
			total += txn.Amount
		}
	}
	if diff := total - 59.97; diff > 0.005 || diff < -0.005 {
		t.Fatalf("bill totals %.2f, want 59.97 (199.90 spent, 139.93 refunded)", total)
	}
}

// A credit dated outside the cycle belongs to the NEXT bill, not this one — the same
// boundary rule a purchase follows. Nothing may be pulled into a bill just because it
// is a credit.
//
// The closing date is EXCLUSIVE: with closing on the 9th, 08/03 is the last day of
// the bill that closes 09/03, and 09/03 itself already belongs to the next one. That
// is what the SQL does (closing_date > date) and what the real card shows — a
// purchase dated 09/03 sits on the April bill on both sides.
func TestCreateTransaction_CardCreditRespectsTheCycleBoundary(t *testing.T) {
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		"personal": {ID: "personal", Name: "Bruno", Type: profile.ProfileTypePersonal},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": cardAccount("card", "personal"),
	}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		"refund": {ID: "refund", ProfileID: "personal", Name: "Reembolso", Type: category.TypeIncome},
	}}
	uc := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo,
		&fakeTransactionRepo{}, &fakeInvoiceRepo{}, nil, nil)

	confirmed := "CONFIRMED"
	refundCat := "refund"
	before, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", CategoryID: &refundCat,
		Type: "INCOME", Status: &confirmed, Amount: 10, Currency: "BRL",
		Description: "estorno no ultimo dia do ciclo", OccurredOn: "2026-03-08",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", CategoryID: &refundCat,
		Type: "INCOME", Status: &confirmed, Amount: 10, Currency: "BRL",
		Description: "estorno ja no ciclo seguinte", OccurredOn: "2026-03-09",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if before.InvoiceID == nil || after.InvoiceID == nil {
		t.Fatal("both credits must join a bill")
	}
	if *before.InvoiceID == *after.InvoiceID {
		t.Fatal("a credit after the closing date joined the bill that had already closed")
	}
}

// A credit split into instalments follows the same rule as any other credit: each
// part belongs to the bill of its own cycle.
//
// The instalment loop carried its own copy of the invoice condition, and fixing only
// the single-transaction path left this one behind — an untested branch that looked
// corrected from the outside. That is how the original defect survived: a rule
// written twice, verified once.
func TestCreateTransaction_CardCreditInInstallmentsJoinsEachBill(t *testing.T) {
	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		"personal": {ID: "personal", Name: "Bruno", Type: profile.ProfileTypePersonal},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		"card": cardAccount("card", "personal"),
	}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		"refund": {ID: "refund", ProfileID: "personal", Name: "Reembolso", Type: category.TypeIncome},
	}}
	txRepo := &fakeTransactionRepo{}
	invoiceRepo := &fakeInvoiceRepo{}
	uc := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, invoiceRepo, nil, nil)

	confirmed := "CONFIRMED"
	refundCat := "refund"
	total := 2
	if _, err := uc.Execute(CreateTransactionInput{
		ProfileID: "personal", BankAccountID: "card", CategoryID: &refundCat,
		Type: "INCOME", Status: &confirmed, Amount: 200, Currency: "BRL",
		Description: "Estorno parcelado", OccurredOn: "2026-02-20",
		InstallmentTotal: &total,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parts := []*transaction.Transaction{}
	for _, txn := range txRepo.created {
		if txn.InstallmentTotal != nil {
			parts = append(parts, txn)
		}
	}
	if len(parts) != 2 {
		t.Fatalf("%d instalments created, want 2", len(parts))
	}
	seen := map[string]bool{}
	for _, part := range parts {
		if part.InvoiceID == nil {
			t.Fatalf("instalment %d joined no bill", *part.InstallmentNumber)
		}
		inv, err := invoiceRepo.FindByID(*part.InvoiceID)
		if err != nil {
			t.Fatalf("instalment %d points at an invoice that does not exist", *part.InstallmentNumber)
		}
		if !inv.ContainsDate(part.OccurredOn) {
			t.Fatalf("instalment dated %s landed on the bill covering %s..%s",
				part.OccurredOn.Format("2006-01-02"),
				inv.OpeningDate.Format("2006-01-02"), inv.ClosingDate.Format("2006-01-02"))
		}
		seen[*part.InvoiceID] = true
	}
	if len(seen) != 2 {
		t.Fatal("both instalments landed on the same bill; they are a month apart")
	}
}
