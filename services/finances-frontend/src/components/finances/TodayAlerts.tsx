'use client';

import { useMemo } from 'react';
import type { Transaction, RecurringTransaction, Invoice, BankAccount, Category } from '@/types/finances';

interface TodayAlertsProps {
  transactions: Transaction[];
  recurringTransactions: RecurringTransaction[];
  invoices: Record<string, Invoice[]>;
  accounts: BankAccount[];
  categories: Category[];
  onPayInvoice?: (invoiceId: string, amount: number) => Promise<void>;
  onConfirmTransaction?: (id: string) => void;
}

const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

const formatLocalDate = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

export default function TodayAlerts({
  transactions,
  recurringTransactions,
  invoices,
  accounts,
  categories,
  onPayInvoice,
  onConfirmTransaction,
}: TodayAlertsProps) {
  const today = formatLocalDate(new Date());

  // Planned transactions due today
  const plannedToday = useMemo(() => {
    return transactions.filter(
      (tx) =>
        tx.status === 'PLANNED' &&
        (tx.occurredOn === today || tx.dueOn === today)
    );
  }, [transactions, today]);

  // Recurring transactions due today
  const recurringToday = useMemo(() => {
    return recurringTransactions.filter(
      (rt) => rt.status === 'ACTIVE' && rt.nextOccurrence === today
    );
  }, [recurringTransactions, today]);

  // Invoices due today (unpaid)
  const invoicesToday = useMemo(() => {
    const result: { invoice: Invoice; account: BankAccount }[] = [];
    Object.entries(invoices).forEach(([accountId, accountInvoices]) => {
      const account = accounts.find((a) => a.id === accountId);
      if (!account) return;
      accountInvoices.forEach((inv) => {
        if (inv.status !== 'PAID' && inv.dueDate === today) {
          result.push({ invoice: inv, account });
        }
      });
    });
    return result;
  }, [invoices, accounts, today]);

  // Calculate totals
  const totals = useMemo(() => {
    let toPay = 0;
    let toReceive = 0;

    plannedToday.forEach((tx) => {
      if (tx.type === 'EXPENSE') toPay += tx.amount;
      else if (tx.type === 'INCOME') toReceive += tx.amount;
    });

    recurringToday.forEach((rt) => {
      if (rt.type === 'EXPENSE') toPay += rt.amount;
      else if (rt.type === 'INCOME') toReceive += rt.amount;
    });

    invoicesToday.forEach(({ invoice }) => {
      toPay += invoice.amount;
    });

    return { toPay, toReceive };
  }, [plannedToday, recurringToday, invoicesToday]);

  const hasAlerts =
    plannedToday.length > 0 ||
    recurringToday.length > 0 ||
    invoicesToday.length > 0;

  if (!hasAlerts) return null;

  const getCategoryName = (categoryId?: string) => {
    if (!categoryId) return '';
    const cat = categories.find((c) => c.id === categoryId);
    return cat?.name || '';
  };

  const getAccountName = (accountId?: string) => {
    if (!accountId) return '';
    const acc = accounts.find((a) => a.id === accountId);
    return acc?.name || '';
  };

  return (
    <div className="bg-gradient-to-r from-amber-500/20 via-orange-500/20 to-red-500/20 border border-amber-400/30 rounded-2xl p-4 backdrop-blur-sm">
      <div className="flex items-center gap-3 mb-3">
        <span className="text-2xl">🔔</span>
        <div className="flex-1">
          <h3 className="text-white font-semibold">Pendencias de Hoje</h3>
          <div className="flex gap-4 text-sm">
            {totals.toPay > 0 && (
              <span className="text-red-400">
                A pagar: {formatCurrency(totals.toPay)}
              </span>
            )}
            {totals.toReceive > 0 && (
              <span className="text-emerald-400">
                A receber: {formatCurrency(totals.toReceive)}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="space-y-2">
        {/* Invoices due today */}
        {invoicesToday.map(({ invoice, account }) => (
          <div
            key={invoice.id}
            className="flex items-center justify-between bg-white/5 rounded-xl p-3"
          >
            <div className="flex items-center gap-3">
              <span className="text-xl">💳</span>
              <div>
                <p className="text-white text-sm font-medium">
                  Fatura {account.name}
                </p>
                <p className="text-white/50 text-xs">Vence hoje</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className="text-red-400 font-semibold">
                {formatCurrency(invoice.amount)}
              </span>
              {onPayInvoice && (
                <button
                  onClick={() => onPayInvoice(invoice.id, invoice.amount)}
                  className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white text-xs rounded-lg transition-colors"
                >
                  Pagar
                </button>
              )}
            </div>
          </div>
        ))}

        {/* Recurring transactions due today */}
        {recurringToday.map((rt) => (
          <div
            key={rt.id}
            className="flex items-center justify-between bg-white/5 rounded-xl p-3"
          >
            <div className="flex items-center gap-3">
              <span className="text-xl">
                {rt.type === 'INCOME' ? '💰' : '📅'}
              </span>
              <div>
                <p className="text-white text-sm font-medium">{rt.description}</p>
                <p className="text-white/50 text-xs">
                  Fixa {rt.type === 'INCOME' ? '(receita)' : '(despesa)'}
                  {getCategoryName(rt.categoryId) && ` - ${getCategoryName(rt.categoryId)}`}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={rt.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'} style={{ fontWeight: 600 }}>
                {rt.type === 'INCOME' ? '+' : '-'}{formatCurrency(rt.amount, rt.currency)}
              </span>
            </div>
          </div>
        ))}

        {/* Planned transactions for today */}
        {plannedToday.map((tx) => (
          <div
            key={tx.id}
            className="flex items-center justify-between bg-white/5 rounded-xl p-3"
          >
            <div className="flex items-center gap-3">
              <span className="text-xl">
                {tx.type === 'INCOME' ? '💵' : tx.type === 'TRANSFER' ? '🔄' : '📋'}
              </span>
              <div>
                <p className="text-white text-sm font-medium">{tx.description}</p>
                <p className="text-white/50 text-xs">
                  Planejado
                  {getCategoryName(tx.categoryId) && ` - ${getCategoryName(tx.categoryId)}`}
                  {getAccountName(tx.bankAccountId) && ` - ${getAccountName(tx.bankAccountId)}`}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={tx.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'} style={{ fontWeight: 600 }}>
                {tx.type === 'INCOME' ? '+' : '-'}{formatCurrency(tx.amount, tx.currency)}
              </span>
              {onConfirmTransaction && (
                <button
                  onClick={() => onConfirmTransaction(tx.id)}
                  className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-xs rounded-lg transition-colors"
                >
                  Confirmar
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
