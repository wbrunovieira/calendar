import type { Transaction, BankAccount } from '@/types/finances';

export interface CashflowSummary {
  income: number; // real income (excludes settlements & exchange accounts)
  incomeCount: number;
  incomeYield: number; // rendimentos (juros/yield)
  incomeYieldCount: number;
  incomeOther: number; // income that is not yield
  incomeOtherCount: number;
  expense: number; // real expenses (excludes settlements & exchange accounts)
  expenseCount: number;
}

/** Yield/interest income ("rendimento da conta", "Rendimento CDB", ...). */
export const isYield = (tx: Transaction): boolean => /rendimento/i.test(tx.description);

/**
 * Internal settlements that are NOT real income/expense and must be excluded from the
 * monthly cashflow:
 *  - credit-card invoice payments. The Pay flow records them as an INCOME credit on the
 *    card side AND as an EXPENSE "Pagamento fatura ..." on the funding account. Counting
 *    either as income/expense is wrong (the card purchases are already the real expense).
 */
export const isSettlement = (tx: Transaction, account?: BankAccount): boolean => {
  if (tx.type === 'INCOME' && account?.type === 'CREDIT_CARD') return true;
  if (/pagamento\s+fatura/i.test(tx.description)) return true;
  return false;
};

/** Summarize confirmed cashflow, separating yield from other income and dropping settlements. */
export const summarizeCashflow = (
  transactions: Transaction[],
  accounts: BankAccount[],
): CashflowSummary => {
  const accountById = new Map(accounts.map((a) => [a.id, a]));
  const exchangeIds = new Set(
    accounts.filter((a) => a.type === 'EXCHANGE').map((a) => a.id),
  );

  let income = 0;
  let incomeCount = 0;
  let incomeYield = 0;
  let incomeYieldCount = 0;
  let expense = 0;
  let expenseCount = 0;

  for (const tx of transactions) {
    if (tx.status !== 'CONFIRMED') continue;
    if (exchangeIds.has(tx.bankAccountId)) continue;
    if (isSettlement(tx, accountById.get(tx.bankAccountId))) continue;

    if (tx.type === 'INCOME') {
      income += tx.amount;
      incomeCount += 1;
      if (isYield(tx)) {
        incomeYield += tx.amount;
        incomeYieldCount += 1;
      }
    } else if (tx.type === 'EXPENSE') {
      expense += tx.amount;
      expenseCount += 1;
    }
  }

  return {
    income,
    incomeCount,
    incomeYield,
    incomeYieldCount,
    incomeOther: income - incomeYield,
    incomeOtherCount: incomeCount - incomeYieldCount,
    expense,
    expenseCount,
  };
};
