import { describe, it, expect } from 'vitest';
import { summarizeCashflow } from './cashflow';
import type { Transaction, BankAccount } from '@/types/finances';

const acc = (id: string, type: BankAccount['type']): BankAccount =>
  ({ id, name: id, type, currentBalance: 0, currency: 'BRL', profileId: 'p1' } as BankAccount);

const tx = (over: Partial<Transaction>): Transaction =>
  ({
    id: Math.random().toString(),
    profileId: 'p1',
    bankAccountId: 'mp',
    type: 'INCOME',
    status: 'CONFIRMED',
    amount: 0,
    currency: 'BRL',
    description: '',
    occurredOn: '2026-06-10',
    createdAt: '',
    updatedAt: '',
    ...over,
  } as Transaction);

const accounts: BankAccount[] = [
  acc('mp', 'CHECKING'),
  acc('mp-card', 'CREDIT_CARD'),
  acc('binance', 'EXCHANGE'),
];

describe('summarizeCashflow', () => {
  it('excludes credit-card invoice payments from income (card-side credit) and expense', () => {
    const s = summarizeCashflow(
      [
        tx({ type: 'INCOME', bankAccountId: 'mp-card', amount: 2108.44, description: 'Pagamento fatura Cartão Mercado Pago' }),
        tx({ type: 'EXPENSE', bankAccountId: 'mp', amount: 2108.44, description: 'Pagamento fatura Cartão Mercado Pago' }),
        tx({ type: 'INCOME', bankAccountId: 'mp', amount: 580.1, description: 'Flavia Guedes - pagamento cliente' }),
      ],
      accounts,
    );
    expect(s.income).toBeCloseTo(580.1, 2);
    expect(s.incomeCount).toBe(1);
    expect(s.expense).toBe(0);
    expect(s.expenseCount).toBe(0);
  });

  it('separates rendimento (yield) from other income', () => {
    const s = summarizeCashflow(
      [
        tx({ type: 'INCOME', amount: 580.1, description: 'Flavia Guedes - pagamento cliente' }),
        tx({ type: 'INCOME', amount: 3.92, description: 'rendimento de conta' }),
        tx({ type: 'INCOME', amount: 27.35, description: 'Rendimento CDB (jan-jun)' }),
        tx({ type: 'INCOME', amount: 0.4, description: 'rendimento da conta' }),
      ],
      accounts,
    );
    expect(s.income).toBeCloseTo(611.77, 2);
    expect(s.incomeYield).toBeCloseTo(31.67, 2);
    expect(s.incomeYieldCount).toBe(3);
    expect(s.incomeOther).toBeCloseTo(580.1, 2);
    expect(s.incomeOtherCount).toBe(1);
  });

  it('excludes EXCHANGE-account transactions and non-confirmed', () => {
    const s = summarizeCashflow(
      [
        tx({ type: 'EXPENSE', bankAccountId: 'binance', amount: 999, description: 'trade' }),
        tx({ type: 'EXPENSE', bankAccountId: 'mp', amount: 50, description: 'mercado', status: 'PLANNED' }),
        tx({ type: 'EXPENSE', bankAccountId: 'mp', amount: 100, description: 'mercado' }),
      ],
      accounts,
    );
    expect(s.expense).toBe(100);
    expect(s.expenseCount).toBe(1);
  });

  it('counts real expenses on checking and credit card', () => {
    const s = summarizeCashflow(
      [
        tx({ type: 'EXPENSE', bankAccountId: 'mp', amount: 109.99, description: 'Tim Celular' }),
        tx({ type: 'EXPENSE', bankAccountId: 'mp-card', amount: 10, description: 'Caldo de mocoto' }),
      ],
      accounts,
    );
    expect(s.expense).toBeCloseTo(119.99, 2);
    expect(s.expenseCount).toBe(2);
  });
});
