package bankaccount

import "fmt"

// GetYieldDescription returns a human-readable description of the yield
func (ba *BankAccount) GetYieldDescription() string {
	if ba.YieldType == nil {
		return ""
	}

	rate := float64(0)
	if ba.YieldRate != nil {
		rate = *ba.YieldRate
	}

	switch *ba.YieldType {
	case YieldTypeFixed:
		return formatRate(rate) + "% a.a."
	case YieldTypeCDIPercentage:
		return formatRate(rate) + "% do CDI"
	case YieldTypeIPCAPlus:
		return "IPCA + " + formatRate(rate) + "%"
	case YieldTypeVariable:
		return "Variável"
	default:
		return ""
	}
}

func formatRate(rate float64) string {
	if rate == float64(int(rate)) {
		return fmt.Sprintf("%.0f", rate)
	}
	return fmt.Sprintf("%.2f", rate)
}

// CalculateReturn calculates the absolute return (current balance - initial balance)
func (ba *BankAccount) CalculateReturn() float64 {
	return ba.CurrentBalance - ba.InitialBalance
}

// CalculateReturnPercentage calculates the percentage return
func (ba *BankAccount) CalculateReturnPercentage() float64 {
	if ba.InitialBalance == 0 {
		return 0
	}
	return (ba.CalculateReturn() / ba.InitialBalance) * 100
}
