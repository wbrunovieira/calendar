package usecases

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/cryptopurchase"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/binance"
)

type SyncTradesUseCase struct {
	binanceClient    *binance.Client
	accountRepo      bankaccount.Repository
	transactionRepo  transaction.Repository
	purchaseRepo     cryptopurchase.Repository
	symbols          []string // e.g., ["SOLBRL"]
}

type SyncTradesResult struct {
	NewBuys  int `json:"newBuys"`
	NewSells int `json:"newSells"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

func NewSyncTradesUseCase(
	client *binance.Client,
	accountRepo bankaccount.Repository,
	transactionRepo transaction.Repository,
	purchaseRepo cryptopurchase.Repository,
	symbols []string,
) *SyncTradesUseCase {
	return &SyncTradesUseCase{
		binanceClient:   client,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		purchaseRepo:    purchaseRepo,
		symbols:         symbols,
	}
}

func (uc *SyncTradesUseCase) Execute(profileID string) (*SyncTradesResult, error) {
	result := &SyncTradesResult{}

	// Find the exchange account for this profile
	accounts, err := uc.accountRepo.FindByProfileID(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find accounts: %w", err)
	}

	var exchangeAccount *bankaccount.BankAccount
	// Map asset -> sub-account (e.g., SOL -> "Binance - Solana (SOL)")
	subAccountMap := map[string]*bankaccount.BankAccount{}

	for _, acc := range accounts {
		if acc.Type == bankaccount.AccountTypeExchange {
			exchangeAccount = acc
		}
	}
	if exchangeAccount == nil {
		return nil, fmt.Errorf("no exchange account found for profile")
	}

	// Find sub-accounts linked to exchange
	for _, acc := range accounts {
		if acc.LinkedAccountID != nil && *acc.LinkedAccountID == exchangeAccount.ID {
			// Extract asset from name e.g., "Binance - Solana (SOL)" -> "SOL"
			if start := strings.LastIndex(acc.Name, "("); start >= 0 {
				if end := strings.LastIndex(acc.Name, ")"); end > start {
					asset := acc.Name[start+1 : end]
					subAccountMap[asset] = acc
				}
			}
		}
	}

	for _, symbol := range uc.symbols {
		trades, err := uc.binanceClient.GetMyTrades(symbol)
		if err != nil {
			log.Printf("Failed to fetch trades for %s: %v", symbol, err)
			result.Errors++
			continue
		}

		// Determine asset and quote from symbol (e.g., SOLBRL -> SOL, BRL)
		asset, quote := parseSymbol(symbol)
		if asset == "" {
			continue
		}

		for _, trade := range trades {
			// Check if trade already exists by externalId
			externalID := fmt.Sprintf("binance-%d", trade.ID)
			existing, _ := uc.transactionRepo.FindByExternalID(externalID)
			if existing != nil {
				result.Skipped++
				continue
			}

			qty, _ := strconv.ParseFloat(trade.Qty, 64)
			price, _ := strconv.ParseFloat(trade.Price, 64)
			total, _ := strconv.ParseFloat(trade.QuoteQty, 64)
			tradeTime := time.Unix(0, trade.Time*int64(time.Millisecond))

			if trade.IsBuyer {
				// BUY: expense on exchange account (BRL out), record crypto purchase
				err := uc.recordBuy(profileID, exchangeAccount, subAccountMap[asset], asset, quote,
					externalID, qty, price, total, tradeTime)
				if err != nil {
					log.Printf("Failed to record buy trade %d: %v", trade.ID, err)
					result.Errors++
					continue
				}
				result.NewBuys++
			} else {
				// SELL: income on exchange account (BRL in)
				err := uc.recordSell(profileID, exchangeAccount, subAccountMap[asset], asset, quote,
					externalID, qty, price, total, tradeTime)
				if err != nil {
					log.Printf("Failed to record sell trade %d: %v", trade.ID, err)
					result.Errors++
					continue
				}
				result.NewSells++
			}
		}
	}

	// After syncing trades, update account balances from Binance
	uc.syncAccountBalances(exchangeAccount, subAccountMap)

	return result, nil
}

func (uc *SyncTradesUseCase) recordBuy(
	profileID string, exchange *bankaccount.BankAccount, subAccount *bankaccount.BankAccount,
	asset, quote, externalID string, qty, price, total float64, occurredOn time.Time,
) error {
	// Create EXPENSE transaction on exchange account
	tx := &transaction.Transaction{
		ProfileID:     profileID,
		BankAccountID: exchange.ID,
		Type:          transaction.TypeExpense,
		Status:        transaction.StatusConfirmed,
		Amount:        math.Round(total*100) / 100,
		Currency:      quote,
		Description:   fmt.Sprintf("Compra %s (bot)", asset),
		Notes:         strPtrHelper(fmt.Sprintf("%.8f %s @ %s %s | Trade ID: %s", qty, asset, quote, formatPrice(price), externalID)),
		OccurredOn:    occurredOn,
		ExternalID:    &externalID,
		Tags:          []string{"crypto", "bot", strings.ToLower(asset)},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := uc.transactionRepo.Create(tx); err != nil {
		return fmt.Errorf("create buy transaction: %w", err)
	}

	// Record crypto purchase with structured data
	// For BRL pairs, exchangeRate = 1 (already in BRL), priceUSD needs conversion
	// But we store the actual purchase price and mark exchange rate appropriately
	var exchangeRate float64
	var priceUSD float64
	if quote == "BRL" {
		// Get current USD/BRL for reference (approximate)
		usdBrl, err := uc.binanceClient.GetTickerPrice("USDTBRL")
		if err == nil && usdBrl > 0 {
			exchangeRate = usdBrl
			priceUSD = price / usdBrl
		} else {
			exchangeRate = 1
			priceUSD = price
		}
	} else {
		exchangeRate = 1
		priceUSD = price
	}

	cp, err := cryptopurchase.NewCryptoPurchase(
		tx.ID, exchange.ID, profileID, asset,
		qty, priceUSD, exchangeRate, occurredOn,
		fmt.Sprintf("Bot buy @ %s %s", quote, formatPrice(price)),
	)
	if err != nil {
		return fmt.Errorf("create crypto purchase: %w", err)
	}
	if err := uc.purchaseRepo.Create(cp); err != nil {
		return fmt.Errorf("persist crypto purchase: %w", err)
	}

	// Update exchange balance
	exchange.CurrentBalance -= math.Round(total*100) / 100
	exchange.UpdatedAt = time.Now()
	uc.accountRepo.Update(exchange)

	return nil
}

func (uc *SyncTradesUseCase) recordSell(
	profileID string, exchange *bankaccount.BankAccount, subAccount *bankaccount.BankAccount,
	asset, quote, externalID string, qty, price, total float64, occurredOn time.Time,
) error {
	// Create INCOME transaction on exchange account
	tx := &transaction.Transaction{
		ProfileID:     profileID,
		BankAccountID: exchange.ID,
		Type:          transaction.TypeIncome,
		Status:        transaction.StatusConfirmed,
		Amount:        math.Round(total*100) / 100,
		Currency:      quote,
		Description:   fmt.Sprintf("Venda %s (bot)", asset),
		Notes:         strPtrHelper(fmt.Sprintf("%.8f %s @ %s %s | Trade ID: %s", qty, asset, quote, formatPrice(price), externalID)),
		OccurredOn:    occurredOn,
		ExternalID:    &externalID,
		Tags:          []string{"crypto", "bot", strings.ToLower(asset)},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := uc.transactionRepo.Create(tx); err != nil {
		return fmt.Errorf("create sell transaction: %w", err)
	}

	// Update exchange balance
	exchange.CurrentBalance += math.Round(total*100) / 100
	exchange.UpdatedAt = time.Now()
	uc.accountRepo.Update(exchange)

	return nil
}

func (uc *SyncTradesUseCase) syncAccountBalances(exchange *bankaccount.BankAccount, subAccounts map[string]*bankaccount.BankAccount) {
	balances, err := uc.binanceClient.GetAccountBalances()
	if err != nil {
		return
	}

	balanceMap := map[string]float64{}
	for _, b := range balances {
		free, _ := strconv.ParseFloat(b.Free, 64)
		locked, _ := strconv.ParseFloat(b.Locked, 64)
		balanceMap[b.Asset] = free + locked
	}

	// Update exchange BRL balance
	if brl, ok := balanceMap["BRL"]; ok {
		exchange.CurrentBalance = math.Round(brl*100) / 100
		exchange.UpdatedAt = time.Now()
		uc.accountRepo.Update(exchange)
	}

	// Update sub-account quotas
	for asset, acc := range subAccounts {
		if qty, ok := balanceMap[asset]; ok {
			acc.NumberOfQuotas = &qty
			// Update balance: qty * quotaPrice
			if acc.QuotaPrice != nil {
				acc.CurrentBalance = qty * *acc.QuotaPrice
			}
			acc.UpdatedAt = time.Now()
			uc.accountRepo.Update(acc)
		}
	}
}

// parseSymbol extracts asset and quote from a Binance symbol
// e.g., "SOLBRL" -> ("SOL", "BRL"), "SOLUSDT" -> ("SOL", "USDT")
func parseSymbol(symbol string) (string, string) {
	quotes := []string{"BRL", "USDT", "BUSD", "BTC", "ETH"}
	for _, q := range quotes {
		if strings.HasSuffix(symbol, q) {
			asset := strings.TrimSuffix(symbol, q)
			if asset != "" {
				return asset, q
			}
		}
	}
	return "", ""
}

func formatPrice(p float64) string {
	if p >= 1 {
		return fmt.Sprintf("%.2f", p)
	}
	return fmt.Sprintf("%.8f", p)
}

func strPtrHelper(s string) *string {
	return &s
}
