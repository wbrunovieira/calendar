package app

import (
	"database/sql"
	"errors"
	"log"
	"os"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/binance"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/brapi"
	httpHandlers "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
	"github.com/brunovieira/calendar-finances/internal/infrastructure/yahoo"
	"github.com/gorilla/mux"
)

// Jobs are the use cases the background loops in main.go drive. They are built
// by the same wiring the HTTP routes use, so a job and a route can never end up
// running against two differently-configured copies of the same use case.
type Jobs struct {
	AutoCloseInvoices *usecases.AutoCloseInvoicesUseCase
	SyncTrades        *usecases.SyncTradesUseCase // nil without Binance credentials
	StockSync         *usecases.StockSyncUseCase
	DividendSync      *usecases.DividendSyncUseCase
	CloseMonth        *usecases.CloseMonthUseCase
}

// App is the wired API: every repository, use case, handler and route the
// service serves.
type App struct {
	Router *mux.Router
	Jobs   Jobs
}

// New wires the whole API against db. It performs no I/O, so tests can build the
// real router — the one main.go serves — without a database.
func New(db *sql.DB) (*App, error) {
	if db == nil {
		return nil, errors.New("app: a database handle is required")
	}

	// Initialize router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
	router.HandleFunc("/", handlers.RootHandler).Methods("GET")

	// Initialize Profile repository and use cases
	profileRepo := persistence.NewProfileRepository(db)
	createProfileUC := usecases.NewCreateProfileUseCase(profileRepo)
	listProfilesUC := usecases.NewListProfilesUseCase(profileRepo)
	getProfileUC := usecases.NewGetProfileUseCase(profileRepo)
	updateProfileUC := usecases.NewUpdateProfileUseCase(profileRepo)
	deleteProfileUC := usecases.NewDeleteProfileUseCase(profileRepo)

	// Initialize Profile handlers
	profileHandler := httpHandlers.NewProfileHandlers(
		createProfileUC,
		listProfilesUC,
		getProfileUC,
		updateProfileUC,
		deleteProfileUC,
	)

	// Initialize repositories
	bankAccountRepo := persistence.NewBankAccountRepository(db)
	categoryRepo := persistence.NewCategoryRepository(db)
	invoiceRepo := persistence.NewInvoiceRepository(db)
	transactionRepo := persistence.NewTransactionRepository(db)
	checkpointRepo := persistence.NewCheckpointRepository(db)

	// Initialize Bank Account use cases and handlers
	createBankAccountUC := usecases.NewCreateBankAccountUseCase(bankAccountRepo)
	listBankAccountsUC := usecases.NewListBankAccountsUseCase(bankAccountRepo)
	getBankAccountUC := usecases.NewGetBankAccountUseCase(bankAccountRepo)
	updateBankAccountUC := usecases.NewUpdateBankAccountUseCase(bankAccountRepo)
	deleteBankAccountUC := usecases.NewDeleteBankAccountUseCase(bankAccountRepo)
	reorderBankAccountsUC := usecases.NewReorderBankAccountsUseCase(bankAccountRepo)
	closeMonthUC := usecases.NewCloseMonthUseCase(bankAccountRepo, transactionRepo, checkpointRepo)
	recalculateBalanceUC := usecases.NewRecalculateBalanceUseCase(bankAccountRepo, transactionRepo, checkpointRepo)
	upcomingMaturitiesUC := usecases.NewListUpcomingMaturitiesUseCase(bankAccountRepo)
	sellPositionUC := usecases.NewSellPositionUseCase(bankAccountRepo, transactionRepo)
	bankAccountHandler := httpHandlers.NewBankAccountHandlers(
		createBankAccountUC,
		listBankAccountsUC,
		getBankAccountUC,
		updateBankAccountUC,
		deleteBankAccountUC,
		reorderBankAccountsUC,
		recalculateBalanceUC,
		closeMonthUC,
		upcomingMaturitiesUC,
		sellPositionUC,
	)
	bankAccountHandler.SetCreditUsageUseCase(
		usecases.NewGetCreditUsageUseCase(bankAccountRepo, invoiceRepo, transactionRepo),
	)

	// Initialize Category use cases and handlers
	createCategoryUC := usecases.NewCreateCategoryUseCase(profileRepo, categoryRepo)
	listCategoriesUC := usecases.NewListCategoriesUseCase(categoryRepo)
	updateCategoryUC := usecases.NewUpdateCategoryUseCase(categoryRepo)
	deleteCategoryUC := usecases.NewDeleteCategoryUseCase(categoryRepo)
	categoryHandler := httpHandlers.NewCategoryHandlers(
		createCategoryUC,
		listCategoriesUC,
		updateCategoryUC,
		deleteCategoryUC,
	)

	// Initialize Transaction use cases and handlers
	createTransactionUC := usecases.NewCreateTransactionUseCase(profileRepo, bankAccountRepo, categoryRepo, transactionRepo, invoiceRepo, recalculateBalanceUC)
	listTransactionsUC := usecases.NewListTransactionsUseCase(transactionRepo)
	getTransactionUC := usecases.NewGetTransactionUseCase(transactionRepo)
	updateTransactionUC := usecases.NewUpdateTransactionUseCase(bankAccountRepo, categoryRepo, transactionRepo, invoiceRepo, recalculateBalanceUC)
	updateTransactionStatusUC := usecases.NewUpdateTransactionStatusUseCase(transactionRepo, bankAccountRepo, recalculateBalanceUC)
	deleteTransactionUC := usecases.NewDeleteTransactionUseCase(transactionRepo, bankAccountRepo, recalculateBalanceUC)
	dailyBalancesUC := usecases.NewGetDailyBalancesUseCase(transactionRepo, bankAccountRepo)
	transactionHandler := httpHandlers.NewTransactionHandlers(
		createTransactionUC,
		listTransactionsUC,
		getTransactionUC,
		updateTransactionUC,
		updateTransactionStatusUC,
		deleteTransactionUC,
	)
	transactionHandler.SetDailyBalancesUseCase(dailyBalancesUC)

	// Initialize Invoice use cases and handlers
	createInvoiceUC := usecases.NewCreateInvoiceUseCase(invoiceRepo, bankAccountRepo)
	listInvoicesUC := usecases.NewListInvoicesUseCase(invoiceRepo, bankAccountRepo, transactionRepo)
	getInvoiceUC := usecases.NewGetInvoiceUseCase(invoiceRepo, transactionRepo)
	getCurrentInvoiceUC := usecases.NewGetCurrentInvoiceUseCase(invoiceRepo, bankAccountRepo, transactionRepo)
	closeInvoiceUC := usecases.NewCloseInvoiceUseCase(invoiceRepo)
	// PayInvoiceUseCaseV2: Creates payment transaction on linked checking account when invoice is paid,
	// credits the card side, and recomputes the card balance from its transactions
	payInvoiceUC := usecases.NewPayInvoiceUseCaseV2(invoiceRepo, bankAccountRepo, transactionRepo, recalculateBalanceUC)
	recalculateInvoiceUC := usecases.NewRecalculateInvoiceAmountUseCase(invoiceRepo, transactionRepo)
	// Deleting a charge changes the bill it belonged to.
	deleteTransactionUC.SetInvoiceRecalculator(recalculateInvoiceUC)
	updateInvoiceUC := usecases.NewUpdateInvoiceUseCase(invoiceRepo)
	invoiceHandler := httpHandlers.NewInvoiceHandlers(
		createInvoiceUC,
		listInvoicesUC,
		getInvoiceUC,
		getCurrentInvoiceUC,
		closeInvoiceUC,
		payInvoiceUC,
		recalculateInvoiceUC,
	)
	invoiceHandler.SetUpdateUseCase(updateInvoiceUC)
	autoCloseInvoicesUC := usecases.NewAutoCloseInvoicesUseCase(invoiceRepo)
	invoiceHandler.SetAutoCloseUseCase(autoCloseInvoicesUC)

	recurringRepo := persistence.NewRecurringTransactionRepository(db)
	recurringService := usecases.NewRecurringTransactionsService(recurringRepo)
	processRecurringUC := usecases.NewProcessRecurringTransactionsUseCase(recurringRepo, createTransactionUC)
	recurringHandler := httpHandlers.NewRecurringTransactionHandlers(recurringService, processRecurringUC)
	recurringHandler.SetPendingUseCase(usecases.NewListPendingRecurringsUseCase(recurringRepo, transactionRepo))

	expenseAnalysisUC := usecases.NewGetExpenseAnalysisUseCase(transactionRepo, categoryRepo, recurringRepo, bankAccountRepo)
	transactionHandler.SetExpenseAnalysisUseCase(expenseAnalysisUC)

	financialSummaryUC := usecases.NewGetFinancialSummaryUseCase(transactionRepo, categoryRepo, bankAccountRepo)
	transactionHandler.SetFinancialSummaryUseCase(financialSummaryUC)

	cashflowSummaryUC := usecases.NewGetCashflowSummaryUseCase(transactionRepo, bankAccountRepo, categoryRepo)
	transactionHandler.SetCashflowSummaryUseCase(cashflowSummaryUC)

	budgetRepo := persistence.NewBudgetTargetRepository(db)
	budgetService := usecases.NewBudgetTargetsService(budgetRepo, transactionRepo, categoryRepo)
	budgetHandler := httpHandlers.NewBudgetHandlers(budgetService)

	goalRepo := persistence.NewGoalRepository(db)
	goalService := usecases.NewGoalsService(goalRepo)
	reorderGoalsUC := usecases.NewReorderGoalsUseCase(goalRepo)
	goalHandler := httpHandlers.NewGoalHandlers(goalService, reorderGoalsUC)

	// Initialize Binance client and crypto sync
	cryptoPurchaseRepo := persistence.NewCryptoPurchaseRepository(db)
	binanceKey := os.Getenv("KEY_BINANCE")
	binanceSecret := os.Getenv("SECRET_BINANCE")
	var cryptoHandler *httpHandlers.CryptoHandlers
	var cryptoPurchaseHandler *httpHandlers.CryptoPurchaseHandlers
	var binanceClient *binance.Client
	var syncTradesUC *usecases.SyncTradesUseCase
	if binanceKey != "" && binanceSecret != "" {
		binanceClient = binance.NewClient(binanceKey, binanceSecret)
		cryptoSyncUC := usecases.NewCryptoSyncUseCase(binanceClient, bankAccountRepo)
		cryptoHandler = httpHandlers.NewCryptoHandlers(cryptoSyncUC)
		cryptoPurchaseHandler = httpHandlers.NewCryptoPurchaseHandlers(cryptoPurchaseRepo, binanceClient)

		// Initialize trade sync
		syncTradesUC = usecases.NewSyncTradesUseCase(
			binanceClient, bankAccountRepo, transactionRepo, cryptoPurchaseRepo,
			[]string{"SOLBRL", "ETHBRL", "BTCBRL", "USDCBRL", "XRPBRL", "BNBBRL"},
			"grid-bot-1",
		)
		cryptoHandler.SetSyncTradesUseCase(syncTradesUC)

		log.Println("✓ Binance API integration enabled")
	} else {
		cryptoPurchaseHandler = httpHandlers.NewCryptoPurchaseHandlers(cryptoPurchaseRepo, nil)
	}

	// Initialize brapi.dev client (B3 prices) and Yahoo Finance (dividends)
	brapiToken := os.Getenv("BRAPI_TOKEN")
	brapiClient := brapi.NewClient(brapiToken)
	yahooClient := yahoo.NewClient()
	stockSyncUC := usecases.NewStockSyncUseCase(brapiClient, bankAccountRepo)
	dividendSyncUC := usecases.NewDividendSyncUseCase(yahooClient, bankAccountRepo, transactionRepo)
	stockHandler := httpHandlers.NewStockHandlers(stockSyncUC)
	stockHandler.SetDividendUseCase(dividendSyncUC)
	log.Println("✓ B3 sync enabled (prices: brapi.dev, dividends: Yahoo Finance)")

	// Initialize Capital Contribution use cases and handlers
	capitalContributionRepo := persistence.NewCapitalContributionRepository(db)
	createCapitalContributionUC := usecases.NewCreateCapitalContributionUseCase(capitalContributionRepo, profileRepo)
	listCapitalContributionsUC := usecases.NewListCapitalContributionsUseCase(capitalContributionRepo)
	getCapitalContributionUC := usecases.NewGetCapitalContributionUseCase(capitalContributionRepo)
	updateCapitalContributionUC := usecases.NewUpdateCapitalContributionUseCase(capitalContributionRepo)
	deleteCapitalContributionUC := usecases.NewDeleteCapitalContributionUseCase(capitalContributionRepo)
	summaryCapitalContributionUC := usecases.NewGetCapitalContributionSummaryUseCase(capitalContributionRepo)
	capitalContributionHandler := httpHandlers.NewCapitalContributionHandlers(
		createCapitalContributionUC,
		listCapitalContributionsUC,
		getCapitalContributionUC,
		updateCapitalContributionUC,
		deleteCapitalContributionUC,
		summaryCapitalContributionUC,
	)

	// Initialize Company Asset use cases and handlers
	companyAssetRepo := persistence.NewCompanyAssetRepository(db)
	createCompanyAssetUC := usecases.NewCreateCompanyAssetUseCase(companyAssetRepo, profileRepo)
	listCompanyAssetsUC := usecases.NewListCompanyAssetsUseCase(companyAssetRepo)
	getCompanyAssetUC := usecases.NewGetCompanyAssetUseCase(companyAssetRepo)
	updateCompanyAssetUC := usecases.NewUpdateCompanyAssetUseCase(companyAssetRepo)
	deleteCompanyAssetUC := usecases.NewDeleteCompanyAssetUseCase(companyAssetRepo)
	companyAssetHandler := httpHandlers.NewCompanyAssetHandlers(
		createCompanyAssetUC,
		listCompanyAssetsUC,
		getCompanyAssetUC,
		updateCompanyAssetUC,
		deleteCompanyAssetUC,
	)

	// Cost Centers
	costCenterRepo := persistence.NewCostCenterRepository(db)
	createCostCenterUC := usecases.NewCreateCostCenterUseCase(costCenterRepo)
	listCostCentersUC := usecases.NewListCostCentersUseCase(costCenterRepo)
	getCostCenterUC := usecases.NewGetCostCenterUseCase(costCenterRepo)
	updateCostCenterUC := usecases.NewUpdateCostCenterUseCase(costCenterRepo)
	deleteCostCenterUC := usecases.NewDeleteCostCenterUseCase(costCenterRepo)
	costCenterHandler := httpHandlers.NewCostCenterHandlers(createCostCenterUC, listCostCentersUC, getCostCenterUC, updateCostCenterUC, deleteCostCenterUC)

	// Marketing Campaigns
	campaignRepo := persistence.NewMarketingCampaignRepository(db)
	createCampaignUC := usecases.NewCreateCampaignUseCase(campaignRepo)
	listCampaignsUC := usecases.NewListCampaignsUseCase(campaignRepo)
	getCampaignUC := usecases.NewGetCampaignUseCase(campaignRepo)
	updateCampaignUC := usecases.NewUpdateCampaignUseCase(campaignRepo)
	deleteCampaignUC := usecases.NewDeleteCampaignUseCase(campaignRepo)
	getCampaignMetricsUC := usecases.NewGetCampaignWithMetricsUseCase(campaignRepo)
	campaignHandler := httpHandlers.NewMarketingCampaignHandlers(createCampaignUC, listCampaignsUC, getCampaignUC, updateCampaignUC, deleteCampaignUC, getCampaignMetricsUC)

	// API v1 routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Invariant report: every stored balance and invoice total against the
	// transactions that justify them. Read-only by design.
	checkInvariantsUC := usecases.NewCheckInvariantsUseCase(bankAccountRepo, transactionRepo, invoiceRepo)
	invariantsHandler := httpHandlers.NewInvariantsHandlers(checkInvariantsUC)
	apiRouter.HandleFunc("/health/invariants", invariantsHandler.Check).Methods("GET")

	// Profile routes
	apiRouter.HandleFunc("/profiles", profileHandler.List).Methods("GET")
	apiRouter.HandleFunc("/profiles", profileHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/profiles/{id}", profileHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/profiles/{id}", profileHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/profiles/{id}", profileHandler.Delete).Methods("DELETE")

	// Bank Account routes
	apiRouter.HandleFunc("/bank-accounts", bankAccountHandler.List).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts", bankAccountHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/bank-accounts/reorder", bankAccountHandler.Reorder).Methods("PUT")
	apiRouter.HandleFunc("/bank-accounts/maturities", bankAccountHandler.UpcomingMaturities).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/bank-accounts/{id}", bankAccountHandler.Delete).Methods("DELETE")
	apiRouter.HandleFunc("/bank-accounts/{id}/recalculate-balance", bankAccountHandler.RecalculateBalance).Methods("POST")
	apiRouter.HandleFunc("/bank-accounts/{id}/sell", bankAccountHandler.Sell).Methods("POST")
	apiRouter.HandleFunc("/bank-accounts/{id}/credit-usage", bankAccountHandler.CreditUsage).Methods("GET")
	apiRouter.HandleFunc("/bank-accounts/close-month", bankAccountHandler.CloseMonth).Methods("POST")

	// Accounts routes (placeholder)
	apiRouter.HandleFunc("/accounts", handlers.NotImplemented).Methods("GET", "POST")

	// Category routes
	apiRouter.HandleFunc("/categories", categoryHandler.List).Methods("GET")
	apiRouter.HandleFunc("/categories", categoryHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/categories/{id}", categoryHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/categories/{id}", categoryHandler.Delete).Methods("DELETE")

	// Transaction routes
	apiRouter.HandleFunc("/transactions/daily-balances", transactionHandler.DailyBalances).Methods("GET")
	apiRouter.HandleFunc("/transactions/cashflow-summary", transactionHandler.CashflowSummary).Methods("GET")
	apiRouter.HandleFunc("/transactions/expense-analysis", transactionHandler.ExpenseAnalysis).Methods("GET")
	apiRouter.HandleFunc("/transactions/financial-summary", transactionHandler.FinancialSummary).Methods("GET")
	apiRouter.HandleFunc("/transactions", transactionHandler.List).Methods("GET")
	apiRouter.HandleFunc("/transactions", transactionHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/transactions/{id}", transactionHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/transactions/{id}", transactionHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/transactions/{id}/status", transactionHandler.UpdateStatus).Methods("PUT")
	apiRouter.HandleFunc("/recurring-transactions", recurringHandler.List).Methods("GET")
	apiRouter.HandleFunc("/recurring-transactions", recurringHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/recurring-transactions/pending", recurringHandler.Pending).Methods("GET")
	apiRouter.HandleFunc("/recurring-transactions/process", recurringHandler.Process).Methods("POST")
	apiRouter.HandleFunc("/recurring-transactions/{id}", recurringHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/recurring-transactions/{id}/status", recurringHandler.UpdateStatus).Methods("PATCH")
	apiRouter.HandleFunc("/recurring-transactions/{id}", recurringHandler.Delete).Methods("DELETE")

	apiRouter.HandleFunc("/budgets/summary", budgetHandler.Summary).Methods("GET")
	apiRouter.HandleFunc("/budgets", budgetHandler.List).Methods("GET")
	apiRouter.HandleFunc("/budgets", budgetHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/budgets/{id}", budgetHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/budgets/{id}", budgetHandler.Delete).Methods("DELETE")

	// Goal routes
	apiRouter.HandleFunc("/goals", goalHandler.List).Methods("GET")
	apiRouter.HandleFunc("/goals", goalHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/goals/reorder", goalHandler.Reorder).Methods("PUT")
	apiRouter.HandleFunc("/goals/{id}", goalHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/goals/{id}", goalHandler.Delete).Methods("DELETE")
	apiRouter.HandleFunc("/goals/{id}/add-amount", goalHandler.AddAmount).Methods("POST")
	apiRouter.HandleFunc("/goals/{id}/status", goalHandler.UpdateStatus).Methods("PATCH")

	apiRouter.HandleFunc("/transactions/{id}", transactionHandler.Delete).Methods("DELETE")

	// Capital Contribution routes (aportes do sócio)
	apiRouter.HandleFunc("/capital-contributions/summary", capitalContributionHandler.Summary).Methods("GET")
	apiRouter.HandleFunc("/capital-contributions", capitalContributionHandler.List).Methods("GET")
	apiRouter.HandleFunc("/capital-contributions", capitalContributionHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/capital-contributions/{id}", capitalContributionHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/capital-contributions/{id}", capitalContributionHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/capital-contributions/{id}", capitalContributionHandler.Delete).Methods("DELETE")

	// Company Asset routes (ativos da empresa)
	apiRouter.HandleFunc("/company-assets", companyAssetHandler.List).Methods("GET")
	apiRouter.HandleFunc("/company-assets", companyAssetHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/company-assets/{id}", companyAssetHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/company-assets/{id}", companyAssetHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/company-assets/{id}", companyAssetHandler.Delete).Methods("DELETE")

	// Cost Center routes (centros de custo)
	apiRouter.HandleFunc("/cost-centers", costCenterHandler.List).Methods("GET")
	apiRouter.HandleFunc("/cost-centers", costCenterHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/cost-centers/{id}", costCenterHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/cost-centers/{id}", costCenterHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/cost-centers/{id}", costCenterHandler.Delete).Methods("DELETE")

	// Marketing Campaign routes (campanhas de marketing)
	apiRouter.HandleFunc("/marketing-campaigns", campaignHandler.List).Methods("GET")
	apiRouter.HandleFunc("/marketing-campaigns", campaignHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/marketing-campaigns/{id}", campaignHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/marketing-campaigns/{id}", campaignHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/marketing-campaigns/{id}", campaignHandler.Delete).Methods("DELETE")
	apiRouter.HandleFunc("/marketing-campaigns/{id}/metrics", campaignHandler.GetWithMetrics).Methods("GET")

	// Invoice routes
	apiRouter.HandleFunc("/invoices", invoiceHandler.List).Methods("GET")
	apiRouter.HandleFunc("/invoices", invoiceHandler.Create).Methods("POST")
	apiRouter.HandleFunc("/invoices/auto-close", invoiceHandler.AutoClose).Methods("POST")
	apiRouter.HandleFunc("/invoices/current", invoiceHandler.GetCurrent).Methods("GET")
	apiRouter.HandleFunc("/invoices/{id}", invoiceHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/invoices/{id}", invoiceHandler.Update).Methods("PUT")
	apiRouter.HandleFunc("/invoices/{id}/close", invoiceHandler.Close).Methods("POST")
	apiRouter.HandleFunc("/invoices/{id}/pay", invoiceHandler.Pay).Methods("POST")
	apiRouter.HandleFunc("/invoices/{id}/recalculate", invoiceHandler.Recalculate).Methods("POST")

	// Crypto routes
	if cryptoHandler != nil {
		apiRouter.HandleFunc("/crypto/sync", cryptoHandler.Sync).Methods("POST")
		apiRouter.HandleFunc("/crypto/sync-trades", cryptoHandler.SyncTrades).Methods("POST")
	}
	apiRouter.HandleFunc("/crypto/purchases", cryptoPurchaseHandler.List).Methods("GET")
	apiRouter.HandleFunc("/crypto/purchases", cryptoPurchaseHandler.Create).Methods("POST")

	// Stock/FII routes (B3 via brapi.dev)
	apiRouter.HandleFunc("/stocks/sync-prices", stockHandler.SyncPrices).Methods("POST")
	apiRouter.HandleFunc("/stocks/sync-dividends", stockHandler.SyncDividends).Methods("POST")

	// Benchmark routes (Yahoo Finance + CDI)
	benchmarkHandler := httpHandlers.NewBenchmarkHandler(yahooClient)
	apiRouter.HandleFunc("/benchmarks/returns", benchmarkHandler.GetReturns()).Methods("GET")

	// FII market analysis routes
	fiiMarketHandler := httpHandlers.NewFIIMarketHandler(yahooClient)
	apiRouter.HandleFunc("/fiis/market", fiiMarketHandler.GetMarketFIIs()).Methods("GET")
	return &App{
		Router: router,
		Jobs: Jobs{
			AutoCloseInvoices: autoCloseInvoicesUC,
			SyncTrades:        syncTradesUC,
			StockSync:         stockSyncUC,
			DividendSync:      dividendSyncUC,
			CloseMonth:        closeMonthUC,
		},
	}, nil
}
