'use client';

import type { BankAccount, Transaction } from '@/types/finances';

interface CashflowSummaryProps {
  transactions: Transaction[];
  accounts: BankAccount[];
}

const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
  }).format(value);

export default function CashflowSummary({ transactions, accounts }: CashflowSummaryProps) {
  const confirmedIncomeTransactions = transactions.filter(
    (transaction) => transaction.status === 'CONFIRMED' && transaction.type === 'INCOME',
  );
  const confirmedExpenseTransactions = transactions.filter(
    (transaction) => transaction.status === 'CONFIRMED' && transaction.type === 'EXPENSE',
  );

  const confirmedIncome = confirmedIncomeTransactions.reduce(
    (total, transaction) => total + transaction.amount,
    0,
  );
  const confirmedExpense = confirmedExpenseTransactions.reduce(
    (total, transaction) => total + transaction.amount,
    0,
  );

  const plannedBalance = transactions
    .filter((transaction) => transaction.status === 'PLANNED')
    .reduce(
      (total, transaction) =>
        transaction.type === 'EXPENSE' ? total - transaction.amount : total + transaction.amount,
      0,
    );

  const accountsBalance = accounts.reduce((total, account) => total + account.currentBalance, 0);
  const netCashflow = confirmedIncome - confirmedExpense;

  const cards = [
    {
      title: 'Entradas confirmadas',
      value: formatCurrency(confirmedIncome),
      highlight: 'from-emerald-500/80 to-emerald-600/80',
      icon: '⬆️',
      description: `${confirmedIncomeTransactions.length} lançamentos confirmados`,
    },
    {
      title: 'Saídas confirmadas',
      value: formatCurrency(confirmedExpense),
      highlight: 'from-rose-500/80 to-rose-600/80',
      icon: '⬇️',
      description: `${confirmedExpenseTransactions.length} lançamentos confirmados`,
    },
    {
      title: 'Fluxo líquido',
      value: formatCurrency(netCashflow),
      highlight:
        netCashflow >= 0 ? 'from-sky-500/80 to-sky-600/80' : 'from-amber-500/80 to-amber-600/80',
      icon: '⚖️',
      description: `Previsto: ${formatCurrency(plannedBalance)}`,
    },
  ];

  return (
    <div className="grid gap-4 lg:grid-cols-4">
      {cards.map((card) => (
        <div
          key={card.title}
          className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-sm flex flex-col gap-4"
        >
          <div
            className={`inline-flex items-center gap-3 px-3 py-1 rounded-full text-sm text-white bg-gradient-to-r ${card.highlight}`}
          >
            <span>{card.icon}</span>
            <span>{card.title}</span>
          </div>
          <div className="text-3xl font-bold text-white">{card.value}</div>
          <p className="text-white/60 text-sm">{card.description}</p>
        </div>
      ))}

      <div className="bg-white/5 border border-white/10 rounded-2xl p-6 backdrop-blur-sm flex flex-col gap-4">
        <div className="inline-flex items-center gap-3 px-3 py-1 rounded-full text-sm text-white bg-gradient-to-r from-purple-500/80 to-indigo-600/80">
          <span>🏦</span>
          <span>Saldo em contas</span>
        </div>
        <div className="text-3xl font-bold text-white">{formatCurrency(accountsBalance)}</div>
        <p className="text-white/60 text-sm">{accounts.length} contas monitoradas</p>
      </div>
    </div>
  );
}
