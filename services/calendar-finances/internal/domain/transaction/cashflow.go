package transaction

import (
	"strings"

	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
)

// maxCategoryDepth stops a malformed parent chain from looping forever.
const maxCategoryDepth = 16

// IsSettlement reports whether the transaction only moves money around instead of
// earning or spending it, and therefore must stay out of a cashflow view.
//
// Three shapes qualify, all recognised structurally rather than by reading the
// description — a description is free text a person types, and a rule that depends on
// the wording silently breaks the month when someone writes it differently:
//
//   - the INCOME credit that lands on a credit card when its bill is paid. The
//     purchases on that card already were the expense.
//   - a TRANSFER into another account of the same owner — paying a card bill from
//     checking, funding an investment. Nothing was earned or spent.
//   - an expense that pays an invoice. Bills entered by hand carry the invoice link
//     but no destination, and they are the majority of the payments in this database.
//
// An EXPENSE carrying a destination is NOT a settlement: that is the source leg of a
// cross-profile transfer, where money genuinely left this profile.
//
// `account` is where the transaction sits and `destination` is the other end of a
// transfer, if any. A caller that cannot resolve them passes nil, and the transaction
// is treated as real — better to count a settlement than to drop real money.
func (t *Transaction) IsSettlement(account, destination *bankaccount.BankAccount) bool {
	if account != nil && account.IsCreditCard() && t.Type == TypeIncome {
		return true
	}
	if t.Type == TypeExpense && t.InvoiceID != nil && (account == nil || !account.IsCreditCard()) {
		return true
	}
	return t.Type == TypeTransfer && destination != nil
}

// IsYieldIn reports whether income came from the balance producing it — interest, a
// dividend, a fund's daily yield — rather than from work or a sale.
//
// The answer is the category's DRE bucket, inherited from ancestors exactly as the
// financial report resolves it: a subcategory "Rendimento CDB" under a FINANCIAL
// "Rendimentos" is yield, and reading only the leaf would disagree with the report on
// the same data. `categories` indexes the profile's categories by id so the chain can
// be walked without a repository call per transaction.
func (t *Transaction) IsYieldIn(cat *category.Category, categories map[string]*category.Category) bool {
	if t.Type != TypeIncome {
		return false
	}

	switch bucket := dreBucketOf(cat, categories); {
	case bucket == category.DREFinancial:
		return true
	case bucket != "":
		// A classified branch is the accounting answer, whatever the wording says.
		return false
	}

	// Nothing in the tree says anything. Almost nothing is classified yet — including
	// the personal "Rendimentos" that receives the daily account interest and every
	// dividend — so trusting the classification alone would silently report zero yield
	// for a whole profile. The wording decides until the categories are filled in, and
	// classifying one is what turns this off for it.
	return looksLikeYield(t.Description)
}

func looksLikeYield(description string) bool {
	return strings.Contains(strings.ToLower(description), "rendiment")
}

// dreBucketOf walks up to the nearest ancestor carrying a classification.
func dreBucketOf(cat *category.Category, categories map[string]*category.Category) category.ClassificationDRE {
	for depth := 0; cat != nil && depth < maxCategoryDepth; depth++ {
		if cat.ClassificationDRE != nil && *cat.ClassificationDRE != "" {
			return *cat.ClassificationDRE
		}
		if cat.ParentID == nil {
			return ""
		}
		cat = categories[*cat.ParentID]
	}
	return ""
}
