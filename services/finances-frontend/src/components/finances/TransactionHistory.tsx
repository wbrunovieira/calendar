'use client';

import { useState, useEffect, useMemo } from 'react';
import { api } from '@/lib/api';
import { formatCurrency, formatDate } from '@/utils/format';
import { transactionStatusConfig } from '@/utils/constants';
import { useToast } from '@/components/ui/Toast';
import type { Category, Transaction } from '@/types/finances';

interface TransactionHistoryProps {
  accountId: string;
  profileId: string;
  categories: Category[];
  selectedInvoiceId?: string;
  accountCurrency?: string;
  isCreditCard?: boolean;
  includeAsDestination?: boolean;
  botFilter?: string;
  onEdit?: (tx: Transaction) => void;
  onDelete?: (tx: Transaction) => void;
  onConfirm?: (tx: Transaction) => Promise<void> | void;
}

export default function TransactionHistory({
  accountId,
  profileId,
  categories,
  selectedInvoiceId,
  accountCurrency = 'BRL',
  isCreditCard = false,
  includeAsDestination = false,
  botFilter,
  onEdit,
  onDelete,
  onConfirm,
}: TransactionHistoryProps) {
  const { confirm } = useToast();
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [dayBalances, setDayBalances] = useState<Record<string, { balance: number; dayTotal: number }>>({});
  const [confirmingId, setConfirmingId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const categoryMap = useMemo(() => {
    const map: Record<string, string> = {};
    categories.forEach((c) => { map[c.id] = c.name; });
    return map;
  }, [categories]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const params = new URLSearchParams({ profileId, bankAccountId: accountId });
        if (selectedInvoiceId) {
          params.set('invoiceId', selectedInvoiceId);
        }
        if (includeAsDestination) {
          params.set('includeAsDestination', 'true');
        }

        const [txData, balanceData] = await Promise.all([
          api.get<{ data: Transaction[] }>(`/transactions?${params}`),
          api.get<{ data: { date: string; balance: number; dayTotal: number }[] }>(`/transactions/daily-balances?${params}`),
        ]);

        setTransactions(txData.data || []);

        {
          const balMap: Record<string, { balance: number; dayTotal: number }> = {};
          for (const entry of (balanceData.data || [])) {
            balMap[entry.date] = { balance: entry.balance, dayTotal: entry.dayTotal };
          }
          setDayBalances(balMap);
        }
      } catch (error) {
        console.warn('Erro ao carregar historico:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [accountId, profileId, selectedInvoiceId, includeAsDestination]);

  const groupedByDay = useMemo(() => {
    const filtered = botFilter
      ? transactions.filter(tx =>
          tx.externalId?.startsWith(botFilter + '-') ||
          tx.description?.includes(`(${botFilter})`)
        )
      : transactions;
    const groups: Record<string, Transaction[]> = {};
    for (const tx of filtered) {
      const dateKey = tx.occurredOn.split('T')[0];
      if (!groups[dateKey]) groups[dateKey] = [];
      groups[dateKey].push(tx);
    }
    return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b));
  }, [transactions, botFilter]);

  const groupedByDayDesc = useMemo(
    () => [...groupedByDay].reverse(),
    [groupedByDay],
  );

  if (loading) {
    return (
      <div className="py-4 text-center text-white/50 text-sm">
        Carregando historico...
      </div>
    );
  }

  if (transactions.length === 0) {
    return (
      <div className="py-4 text-center text-white/40 text-sm">
        Nenhuma transacao encontrada para esta conta.
      </div>
    );
  }

  return (
    <div className={`space-y-2 max-h-[28rem] overflow-y-auto scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent ${isCreditCard ? 'mt-2 border-l-2 border-emerald-500/20 pl-3 mr-6 bg-emerald-950/10 rounded-r-lg py-2 pr-2' : ''}`}>
      {groupedByDayDesc.map(([dateKey, dayTransactions]) => {
        const balEntry = dayBalances[dateKey];
        const endOfDayBalance = balEntry?.balance ?? 0;

        const dayTotal = isCreditCard
          ? dayTransactions.reduce((sum, tx) => sum + tx.amount, 0)
          : (balEntry?.dayTotal ?? 0);

        return (
          <div key={dateKey}>
            <div className={`flex items-center justify-between px-3 py-1.5 rounded-t-lg border-b ${isCreditCard ? 'bg-white/[0.04] border-white/[0.06]' : 'bg-white/10 border-white/10'}`}>
              <span className={`text-xs font-semibold ${isCreditCard ? 'text-white/50' : 'text-white/70'}`}>{formatDate(dateKey)}</span>
              <div className="flex items-center gap-3">
                {isCreditCard ? (
                  <span className="text-xs font-semibold text-orange-400/80">
                    {formatCurrency(dayTotal, accountCurrency)}
                  </span>
                ) : (
                  <span className={`text-xs font-semibold ${dayTotal >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
                    {dayTotal >= 0 ? '+' : ''}{formatCurrency(Math.abs(dayTotal), accountCurrency)}
                  </span>
                )}
                {!isCreditCard && (
                  <span className={`text-xs font-medium ${endOfDayBalance >= 0 ? 'text-white/60' : 'text-red-300'}`}>
                    Saldo: {formatCurrency(endOfDayBalance, accountCurrency)}
                  </span>
                )}
              </div>
            </div>
            <div className="space-y-0.5 mt-0.5">
              {dayTransactions.map((tx) => {
                const status = transactionStatusConfig[tx.status] || transactionStatusConfig.PLANNED;
                const isExpense = tx.type === 'EXPENSE';
                const isIncome = tx.type === 'INCOME';
                return (
                  <div
                    key={tx.id}
                    className={`flex items-center justify-between py-1.5 px-3 rounded-lg ${isCreditCard ? 'bg-white/[0.02] hover:bg-white/[0.05]' : 'bg-white/5'}`}
                  >
                    <div className="flex-1 min-w-0">
                      <p className={`truncate ${isCreditCard ? 'text-white/80 text-xs' : 'text-white text-sm'}`}>{tx.description}</p>
                      <div className="flex items-center gap-2 text-white/40 text-xs">
                        {tx.categoryId && categoryMap[tx.categoryId] && (
                          <span className="truncate">{categoryMap[tx.categoryId]}</span>
                        )}
                      </div>
                      {tx.notes && (
                        <p className="text-white/30 text-[10px] mt-0.5 truncate">{tx.notes}</p>
                      )}
                    </div>
                    <div className="flex items-center gap-2 ml-3 shrink-0">
                      <div className="text-right">
                        <p className={`font-semibold ${isCreditCard ? 'text-xs' : 'text-sm'} ${isExpense ? 'text-red-400' : isIncome ? 'text-emerald-400' : 'text-blue-400'}`}>
                          {isExpense ? '-' : isIncome ? '+' : ''}{formatCurrency(tx.amount, tx.currency)}
                        </p>
                      </div>
                      <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${status.color}`}>
                        {status.label}
                      </span>
                      {onConfirm && tx.status === 'PLANNED' && (
                        <button
                          type="button"
                          disabled={!!confirmingId}
                          onClick={async () => {
                            if (confirmingId) return;
                            setConfirmingId(tx.id);
                            try { await onConfirm(tx); } finally { setConfirmingId(null); }
                          }}
                          className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 hover:bg-emerald-500/30 transition-colors disabled:opacity-50"
                          title="Confirmar lançamento"
                        >
                          {confirmingId === tx.id ? 'Confirmando...' : 'Confirmar'}
                        </button>
                      )}
                      {onEdit && (
                        <button
                          type="button"
                          onClick={() => onEdit(tx)}
                          className="text-white/30 hover:text-white/70 transition-colors p-1"
                          title="Editar lançamento"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                          </svg>
                        </button>
                      )}
                      {onDelete && (
                        <button
                          type="button"
                          onClick={async () => {
                            const ok = await confirm(
                              `Excluir "${tx.description}"?`,
                              `${formatCurrency(tx.amount)} sera removido permanentemente.`,
                            );
                            if (ok) onDelete(tx);
                          }}
                          className="text-white/30 hover:text-red-400 transition-colors p-1"
                          title="Excluir lançamento"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
