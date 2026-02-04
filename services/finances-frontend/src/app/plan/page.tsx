'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout, { ProfileTheme } from '@/components/layout/AppLayout';
import TransactionForm from '@/components/finances/TransactionForm';
import type { Profile, BankAccount, RecurringTransaction, BudgetSummaryItem, Category, Transaction, TransactionFormData } from '@/types/finances';

const API_BASE = 'http://localhost:3335/api/v1';

type Range = 'month' | '30' | '90';

interface ForecastItem {
  id: string;
  recurringId: string;
  date: string;
  description: string;
  amount: number;
  type: 'INCOME' | 'EXPENSE';
  categoryId?: string;
  bankAccountId?: string;
}

export default function PlanPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [summary, setSummary] = useState<BudgetSummaryItem[]>([]);
  const [range, setRange] = useState<Range>('month');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState<string | null>(null);
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(null);
  const [showTransactionForm, setShowTransactionForm] = useState(false);
  const [savingTransaction, setSavingTransaction] = useState(false);

  const { startDate, endDate } = useMemo(() => {
    const now = new Date();
    if (range === 'month') {
      // Current month: 1st to last day
      const start = new Date(now.getFullYear(), now.getMonth(), 1);
      const end = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59);
      return { startDate: start, endDate: end };
    }
    // 30 or 90 days from today
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const end = new Date(start);
    end.setDate(end.getDate() + (range === '30' ? 30 : 90));
    return { startDate: start, endDate: end };
  }, [range]);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/profiles`);
        const data = await res.json();
        const list: Profile[] = data.data || [];
        setProfiles(list);
        if (list.length > 0) setSelectedProfileId(list[0].id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  const loadData = useCallback(async () => {
    if (!selectedProfileId) return;
    try {
      setLoading(true);
      setError(null);
      const month = new Date().toISOString().slice(0, 7);
      const fromDate = startDate.toISOString().slice(0, 10);
      const toDate = endDate.toISOString().slice(0, 10);
      const [accRes, recRes, sumRes, catRes, txRes] = await Promise.all([
        fetch(`${API_BASE}/bank-accounts`),
        fetch(`${API_BASE}/recurring-transactions?profileId=${selectedProfileId}`),
        fetch(`${API_BASE}/budgets/summary?profileId=${selectedProfileId}&period=${month}`),
        fetch(`${API_BASE}/categories?profileId=${selectedProfileId}`),
        fetch(`${API_BASE}/transactions?profileId=${selectedProfileId}&from=${fromDate}&to=${toDate}`),
      ]);
      if (!accRes.ok) throw new Error(`accounts ${accRes.status}`);
      if (!recRes.ok) throw new Error(`recurring ${recRes.status}`);
      if (!sumRes.ok) throw new Error(`summary ${sumRes.status}`);
      if (!catRes.ok) throw new Error(`categories ${catRes.status}`);
      if (!txRes.ok) throw new Error(`transactions ${txRes.status}`);
      const accData = await accRes.json();
      const recData = await recRes.json();
      const sumData = await sumRes.json();
      const catData = await catRes.json();
      const txData = await txRes.json();
      setAccounts(accData.data || []);
      setRecurrings(recData.data || []);
      setSummary(sumData.data || []);
      setCategories(catData.data || []);
      setTransactions(txData.data || []);
    } catch (e) {
      console.warn('Erro ao carregar dados do planejamento', e);
      setError('Nao foi possivel carregar dados do planejamento.');
      setAccounts([]);
      setRecurrings([]);
      setSummary([]);
      setCategories([]);
      setTransactions([]);
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId, startDate, endDate]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const forecast = useMemo(() => {
    const start = new Date(startDate);
    start.setHours(0, 0, 0, 0);
    const end = new Date(endDate);
    end.setHours(23, 59, 59, 999);
    const items: ForecastItem[] = [];

    const parseRule = (rule: string) => {
      const map = new Map<string, string>();
      rule.split(';').forEach((kv) => {
        const [k, v] = kv.split('=');
        if (k && v) map.set(k.toUpperCase(), v.toUpperCase());
      });
      return map;
    };

    recurrings.forEach((r) => {
      // Skip paused or cancelled recurring transactions
      if (r.status !== 'ACTIVE') return;

      const rule = parseRule(r.recurrenceRule || '');
      const freq = rule.get('FREQ') || 'MONTHLY';
      const byMonthDay = rule.get('BYMONTHDAY');

      // Parse dates - handle timezone by using local date parts
      const parseLocalDate = (dateStr: string) => {
        const [year, month, day] = dateStr.slice(0, 10).split('-').map(Number);
        return new Date(year, month - 1, day, 0, 0, 0, 0);
      };

      const recStartOn = parseLocalDate(r.startOn);
      const recEndOn = r.endOn ? parseLocalDate(r.endOn) : undefined;
      if (recEndOn) recEndOn.setHours(23, 59, 59, 999);

      // Start from nextOccurrence
      let cur = parseLocalDate(r.nextOccurrence);

      const addDays = (d: Date, n: number) => {
        const c = new Date(d);
        c.setDate(c.getDate() + n);
        return c;
      };
      const addMonths = (d: Date, n: number) => {
        const c = new Date(d);
        c.setMonth(c.getMonth() + n);
        if (byMonthDay) {
          c.setDate(Number(byMonthDay));
        }
        return c;
      };

      let idx = 0;
      while (cur <= end && idx < 100) {
        idx++;
        // Check if within range and recurring period
        if (cur >= start && cur >= recStartOn && (!recEndOn || cur <= recEndOn)) {
          items.push({
            id: `${r.id}-${cur.toISOString().slice(0, 10)}`,
            recurringId: r.id,
            date: cur.toISOString().slice(0, 10),
            description: r.description,
            amount: r.amount,
            type: r.type as 'INCOME' | 'EXPENSE',
            categoryId: r.categoryId,
            bankAccountId: r.bankAccountId,
          });
        }
        // Advance to next occurrence
        if (freq === 'DAILY') {
          cur = addDays(cur, 1);
        } else if (freq === 'WEEKLY') {
          cur = addDays(cur, 7);
        } else {
          cur = addMonths(cur, 1);
        }
        if (recEndOn && cur > recEndOn) break;
      }
    });

    items.sort((a, b) => a.date.localeCompare(b.date));
    const totals = items.reduce(
      (acc, it) => {
        const sign = it.type === 'EXPENSE' ? -1 : 1;
        return {
          ...acc,
          total: acc.total + sign * it.amount,
          out: acc.out + (it.type === 'EXPENSE' ? it.amount : 0),
          inc: acc.inc + (it.type === 'INCOME' ? it.amount : 0),
        };
      },
      { total: 0, out: 0, inc: 0 },
    );

    return { items, totals };
  }, [recurrings, startDate, endDate]);

  const totalBalance = useMemo(
    () => accounts.filter((a) => a.profileId === selectedProfileId).reduce((sum, a) => sum + a.currentBalance, 0),
    [accounts, selectedProfileId]
  );

  const handleConfirm = async (item: ForecastItem) => {
    if (!selectedProfileId) return;
    setConfirming(item.id);
    try {
      // Check if there's an existing PLANNED transaction for this item
      const existingTx = transactions.find(
        (tx) =>
          tx.description === item.description &&
          tx.occurredOn.slice(0, 10) === item.date &&
          tx.status === 'PLANNED'
      );

      if (existingTx) {
        // Update existing transaction status to CONFIRMED
        const res = await fetch(`${API_BASE}/transactions/${existingTx.id}/status`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ status: 'CONFIRMED' }),
        });

        if (!res.ok) {
          const errorText = await res.text();
          throw new Error(errorText || `status ${res.status}`);
        }
      } else {
        // Create new confirmed transaction
        const payload = {
          profileId: selectedProfileId,
          bankAccountId: item.bankAccountId,
          categoryId: item.categoryId,
          type: item.type,
          amount: item.amount,
          currency: 'BRL',
          description: item.description,
          occurredOn: item.date,
          status: 'CONFIRMED',
        };

        const cleanPayload = Object.fromEntries(
          Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
        );

        const res = await fetch(`${API_BASE}/transactions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(cleanPayload),
        });

        if (!res.ok) {
          const errorText = await res.text();
          throw new Error(errorText || `status ${res.status}`);
        }
      }

      await loadData();
    } catch (e) {
      console.warn('Erro ao confirmar', e);
      alert('Erro ao confirmar lancamento.');
    } finally {
      setConfirming(null);
    }
  };

  const getCategoryName = (categoryId?: string) => {
    if (!categoryId) return null;
    const cat = categories.find((c) => c.id === categoryId);
    return cat?.name || null;
  };

  // Set of already confirmed items (description + date) - only CONFIRMED status
  const confirmedSet = useMemo(() => {
    const set = new Set<string>();
    transactions
      .filter((tx) => tx.status === 'CONFIRMED')
      .forEach((tx) => {
        const date = tx.occurredOn.slice(0, 10);
        set.add(`${tx.description}|${date}`);
      });
    return set;
  }, [transactions]);

  const isConfirmed = (item: ForecastItem) => {
    return confirmedSet.has(`${item.description}|${item.date}`);
  };

  // PLANNED transactions (not from recurring)
  const plannedTransactions = useMemo(() => {
    const start = startDate.toISOString().slice(0, 10);
    const end = endDate.toISOString().slice(0, 10);
    return transactions.filter((tx) => {
      const date = tx.occurredOn.slice(0, 10);
      return tx.status === 'PLANNED' && date >= start && date <= end;
    }).sort((a, b) => a.occurredOn.localeCompare(b.occurredOn));
  }, [transactions, startDate, endDate]);

  const handleConfirmPlanned = async (tx: Transaction) => {
    setConfirming(tx.id);
    try {
      const res = await fetch(`${API_BASE}/transactions/${tx.id}/status`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: 'CONFIRMED' }),
      });
      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(errorText || `status ${res.status}`);
      }
      await loadData();
    } catch (e) {
      console.warn('Erro ao confirmar', e);
      alert('Erro ao confirmar lancamento.');
    } finally {
      setConfirming(null);
    }
  };

  const handleEditPlanned = (tx: Transaction) => {
    setEditingTransaction(tx);
    setShowTransactionForm(true);
  };

  const handleTransactionSaved = () => {
    setShowTransactionForm(false);
    setEditingTransaction(null);
    loadData();
  };

  const handleCreateTransaction = async (payload: TransactionFormData) => {
    if (!selectedProfileId) return;
    try {
      setSavingTransaction(true);
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      const response = await fetch(`${API_BASE}/transactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cleanPayload),
      });
      if (!response.ok) throw new Error('Erro ao criar transacao');
      handleTransactionSaved();
    } catch (err) {
      console.error(err);
      alert('Erro ao criar transacao');
    } finally {
      setSavingTransaction(false);
    }
  };

  const handleUpdateTransaction = async (id: string, payload: TransactionFormData) => {
    if (!selectedProfileId) return;
    try {
      setSavingTransaction(true);
      const cleanPayload = Object.fromEntries(
        Object.entries(payload).filter(([, v]) => v !== undefined && v !== null)
      );
      const response = await fetch(`${API_BASE}/transactions/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cleanPayload),
      });
      if (!response.ok) throw new Error('Erro ao atualizar transacao');
      handleTransactionSaved();
    } catch (err) {
      console.error(err);
      alert('Erro ao atualizar transacao');
    } finally {
      setSavingTransaction(false);
    }
  };

  // Combined totals for summary (recurring + planned)
  const combinedTotals = useMemo(() => {
    const plannedIncome = plannedTransactions
      .filter((tx) => tx.type === 'INCOME')
      .reduce((sum, tx) => sum + tx.amount, 0);
    const plannedExpense = plannedTransactions
      .filter((tx) => tx.type === 'EXPENSE')
      .reduce((sum, tx) => sum + tx.amount, 0);
    return {
      inc: forecast.totals.inc + plannedIncome,
      out: forecast.totals.out + plannedExpense,
      total: forecast.totals.total + plannedIncome - plannedExpense,
    };
  }, [forecast.totals, plannedTransactions]);

  const selectedProfile = profiles.find((p) => p.id === selectedProfileId);
  const profileTheme: ProfileTheme = selectedProfile?.type === 'BUSINESS' ? 'business' : 'personal';

  return (
    <AppLayout profileTheme={profileTheme} profileName={selectedProfile?.name}>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Planejamento</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Perfil:</span>
              <div className="flex flex-wrap gap-2">
                {profiles.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => setSelectedProfileId(p.id)}
                    className={`px-3 py-1.5 rounded-xl border ${selectedProfileId === p.id ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}
                  >
                    {p.name}
                  </button>
                ))}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Horizonte:</span>
              <button
                onClick={() => setRange('month')}
                className={`px-3 py-1.5 rounded-xl text-sm border ${range === 'month' ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}
              >
                Mes
              </button>
              <button
                onClick={() => setRange('30')}
                className={`px-3 py-1.5 rounded-xl text-sm border ${range === '30' ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}
              >
                30 dias
              </button>
              <button
                onClick={() => setRange('90')}
                className={`px-3 py-1.5 rounded-xl text-sm border ${range === '90' ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'}`}
              >
                90 dias
              </button>
            </div>
          </div>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              {loading && <p className="text-white/70">Carregando...</p>}
              {error && <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3 mb-3">{error}</div>}
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-white font-semibold">Agenda de Recorrentes</h3>
                <span className="text-white/50 text-sm">{recurrings.length} regras cadastradas</span>
              </div>
              {forecast.items.length === 0 ? (
                <div className="text-white/70 space-y-2">
                  <p>Nenhum item no periodo {range === 'month' ? 'deste mes' : `de ${range} dias`}.</p>
                  {recurrings.length === 0 && (
                    <p className="text-sm">
                      <Link href="/recurring" className="text-emerald-400 hover:underline">
                        Cadastre transacoes recorrentes
                      </Link>{' '}
                      para ver a previsao aqui.
                    </p>
                  )}
                </div>
              ) : (
                <div className="space-y-2">
                  {forecast.items.map((it) => (
                    <div
                      key={it.id}
                      className="flex items-center justify-between border border-white/10 bg-white/5 rounded-xl px-4 py-3"
                    >
                      <div className="flex items-center gap-4">
                        <div className="text-white/60 text-sm w-24">
                          {new Date(it.date + 'T12:00:00').toLocaleDateString('pt-BR')}
                        </div>
                        <div>
                          <p className="text-white font-medium">{it.description}</p>
                          {getCategoryName(it.categoryId) && (
                            <p className="text-white/50 text-xs">{getCategoryName(it.categoryId)}</p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className={`text-sm font-semibold ${it.type === 'EXPENSE' ? 'text-rose-300' : 'text-emerald-300'}`}>
                          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(it.amount * (it.type === 'EXPENSE' ? -1 : 1))}
                        </div>
                        {isConfirmed(it) ? (
                          <span className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-white/10 text-white/50">
                            Confirmado
                          </span>
                        ) : (
                          <button
                            onClick={() => handleConfirm(it)}
                            disabled={confirming === it.id}
                            className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
                              confirming === it.id
                                ? 'bg-white/10 text-white/40'
                                : 'bg-emerald-500/80 hover:bg-emerald-500 text-white'
                            }`}
                            title="Confirmar e criar lancamento"
                          >
                            {confirming === it.id ? '...' : 'Confirmar'}
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Planned Transactions Section */}
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-white font-semibold">Transacoes Planejadas</h3>
                <span className="text-white/50 text-sm">{plannedTransactions.length} lancamentos</span>
              </div>
              {plannedTransactions.length === 0 ? (
                <p className="text-white/70 text-sm">
                  Nenhuma transacao planejada no periodo.
                </p>
              ) : (
                <div className="space-y-2">
                  {plannedTransactions.map((tx) => (
                    <div
                      key={tx.id}
                      className="flex items-center justify-between border border-white/10 bg-white/5 rounded-xl px-4 py-3"
                    >
                      <div className="flex items-center gap-4">
                        <div className="text-white/60 text-sm w-24">
                          {new Date(tx.occurredOn.slice(0, 10) + 'T12:00:00').toLocaleDateString('pt-BR')}
                        </div>
                        <div>
                          <p className="text-white font-medium">{tx.description}</p>
                          {getCategoryName(tx.categoryId) && (
                            <p className="text-white/50 text-xs">{getCategoryName(tx.categoryId)}</p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className={`text-sm font-semibold ${tx.type === 'EXPENSE' ? 'text-rose-300' : 'text-emerald-300'}`}>
                          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(tx.amount * (tx.type === 'EXPENSE' ? -1 : 1))}
                        </div>
                        <button
                          onClick={() => handleEditPlanned(tx)}
                          className="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors bg-white/10 hover:bg-white/20 text-white"
                          title="Editar transacao"
                        >
                          Editar
                        </button>
                        <button
                          onClick={() => handleConfirmPlanned(tx)}
                          disabled={confirming === tx.id}
                          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
                            confirming === tx.id
                              ? 'bg-white/10 text-white/40'
                              : 'bg-emerald-500/80 hover:bg-emerald-500 text-white'
                          }`}
                          title="Confirmar pagamento"
                        >
                          {confirming === tx.id ? '...' : 'Confirmar'}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <div className="space-y-6">
            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-white font-semibold mb-3">Resumo do periodo</h3>
              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between text-white/80">
                  <span>Saldo atual</span>
                  <span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(totalBalance)}</span>
                </div>
                <div className="flex items-center justify-between text-emerald-200">
                  <span>Receitas previstas</span>
                  <span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(combinedTotals.inc)}</span>
                </div>
                <div className="flex items-center justify-between text-rose-200">
                  <span>Despesas previstas</span>
                  <span className="font-semibold">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(combinedTotals.out)}</span>
                </div>
                <div className="border-t border-white/10 pt-2 mt-2">
                  <div className="flex items-center justify-between text-white">
                    <span>Saldo projetado</span>
                    <span className={`font-semibold ${totalBalance + combinedTotals.total >= 0 ? 'text-emerald-300' : 'text-rose-300'}`}>
                      {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(totalBalance + combinedTotals.total)}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
              <h3 className="text-white font-semibold mb-3">Orcamentos do mes</h3>
              {summary.length === 0 ? (
                <p className="text-white/70 text-sm">Nenhuma meta cadastrada.</p>
              ) : (
                <div className="space-y-2 text-sm">
                  {summary.map((s) => {
                    const cat = categories.find((c) => c.id === s.target.categoryId);
                    const name = cat ? cat.name : s.target.categoryId;
                    return (
                      <div key={s.target.id} className="flex items-center justify-between text-white/80">
                        <span>{name}</span>
                        <span className="font-semibold">
                          {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(s.remaining)} restantes
                        </span>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Transaction Form Modal */}
      {showTransactionForm && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-zinc-900 rounded-2xl border border-white/10 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-xl font-bold text-white">
                  {editingTransaction ? 'Editar Transacao' : 'Nova Transacao'}
                </h3>
                <button
                  onClick={() => {
                    setShowTransactionForm(false);
                    setEditingTransaction(null);
                  }}
                  className="text-white/60 hover:text-white text-2xl"
                >
                  &times;
                </button>
              </div>
              <TransactionForm
                isOpen={showTransactionForm}
                onClose={() => {
                  setShowTransactionForm(false);
                  setEditingTransaction(null);
                }}
                onSave={handleCreateTransaction}
                onUpdate={handleUpdateTransaction}
                accounts={accounts}
                categories={categories}
                defaultProfileId={selectedProfileId ?? ''}
                loading={savingTransaction}
                profiles={profiles.map((p) => ({ id: p.id, name: p.name }))}
                editingTransaction={editingTransaction}
              />
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
