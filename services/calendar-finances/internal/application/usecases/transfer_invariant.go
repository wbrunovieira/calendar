package usecases

import (
	"fmt"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/transaction"
)

// TransferInvariant is one movement that does not have both of its sides.
//
// This is the double-entry property, and its absence was the one gap in the
// invariants report: accounts, invoices, cycles, instalments and payments were
// all checked, and nothing asked whether money that left an account arrived
// anywhere. The balance arithmetic already counts both legs of a same-profile
// transfer — source minus, destination plus — so a transfer pointing at nothing
// drains the source, credits nobody, and leaves every other check green.
//
// It is the shape of three open bugs: deleting a TRANSFER not reversing the
// destination, a cross-profile credit landing outside every bill, and a
// contribution crediting an account that never received the money.
type TransferInvariant struct {
	TransactionID string  `json:"transactionId"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	// Problem says, in one line, which side is missing and how.
	Problem string `json:"problem"`
}

// settled reports whether this row claims money actually moved. A cancelled or
// reversed row moved nothing, so its counterpart is not owed.
func settled(t *transaction.Transaction) bool {
	return t.Status != transaction.StatusCancelled && t.Status != transaction.StatusReversed
}

// checkTransfers walks the ledger looking for one-sided money.
//
// It takes the rows and accounts already in hand rather than querying: the
// report builds this from the same list it uses for the other checks, and a
// second pass over the database to answer the same question is how two checks
// start disagreeing about what is in it.
func checkTransfers(txs []*transaction.Transaction, accounts map[string]*bankaccount.BankAccount) *CheckInvariantsResult {
	result := &CheckInvariantsResult{OK: true}
	byID := make(map[string]*transaction.Transaction, len(txs))
	for _, t := range txs {
		byID[t.ID] = t
	}

	report := func(t *transaction.Transaction, problem string) {
		result.TransferDrifts = append(result.TransferDrifts, TransferInvariant{
			TransactionID: t.ID, Description: t.Description, Amount: t.Amount, Problem: problem,
		})
		result.OK = false
	}

	for _, t := range txs {
		if !settled(t) {
			continue
		}

		if t.Type == transaction.TypeTransfer && t.LinkedTransactionID == nil {
			switch {
			case t.DestinationAccountID == nil || *t.DestinationAccountID == "":
				report(t, "transferencia sem conta de destino: o dinheiro saiu e nao chegou em lugar nenhum")
			case accounts[*t.DestinationAccountID] == nil:
				report(t, fmt.Sprintf("conta de destino %s nao existe", *t.DestinationAccountID))
			}
		}

		if t.LinkedTransactionID == nil {
			continue
		}
		sibling := byID[*t.LinkedTransactionID]
		switch {
		case sibling == nil:
			report(t, fmt.Sprintf("a outra perna (%s) nao existe: o dinheiro atravessou so de um lado", *t.LinkedTransactionID))
		case !settled(sibling):
			report(t, fmt.Sprintf("a outra perna (%s) esta %s enquanto esta continua valendo", sibling.ID, sibling.Status))
		case sibling.LinkedTransactionID == nil || *sibling.LinkedTransactionID != t.ID:
			report(t, fmt.Sprintf("o vinculo com %s nao e mutuo: so este lado aponta para o outro", sibling.ID))
		case round2(sibling.Amount) != round2(t.Amount):
			// Reported once, from the lower id. Both legs can see this one, and a
			// daily message that prints one problem twice teaches the reader to
			// skim it. The cases above are each visible from one side only.
			if t.ID < sibling.ID {
				report(t, fmt.Sprintf("as duas pernas discordam do valor: %.2f aqui e %.2f em %s", t.Amount, sibling.Amount, sibling.ID))
			}
		}
	}

	return result
}
