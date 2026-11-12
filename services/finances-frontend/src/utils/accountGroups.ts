import type { AccountType, BankAccount } from '@/types/finances';

/**
 * Accounts split by what they MEAN, so a debt is never reported as money
 * available.
 *
 * The home screen used to group them by exclusion — "not a credit card and not
 * an investment is available" — which made every type the backend did not have
 * yet default into cash. On 12/11/2026 that put two LIABILITY accounts (a loan
 * and a debt to family, together R$ 3.865,68 owed) inside a card headed
 * "Disponivel", which then read -R$ 3.414,81 while R$ 450,87 was what could
 * actually be spent.
 *
 * Every group is a POSITIVE list. Anything unrecognised comes back in
 * `unclassified` — visible, never silently folded into the cash total.
 */
export interface AccountGroups {
  /** What can actually be spent today. */
  cash: BankAccount[];
  /** Debts that are not credit cards: loans, money owed to people. */
  liabilities: BankAccount[];
  creditCards: BankAccount[];
  investments: BankAccount[];
  /** Types this function was never taught. Shown apart rather than assumed. */
  unclassified: BankAccount[];
}

const CASH_TYPES: AccountType[] = ['CHECKING', 'SAVINGS', 'CASH', 'EXCHANGE', 'WALLET', 'OTHER'];

/** A deactivated account is not part of any picture — it is off the screen. */
const isLive = (account: BankAccount) => account.isActive !== false;

export function groupAccounts(accounts: BankAccount[]): AccountGroups {
  const groups: AccountGroups = {
    cash: [],
    liabilities: [],
    creditCards: [],
    investments: [],
    unclassified: [],
  };

  for (const account of accounts.filter(isLive)) {
    if (account.type === 'CREDIT_CARD') groups.creditCards.push(account);
    else if (account.type === 'INVESTMENT') groups.investments.push(account);
    else if (account.type === 'LIABILITY') groups.liabilities.push(account);
    else if (CASH_TYPES.includes(account.type)) groups.cash.push(account);
    else groups.unclassified.push(account);
  }

  return groups;
}

/**
 * Whether this account holds money that is actually available — as opposed to a
 * debt (LIABILITY, CREDIT_CARD) or something put away (INVESTMENT).
 *
 * Use it anywhere a "saldo em contas" total is built, so that total means the
 * same thing on every screen.
 */
export function isCashLike(account: BankAccount): boolean {
  return CASH_TYPES.includes(account.type);
}

export function sumBalances(accounts: BankAccount[]): number {
  return accounts.reduce((total, account) => total + account.currentBalance, 0);
}

/**
 * What is owed on these accounts, as a positive number.
 *
 * Liability balances are stored negative, so this is their sum flipped. A
 * settled account reports zero and an overpaid one reports a negative debt —
 * neither is rounded away, because hiding an overpayment is how it stops being
 * investigated.
 */
export function debtTotal(liabilities: BankAccount[]): number {
  const owed = -sumBalances(liabilities);
  // Negating a zero sum yields -0, which formats as "-R$ 0,00" — a settled debt
  // rendered as if something were wrong with it.
  return owed === 0 ? 0 : owed;
}
