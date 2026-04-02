package usecases

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

type DividendSyncUseCase struct {
	brapiClient     BrapiClient
	accountRepo     bankaccount.Repository
	transactionRepo transaction.Repository
}

type DividendSyncResult struct {
	NewDividends int     `json:"newDividends"`
	Skipped      int     `json:"skipped"`
	Errors       int     `json:"errors"`
	TotalAmount  float64 `json:"totalAmount"`
}

func NewDividendSyncUseCase(
	client BrapiClient,
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
) *DividendSyncUseCase {
	return &DividendSyncUseCase{
		brapiClient:     client,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// Execute checks for new dividends since the given date and creates income transactions.
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
		dividends, err := uc.brapiClient.GetDividends(ta.ticker)
		if err != nil {
			log.Printf("Failed to fetch dividends for %s: %v", ta.ticker, err)
			result.Errors++
			continue
		}

		for _, div := range dividends {
			// Parse payment date
			payDate, err := time.Parse("2006-01-02", div.PaymentDate)
			if err != nil {
				continue
			}

			// Only process dividends after since date
			if payDate.Before(since) {
				continue
			}

			// Check if already recorded
			externalID := dividendExternalID(ta.ticker, div.PaymentDate, div.Rate)
			existing, _ := uc.transactionRepo.FindByExternalID(externalID)
			if existing != nil {
				result.Skipped++
				continue
			}

			// Calculate amount: quotas * rate per share
			quotas := *ta.account.NumberOfQuotas
			amount := math.Round(quotas*div.Rate*100) / 100

			tx := &transaction.Transaction{
				ProfileID:     profileID,
				BankAccountID: parentID,
				Type:          transaction.TypeIncome,
				Status:        transaction.StatusConfirmed,
				Amount:        amount,
				Currency:      "BRL",
				Description:   fmt.Sprintf("%s %s", div.Label, ta.ticker),
				Notes:         strPtrHelper(fmt.Sprintf("%.2f/cota × %.0f cotas", div.Rate, quotas)),
				OccurredOn:    payDate,
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

func dividendExternalID(ticker, paymentDate string, rate float64) string {
	return fmt.Sprintf("dividend-%s-%s-%.2f", ticker, paymentDate, rate)
}
