/**
 * How much of a credit card's limit is actually committed.
 *
 * The card's `currentBalance` is the whole outstanding debt — the open invoice, any
 * closed invoice still unpaid, and whatever an installment plan or revolving credit
 * left behind. That is what the bank blocks against the limit, so it is what
 * availability has to be derived from. Deriving it from the open invoice alone makes
 * a card with older debt look emptier than it is.
 *
 * A card owing money carries a negative balance; a positive one is a credit (a refund
 * that landed after the bill was paid) and commits nothing.
 */
export function getCardUsage(account: {
  creditLimit?: number;
  currentBalance: number;
  outstanding?: number;
  availableCredit?: number;
  creditUsagePercent?: number;
}): { outstanding: number; available: number; usagePercent: number } {
  // The API derives these on the entity, so every client reads the same numbers.
  // The local fallback keeps older payloads rendering instead of showing zeros.
  if (account.availableCredit !== undefined && account.outstanding !== undefined) {
    return {
      outstanding: account.outstanding,
      available: account.availableCredit,
      usagePercent: account.creditUsagePercent ?? 0,
    };
  }

  const limit = account.creditLimit || 0;
  const outstanding = Math.max(0, -(account.currentBalance || 0));

  return {
    outstanding,
    available: limit - outstanding,
    usagePercent: limit > 0 ? (outstanding / limit) * 100 : 0,
  };
}
