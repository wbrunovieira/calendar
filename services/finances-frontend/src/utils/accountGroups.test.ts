import { describe, expect, it } from 'vitest';
import { groupAccounts, sumBalances, debtTotal, isCashLike } from './accountGroups';
import type { AccountType, BankAccount } from '@/types/finances';

const account = (name: string, type: AccountType, currentBalance: number, isActive = true): BankAccount =>
  ({ id: name, name, type, currentBalance, isActive }) as BankAccount;

// The real home screen on 12/11/2026: a card headed "Disponivel" showed
// -R$ 3.414,81 while R$ 450,87 was what the user could actually spend. It was
// summing two LIABILITY accounts — a loan and a debt to family — in with the
// cash, because the filter was written as "not a card and not an investment".
// Anything the backend adds later lands in cash by default, which is how a debt
// ended up being reported as money available.
describe('groupAccounts', () => {
  const accounts = [
    account('Mercado Pago', 'CHECKING', 447.86),
    account('Nubank', 'CHECKING', 0.51),
    account('Dinheiro (Caixa)', 'CASH', 2.5),
    account('Binance', 'EXCHANGE', 0),
    account('Divida com as irmas', 'LIABILITY', -875),
    account('Emprestimo pessoal Mercado Pago', 'LIABILITY', -2990.68),
    account('Cartao Mercado Pago', 'CREDIT_CARD', -1059.22),
    account('Investimentos Mercado Pago', 'INVESTMENT', 6335.67),
  ];

  it('keeps debts out of what is available', () => {
    const { cash, liabilities } = groupAccounts(accounts);

    expect(sumBalances(cash)).toBeCloseTo(450.87, 2);
    expect(liabilities.map((a) => a.name)).toEqual([
      'Divida com as irmas',
      'Emprestimo pessoal Mercado Pago',
    ]);
    expect(debtTotal(liabilities)).toBeCloseTo(3865.68, 2);
  });

  it('separates cards and investments', () => {
    const { creditCards, investments } = groupAccounts(accounts);

    expect(creditCards.map((a) => a.name)).toEqual(['Cartao Mercado Pago']);
    expect(sumBalances(investments)).toBeCloseTo(6335.67, 2);
  });

  // "[DUPLICADA - nao usar] Emprestimo..." was rendered on the home screen with
  // isActive false. It carried zero, so no total was wrong — but a line that
  // shouts DUPLICADA on the first screen is how trust in the rest goes.
  it('leaves deactivated accounts off the screen entirely', () => {
    const groups = groupAccounts([
      ...accounts,
      account('[DUPLICADA - nao usar] Emprestimo', 'LIABILITY', 0, false),
      account('CDB Banco Arbi (DUPLICADO)', 'INVESTMENT', 0, false),
      account('conta morta', 'CHECKING', 99, false),
    ]);

    const everyName = [
      ...groups.cash,
      ...groups.liabilities,
      ...groups.creditCards,
      ...groups.investments,
      ...groups.unclassified,
    ].map((a) => a.name);

    expect(everyName).not.toContain('[DUPLICADA - nao usar] Emprestimo');
    expect(everyName).not.toContain('CDB Banco Arbi (DUPLICADO)');
    expect(everyName).not.toContain('conta morta');
    expect(sumBalances(groups.cash)).toBeCloseTo(450.87, 2);
  });

  // A type nobody taught this function about must NOT default into cash — that
  // default is the whole bug. It is surfaced on its own so the money is still
  // visible instead of silently dropped.
  it('does not let an unknown type pass itself off as cash', () => {
    const groups = groupAccounts([
      account('Mercado Pago', 'CHECKING', 447.86),
      account('coisa nova', 'FINANCIAMENTO' as AccountType, -5000),
    ]);

    expect(sumBalances(groups.cash)).toBeCloseTo(447.86, 2);
    expect(groups.unclassified.map((a) => a.name)).toEqual(['coisa nova']);
  });

  it('reports an account with no isActive flag as active', () => {
    const groups = groupAccounts([{ id: 'x', name: 'sem flag', type: 'CHECKING', currentBalance: 10 } as BankAccount]);

    expect(groups.cash.map((a) => a.name)).toEqual(['sem flag']);
  });
});

// Net worth must not change just because the pieces were renamed: the same
// money, grouped honestly. 450,87 + 9.391,85 - 5.257,10 - 3.865,68 = 719,94.
describe('debtTotal', () => {
  it('reports a debt as a positive amount owed', () => {
    expect(debtTotal([account('a', 'LIABILITY', -875), account('b', 'LIABILITY', -2990.68)])).toBeCloseTo(3865.68, 2);
  });

  it('does not invent debt when a liability account is settled or overpaid', () => {
    expect(debtTotal([account('quitada', 'LIABILITY', 0)])).toBe(0);
    expect(debtTotal([account('paguei demais', 'LIABILITY', 120)])).toBeCloseTo(-120, 2);
  });
});

// The same total is built on the accounts screen ("Saldo em contas"), where it
// had the same defect: everything that was not a credit card counted as
// balance, debts included.
describe('isCashLike', () => {
  it('counts only money that can be spent', () => {
    expect(isCashLike(account('conta', 'CHECKING', 1))).toBe(true);
    expect(isCashLike(account('caixa', 'CASH', 1))).toBe(true);
    expect(isCashLike(account('binance', 'EXCHANGE', 1))).toBe(true);
  });

  it('refuses debts and anything put away', () => {
    expect(isCashLike(account('emprestimo', 'LIABILITY', -2990.68))).toBe(false);
    expect(isCashLike(account('cartao', 'CREDIT_CARD', -1059.22))).toBe(false);
    expect(isCashLike(account('cdb', 'INVESTMENT', 3043.87))).toBe(false);
  });

  it('refuses a type it was never taught instead of assuming it is money', () => {
    expect(isCashLike(account('coisa nova', 'FINANCIAMENTO' as AccountType, -5000))).toBe(false);
  });
});
