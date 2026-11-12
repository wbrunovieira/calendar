import { describe, expect, it } from 'vitest';
import { datesForConfirmedPayment } from './confirmRecurring';

// "Ja paguei" writes a CONFIRMED expense. It used to date that write with the
// recurrence's next occurrence, which was always today or later — so the date
// was effectively today and nobody noticed.
//
// Now that an overdue recurrence reaches this button with its real (past) date,
// the same line would post a payment dated weeks ago: the month attribution
// moves, and create_transaction has no closed-month guard, so it can land inside
// a month already checkpointed.
describe('datesForConfirmedPayment', () => {
  it('dates the payment today, not on the day it was due', () => {
    const { occurredOn, dueOn } = datesForConfirmedPayment('2026-01-20T00:00:00Z', '2026-11-12');

    expect(occurredOn).toBe('2026-11-12');
    expect(dueOn).toBe('2026-01-20');
  });

  it('keeps the obligation date, so nothing about when it was due is lost', () => {
    expect(datesForConfirmedPayment('2026-11-20', '2026-11-12').dueOn).toBe('2026-11-20');
  });

  it('reads a bare day as well as a timestamp', () => {
    expect(datesForConfirmedPayment('2026-11-20', '2026-11-12').occurredOn).toBe('2026-11-12');
  });

  it('is unremarkable when the bill is due today', () => {
    expect(datesForConfirmedPayment('2026-11-12T00:00:00Z', '2026-11-12')).toEqual({
      occurredOn: '2026-11-12',
      dueOn: '2026-11-12',
    });
  });
});
