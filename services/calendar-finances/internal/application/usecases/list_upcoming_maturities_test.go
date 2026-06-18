package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

// fakeMaturityRepo is a minimal bankaccount.Repository for maturity-alert tests.
type fakeMaturityRepo struct {
	accounts []*bankaccount.BankAccount
	err      error
}

func (f *fakeMaturityRepo) Create(*bankaccount.BankAccount) error          { return nil }
func (f *fakeMaturityRepo) FindByID(string) (*bankaccount.BankAccount, error) { return nil, nil }
func (f *fakeMaturityRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return f.accounts, f.err
}
func (f *fakeMaturityRepo) FindAll() ([]*bankaccount.BankAccount, error) { return f.accounts, f.err }
func (f *fakeMaturityRepo) Update(*bankaccount.BankAccount) error        { return nil }
func (f *fakeMaturityRepo) Delete(string) error                         { return nil }
func (f *fakeMaturityRepo) UpdateDisplayOrders([]bankaccount.DisplayOrderUpdate) error { return nil }

func ptrTime(t time.Time) *time.Time { return &t }

func mkInvestment(id string, active bool, maturity *time.Time) *bankaccount.BankAccount {
	return &bankaccount.BankAccount{
		ID:             id,
		ProfileID:      "p1",
		Name:           id,
		Type:           bankaccount.AccountTypeInvestment,
		IsActive:       active,
		MaturityDate:   maturity,
		CurrentBalance: 1000,
		Currency:       "BRL",
	}
}

var refNow = time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)

func runMaturities(t *testing.T, accounts []*bankaccount.BankAccount, withinDays int) []MaturityAlert {
	t.Helper()
	uc := NewListUpcomingMaturitiesUseCase(&fakeMaturityRepo{accounts: accounts})
	got, err := uc.Execute(ListUpcomingMaturitiesInput{ProfileID: "p1", WithinDays: withinDays, Now: refNow})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return got
}

func TestUpcomingMaturities_IncludesWithinWindow(t *testing.T) {
	mat := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // 13 days ahead
	got := runMaturities(t, []*bankaccount.BankAccount{mkInvestment("cdb", true, ptrTime(mat))}, 30)
	if len(got) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(got))
	}
	if got[0].AccountID != "cdb" {
		t.Errorf("expected account cdb, got %s", got[0].AccountID)
	}
	if got[0].DaysToMaturity != 13 {
		t.Errorf("expected 13 days, got %d", got[0].DaysToMaturity)
	}
	if got[0].IsMatured {
		t.Error("should not be matured")
	}
}

func TestUpcomingMaturities_ExcludesBeyondWindow(t *testing.T) {
	mat := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC) // far ahead
	got := runMaturities(t, []*bankaccount.BankAccount{mkInvestment("cdb", true, ptrTime(mat))}, 30)
	if len(got) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(got))
	}
}

func TestUpcomingMaturities_IncludesAlreadyMatured(t *testing.T) {
	mat := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC) // 8 days ago
	got := runMaturities(t, []*bankaccount.BankAccount{mkInvestment("cdb", true, ptrTime(mat))}, 30)
	if len(got) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(got))
	}
	if !got[0].IsMatured {
		t.Error("expected matured=true")
	}
	if got[0].DaysToMaturity != -8 {
		t.Errorf("expected -8 days, got %d", got[0].DaysToMaturity)
	}
}

func TestUpcomingMaturities_ExcludesNoMaturityDate(t *testing.T) {
	got := runMaturities(t, []*bankaccount.BankAccount{mkInvestment("cdb", true, nil)}, 30)
	if len(got) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(got))
	}
}

func TestUpcomingMaturities_ExcludesInactive(t *testing.T) {
	mat := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	got := runMaturities(t, []*bankaccount.BankAccount{mkInvestment("cdb", false, ptrTime(mat))}, 30)
	if len(got) != 0 {
		t.Fatalf("expected 0 alerts (inactive), got %d", len(got))
	}
}

func TestUpcomingMaturities_ExcludesNonInvestment(t *testing.T) {
	mat := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	acc := mkInvestment("checking", true, ptrTime(mat))
	acc.Type = bankaccount.AccountTypeChecking
	got := runMaturities(t, []*bankaccount.BankAccount{acc}, 30)
	if len(got) != 0 {
		t.Fatalf("expected 0 alerts (non-investment), got %d", len(got))
	}
}

func TestUpcomingMaturities_SortedByMaturityAsc(t *testing.T) {
	m1 := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	m2 := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	m3 := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC) // matured
	got := runMaturities(t, []*bankaccount.BankAccount{
		mkInvestment("a", true, ptrTime(m1)),
		mkInvestment("b", true, ptrTime(m2)),
		mkInvestment("c", true, ptrTime(m3)),
	}, 30)
	if len(got) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(got))
	}
	order := []string{got[0].AccountID, got[1].AccountID, got[2].AccountID}
	want := []string{"c", "b", "a"}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("position %d: expected %s, got %s (order=%v)", i, want[i], order[i], order)
		}
	}
}

func TestUpcomingMaturities_DefaultWindowWhenZero(t *testing.T) {
	within := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) // 22 days -> within default 30
	beyond := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC) // 37 days -> beyond default 30
	got := runMaturities(t, []*bankaccount.BankAccount{
		mkInvestment("in", true, ptrTime(within)),
		mkInvestment("out", true, ptrTime(beyond)),
	}, 0)
	if len(got) != 1 || got[0].AccountID != "in" {
		t.Fatalf("expected only 'in' with default 30-day window, got %+v", got)
	}
}

func TestUpcomingMaturities_PropagatesRepoError(t *testing.T) {
	uc := NewListUpcomingMaturitiesUseCase(&fakeMaturityRepo{err: errors.New("db down")})
	_, err := uc.Execute(ListUpcomingMaturitiesInput{ProfileID: "p1", Now: refNow})
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}
