package usecases

import (
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
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
