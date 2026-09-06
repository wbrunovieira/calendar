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
 * What a single bill still owes.
 *
 * The API derives `amountRemaining` on every read; the fallback covers payloads from
 * before that field existed. A bill marked PAID with no recorded paidAmount owes
 * nothing — the status is the only evidence there is.
 *
 * Use this anywhere a bill's debt is shown or paid. Sending `invoice.amount` to the
 * pay endpoint on a partially paid bill debits the funding account a second time for
 * money that is no longer owed.
 */
export function amountLeftOn(invoice: Invoice): number {
  if (typeof invoice.amountRemaining === 'number') return Math.max(0, invoice.amountRemaining);
  const paid = invoice.paidAmount ?? (invoice.status === 'PAID' ? invoice.amount : 0);
  return Math.max(0, invoice.amount - paid);
}

/**
 * What is still owed across a card's bills.
 *
 * Always billed minus paid, whatever the status. Branching on status instead put a
 * partially paid bill in the "owes everything" case and counted money that had
 * already left the checking account.
 *
 * A bill marked PAID with no paidAmount recorded owes nothing — the status is the
 * only evidence there is. Everything else defaults to nothing paid yet, which also
 * covers legacy rows where a partial payment was stored as PAID.
 */
export function outstandingOn(invoices: Invoice[]): number {
  return invoices.reduce((total, inv) => total + amountLeftOn(inv), 0);
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
