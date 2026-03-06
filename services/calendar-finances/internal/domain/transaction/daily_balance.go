package transaction

import (
	"sort"
)

type DailyBalance struct {
	Date     string  `json:"date"`
	Balance  float64 `json:"balance"`
	DayTotal float64 `json:"dayTotal"`
}

// CalculateDailyBalances computes running daily balances for a specific bank account.
// bankAccountID is used to determine the sign of transfers (outgoing vs incoming).
// If empty, transfers with DestinationAccountID are treated as outgoing.
func CalculateDailyBalances(txs []*Transaction, currentBalance float64, bankAccountID string) []DailyBalance {
	if len(txs) == 0 {
		return nil
	}

	// Separate confirmed and planned, ignore cancelled
	type dayEntry struct {
		confirmedTotal float64
		plannedTotal   float64
	}

	days := map[string]*dayEntry{}
	var confirmedSum float64

	for _, tx := range txs {
		if tx.Status == StatusCancelled {
			continue
		}

		dateStr := tx.OccurredOn.Format("2006-01-02")
		if days[dateStr] == nil {
			days[dateStr] = &dayEntry{}
		}

		amount := txAmountForAccount(tx, bankAccountID)

		if tx.Status == StatusConfirmed {
			days[dateStr].confirmedTotal += amount
			confirmedSum += amount
		} else if tx.Status == StatusPlanned {
			days[dateStr].plannedTotal += amount
		}
	}

	// Collect and sort dates
	dateKeys := make([]string, 0, len(days))
	for d := range days {
		dateKeys = append(dateKeys, d)
	}
	sort.Strings(dateKeys)

	// Derive starting balance: currentBalance is the balance after all confirmed txs
	startBalance := round2(currentBalance - confirmedSum)

	result := make([]DailyBalance, 0, len(dateKeys))
	runningBalance := startBalance

	for _, d := range dateKeys {
		entry := days[d]
		dayTotal := round2(entry.confirmedTotal + entry.plannedTotal)
		runningBalance = round2(runningBalance + dayTotal)

		result = append(result, DailyBalance{
			Date:     d,
			Balance:  runningBalance,
			DayTotal: dayTotal,
		})
	}

	return result
}

func txAmountForAccount(tx *Transaction, bankAccountID string) float64 {
	switch tx.Type {
	case TypeIncome:
		return tx.Amount
	case TypeExpense:
		return -tx.Amount
	case TypeTransfer:
		// If this account is the destination, the transfer is incoming (+)
		if bankAccountID != "" && tx.DestinationAccountID != nil && *tx.DestinationAccountID == bankAccountID {
			return tx.Amount
		}
		// Otherwise it's outgoing (-)
		if tx.DestinationAccountID != nil {
			return -tx.Amount
		}
		return tx.Amount
	}
	return 0
}
