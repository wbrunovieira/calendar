import type { Transaction, RecurringTransaction } from '@/types/finances';

/** Normalize any date string to YYYY-MM-DD. */
export const extractDateStr = (dateStr: string): string => {
  if (!dateStr) return '';
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr;
  if (dateStr.includes('T')) return dateStr.split('T')[0];
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '';
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

/** Absolute difference in whole days between two YYYY-MM-DD dates. */
const daysBetween = (a: string, b: string): number => {
  const [ay, am, ad] = a.split('-').map(Number);
  const [by, bm, bd] = b.split('-').map(Number);
  if (!ay || !by) return 0;
  const da = new Date(ay, am - 1, ad).getTime();
  const db = new Date(by, bm - 1, bd).getTime();
  return Math.floor(Math.abs(da - db) / (1000 * 60 * 60 * 24));
};

/**
 * Is a recurring obligation already satisfied for its current occurrence?
 *
 * Matches a non-cancelled transaction (CONFIRMED or PLANNED) of the same type +
 * description, on the same account (when the recurring declares one), occurring in
 * the same month as the occurrence (or within 7 days).
 */
export const isRecurringSatisfied = (
  rt: RecurringTransaction,
  transactions: Transaction[],
): boolean => {
  const rtNextDate = extractDateStr(rt.nextOccurrence);
  const rtDesc = rt.description.toLowerCase().trim();

  return transactions.some((tx) => {
    if (tx.status !== 'CONFIRMED' && tx.status !== 'PLANNED') return false;
    if (tx.type !== rt.type) return false;

    // NOTE: amount is intentionally NOT compared. Recurring expenses vary month to
    // month (e.g. "Tim Celular" billed 99 but actually 109,99). Matching by amount
    // kept the obligation "pending" after it was paid at the real amount.
    if (rt.bankAccountId && tx.bankAccountId !== rt.bankAccountId) return false;

    const txDesc = tx.description.toLowerCase().trim();
    if (txDesc !== rtDesc) return false;

    const txDate = extractDateStr(tx.occurredOn);
    if (!txDate || !rtNextDate) return false;

    if (txDate.substring(0, 7) === rtNextDate.substring(0, 7)) return true;
    return daysBetween(txDate, rtNextDate) <= 7;
  });
};

/**
 * Active recurring obligations that are due (nextOccurrence <= today) and not yet
 * satisfied by a matching transaction.
 */
export const findPendingRecurrings = (
  recurringTransactions: RecurringTransaction[],
  transactions: Transaction[],
  today: string,
): RecurringTransaction[] => {
  return recurringTransactions
    .filter((rt) => {
      if (rt.status !== 'ACTIVE') return false;
      const nextDate = extractDateStr(rt.nextOccurrence);
      if (!nextDate || nextDate > today) return false;
      return !isRecurringSatisfied(rt, transactions);
    })
    .sort((a, b) =>
      extractDateStr(a.nextOccurrence).localeCompare(extractDateStr(b.nextOccurrence)),
    );
};
