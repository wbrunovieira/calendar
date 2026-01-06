'use client';

import type { BankAccount, Transaction } from '@/types/finances';

// Format date to YYYY-MM-DD without timezone conversion
const formatLocalDate = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

interface Props {
  accounts: BankAccount[];
  transactions: Transaction[];
}

export default function SafeToSpend({ accounts, transactions }: Props) {
  // Available balance from regular accounts (checking, savings, cash) - not credit cards or investments
  const availableBalance = accounts
    .filter((acc) => acc.type !== 'CREDIT_CARD' && acc.type !== 'INVESTMENT')
    .reduce((sum, acc) => sum + acc.currentBalance, 0);

  // Get current month expenses (confirmed)
  const now = new Date();
  const monthStart = formatLocalDate(new Date(now.getFullYear(), now.getMonth(), 1));
  const monthEnd = formatLocalDate(new Date(now.getFullYear(), now.getMonth() + 1, 0));

  const monthExpenses = transactions
    .filter((tx) => {
      const date = tx.occurredOn.slice(0, 10);
      return (
        tx.type === 'EXPENSE' &&
        tx.status === 'CONFIRMED' &&
        date >= monthStart &&
        date <= monthEnd
      );
    })
    .reduce((sum, tx) => sum + tx.amount, 0);

  const monthIncome = transactions
    .filter((tx) => {
      const date = tx.occurredOn.slice(0, 10);
      return (
        tx.type === 'INCOME' &&
        tx.status === 'CONFIRMED' &&
        date >= monthStart &&
        date <= monthEnd
      );
    })
    .reduce((sum, tx) => sum + tx.amount, 0);

  // Remaining = Available balance (already reflects past transactions)
  // But we show monthly context
  const remaining = availableBalance;

  const daysInMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
  const today = now.getDate();
  const daysLeft = Math.max(1, daysInMonth - today + 1);
  const perDay = remaining / daysLeft;
  const perWeek = perDay * 7;

  return (
    <div className="grid gap-4 sm:grid-cols-3">
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <p className="text-white/60 text-sm">Saldo disponivel</p>
        <p className={`text-2xl font-bold ${remaining < 0 ? 'text-rose-200' : 'text-white'}`}>
          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(remaining)}
        </p>
        <p className="text-white/40 text-xs mt-1">
          +{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(monthIncome)} entradas /
          -{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(monthExpenses)} saidas
        </p>
      </div>
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <p className="text-white/60 text-sm">Safe-to-spend diario</p>
        <p className={`text-2xl font-bold ${perDay < 0 ? 'text-rose-200' : 'text-white'}`}>
          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(perDay)}
        </p>
        <p className="text-white/40 text-xs mt-1">{daysLeft} dias restantes no mes</p>
      </div>
      <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
        <p className="text-white/60 text-sm">Safe-to-spend semanal</p>
        <p className={`text-2xl font-bold ${perWeek < 0 ? 'text-rose-200' : 'text-white'}`}>
          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(perWeek)}
        </p>
        <p className="text-white/40 text-xs mt-1">baseado no saldo atual</p>
      </div>
    </div>
  );
}

