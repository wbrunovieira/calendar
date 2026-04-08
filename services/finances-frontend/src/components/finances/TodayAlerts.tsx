'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import type { Transaction, RecurringTransaction, Invoice, BankAccount, Category } from '@/types/finances';
import { formatCurrency } from '@/utils/format';

interface TodayAlertsProps {
  transactions: Transaction[];
  recurringTransactions: RecurringTransaction[];
  invoices: Record<string, Invoice[]>;
  accounts: BankAccount[];
  categories: Category[];
  onPayInvoice?: (invoiceId: string, amount: number) => Promise<void>;
  onConfirmTransaction?: (id: string) => void;
  onConfirmRecurring?: (recurring: RecurringTransaction) => Promise<void>;
  onEditTransaction?: (transaction: Transaction) => void;
  onDeleteTransaction?: (id: string) => void;
}

const formatLocalDate = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const formatDisplayDate = (dateStr: string) => {
  if (!dateStr) return '';
  // Extract date part if ISO format
  const cleanDate = dateStr.includes('T') ? dateStr.split('T')[0] : dateStr;
  const [year, month, day] = cleanDate.split('-').map(Number);
  if (!year || !month || !day) return '';
  return `${String(day).padStart(2, '0')}/${String(month).padStart(2, '0')}`;
};

const getDaysOverdue = (dateStr: string, todayStr: string) => {
  if (!dateStr) return 0;
  // Extract date part if ISO format
  const cleanDate = dateStr.includes('T') ? dateStr.split('T')[0] : dateStr;
  const [year, month, day] = cleanDate.split('-').map(Number);
  if (!year || !month || !day) return 0;

  const [todayYear, todayMonth, todayDay] = todayStr.split('-').map(Number);

  const date = new Date(year, month - 1, day);
  const todayDate = new Date(todayYear, todayMonth - 1, todayDay);

  const diffTime = todayDate.getTime() - date.getTime();
  const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));
  return diffDays;
};

// Extract YYYY-MM-DD from any date string
const extractDateStr = (dateStr: string): string => {
  if (!dateStr) return '';
  // If it's already YYYY-MM-DD format
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr;
  // If it's ISO format, extract the date part
  if (dateStr.includes('T')) return dateStr.split('T')[0];
  // Try to parse and format
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return '';
  return formatLocalDate(d);
};

export default function TodayAlerts({
  transactions,
  recurringTransactions,
  invoices,
  accounts,
  onPayInvoice,
  onConfirmTransaction,
  onConfirmRecurring,
  onEditTransaction,
  onDeleteTransaction,
}: TodayAlertsProps) {
  const today = formatLocalDate(new Date());

  // Planned transactions due today or overdue
  const plannedPending = useMemo(() => {
    return transactions.filter((tx) => {
      if (tx.status !== 'PLANNED') return false;
      const occurredDate = extractDateStr(tx.occurredOn);
      const dueDate = tx.dueOn ? extractDateStr(tx.dueOn) : '';
      return occurredDate <= today || (dueDate && dueDate <= today);
    }).sort((a, b) => {
      const dateA = extractDateStr(a.dueOn || a.occurredOn);
      const dateB = extractDateStr(b.dueOn || b.occurredOn);
      return dateA.localeCompare(dateB);
    });
  }, [transactions, today]);

  // Recurring transactions due today or overdue
  // Filter out recurring transactions that have a corresponding confirmed transaction
  const recurringPending = useMemo(() => {
    return recurringTransactions.filter((rt) => {
      if (rt.status !== 'ACTIVE') return false;
      const nextDate = extractDateStr(rt.nextOccurrence);
      if (!nextDate || nextDate > today) return false;

      // Check if there's a confirmed or planned transaction that matches this recurring
      // Match by: same description (case insensitive), same amount
      // and occurred within a reasonable window (same month or up to 7 days difference)
      const hasMatchingTransaction = transactions.some((tx) => {
        if (tx.status !== 'CONFIRMED' && tx.status !== 'PLANNED') return false;
        if (tx.type !== rt.type) return false;
        if (Math.abs(tx.amount - rt.amount) > 0.01) return false; // Allow small float difference

        // Check if descriptions match (case insensitive, trimmed)
        const txDesc = tx.description.toLowerCase().trim();
        const rtDesc = rt.description.toLowerCase().trim();
        if (txDesc !== rtDesc) return false;

        // Check if the transaction occurred within a reasonable window
        const txDate = extractDateStr(tx.occurredOn);
        const rtNextDate = extractDateStr(rt.nextOccurrence);
        if (!txDate || !rtNextDate) return false;

        // Same month check or within 7 days
        const txMonth = txDate.substring(0, 7); // YYYY-MM
        const rtMonth = rtNextDate.substring(0, 7);
        if (txMonth === rtMonth) return true;

        // Or within 7 days difference
        const daysDiff = Math.abs(getDaysOverdue(txDate, rtNextDate));
        return daysDiff <= 7;
      });

      return !hasMatchingTransaction;
    }).sort((a, b) => {
      const dateA = extractDateStr(a.nextOccurrence);
      const dateB = extractDateStr(b.nextOccurrence);
      return dateA.localeCompare(dateB);
    });
  }, [recurringTransactions, transactions, today]);

  // Invoices due today or overdue (unpaid)
  const invoicesPending = useMemo(() => {
    const result: { invoice: Invoice; account: BankAccount; isOverdue: boolean; daysOverdue: number }[] = [];
    Object.entries(invoices).forEach(([accountId, accountInvoices]) => {
      const account = accounts.find((a) => a.id === accountId);
      if (!account) return;
      accountInvoices.forEach((inv) => {
        const dueDate = extractDateStr(inv.dueDate);
        if (inv.status !== 'PAID' && dueDate && dueDate <= today && inv.amount > 0) {
          const daysOverdue = getDaysOverdue(inv.dueDate, today);
          result.push({
            invoice: inv,
            account,
            isOverdue: dueDate < today,
            daysOverdue
          });
        }
      });
    });
    return result.sort((a, b) => {
      const dateA = extractDateStr(a.invoice.dueDate);
      const dateB = extractDateStr(b.invoice.dueDate);
      return dateA.localeCompare(dateB);
    });
  }, [invoices, accounts, today]);

  // Separate today vs overdue
  const overdueInvoices = invoicesPending.filter(i => i.isOverdue);
  const todayInvoices = invoicesPending.filter(i => !i.isOverdue);

  const plannedOverdue = plannedPending.filter(tx => {
    const date = extractDateStr(tx.dueOn || tx.occurredOn);
    return date < today;
  });
  const plannedToday = plannedPending.filter(tx => {
    const date = extractDateStr(tx.dueOn || tx.occurredOn);
    return date === today;
  });

  const recurringOverdue = recurringPending.filter(rt => {
    const date = extractDateStr(rt.nextOccurrence);
    return date < today;
  });
  const recurringToday = recurringPending.filter(rt => {
    const date = extractDateStr(rt.nextOccurrence);
    return date === today;
  });

  // Calculate totals
  const totals = useMemo(() => {
    let toPay = 0;
    let toReceive = 0;
    let overduePay = 0;
    let overdueReceive = 0;

    plannedToday.forEach((tx) => {
      if (tx.type === 'EXPENSE') toPay += tx.amount;
      else if (tx.type === 'INCOME') toReceive += tx.amount;
    });

    plannedOverdue.forEach((tx) => {
      if (tx.type === 'EXPENSE') overduePay += tx.amount;
      else if (tx.type === 'INCOME') overdueReceive += tx.amount;
    });

    recurringToday.forEach((rt) => {
      if (rt.type === 'EXPENSE') toPay += rt.amount;
      else if (rt.type === 'INCOME') toReceive += rt.amount;
    });

    recurringOverdue.forEach((rt) => {
      if (rt.type === 'EXPENSE') overduePay += rt.amount;
      else if (rt.type === 'INCOME') overdueReceive += rt.amount;
    });

    todayInvoices.forEach(({ invoice }) => {
      toPay += invoice.amount;
    });

    overdueInvoices.forEach(({ invoice }) => {
      overduePay += invoice.amount;
    });

    return { toPay, toReceive, overduePay, overdueReceive };
  }, [plannedToday, plannedOverdue, recurringToday, recurringOverdue, todayInvoices, overdueInvoices]);

  // Loading state for async actions (prevents double clicks)
  const [loadingAction, setLoadingAction] = useState<string | null>(null);

  const handlePayInvoice = async (invoiceId: string, amount: number) => {
    if (loadingAction || !onPayInvoice) return;
    setLoadingAction(invoiceId);
    try {
      await onPayInvoice(invoiceId, amount);
    } finally {
      setLoadingAction(null);
    }
  };

  const handleConfirmRecurring = async (rt: RecurringTransaction) => {
    if (loadingAction || !onConfirmRecurring) return;
    setLoadingAction(rt.id);
    try {
      await onConfirmRecurring(rt);
    } finally {
      setLoadingAction(null);
    }
  };

  const handleConfirmTransaction = async (id: string) => {
    if (loadingAction || !onConfirmTransaction) return;
    setLoadingAction(id);
    try {
      await onConfirmTransaction(id);
    } finally {
      setLoadingAction(null);
    }
  };

  // State for dismissed review reminders (persisted in session)
  const [dismissedReviews, setDismissedReviews] = useState<Set<string>>(() => {
    if (typeof window !== 'undefined') {
      const stored = sessionStorage.getItem('dismissedReviewAlerts');
      if (stored) {
        return new Set(JSON.parse(stored));
      }
    }
    return new Set();
  });

  const dismissReview = (id: string) => {
    const newDismissed = new Set(dismissedReviews);
    newDismissed.add(id);
    setDismissedReviews(newDismissed);
    if (typeof window !== 'undefined') {
      sessionStorage.setItem('dismissedReviewAlerts', JSON.stringify([...newDismissed]));
    }
  };

  // State for dismissed transaction reminders (persisted in session)
  const [dismissedTxReminders, setDismissedTxReminders] = useState<Set<string>>(() => {
    if (typeof window !== 'undefined') {
      const stored = sessionStorage.getItem('dismissedTxReminderAlerts');
      if (stored) {
        return new Set(JSON.parse(stored));
      }
    }
    return new Set();
  });

  const dismissTxReminder = (id: string) => {
    const newDismissed = new Set(dismissedTxReminders);
    newDismissed.add(id);
    setDismissedTxReminders(newDismissed);
    if (typeof window !== 'undefined') {
      sessionStorage.setItem('dismissedTxReminderAlerts', JSON.stringify([...newDismissed]));
    }
  };

  // Paused transactions with review dates approaching (10, 5, 1, 0 days)
  const reviewReminders = useMemo(() => {
    const alertDays = [10, 5, 1, 0];
    const results: { rt: RecurringTransaction; daysUntil: number }[] = [];

    recurringTransactions.forEach((rt) => {
      if (rt.status !== 'PAUSED' || !rt.reviewOn) return;
      if (dismissedReviews.has(rt.id)) return;

      const reviewDate = extractDateStr(rt.reviewOn);
      if (!reviewDate) return;

      const daysUntil = -getDaysOverdue(reviewDate, today); // Negative because we want days until, not days since

      // Show alert if within any of our thresholds
      if (alertDays.includes(daysUntil) || daysUntil < 0) {
        results.push({ rt, daysUntil });
      }
    });

    return results.sort((a, b) => a.daysUntil - b.daysUntil);
  }, [recurringTransactions, today, dismissedReviews]);

  // Transaction reminders with reminderOn dates approaching (10, 5, 1, 0 days) or overdue
  const transactionReminders = useMemo(() => {
    const alertDays = [10, 5, 1, 0];
    const results: { tx: Transaction; daysUntil: number }[] = [];

    transactions.forEach((tx) => {
      // Only show reminders for PLANNED transactions with reminderOn set
      if (tx.status !== 'PLANNED' || !tx.reminderOn) return;
      if (dismissedTxReminders.has(tx.id)) return;

      const reminderDate = extractDateStr(tx.reminderOn);
      if (!reminderDate) return;

      const daysUntil = -getDaysOverdue(reminderDate, today); // Negative because we want days until, not days since

      // Show alert if within any of our thresholds or if overdue
      if (alertDays.includes(daysUntil) || daysUntil < 0) {
        results.push({ tx, daysUntil });
      }
    });

    return results.sort((a, b) => a.daysUntil - b.daysUntil);
  }, [transactions, today, dismissedTxReminders]);

  const hasAlerts =
    plannedPending.length > 0 ||
    recurringPending.length > 0 ||
    invoicesPending.length > 0 ||
    reviewReminders.length > 0 ||
    transactionReminders.length > 0;

  const hasOverdue =
    overdueInvoices.length > 0 ||
    plannedOverdue.length > 0 ||
    recurringOverdue.length > 0;

  const getAccountName = (accountId?: string) => {
    if (!accountId) return '';
    const acc = accounts.find((a) => a.id === accountId);
    return acc?.name || '';
  };

  // No pending items - show friendly message
  if (!hasAlerts) {
    return (
      <div className="bg-gradient-to-r from-emerald-500/20 via-teal-500/20 to-cyan-500/20 border border-emerald-400/30 rounded-2xl p-4 backdrop-blur-sm">
        <div className="flex items-center gap-3">
          <span className="text-2xl">✨</span>
          <div>
            <h3 className="text-white font-semibold">Tudo em dia!</h3>
            <p className="text-white/60 text-sm">
              Nenhuma pendencia para hoje. Continue assim!
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Overdue Items - Red alert */}
      {hasOverdue && (
        <div className="bg-gradient-to-r from-red-500/20 via-red-600/20 to-rose-500/20 border border-red-400/30 rounded-2xl p-4 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-3">
            <span className="text-2xl">⚠️</span>
            <div className="flex-1">
              <h3 className="text-white font-semibold">Itens Atrasados</h3>
              <div className="flex gap-4 text-sm">
                {totals.overduePay > 0 && (
                  <span className="text-red-400">
                    A pagar: {formatCurrency(totals.overduePay)}
                  </span>
                )}
                {totals.overdueReceive > 0 && (
                  <span className="text-emerald-400">
                    A receber: {formatCurrency(totals.overdueReceive)}
                  </span>
                )}
              </div>
            </div>
          </div>

          <div className="space-y-2">
            {/* Overdue Invoices */}
            {overdueInvoices.map(({ invoice, account, daysOverdue }) => (
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
                    <p className="text-red-400 text-xs">
                      {daysOverdue === 1 ? 'Venceu ontem' : `Venceu ha ${daysOverdue} dias`} ({formatDisplayDate(invoice.dueDate)})
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-red-400 font-semibold">
                    {formatCurrency(invoice.amount)}
                  </span>
                  {onPayInvoice && (
                    <button
                      onClick={() => handlePayInvoice(invoice.id, invoice.amount)}
                      disabled={!!loadingAction}
                      className="px-3 py-1.5 bg-red-600 hover:bg-red-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                    >
                      {loadingAction === invoice.id ? 'Pagando...' : 'Pagar'}
                    </button>
                  )}
                </div>
              </div>
            ))}

            {/* Overdue Recurring */}
            {recurringOverdue.map((rt) => {
              const daysOverdue = getDaysOverdue(rt.nextOccurrence, today);
              return (
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
                      <p className="text-red-400 text-xs">
                        Fixa atrasada - {daysOverdue === 1 ? 'ontem' : `${daysOverdue} dias`} ({formatDisplayDate(rt.nextOccurrence)})
                        {getAccountName(rt.bankAccountId) && <span className="text-white/50"> • {getAccountName(rt.bankAccountId)}</span>}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className={rt.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'} style={{ fontWeight: 600 }}>
                      {rt.type === 'INCOME' ? '+' : '-'}{formatCurrency(rt.amount, rt.currency)}
                    </span>
                    {onConfirmRecurring && (
                      <button
                        onClick={() => handleConfirmRecurring(rt)}
                        disabled={!!loadingAction}
                        className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                      >
                        {loadingAction === rt.id ? 'Processando...' : (rt.type === 'INCOME' ? 'Ja recebi' : 'Ja paguei')}
                      </button>
                    )}
                  </div>
                </div>
              );
            })}

            {/* Overdue Planned */}
            {plannedOverdue.map((tx) => {
              const date = tx.dueOn || tx.occurredOn;
              const daysOverdue = getDaysOverdue(date, today);
              return (
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
                      <p className="text-red-400 text-xs">
                        Atrasado - {daysOverdue === 1 ? 'ontem' : `${daysOverdue} dias`} ({formatDisplayDate(date)})
                        {getAccountName(tx.bankAccountId) && <span className="text-white/50"> • {getAccountName(tx.bankAccountId)}</span>}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className={tx.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'} style={{ fontWeight: 600 }}>
                      {tx.type === 'INCOME' ? '+' : '-'}{formatCurrency(tx.amount, tx.currency)}
                    </span>
                    {onEditTransaction && (
                      <button
                        onClick={() => onEditTransaction(tx)}
                        className="px-3 py-1.5 bg-white/10 hover:bg-white/20 text-white/70 text-xs rounded-lg transition-colors"
                      >
                        Editar
                      </button>
                    )}
                    {onConfirmTransaction && (
                      <button
                        onClick={() => handleConfirmTransaction(tx.id)}
                        disabled={!!loadingAction}
                        className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                      >
                        {loadingAction === tx.id ? 'Confirmando...' : 'Confirmar'}
                      </button>
                    )}
                    {onDeleteTransaction && (
                      <button
                        onClick={() => onDeleteTransaction(tx.id)}
                        className="px-2 py-1.5 bg-red-500/20 hover:bg-red-500/40 text-red-400 hover:text-red-300 text-xs rounded-lg transition-colors"
                        title="Excluir lançamento"
                      >
                        🗑️
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Today Items - Amber alert */}
      {(todayInvoices.length > 0 || recurringToday.length > 0 || plannedToday.length > 0) && (
        <div className="bg-gradient-to-r from-amber-500/20 via-orange-500/20 to-yellow-500/20 border border-amber-400/30 rounded-2xl p-4 backdrop-blur-sm">
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
            {todayInvoices.map(({ invoice, account }) => (
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
                      onClick={() => handlePayInvoice(invoice.id, invoice.amount)}
                      disabled={!!loadingAction}
                      className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                    >
                      {loadingAction === invoice.id ? 'Pagando...' : 'Pagar'}
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
                      {getAccountName(rt.bankAccountId) && ` • ${getAccountName(rt.bankAccountId)}`}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className={rt.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'} style={{ fontWeight: 600 }}>
                    {rt.type === 'INCOME' ? '+' : '-'}{formatCurrency(rt.amount, rt.currency)}
                  </span>
                  {onConfirmRecurring && (
                    <button
                      onClick={() => onConfirmRecurring(rt)}
                      className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white text-xs rounded-lg transition-colors"
                    >
                      {rt.type === 'INCOME' ? 'Ja recebi' : 'Ja paguei'}
                    </button>
                  )}
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
                      {getAccountName(tx.bankAccountId) && ` • ${getAccountName(tx.bankAccountId)}`}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className={tx.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'} style={{ fontWeight: 600 }}>
                    {tx.type === 'INCOME' ? '+' : '-'}{formatCurrency(tx.amount, tx.currency)}
                  </span>
                  {onEditTransaction && (
                    <button
                      onClick={() => onEditTransaction(tx)}
                      className="px-3 py-1.5 bg-white/10 hover:bg-white/20 text-white/70 text-xs rounded-lg transition-colors"
                    >
                      Editar
                    </button>
                  )}
                  {onConfirmTransaction && (
                    <button
                      onClick={() => onConfirmTransaction(tx.id)}
                      className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-xs rounded-lg transition-colors"
                    >
                      Confirmar
                    </button>
                  )}
                  {onDeleteTransaction && (
                    <button
                      onClick={() => onDeleteTransaction(tx.id)}
                      className="px-2 py-1.5 bg-red-500/20 hover:bg-red-500/40 text-red-400 hover:text-red-300 text-xs rounded-lg transition-colors"
                      title="Excluir lançamento"
                    >
                      🗑️
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Review Reminders for Paused Transactions - Purple alert */}
      {reviewReminders.length > 0 && (
        <div className="bg-gradient-to-r from-purple-500/20 via-violet-500/20 to-indigo-500/20 border border-purple-400/30 rounded-2xl p-4 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-3">
            <span className="text-2xl">⏸️</span>
            <div className="flex-1">
              <h3 className="text-white font-semibold">Revisao de Assinaturas Pausadas</h3>
              <p className="text-white/60 text-sm">
                {reviewReminders.length} {reviewReminders.length === 1 ? 'assinatura precisa' : 'assinaturas precisam'} de revisao
              </p>
            </div>
            <Link
              href="/recurring"
              className="text-purple-300 hover:text-purple-200 text-sm underline"
            >
              Ver todas
            </Link>
          </div>

          <div className="space-y-2">
            {reviewReminders.map(({ rt, daysUntil }) => {
              const getReviewMessage = () => {
                if (daysUntil < 0) {
                  const daysOverdue = Math.abs(daysUntil);
                  return <span className="text-red-400">Revisao atrasada ha {daysOverdue} {daysOverdue === 1 ? 'dia' : 'dias'}</span>;
                }
                if (daysUntil === 0) return <span className="text-amber-400">Revisar hoje!</span>;
                if (daysUntil === 1) return <span className="text-amber-400">Revisar amanha</span>;
                return <span className="text-purple-300">Revisar em {daysUntil} dias</span>;
              };

              return (
                <div
                  key={rt.id}
                  className="flex items-center justify-between bg-white/5 rounded-xl p-3"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-xl">
                      {rt.type === 'INCOME' ? '💰' : '📋'}
                    </span>
                    <div>
                      <p className="text-white text-sm font-medium">{rt.description}</p>
                      <p className="text-xs">
                        {getReviewMessage()}
                        <span className="text-white/50"> • {formatDisplayDate(rt.reviewOn!)} • {formatCurrency(rt.amount, rt.currency)}/mes</span>
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => dismissReview(rt.id)}
                      className="px-3 py-1.5 bg-white/10 hover:bg-white/20 text-white/70 text-xs rounded-lg transition-colors"
                      title="Ignorar por agora"
                    >
                      Ignorar
                    </button>
                    <Link
                      href="/recurring"
                      className="px-3 py-1.5 bg-purple-600 hover:bg-purple-700 text-white text-xs rounded-lg transition-colors"
                    >
                      Revisar
                    </Link>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Transaction Reminders - Cyan alert */}
      {transactionReminders.length > 0 && (
        <div className="bg-gradient-to-r from-cyan-500/20 via-teal-500/20 to-emerald-500/20 border border-cyan-400/30 rounded-2xl p-4 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-3">
            <span className="text-2xl">🔔</span>
            <div className="flex-1">
              <h3 className="text-white font-semibold">Lembretes de Lancamentos</h3>
              <p className="text-white/60 text-sm">
                {transactionReminders.length} {transactionReminders.length === 1 ? 'lembrete ativo' : 'lembretes ativos'}
              </p>
            </div>
          </div>

          <div className="space-y-2">
            {transactionReminders.map(({ tx, daysUntil }) => {
              const getReminderMessage = () => {
                if (daysUntil < 0) {
                  const daysOverdue = Math.abs(daysUntil);
                  return <span className="text-red-400">Lembrete atrasado ha {daysOverdue} {daysOverdue === 1 ? 'dia' : 'dias'}</span>;
                }
                if (daysUntil === 0) return <span className="text-amber-400">Lembrete para hoje!</span>;
                if (daysUntil === 1) return <span className="text-amber-400">Lembrete para amanha</span>;
                return <span className="text-cyan-300">Lembrete em {daysUntil} dias</span>;
              };

              return (
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
                      <p className="text-xs">
                        {getReminderMessage()}
                        <span className="text-white/50"> • {formatDisplayDate(tx.reminderOn!)} • </span>
                        <span className={tx.type === 'INCOME' ? 'text-emerald-400' : 'text-red-400'}>
                          {formatCurrency(tx.amount, tx.currency)}
                        </span>
                        {getAccountName(tx.bankAccountId) && <span className="text-white/50"> • {getAccountName(tx.bankAccountId)}</span>}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => dismissTxReminder(tx.id)}
                      className="px-3 py-1.5 bg-white/10 hover:bg-white/20 text-white/70 text-xs rounded-lg transition-colors"
                      title="Ignorar por agora"
                    >
                      Ignorar
                    </button>
                    {onEditTransaction && (
                      <button
                        onClick={() => onEditTransaction(tx)}
                        className="px-3 py-1.5 bg-white/10 hover:bg-white/20 text-white/70 text-xs rounded-lg transition-colors"
                      >
                        Editar
                      </button>
                    )}
                    {onConfirmTransaction && (
                      <button
                        onClick={() => handleConfirmTransaction(tx.id)}
                        disabled={!!loadingAction}
                        className="px-3 py-1.5 bg-cyan-600 hover:bg-cyan-700 disabled:bg-gray-600 text-white text-xs rounded-lg transition-colors"
                      >
                        {loadingAction === tx.id ? 'Confirmando...' : 'Confirmar'}
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
