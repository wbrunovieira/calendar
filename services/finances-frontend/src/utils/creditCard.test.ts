import { describe, expect, it } from 'vitest';
import { getCardUsage } from './creditCard';
import type { Invoice } from '@/types/finances';

const invoice = (amount: number, status: Invoice['status']): Invoice =>
  ({
    id: `${status}-${amount}`,
    bankAccountId: 'card',
    referenceDate: '2026-09-01',
    openingDate: '2026-08-27',
    closingDate: '2026-09-27',
    dueDate: '2026-10-03',
    amount,
    confirmedAmount: amount,
    plannedAmount: 0,
    status,
  }) as Invoice;

describe('getCardUsage', () => {
  // The bug this replaces: availability came from the OPEN invoice alone, so a closed
  // bill still unpaid or an instalment plan was invisible. The real card: R$400 limit,
  // R$253,48 rolled into instalments plus R$126,75 in the open cycle. The bank shows
  // R$19,77 available and 95% used; the panel offered R$273,26 and drew 32%.
  it('derives availability from every unpaid invoice, not just the open one', () => {
    const usage = getCardUsage({ creditLimit: 400 }, [
      invoice(253.48, 'CLOSED'),
      invoice(126.75, 'OPEN'),
      invoice(2702.66, 'PAID'),
    ]);

    expect(usage.outstanding).toBeCloseTo(380.23, 2);
    expect(usage.available).toBeCloseTo(19.77, 2);
    expect(Math.round(usage.usagePercent)).toBe(95);
  });

  it('treats a card with no unpaid invoice as fully available', () => {
    const usage = getCardUsage({ creditLimit: 400 }, [invoice(2702.66, 'PAID')]);

    expect(usage.outstanding).toBe(0);
    expect(usage.available).toBe(400);
    expect(usage.usagePercent).toBe(0);
  });

  it('ignores an invoice with nothing on it', () => {
    const usage = getCardUsage({ creditLimit: 400 }, [invoice(0, 'OPEN')]);

    expect(usage.outstanding).toBe(0);
  });

  it('reports a negative availability when the limit is exceeded', () => {
    const usage = getCardUsage({ creditLimit: 400 }, [invoice(450, 'CLOSED')]);

    expect(usage.available).toBeCloseTo(-50, 2);
    expect(usage.usagePercent).toBeGreaterThan(100);
  });

  it('does not divide by zero when no limit is configured', () => {
    const usage = getCardUsage({}, [invoice(120, 'OPEN')]);

    expect(usage.usagePercent).toBe(0);
    expect(usage.available).toBeCloseTo(-120, 2);
  });

  it('handles a card with no invoices at all', () => {
    const usage = getCardUsage({ creditLimit: 400 }, []);

    expect(usage.outstanding).toBe(0);
    expect(usage.available).toBe(400);
  });
});
