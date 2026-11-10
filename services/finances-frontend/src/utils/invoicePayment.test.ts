import { describe, it, expect } from 'vitest';
import { describeInvoicePayment } from './invoicePayment';
import { amountLeftOn } from './creditCard';
import type { BankAccount, Invoice } from '@/types/finances';

function bill(over: Partial<Invoice> = {}): Invoice {
  return {
    id: 'inv',
    bankAccountId: 'card',
    referenceDate: '2026-10-01',
    openingDate: '2026-09-10',
    closingDate: '2026-10-09',
    dueDate: '2026-10-14',
    amount: 1649.35,
    confirmedAmount: 1649.35,
    plannedAmount: 0,
    status: 'CLOSED',
    ...over,
  } as Invoice;
}

const card = { id: 'card', linkedAccountId: 'checking' } as BankAccount;
const accounts = [{ id: 'checking', name: 'Mercado Pago' }] as BankAccount[];

describe('describeInvoicePayment', () => {
  it('names the amount and the account the money will leave', () => {
    expect(describeInvoicePayment(bill(), card, accounts)).toEqual({
      amount: 1649.35,
      fundingAccountName: 'Mercado Pago',
      blockedReason: null,
      noFundingAccount: false,
    });
  });

  it('offers only what is left after a partial payment', () => {
    const preview = describeInvoicePayment(
      bill({ status: 'PARTIALLY_PAID', paidAmount: 1000, amountRemaining: 649.35 }),
      card,
      accounts,
    );

    expect(preview.amount).toBe(649.35);
    expect(preview.blockedReason).toBeNull();
  });

  // The case that started this: the bank converted the bill into instalments, so
  // nobody paid it and the instalments are already on the books. Recording a
  // payment here debits an account that nothing left.
  it('blocks a bill the bank converted into instalments', () => {
    const preview = describeInvoicePayment(bill({ status: 'REFINANCED' }), card, accounts);

    expect(preview.blockedReason).toContain('convertida em parcelas');
    expect(preview.fundingAccountName).toBeNull();
  });

  it('blocks a bill that owes nothing', () => {
    const preview = describeInvoicePayment(
      bill({ status: 'PAID', paidAmount: 1649.35, amountRemaining: 0 }),
      card,
      accounts,
    );

    expect(preview.amount).toBe(0);
    expect(preview.blockedReason).toContain('não deve nada');
  });

  // Not a refusal: PayInvoiceUseCaseV2 handles this on purpose, marking the bill
  // paid and skipping the transfer. Blocking it made such cards unpayable from
  // the screen while the API was willing.
  it('allows a card with no funding account, and says no account is named', () => {
    const preview = describeInvoicePayment(bill(), { id: 'card' } as BankAccount, accounts);

    expect(preview.blockedReason).toBeNull();
    expect(preview.noFundingAccount).toBe(true);
    expect(preview.fundingAccountName).toBeNull();
  });

  it('says the same when the linked account is not among the ones loaded', () => {
    const orphan = { id: 'card', linkedAccountId: 'gone' } as BankAccount;
    const preview = describeInvoicePayment(bill(), orphan, accounts);

    expect(preview.blockedReason).toBeNull();
    expect(preview.noFundingAccount).toBe(true);
  });
});

describe('amountLeftOn with a converted bill', () => {
  // The derived field from the API already answers zero; this covers the payload
  // that predates it, where only the status says what happened.
  it('owes nothing even when a partial payment was recorded', () => {
    expect(amountLeftOn(bill({ status: 'REFINANCED', amount: 3292.96, paidAmount: 3200 }))).toBe(0);
  });

  it('still owes the remainder on a merely partially paid bill', () => {
    expect(amountLeftOn(bill({ status: 'PARTIALLY_PAID', amount: 3292.96, paidAmount: 3200 }))).toBeCloseTo(
      92.96,
      2,
    );
  });
});

// The REFINANCED check sits BEFORE the amountRemaining early return. The API
// never emits a converted bill with a non-zero remainder, but a payload that
// did would otherwise be reported as debt on a settled bill.
describe('amountLeftOn ordering', () => {
  it('answers zero for a converted bill even if a remainder is present', () => {
    expect(
      amountLeftOn(bill({ status: 'REFINANCED', amount: 1649.35, amountRemaining: 1649.35 })),
    ).toBe(0);
  });
});
