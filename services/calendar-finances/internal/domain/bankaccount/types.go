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
	// AccountTypeLiability is money OWED to someone who is not a card issuer: a
	// sister who covered a bill, a loan, a client advance still to return.
	//
	// The ledger could not express one. The only debt it knew was a credit card, so
	// a real R$ 875 owed to Bruno's sisters had to live in the notes field of a
	// forecast entry — with a paragraph explaining that the creditor had changed,
	// because the model had nowhere to say it.
	//
	// It carries what is owed as a NEGATIVE balance, the same convention a card
	// uses. It is not a card: no bill, no closing day, no limit.
	AccountTypeLiability AccountType = "LIABILITY"
)

// IsCash reports whether the balance of an account of this type is money that can
// be spent right now.
//
// It exists because the question gets asked constantly — "how much is in cash?" —
// and answering it by summing account balances is wrong in two directions at once:
// it leaves out nothing, and it counts investments that cannot be withdrawn and
// debts that are not money. I made that mistake by hand, and the point of putting
// it here is that the next caller does not have to remember.
func (t AccountType) IsCash() bool {
	switch t {
	case AccountTypeChecking, AccountTypeCash:
		return true
	default:
		return false
	}
}

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
