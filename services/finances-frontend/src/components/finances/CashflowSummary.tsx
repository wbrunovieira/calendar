'use client';

import type { BankAccount, Transaction, Invoice } from '@/types/finances';
import { formatCurrency } from '@/utils/format';

interface CashflowSummaryProps {
  transactions: Transaction[];
  accounts: BankAccount[];
  currentInvoices?: Record<string, Invoice>;
}

const getCurrentMonthYear = () => {
  const now = new Date();
  const month = now.toLocaleDateString('pt-BR', { month: 'short' }).replace('.', '');
  const year = now.getFullYear();
  return `${month.charAt(0).toUpperCase() + month.slice(1)}/${year}`;
};

export default function CashflowSummary({ transactions, accounts, currentInvoices = {} }: CashflowSummaryProps) {
  const currentPeriod = getCurrentMonthYear();

  // Exclude trades from EXCHANGE accounts (crypto bots) from income/expense totals
  const exchangeAccountIds = new Set(
    accounts.filter((acc) => acc.type === 'EXCHANGE').map((acc) => acc.id),
  );

  const confirmedIncomeTransactions = transactions.filter(
    (transaction) => transaction.status === 'CONFIRMED' && transaction.type === 'INCOME' && !exchangeAccountIds.has(transaction.bankAccountId),
  );
  const confirmedExpenseTransactions = transactions.filter(
    (transaction) => transaction.status === 'CONFIRMED' && transaction.type === 'EXPENSE' && !exchangeAccountIds.has(transaction.bankAccountId),
  );

  const confirmedIncome = confirmedIncomeTransactions.reduce(
    (total, transaction) => total + transaction.amount,
    0,
  );
  const confirmedExpense = confirmedExpenseTransactions.reduce(
    (total, transaction) => total + transaction.amount,
    0,
  );

  // Separate accounts by type
  const regularAccounts = accounts.filter(
    (acc) => acc.type !== 'CREDIT_CARD' && acc.type !== 'INVESTMENT'
  );
  const creditCards = accounts.filter((acc) => acc.type === 'CREDIT_CARD');
  const investments = accounts.filter((acc) => acc.type === 'INVESTMENT');

  // Calculate balances
  const availableBalance = regularAccounts.reduce(
    (total, acc) => total + acc.currentBalance,
    0
  );

  const creditCardDebt = creditCards.reduce((total, acc) => {
    const invoice = currentInvoices[acc.id];
    return total + (invoice?.amount || 0);
  }, 0);

  const investmentTotal = investments.reduce(
    (total, acc) => total + acc.currentBalance,
    0
  );

  // Net worth = available + investments - credit card debt
  const netWorth = availableBalance + investmentTotal - creditCardDebt;

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
      {/* Entradas */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">⬆️</span>
          <span className="text-white/70 text-sm">Entradas</span>
          <span className="text-white/40 text-xs ml-auto">{currentPeriod}</span>
        </div>
        <div className="text-2xl font-bold text-emerald-400">
          {formatCurrency(confirmedIncome)}
        </div>
        <p className="text-white/50 text-xs mt-1">
          {confirmedIncomeTransactions.length} confirmadas
        </p>
      </div>

      {/* Saidas */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">⬇️</span>
          <span className="text-white/70 text-sm">Saidas</span>
          <span className="text-white/40 text-xs ml-auto">{currentPeriod}</span>
        </div>
        <div className="text-2xl font-bold text-rose-400">
          {formatCurrency(confirmedExpense)}
        </div>
        <p className="text-white/50 text-xs mt-1">
          {confirmedExpenseTransactions.length} confirmadas
        </p>
      </div>

      {/* Saldo Disponivel (contas correntes, poupanca, dinheiro) */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">🏦</span>
          <span className="text-white/70 text-sm">Disponivel</span>
        </div>
        <div className="space-y-1 mb-2">
          {regularAccounts.map((acc) => (
            <div key={acc.id} className="flex justify-between text-sm">
              <span className="text-white/60 truncate mr-2">{acc.name}</span>
              <span className={acc.currentBalance >= 0 ? 'text-white/80' : 'text-rose-400'}>
                {formatCurrency(acc.currentBalance)}
              </span>
            </div>
          ))}
        </div>
        <div className="border-t border-white/10 pt-2">
          <div className="flex justify-between">
            <span className="text-white/70 text-sm font-medium">Total</span>
            <span className={`text-lg font-bold ${availableBalance >= 0 ? 'text-white' : 'text-rose-400'}`}>
              {formatCurrency(availableBalance)}
            </span>
          </div>
        </div>
      </div>

      {/* Cartoes de Credito */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">💳</span>
          <span className="text-white/70 text-sm">Faturas</span>
        </div>
        <div className={`text-2xl font-bold ${creditCardDebt > 0 ? 'text-amber-400' : 'text-white'}`}>
          {formatCurrency(creditCardDebt)}
        </div>
        <p className="text-white/50 text-xs mt-1">
          {creditCards.length} {creditCards.length === 1 ? 'cartao' : 'cartoes'}
        </p>
      </div>

      {/* Patrimonio Liquido */}
      <div className="bg-gradient-to-br from-purple-900/30 to-indigo-900/30 border border-purple-500/20 rounded-2xl p-5 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-lg">💰</span>
          <span className="text-white/70 text-sm">Patrimonio</span>
        </div>
        <div className={`text-2xl font-bold ${netWorth >= 0 ? 'text-purple-300' : 'text-rose-400'}`}>
          {formatCurrency(netWorth)}
        </div>
        <p className="text-white/50 text-xs mt-1">
          {investments.length > 0 && `+ ${formatCurrency(investmentTotal)} investidos`}
          {investments.length === 0 && 'Disponivel - Faturas'}
        </p>
      </div>
    </div>
  );
}
