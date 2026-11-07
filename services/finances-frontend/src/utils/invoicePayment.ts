import type { BankAccount, Invoice } from '@/types/finances';
import { amountLeftOn } from './creditCard';

export interface PaymentPreview {
  /** What the debit will be worth. */
  amount: number;
  /** The account the money will leave, by name. */
  fundingAccountName: string | null;
  /** Why this payment should not be recorded, if it should not. */
  blockedReason: string | null;
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
  if (!funding) {
    // Better to say so than to let the entry land somewhere the user did not pick.
    return {
      amount,
      fundingAccountName: null,
      blockedReason:
        'Este cartão não tem conta de pagamento vinculada, então não dá para dizer de onde o dinheiro sairia.',
    };
  }

  return { amount, fundingAccountName: funding.name, blockedReason: null };
}
