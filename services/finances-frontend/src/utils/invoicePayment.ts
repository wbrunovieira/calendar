import type { BankAccount, Invoice } from '@/types/finances';
import { amountLeftOn } from './creditCard';

export interface PaymentPreview {
  /** What the debit will be worth. */
  amount: number;
  /** The account the money will leave, by name. */
  fundingAccountName: string | null;
  /** Why this payment should not be recorded, if it should not. */
  blockedReason: string | null;
  /**
   * The card names no funding account. Not a refusal — the API marks the bill
   * paid and skips the transfer — but the user should know no money will be
   * recorded as leaving anywhere.
   */
  noFundingAccount?: boolean;
}

/**
 * Describes the ledger entry "Pagar" is about to create, before it is created.
 *
 * Paying a bill writes a real debit against a real account. When the bill was in
 * fact settled outside the system — the bank converted it into instalments, say —
 * that debit is money that never moved, and it breaks a balance that reconciles
 * to the cent against the bank. The button gave no hint of any of this: one
 * click, one irreversible entry.
 *
 * Saying the amount and the account out loud is what turns it into a decision.
 */
export function describeInvoicePayment(
  invoice: Invoice,
  card: Pick<BankAccount, 'id' | 'linkedAccountId'> | undefined,
  accounts: Pick<BankAccount, 'id' | 'name'>[],
): PaymentPreview {
  const amount = amountLeftOn(invoice);

  if (invoice.status === 'REFINANCED') {
    return {
      amount,
      fundingAccountName: null,
      blockedReason:
        'Esta fatura foi convertida em parcelas pelo banco. Ninguém a pagou, e as parcelas já estão lançadas.',
    };
  }
  if (amount <= 0) {
    return { amount, fundingAccountName: null, blockedReason: 'Esta fatura não deve nada.' };
  }

  const fundingId = card?.linkedAccountId;
  const funding = fundingId ? accounts.find((a) => a.id === fundingId) : undefined;

  // No funding account is not a refusal: the API handles that case on purpose,
  // marking the bill paid and skipping the transfer. Blocking it here made cards
  // without a linked account unpayable from the screen while the backend was
  // perfectly willing. Saying so is enough.
  return {
    amount,
    fundingAccountName: funding?.name ?? null,
    blockedReason: null,
    noFundingAccount: !funding,
  };
}
