package usecases

import (
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// CashflowSummaryInput asks "what actually came in and went out this period".
type CashflowSummaryInput struct {
	ProfileID string
	From      string // YYYY-MM-DD
	To        string // YYYY-MM-DD
}

// CashflowSummaryOutput separates yield from earned income because the two answer
// different questions: how much the balance produced by itself, and how much work or
// sales brought in.
type CashflowSummaryOutput struct {
	Income      float64 `json:"income"`
	IncomeYield float64 `json:"incomeYield"`
	IncomeOther float64 `json:"incomeOther"`
	Expense     float64 `json:"expense"`
	Net         float64 `json:"net"`
}

// GetCashflowSummaryUseCase answers the monthly cashflow with the classification rules
// living in the domain, so every client — the dashboard, an agent, a report — reads
// the same numbers instead of re-deriving them from descriptions.
type GetCashflowSummaryUseCase struct {
	txRepo       transaction.Repository
	accountRepo  bankaccount.Repository
	categoryRepo category.Repository
}

func NewGetCashflowSummaryUseCase(
	txRepo transaction.Repository,
	accountRepo bankaccount.Repository,
	categoryRepo category.Repository,
) *GetCashflowSummaryUseCase {
	return &GetCashflowSummaryUseCase{txRepo: txRepo, accountRepo: accountRepo, categoryRepo: categoryRepo}
}

func (uc *GetCashflowSummaryUseCase) Execute(input CashflowSummaryInput) (*CashflowSummaryOutput, error) {
	if input.ProfileID == "" {
		return nil, errors.New("profileId is required")
	}

	from, err := parseDate(input.From)
	if err != nil {
		return nil, errors.New("from must be YYYY-MM-DD")
	}
	to, err := parseDate(input.To)
	if err != nil {
		return nil, errors.New("to must be YYYY-MM-DD")
	}

	accounts, err := uc.accountRepo.FindByProfileID(input.ProfileID)
	if err != nil {
		return nil, err
	}
	accountByID := make(map[string]*bankaccount.BankAccount, len(accounts))
	for _, a := range accounts {
		accountByID[a.ID] = a
	}

	categories, err := uc.categoryRepo.ListByProfile(input.ProfileID)
	if err != nil {
		return nil, err
	}
	categoryByID := make(map[string]*category.Category, len(categories))
	for _, c := range categories {
		categoryByID[c.ID] = c
	}

	confirmed := transaction.StatusConfirmed
	txs, err := uc.txRepo.List(transaction.ListFilter{
		ProfileID:    input.ProfileID,
		Status:       &confirmed,
		OccurredFrom: &from,
		OccurredTo:   &to,
	})
	if err != nil {
		return nil, err
	}

	out := &CashflowSummaryOutput{}
	for _, tx := range txs {
		account := accountByID[tx.BankAccountID]

		// An exchange holds crypto the owner already owns; trading inside it is not
		// household cashflow, so nothing there counts either way.
		if account != nil && account.IsExchange() {
			continue
		}

		var destination *bankaccount.BankAccount
		if tx.DestinationAccountID != nil {
			destination = accountByID[*tx.DestinationAccountID]
		}
		if tx.IsSettlement(account, destination) {
			continue
		}

		switch tx.Type {
		case transaction.TypeIncome:
			var cat *category.Category
			if tx.CategoryID != nil {
				cat = categoryByID[*tx.CategoryID]
			}
			if tx.IsYield(cat) {
				out.IncomeYield += tx.Amount
			} else {
				out.IncomeOther += tx.Amount
			}
		case transaction.TypeExpense:
			out.Expense += tx.Amount
		}
	}

	out.Income = round2(out.IncomeYield + out.IncomeOther)
	out.IncomeYield = round2(out.IncomeYield)
	out.IncomeOther = round2(out.IncomeOther)
	out.Expense = round2(out.Expense)
	out.Net = round2(out.Income - out.Expense)

	return out, nil
}
