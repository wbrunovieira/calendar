package bankaccount

type AccountType string

const (
	AccountTypeChecking   AccountType = "CHECKING"
	AccountTypeSavings    AccountType = "SAVINGS"
	AccountTypeInvestment AccountType = "INVESTMENT"
	AccountTypeCreditCard AccountType = "CREDIT_CARD"
	AccountTypeCash       AccountType = "CASH"
	AccountTypeExchange   AccountType = "EXCHANGE" // Corretoras de cripto (Binance, OKX, Rabbit)
	AccountTypeWallet     AccountType = "WALLET"   // Carteiras de cripto (Ledger, MetaMask)
	AccountTypeOther      AccountType = "OTHER"
)

// InvestmentType represents the type of investment product
type InvestmentType string

const (
	InvestmentTypeSavingsBox InvestmentType = "SAVINGS_BOX" // Caixinha (Nubank, etc.)
	InvestmentTypeCDB        InvestmentType = "CDB"         // Certificado de Depósito Bancário
	InvestmentTypeLCI        InvestmentType = "LCI"         // Letra de Crédito Imobiliário
	InvestmentTypeLCA        InvestmentType = "LCA"         // Letra de Crédito do Agronegócio
	InvestmentTypeStocks     InvestmentType = "STOCKS"      // Ações
	InvestmentTypeFunds      InvestmentType = "FUNDS"       // Fundos de investimento
	InvestmentTypeFII        InvestmentType = "FII"         // Fundos Imobiliários
	InvestmentTypeCrypto     InvestmentType = "CRYPTO"      // Criptomoedas
	InvestmentTypeTreasury   InvestmentType = "TREASURY"    // Tesouro Direto
	InvestmentTypeOther      InvestmentType = "OTHER"       // Outros
)

// YieldType represents how the yield/return is calculated
type YieldType string

const (
	YieldTypeFixed         YieldType = "FIXED"          // Taxa fixa (ex: 12% a.a.)
	YieldTypeCDIPercentage YieldType = "CDI_PERCENTAGE" // Percentual do CDI (ex: 100% CDI)
	YieldTypeIPCAPlus      YieldType = "IPCA_PLUS"      // IPCA + taxa (ex: IPCA + 5%)
	YieldTypeVariable      YieldType = "VARIABLE"       // Taxa variável (ações, fundos, crypto)
)

func IsValidInvestmentType(investmentType InvestmentType) bool {
	switch investmentType {
	case InvestmentTypeSavingsBox, InvestmentTypeCDB, InvestmentTypeLCI,
		InvestmentTypeLCA, InvestmentTypeStocks, InvestmentTypeFunds,
		InvestmentTypeFII, InvestmentTypeCrypto, InvestmentTypeTreasury, InvestmentTypeOther:
		return true
	default:
		return false
	}
}

func IsValidYieldType(yieldType YieldType) bool {
	switch yieldType {
	case YieldTypeFixed, YieldTypeCDIPercentage, YieldTypeIPCAPlus, YieldTypeVariable:
		return true
	default:
		return false
	}
}
