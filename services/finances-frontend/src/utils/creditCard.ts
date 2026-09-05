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

  const outstanding = invoices
    .filter((inv) => inv.status !== 'PAID' && inv.amount > 0)
    .reduce((total, inv) => total + inv.amount, 0);

  return {
    outstanding,
    available: limit - outstanding,
    usagePercent: limit > 0 ? (outstanding / limit) * 100 : 0,
  };
}
