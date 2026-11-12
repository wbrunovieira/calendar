import { describe, expect, it } from 'vitest';
import { groupAccounts, sumBalances, debtTotal, isCashLike, isDebtType, computeNetWorth, sumHeldByCurrency } from './accountGroups';
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

// A deactivated CARD is not like a deactivated account. Its debt outlives the
// row: `getCardsOutstanding` keys off the card list, so dropping the card drops
// its unpaid invoice from the total and net worth jumps by exactly that amount,
// with nothing on screen saying so.
describe('groupAccounts and deactivated credit cards', () => {
  it('keeps a deactivated card, because what it owes did not go away', () => {
    const groups = groupAccounts([
      account('Cartao antigo', 'CREDIT_CARD', -500, false),
      account('Cartao atual', 'CREDIT_CARD', -100),
      account('conta morta', 'CHECKING', 99, false),
    ]);

    expect(groups.creditCards.map((a) => a.name)).toEqual(['Cartao antigo', 'Cartao atual']);
    expect(groups.cash).toHaveLength(0);
  });
});

describe('computeNetWorth', () => {
  const invoice = (amount: number) =>
    [{ id: 'i', bankAccountId: 'c', amount, confirmedAmount: amount, plannedAmount: 0, status: 'OPEN' }] as never;

  // The real home screen, 12/11/2026. The number itself never moved when the
  // grouping was fixed — only its decomposition — so it is pinned here.
  it('reproduces the real figures', () => {
    const groups = groupAccounts([
      account('Mercado Pago', 'CHECKING', 447.86),
      account('Nubank', 'CHECKING', 0.51),
      account('Dinheiro (Caixa)', 'CASH', 2.5),
      account('Investimentos', 'INVESTMENT', 9391.85),
      account('Divida com as irmas', 'LIABILITY', -875),
      account('Emprestimo pessoal Mercado Pago', 'LIABILITY', -2990.68),
      account('cartoes', 'CREDIT_CARD', 0),
    ]);

    const parts = computeNetWorth(groups, { cartoes: invoice(5257.1) });

    expect(parts.available).toBeCloseTo(450.87, 2);
    expect(parts.invested).toBeCloseTo(9391.85, 2);
    expect(parts.cardDebt).toBeCloseTo(5257.1, 2);
    expect(parts.loanDebt).toBeCloseTo(3865.68, 2);
    expect(parts.netWorth).toBeCloseTo(719.94, 2);
  });

  // The mutation that got through the first time: drop `- loanDebt` and every
  // test still passed, because the arithmetic lived in the component and the
  // component had none.
  it('subtracts loans from net worth', () => {
    const groups = groupAccounts([
      account('conta', 'CHECKING', 1000),
      account('emprestimo', 'LIABILITY', -400),
    ]);

    expect(computeNetWorth(groups, {}).netWorth).toBeCloseTo(600, 2);
  });

  it('does not count a debt as money available', () => {
    const groups = groupAccounts([
      account('conta', 'CHECKING', 1000),
      account('emprestimo', 'LIABILITY', -400),
    ]);

    expect(computeNetWorth(groups, {}).available).toBeCloseTo(1000, 2);
  });

  // Unclassified money is not available and must not be lost either: it has to
  // reach the total, and be reported as its own term so the printed equation
  // adds up.
  it('counts an unclassified account without calling it available', () => {
    const groups = groupAccounts([
      account('conta', 'CHECKING', 100),
      account('coisa nova', 'FINANCIAMENTO' as AccountType, -50),
    ]);

    const parts = computeNetWorth(groups, {});

    expect(parts.available).toBeCloseTo(100, 2);
    expect(parts.unclassified).toBeCloseTo(-50, 2);
    expect(parts.netWorth).toBeCloseTo(50, 2);
  });

  it('adds back an overpaid liability instead of pretending it is a debt', () => {
    const groups = groupAccounts([
      account('conta', 'CHECKING', 100),
      account('paguei demais', 'LIABILITY', 120),
    ]);

    const parts = computeNetWorth(groups, {});

    expect(parts.loanDebt).toBeCloseTo(-120, 2);
    expect(parts.netWorth).toBeCloseTo(220, 2);
  });

  it('still subtracts what a deactivated card owes', () => {
    const groups = groupAccounts([
      account('conta', 'CHECKING', 1000),
      account('cartao desativado', 'CREDIT_CARD', 0, false),
    ]);

    expect(computeNetWorth(groups, { 'cartao desativado': invoice(300) }).netWorth).toBeCloseTo(700, 2);
  });

  // Every printed term must reach the printed total, or the panel shows an
  // equation that does not close — which is the bug being fixed, one level up.
  it('reports terms that add up to the total it reports', () => {
    const groups = groupAccounts([
      account('conta', 'CHECKING', 447.86),
      account('investido', 'INVESTMENT', 9391.85),
      account('emprestimo', 'LIABILITY', -2990.68),
      account('coisa nova', 'FINANCIAMENTO' as AccountType, -7),
      account('cartao', 'CREDIT_CARD', 0),
    ]);

    const p = computeNetWorth(groups, { cartao: invoice(5257.1) });

    expect(p.available + p.invested + p.unclassified - p.cardDebt - p.loanDebt).toBeCloseTo(p.netWorth, 2);
  });
});

// The accounts screen lists everything and totals what is held. It cannot filter
// by a positive cash list: an unclassified type would vanish from the list AND
// from the total in one move, which is the regression that replaces this bug.
describe('isDebtType', () => {
  it('names what is owed', () => {
    expect(isDebtType(account('cartao', 'CREDIT_CARD', -100))).toBe(true);
    expect(isDebtType(account('emprestimo', 'LIABILITY', -100))).toBe(true);
  });

  it('leaves everything else alone, including a type it was never taught', () => {
    expect(isDebtType(account('conta', 'CHECKING', 1))).toBe(false);
    expect(isDebtType(account('cdb', 'INVESTMENT', 1))).toBe(false);
    expect(isDebtType(account('coisa nova', 'FINANCIAMENTO' as AccountType, 1))).toBe(false);
  });
});

// The "Saldo em contas" total on the accounts screen. It was built inline in the
// page, so nothing proved it excluded deactivated accounts — the screen prints
// "(desativada, fora do total)" next to them, and that label could have started
// lying without a single test going red.
describe('sumHeldByCurrency', () => {
  const withCurrency = (name: string, type: AccountType, balance: number, currency: string, isActive = true) =>
    ({ id: name, name, type, currentBalance: balance, currency, isActive }) as BankAccount;

  it('totals what is held, per currency', () => {
    const totals = sumHeldByCurrency([
      withCurrency('Mercado Pago', 'CHECKING', 447.86, 'BRL'),
      withCurrency('Dinheiro', 'CASH', 2.5, 'BRL'),
      withCurrency('Wise EUR', 'CHECKING', 30, 'EUR'),
    ]);

    expect(totals).toEqual({ BRL: 450.36, EUR: 30 });
  });

  it('leaves debts out', () => {
    const totals = sumHeldByCurrency([
      withCurrency('Mercado Pago', 'CHECKING', 1000, 'BRL'),
      withCurrency('Emprestimo', 'LIABILITY', -2990.68, 'BRL'),
      withCurrency('Cartao', 'CREDIT_CARD', -1059.22, 'BRL'),
    ]);

    expect(totals).toEqual({ BRL: 1000 });
  });

  it('leaves deactivated accounts out — the label on screen says so', () => {
    const totals = sumHeldByCurrency([
      withCurrency('Mercado Pago', 'CHECKING', 1000, 'BRL'),
      withCurrency('conta morta', 'CHECKING', 99, 'BRL', false),
    ]);

    expect(totals).toEqual({ BRL: 1000 });
  });

  // Not a positive cash list: a type nobody classified has to stay counted, or
  // it disappears from the screen and the total in one move.
  it('still counts a type it was never taught', () => {
    const totals = sumHeldByCurrency([
      withCurrency('coisa nova', 'FINANCIAMENTO' as AccountType, 42, 'BRL'),
    ]);

    expect(totals).toEqual({ BRL: 42 });
  });

  it('treats a missing currency as BRL', () => {
    expect(sumHeldByCurrency([{ id: 'x', name: 'x', type: 'CHECKING', currentBalance: 5 } as BankAccount])).toEqual({ BRL: 5 });
  });
});
