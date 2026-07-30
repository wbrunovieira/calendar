package usecases

import (
	"math"
	"strings"
	"testing"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// sellAccountRepo is a minimal in-memory bankaccount.Repository for sell tests.
// FindByID returns the stored pointer, so mutations by the use case are visible
// to assertions without Update having to persist anything.
type sellAccountRepo struct {
	accounts map[string]*bankaccount.BankAccount
}

func newSellAccountRepo(accs ...*bankaccount.BankAccount) *sellAccountRepo {
	m := make(map[string]*bankaccount.BankAccount, len(accs))
	for _, a := range accs {
		m[a.ID] = a
	}
	return &sellAccountRepo{accounts: m}
}

func (r *sellAccountRepo) Create(a *bankaccount.BankAccount) error { r.accounts[a.ID] = a; return nil }
func (r *sellAccountRepo) FindByID(id string) (*bankaccount.BankAccount, error) {
	if a, ok := r.accounts[id]; ok {
		return a, nil
	}
	return nil, ErrBankAccountNotFound
}
func (r *sellAccountRepo) FindByProfileID(string) ([]*bankaccount.BankAccount, error) {
	return nil, nil
}
func (r *sellAccountRepo) FindAll() ([]*bankaccount.BankAccount, error)               { return nil, nil }
func (r *sellAccountRepo) Update(a *bankaccount.BankAccount) error                    { r.accounts[a.ID] = a; return nil }
func (r *sellAccountRepo) Delete(string) error                                        { return nil }
func (r *sellAccountRepo) UpdateDisplayOrders([]bankaccount.DisplayOrderUpdate) error { return nil }

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// linkedPosition builds a FII position with `quotas` shares @ `price`, linked to
// a cash account holding `cashBalance`. Returns (position, cash).
func linkedPosition(t *testing.T, quotas, price, cashBalance float64) (*bankaccount.BankAccount, *bankaccount.BankAccount) {
	t.Helper()
	cash, err := bankaccount.NewBankAccount("profile-1", "Clear", bankaccount.AccountTypeChecking, cashBalance, "BRL")
	if err != nil {
		t.Fatalf("cash NewBankAccount: %v", err)
	}
	pos, err := bankaccount.NewBankAccount("profile-1", "SNAG11", bankaccount.AccountTypeInvestment, 0, "BRL")
	if err != nil {
		t.Fatalf("position NewBankAccount: %v", err)
	}
	if err := pos.SetQuotasFromPrice(quotas, price); err != nil {
		t.Fatalf("SetQuotasFromPrice: %v", err)
	}
	if err := pos.SetLinkedAccount(cash); err != nil {
		t.Fatalf("SetLinkedAccount: %v", err)
	}
	return pos, cash
}

func TestSellPosition_FullSell_ClosesPositionAndCreditsCash(t *testing.T) {
	pos, cash := linkedPosition(t, 120, 9.84, 182.04)
	accRepo := newSellAccountRepo(pos, cash)
	txRepo := &FakeTransactionRepository{}
	uc := NewSellPositionUseCase(accRepo, txRepo)

	res, err := uc.Execute(pos.ID, SellPositionInput{Quantity: 120, UnitPrice: 9.84, OccurredOn: "2026-07-30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Position closed.
	if pos.NumberOfQuotas == nil || *pos.NumberOfQuotas != 0 {
		t.Errorf("position quotas = %v, want 0", pos.NumberOfQuotas)
	}
	if pos.CurrentBalance != 0 {
		t.Errorf("position balance = %.2f, want 0", pos.CurrentBalance)
	}

	// Cash credited with the proceeds (182.04 + 1180.80).
	if !approxEqual(cash.CurrentBalance, 182.04+1180.80) {
		t.Errorf("cash balance = %.2f, want %.2f", cash.CurrentBalance, 182.04+1180.80)
	}

	// One transfer transaction recorded, position -> cash.
	if len(txRepo.transactions) != 1 {
		t.Fatalf("created %d transactions, want 1", len(txRepo.transactions))
	}
	tx := txRepo.transactions[0]
	if tx.Type != transaction.TypeTransfer {
		t.Errorf("tx type = %s, want TRANSFER", tx.Type)
	}
	if tx.BankAccountID != pos.ID {
		t.Errorf("tx source = %s, want position %s", tx.BankAccountID, pos.ID)
	}
	if tx.DestinationAccountID == nil || *tx.DestinationAccountID != cash.ID {
		t.Errorf("tx destination = %v, want cash %s", tx.DestinationAccountID, cash.ID)
	}
	if tx.Status != transaction.StatusConfirmed {
		t.Errorf("tx status = %s, want CONFIRMED", tx.Status)
	}
	if !approxEqual(tx.Amount, 1180.80) {
		t.Errorf("tx amount = %.2f, want 1180.80", tx.Amount)
	}
	if !strings.Contains(tx.Description, "Venda SNAG11 - 120 cotas @ R$9.84") {
		t.Errorf("tx description = %q", tx.Description)
	}
	if res.Transaction != tx || res.Account != pos {
		t.Error("result should point at the created transaction and updated position")
	}
}

func TestSellPosition_PartialSell_KeepsRemainder(t *testing.T) {
	pos, cash := linkedPosition(t, 120, 9.84, 0)
	uc := NewSellPositionUseCase(newSellAccountRepo(pos, cash), &FakeTransactionRepository{})

	if _, err := uc.Execute(pos.ID, SellPositionInput{Quantity: 73, UnitPrice: 9.84, OccurredOn: "2026-07-30"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pos.NumberOfQuotas == nil || *pos.NumberOfQuotas != 47 {
		t.Errorf("remaining quotas = %v, want 47", pos.NumberOfQuotas)
	}
	if !approxEqual(cash.CurrentBalance, 73*9.84) {
		t.Errorf("cash balance = %.2f, want %.2f", cash.CurrentBalance, 73*9.84)
	}
}

func TestSellPosition_OversellRejected_NoMutation(t *testing.T) {
	pos, cash := linkedPosition(t, 10, 100, 0)
	txRepo := &FakeTransactionRepository{}
	uc := NewSellPositionUseCase(newSellAccountRepo(pos, cash), txRepo)

	_, err := uc.Execute(pos.ID, SellPositionInput{Quantity: 11, UnitPrice: 100, OccurredOn: "2026-07-30"})
	if err != ErrCannotSellMoreThanHeld {
		t.Fatalf("error = %v, want ErrCannotSellMoreThanHeld", err)
	}
	if pos.NumberOfQuotas == nil || *pos.NumberOfQuotas != 10 {
		t.Errorf("position mutated on rejected sell: %v", pos.NumberOfQuotas)
	}
	if len(txRepo.transactions) != 0 {
		t.Errorf("created %d transactions on rejected sell, want 0", len(txRepo.transactions))
	}
	if cash.CurrentBalance != 0 {
		t.Errorf("cash mutated on rejected sell: %.2f", cash.CurrentBalance)
	}
}

func TestSellPosition_NotLinked_Rejected(t *testing.T) {
	pos, err := bankaccount.NewBankAccount("profile-1", "SNAG11", bankaccount.AccountTypeInvestment, 0, "BRL")
	if err != nil {
		t.Fatalf("NewBankAccount: %v", err)
	}
	if err := pos.SetQuotasFromPrice(120, 9.84); err != nil {
		t.Fatalf("SetQuotasFromPrice: %v", err)
	}
	uc := NewSellPositionUseCase(newSellAccountRepo(pos), &FakeTransactionRepository{})

	if _, err := uc.Execute(pos.ID, SellPositionInput{Quantity: 1, UnitPrice: 9.84, OccurredOn: "2026-07-30"}); err != ErrAccountNotLinked {
		t.Fatalf("error = %v, want ErrAccountNotLinked", err)
	}
}

func TestSellPosition_NoQuotas_Rejected(t *testing.T) {
	acc, err := bankaccount.NewBankAccount("profile-1", "Checking", bankaccount.AccountTypeChecking, 500, "BRL")
	if err != nil {
		t.Fatalf("NewBankAccount: %v", err)
	}
	uc := NewSellPositionUseCase(newSellAccountRepo(acc), &FakeTransactionRepository{})

	if _, err := uc.Execute(acc.ID, SellPositionInput{Quantity: 1, UnitPrice: 10, OccurredOn: "2026-07-30"}); err != ErrPositionHasNoQuotas {
		t.Fatalf("error = %v, want ErrPositionHasNoQuotas", err)
	}
}

func TestSellPosition_AccountNotFound(t *testing.T) {
	uc := NewSellPositionUseCase(newSellAccountRepo(), &FakeTransactionRepository{})

	if _, err := uc.Execute("missing", SellPositionInput{Quantity: 1, UnitPrice: 10, OccurredOn: "2026-07-30"}); err != ErrBankAccountNotFound {
		t.Fatalf("error = %v, want ErrBankAccountNotFound", err)
	}
}

func TestSellPosition_InvalidInput(t *testing.T) {
	pos, cash := linkedPosition(t, 10, 100, 0)
	uc := NewSellPositionUseCase(newSellAccountRepo(pos, cash), &FakeTransactionRepository{})

	if _, err := uc.Execute(pos.ID, SellPositionInput{Quantity: 0, UnitPrice: 100, OccurredOn: "2026-07-30"}); err != ErrInvalidInput {
		t.Errorf("zero quantity error = %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Execute(pos.ID, SellPositionInput{Quantity: 1, UnitPrice: 0, OccurredOn: "2026-07-30"}); err != ErrInvalidInput {
		t.Errorf("zero price error = %v, want ErrInvalidInput", err)
	}
}
