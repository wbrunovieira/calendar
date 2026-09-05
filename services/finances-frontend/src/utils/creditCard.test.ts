import { describe, expect, it } from 'vitest';
import { getCardUsage } from './creditCard';

describe('getCardUsage', () => {
  // The bug this replaces: available was computed from the OPEN invoice only, so a
  // closed-unpaid invoice, an installment plan or revolving credit made the card look
  // far emptier than it is. Real case: R$400 limit, R$380,23 owed, bank shows R$19,77.
  it('derives availability from the total owed, not from the open invoice', () => {
    const usage = getCardUsage({ creditLimit: 400, currentBalance: -380.23 });

    expect(usage.outstanding).toBeCloseTo(380.23, 2);
    expect(usage.available).toBeCloseTo(19.77, 2);
    expect(Math.round(usage.usagePercent)).toBe(95);
  });

  it('treats a card with no debt as fully available', () => {
    const usage = getCardUsage({ creditLimit: 400, currentBalance: 0 });

    expect(usage.outstanding).toBe(0);
    expect(usage.available).toBe(400);
    expect(usage.usagePercent).toBe(0);
  });

  // A card can hold a credit balance (a refund landing after the bill was paid).
  it('never reports negative usage when the card holds a credit', () => {
    const usage = getCardUsage({ creditLimit: 400, currentBalance: 25 });

    expect(usage.outstanding).toBe(0);
    expect(usage.available).toBe(400);
    expect(usage.usagePercent).toBe(0);
  });

  it('reports a negative availability when the limit is exceeded', () => {
    const usage = getCardUsage({ creditLimit: 400, currentBalance: -450 });

    expect(usage.available).toBeCloseTo(-50, 2);
    expect(usage.usagePercent).toBeGreaterThan(100);
  });

  it('does not divide by zero when no limit is configured', () => {
    const usage = getCardUsage({ currentBalance: -120 });

    expect(usage.usagePercent).toBe(0);
    expect(usage.available).toBeCloseTo(-120, 2);
  });
});
