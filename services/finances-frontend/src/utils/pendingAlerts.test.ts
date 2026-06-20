import { describe, it, expect } from 'vitest';
import { findPendingRecurrings } from './pendingAlerts';
import type { Transaction, RecurringTransaction } from '@/types/finances';

const TODAY = '2026-06-25';

const recurring = (over: Partial<RecurringTransaction> = {}): RecurringTransaction => ({
  id: 'rec-1',
  profileId: 'p1',
  bankAccountId: 'acc-mp',
  type: 'EXPENSE',
  amount: 99,
  currency: 'BRL',
  description: 'Tim Celular',
  recurrenceRule: 'FREQ=MONTHLY;BYMONTHDAY=20',
  startOn: '2026-01-20',
  nextOccurrence: '2026-06-20',
  status: 'ACTIVE',
  createdAt: '',
  updatedAt: '',
  ...over,
});

const tx = (over: Partial<Transaction> = {}): Transaction => ({
  id: 'tx-1',
  profileId: 'p1',
  bankAccountId: 'acc-mp',
  type: 'EXPENSE',
  status: 'CONFIRMED',
  amount: 99,
  currency: 'BRL',
  description: 'Tim Celular',
  occurredOn: '2026-06-20',
  createdAt: '',
  updatedAt: '',
  ...over,
});

describe('findPendingRecurrings', () => {
  // Regression guard for the variable-amount bug: a recurring expense that varies
  // each month (Tim billed 99 but actually 109,99) must NOT stay pending once paid
  // at the real amount. The previous logic matched by amount and kept it pending.
  it('does NOT flag a recurring paid at a DIFFERENT amount in the same month', () => {
    const result = findPendingRecurrings(
      [recurring({ amount: 99 })],
      [tx({ amount: 109.99 })],
      TODAY,
    );
    expect(result).toHaveLength(0);
  });

  it('flags a due recurring with no matching transaction', () => {
    const result = findPendingRecurrings([recurring()], [], TODAY);
    expect(result).toHaveLength(1);
  });

  it('does not flag a recurring whose nextOccurrence is in the future', () => {
    const result = findPendingRecurrings(
      [recurring({ nextOccurrence: '2026-07-20' })],
      [],
      TODAY,
    );
    expect(result).toHaveLength(0);
  });

  it('ignores paused / cancelled recurrings', () => {
    expect(findPendingRecurrings([recurring({ status: 'PAUSED' })], [], TODAY)).toHaveLength(0);
    expect(findPendingRecurrings([recurring({ status: 'CANCELLED' })], [], TODAY)).toHaveLength(0);
  });

  it('does not match a transaction on a different account', () => {
    const result = findPendingRecurrings(
      [recurring({ bankAccountId: 'acc-mp' })],
      [tx({ amount: 109.99, bankAccountId: 'acc-nubank' })],
      TODAY,
    );
    expect(result).toHaveLength(1);
  });

  it('matches a PLANNED transaction (obligation generated but not yet confirmed)', () => {
    const result = findPendingRecurrings(
      [recurring()],
      [tx({ status: 'PLANNED', amount: 99 })],
      TODAY,
    );
    expect(result).toHaveLength(0);
  });

  it('does not match a different description', () => {
    const result = findPendingRecurrings(
      [recurring({ description: 'Tim Celular' })],
      [tx({ description: 'Vivo Celular', amount: 109.99 })],
      TODAY,
    );
    expect(result).toHaveLength(1);
  });
});
