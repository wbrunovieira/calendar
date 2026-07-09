package usecases

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/yahoo"
)

// DividendProvider abstracts the dividend data source (Yahoo Finance) for testability.
type DividendProvider interface {
	GetDividends(ticker string, from time.Time) ([]yahoo.Dividend, error)
}

type DividendSyncUseCase struct {
	dividendProvider DividendProvider
	accountRepo      bankaccount.Repository
	transactionRepo  transaction.Repository
}

type DividendSyncResult struct {
	NewDividends int     `json:"newDividends"`
	Skipped      int     `json:"skipped"`
	Errors       int     `json:"errors"`
	TotalAmount  float64 `json:"totalAmount"`
}

func NewDividendSyncUseCase(
	provider DividendProvider,
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
) *DividendSyncUseCase {
	return &DividendSyncUseCase{
		dividendProvider: provider,
		accountRepo:      accountRepo,
		transactionRepo:  transactionRepo,
	}
}

// Execute checks for new dividends since the given date and creates income transactions.
// Dividend dates are ex-dividend dates (Yahoo does not expose payment dates).
func (uc *DividendSyncUseCase) Execute(profileID string, since time.Time) (*DividendSyncResult, error) {
	accounts, err := uc.accountRepo.FindByProfileID(profileID)
	if err != nil {
		return nil, err
	}

	// Find the parent broker account (first INVESTMENT without LinkedAccountID that has sub-accounts)
	parentID := ""
	type tickerAccount struct {
		ticker  string
		account *bankaccount.BankAccount
	}
	var tickerAccounts []tickerAccount

	for _, acc := range accounts {
		if acc.LinkedAccountID == nil {
			continue
		}
		if !acc.HasQuotas() {
			continue
		}
		ticker := detectTicker(acc.Name)
		if ticker == "" {
			continue
		}
		parentID = *acc.LinkedAccountID
		tickerAccounts = append(tickerAccounts, tickerAccount{ticker: ticker, account: acc})
	}

	if len(tickerAccounts) == 0 {
		return &DividendSyncResult{}, nil
	}

	result := &DividendSyncResult{}

	for _, ta := range tickerAccounts {
		dividends, err := uc.dividendProvider.GetDividends(ta.ticker, since)
		if err != nil {
			log.Printf("Failed to fetch dividends for %s: %v", ta.ticker, err)
			result.Errors++
			continue
		}

		for _, div := range dividends {
			// Only process dividends after since date
			if div.Date.Before(since) {
				continue
			}

			dateStr := div.Date.Format("2006-01-02")

			// Check if already recorded. On lookup failure, skip creation —
			// creating blindly would duplicate the dividend.
			externalID := dividendExternalID(ta.ticker, dateStr, div.Amount)
			existing, err := uc.transactionRepo.FindByExternalID(externalID)
			if err != nil && !errors.Is(err, transaction.ErrNotFound) {
				log.Printf("Failed to check existing dividend %s: %v", externalID, err)
				result.Errors++
				continue
			}
			if existing != nil {
				result.Skipped++
				continue
			}

			// Calculate amount: quotas * rate per share
			quotas := *ta.account.NumberOfQuotas
			amount := math.Round(quotas*div.Amount*100) / 100

			tx := &transaction.Transaction{
				ProfileID:     profileID,
				BankAccountID: parentID,
				Type:          transaction.TypeIncome,
				Status:        transaction.StatusConfirmed,
				Amount:        amount,
				Currency:      "BRL",
				Description:   fmt.Sprintf("Dividendo %s", ta.ticker),
				Notes:         strPtrHelper(fmt.Sprintf("%.2f/cota × %.0f cotas (data-ex %s)", div.Amount, quotas, dateStr)),
				OccurredOn:    div.Date,
				ExternalID:    &externalID,
				Tags:          []string{"dividendo", ta.ticker},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			if err := uc.transactionRepo.Create(tx); err != nil {
				log.Printf("Failed to create dividend transaction for %s: %v", ta.ticker, err)
				result.Errors++
				continue
			}

			result.NewDividends++
			result.TotalAmount += amount
		}
	}

	return result, nil
}

func dividendExternalID(ticker, date string, rate float64) string {
	return fmt.Sprintf("dividend-%s-%s-%.2f", ticker, date, rate)
}
