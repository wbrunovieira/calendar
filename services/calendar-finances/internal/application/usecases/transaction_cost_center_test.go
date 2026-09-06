package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/costcenter"
	"github.com/brunovieira/calendar-finances/internal/domain/profile"
)

// Embedding the interface keeps the fake honest: a call it does not implement
// panics instead of returning a silent zero.
type fakeCostCenterRepo struct {
	costcenter.Repository
	centers map[string]*costcenter.CostCenter
}

func (f *fakeCostCenterRepo) FindByID(id string) (*costcenter.CostCenter, error) {
	if c, ok := f.centers[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

type costCenterFixture struct {
	useCase   *CreateTransactionUseCase
	txRepo    *fakeTransactionRepo
	profileID string
	accountID string
	catID     string
	clientID  string
	otherID   string
}

func newCostCenterFixture(t *testing.T) *costCenterFixture {
	t.Helper()
	const (
		wbProfile     = "profile-wb"
		otherProfile  = "profile-pessoal"
		accountID     = "account-wb"
		categoryID    = "cat-receita"
		clientCenter  = "cc-cliente-acme"
		foreignCenter = "cc-de-outro-perfil"
	)
	now := time.Now()

	profileRepo := &fakeProfileRepo{profiles: map[string]*profile.Profile{
		wbProfile:    {ID: wbProfile, CalendarID: "cal-wb", Name: "WB Digital", Type: profile.ProfileTypeBusiness, IsActive: true, CreatedAt: now, UpdatedAt: now},
		otherProfile: {ID: otherProfile, CalendarID: "cal-pf", Name: "Bruno Pessoal", Type: profile.ProfileTypePersonal, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}
	accountRepo := &fakeAccountRepo{accounts: map[string]*bankaccount.BankAccount{
		accountID: {ID: accountID, ProfileID: wbProfile, Name: "Nubank Juridica", Type: bankaccount.AccountTypeChecking, Currency: "BRL", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}
	categoryRepo := &fakeCategoryRepo{categories: map[string]*category.Category{
		categoryID: {ID: categoryID, ProfileID: wbProfile, Name: "Projetos/Servicos", Type: category.TypeIncome, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}}
	centerRepo := &fakeCostCenterRepo{centers: map[string]*costcenter.CostCenter{
		clientCenter:  {ID: clientCenter, ProfileID: wbProfile, Name: "Acme", Type: costcenter.TypeClient, IsActive: true},
		foreignCenter: {ID: foreignCenter, ProfileID: otherProfile, Name: "Centro alheio", Type: costcenter.TypeClient, IsActive: true},
	}}
	txRepo := &fakeTransactionRepo{}

	uc := NewCreateTransactionUseCase(profileRepo, accountRepo, categoryRepo, txRepo, &fakeInvoiceRepo{}, nil, centerRepo)

	return &costCenterFixture{
		useCase: uc, txRepo: txRepo,
		profileID: wbProfile, accountID: accountID, catID: categoryID,
		clientID: clientCenter, otherID: foreignCenter,
	}
}

func (f *costCenterFixture) income(t *testing.T, costCenterID *string) error {
	t.Helper()
	confirmed := "CONFIRMED"
	_, err := f.useCase.Execute(CreateTransactionInput{
		ProfileID:     f.profileID,
		BankAccountID: f.accountID,
		CategoryID:    &f.catID,
		CostCenterID:  costCenterID,
		Type:          "INCOME",
		Status:        &confirmed,
		Amount:        1500,
		Currency:      "BRL",
		Description:   "Pix cliente Acme",
		OccurredOn:    "2026-09-01",
	})
	return err
}

// Revenue that does not say who paid it cannot be read back per client. The
// column and its foreign key already exist; nothing was writing them.
func TestCreateTransaction_StoresTheCostCenter(t *testing.T) {
	f := newCostCenterFixture(t)

	if err := f.income(t, &f.clientID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(f.txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(f.txRepo.created))
	}
	got := f.txRepo.created[0]
	if got.CostCenterID == nil {
		t.Fatal("the transaction was stored with no cost center")
	}
	if *got.CostCenterID != f.clientID {
		t.Errorf("CostCenterID = %q, want %q", *got.CostCenterID, f.clientID)
	}
}

// Same rule categories already follow: a cost center from another profile would
// let one company's revenue be filed under another's client.
func TestCreateTransaction_RejectsACostCenterFromAnotherProfile(t *testing.T) {
	f := newCostCenterFixture(t)

	err := f.income(t, &f.otherID)
	if err == nil {
		t.Fatal("a cost center from another profile was accepted")
	}
	if !errors.Is(err, ErrCostCenterNotFound) {
		t.Errorf("err = %v, want ErrCostCenterNotFound", err)
	}
	if len(f.txRepo.created) != 0 {
		t.Error("the transaction was written despite the invalid cost center")
	}
}

func TestCreateTransaction_RejectsAnUnknownCostCenter(t *testing.T) {
	f := newCostCenterFixture(t)
	unknown := "cc-inexistente"

	if err := f.income(t, &unknown); !errors.Is(err, ErrCostCenterNotFound) {
		t.Errorf("err = %v, want ErrCostCenterNotFound", err)
	}
}

// The field is optional: every existing caller omits it, and none of them may
// start failing.
func TestCreateTransaction_WithoutACostCenterStillWorks(t *testing.T) {
	f := newCostCenterFixture(t)

	if err := f.income(t, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.txRepo.created) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(f.txRepo.created))
	}
	if f.txRepo.created[0].CostCenterID != nil {
		t.Error("a cost center appeared out of nowhere")
	}
}
