package transaction

import (
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/category"
)

// IsSettlement reports whether the transaction only moves money around instead of
// earning or spending it, and therefore must stay out of a cashflow view.
//
// Two shapes qualify, and both are recognised structurally rather than by reading the
// description — a description is free text a person types, and a rule that depends on
// the wording silently breaks the month when someone writes it differently:
//
//   - the INCOME credit that lands on a credit card when its bill is paid. The
//     purchases on that card already were the expense; counting the payment as income
//     would cancel them out.
//   - money moving INTO another account of the same owner — paying a card bill from a
//     checking account, funding an investment. Nothing was earned or spent.
//
// `account` is where the transaction sits and `destination` is the other end of a
// transfer, if any. A caller that cannot resolve them passes nil, and the transaction
// is treated as real — better to count a settlement than to drop real money.
func (t *Transaction) IsSettlement(account, destination *bankaccount.BankAccount) bool {
	if account != nil && account.IsCreditCard() && t.Type == TypeIncome {
		return true
	}

	return destination != nil
}

// IsYield reports whether income came from the balance producing it — interest, a
// dividend, a fund's daily yield — rather than from work or a sale.
//
// The answer is the category's DRE bucket, which is the accounting classification the
// user already maintains. Reading it off the description instead would call
// "Rendimento da venda do carro" yield and miss "juros da poupanca", and the two
// buckets are taxed and read differently.
func (t *Transaction) IsYield(cat *category.Category) bool {
	if t.Type != TypeIncome || cat == nil || cat.ClassificationDRE == nil {
		return false
	}
	return *cat.ClassificationDRE == category.DREFinancial
}
