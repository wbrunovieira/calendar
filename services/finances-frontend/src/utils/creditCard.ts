import type { BankAccount, Invoice } from '@/types/finances';

/**
 * How much of a credit card's limit is actually committed.
 *
 * Outstanding is everything still owed: the open invoice AND any earlier one not yet
 * paid — a closed bill, whatever an instalment plan or revolving credit left behind.
 * That is what the issuer blocks against the limit. Deriving it from the open invoice
 * alone makes a card carrying older debt look far emptier than it is; deriving it from
 * the account balance is worse, because this system only moves a card's balance when
 * an invoice is paid.
 *
 * The API answers the same question at GET /bank-accounts/{id}/credit-usage, which is
 * the canonical source for anything that is not this screen.
 */
export function getCardUsage(
  account: Pick<BankAccount, 'creditLimit'>,
  invoices: Invoice[],
): { outstanding: number; available: number; usagePercent: number } {
  const limit = account.creditLimit || 0;

  const outstanding = outstandingOn(invoices);

  return {
    outstanding,
    available: limit - outstanding,
    usagePercent: limit > 0 ? (outstanding / limit) * 100 : 0,
  };
}

/**
 * What is still owed across a card's bills.
 *
 * A bill is marked PAID the moment anything is paid against it, however small, so a
 * partial payment has to keep its remainder in view — dropping the bill whole is how
 * a card owing R$301,74 reports itself as empty.
 */
export function outstandingOn(invoices: Invoice[]): number {
  return invoices.reduce((total, inv) => {
    if (inv.status !== 'PAID') return total + Math.max(0, inv.amount);
    return total + Math.max(0, inv.amount - (inv.paidAmount ?? inv.amount));
  }, 0);
}

/**
 * Total credit-card debt across several cards — what net worth has to subtract.
 * Counting only the open invoice leaves out a closed bill still unpaid or an
 * instalment plan, and overstates net worth by exactly that much.
 */
export function getCardsOutstanding(
  cards: { id: string }[],
  invoicesByCard: Record<string, Invoice[]>,
): number {
  return cards.reduce((total, card) => total + outstandingOn(invoicesByCard[card.id] ?? []), 0);
}
