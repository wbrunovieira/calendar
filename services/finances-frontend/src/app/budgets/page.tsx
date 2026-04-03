'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import { useToast } from '@/components/ui/Toast';
import { api } from '@/lib/api';
import type { BudgetTarget, BudgetSummaryItem, Category, RecurringTransaction, Transaction } from '@/types/finances';
import { parseLocalDate } from '@/utils/format';

// Calculate pace tracking metrics
const calculatePaceMetrics = (period: string, spent: number, budget: number, pendingRecurring: number = 0) => {
  const [year, month] = period.split('-').map(Number);
  const lastDay = new Date(year, month, 0);
  const today = new Date();

  const totalDays = lastDay.getDate();
  const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year
    ? today.getDate()
    : (today > lastDay ? totalDays : 0);

  const daysRemaining = Math.max(0, totalDays - currentDay);
  const percentMonth = totalDays > 0 ? (currentDay / totalDays) * 100 : 0;
  const percentSpent = budget > 0 ? (spent / budget) * 100 : 0;

  // Projection: spent + pending recurring expenses that will still happen this month
  const projection = spent + pendingRecurring;

  // Safe to spend per day (excluding pending recurring - those are fixed)
  const remaining = budget - spent - pendingRecurring;
  const safePerDay = daysRemaining > 0 ? remaining / daysRemaining : remaining;

  // Status: on track if spent% is within 10% of month%
  let status: 'on_track' | 'ahead' | 'behind' = 'on_track';
  if (percentSpent > percentMonth + 10) {
    status = 'behind'; // spending too fast
  } else if (percentSpent < percentMonth - 10) {
    status = 'ahead'; // underspending (good)
  }

  return {
    percentMonth: Math.round(percentMonth),
    percentSpent: Math.round(percentSpent),
    projection,
    safePerDay,
    daysRemaining,
    status,
    pendingRecurring,
  };
};

export default function BudgetsPage() {
  // Use shared profile context
  const { selectedProfileId, selectedProfile } = useProfile();
  const { toast } = useToast();

  const [period, setPeriod] = useState<string>(() => new Date().toISOString().slice(0, 7));
  const [targets, setTargets] = useState<BudgetTarget[]>([]);
  const [summary, setSummary] = useState<BudgetSummaryItem[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedCards, setExpandedCards] = useState<Set<string>>(new Set());
  const [form, setForm] = useState<{ categoryId: string; period: string; amount: string; notes: string; isRecurring: boolean }>({ categoryId: '', period, amount: '', notes: '', isRecurring: true });
  const [editingId, setEditingId] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedProfileId) return;
    (async () => {
      try {
        setLoading(true);
        setError(null);
        // Calculate date range for transactions
        const [year, month] = period.split('-').map(Number);
        const fromDate = `${period}-01`;
        const lastDay = new Date(year, month, 0).getDate();
        const toDate = `${period}-${String(lastDay).padStart(2, '0')}`;

        const [listData, sumData, catData] = await Promise.all([
          api.get<{ data: BudgetTarget[] }>(`/budgets?profileId=${selectedProfileId}`),
          api.get<{ data: BudgetSummaryItem[] }>(`/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
          api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}&type=EXPENSE`),
        ]);

        let recData: { data: RecurringTransaction[] } = { data: [] };
        let txData: { data: Transaction[] } = { data: [] };
        try {
          recData = await api.get<{ data: RecurringTransaction[] }>(`/recurring-transactions?profileId=${selectedProfileId}`);
        } catch { /* optional */ }
        try {
          txData = await api.get<{ data: Transaction[] }>(`/transactions?profileId=${selectedProfileId}&from=${fromDate}&to=${toDate}&type=EXPENSE`);
        } catch { /* optional */ }

        setTargets(listData.data || []);
        setSummary(sumData.data || []);
        setCategories(catData.data || []);
        setRecurrings(recData.data || []);
        setTransactions(txData.data || []);
      } catch (e) {
        console.warn('Erro ao carregar orçamentos', e);
        setTargets([]);
        setSummary([]);
        setCategories([]);
        setRecurrings([]);
        setTransactions([]);
        setError('Não foi possível carregar os orçamentos.');
      } finally {
        setLoading(false);
      }
    })();
  }, [selectedProfileId, period]);

  // Calculate pending recurring expenses per budget category
  const pendingRecurringByCategory = useMemo(() => {
    const [year, month] = period.split('-').map(Number);
    const today = new Date();
    const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year
      ? today.getDate()
      : 0;

    // Helper to check if a category belongs to a budget category (direct or via parent)
    const getCategoryChain = (categoryId: string): string[] => {
      const chain: string[] = [categoryId];
      let current = categories.find((c) => c.id === categoryId);
      while (current?.parentId) {
        chain.push(current.parentId);
        current = categories.find((c) => c.id === current?.parentId);
      }
      return chain;
    };

    // Map each budget category to its pending recurring amount
    const pendingMap: Record<string, number> = {};

    summary.forEach((s) => {
      const budgetCategoryId = s.target.categoryId;
      let pendingAmount = 0;

      recurrings.forEach((r) => {
        if (r.status !== 'ACTIVE' || r.type !== 'EXPENSE') return;
        if (!r.categoryId) return;

        // Check if this recurring belongs to this budget category
        const chain = getCategoryChain(r.categoryId);
        if (!chain.includes(budgetCategoryId)) return;

        // Parse recurrence rule to get the day
        const rule = new Map<string, string>();
        (r.recurrenceRule || '').split(';').forEach((kv) => {
          const [k, v] = kv.split('=');
          if (k && v) rule.set(k.toUpperCase(), v.toUpperCase());
        });

        const byMonthDay = rule.get('BYMONTHDAY');
        if (!byMonthDay) return;

        const dayOfMonth = parseInt(byMonthDay, 10);
        // If this day hasn't passed yet, it's pending
        if (dayOfMonth > currentDay) {
          pendingAmount += r.amount;
        }
      });

      pendingMap[budgetCategoryId] = pendingAmount;
    });

    return pendingMap;
  }, [period, categories, recurrings, summary]);

  // Get transactions and pending recurrings for a specific budget category
  const getTransactionsForCategory = (budgetCategoryId: string) => {
    const [year, month] = period.split('-').map(Number);
    const today = new Date();
    const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year
      ? today.getDate()
      : 0;

    // Helper to check if a category belongs to this budget category
    const getCategoryChain = (categoryId: string): string[] => {
      const chain: string[] = [categoryId];
      let current = categories.find((c) => c.id === categoryId);
      while (current?.parentId) {
        chain.push(current.parentId);
        current = categories.find((c) => c.id === current?.parentId);
      }
      return chain;
    };

    // Get completed transactions for this category
    const completedTx = transactions.filter((tx) => {
      if (!tx.categoryId || tx.status !== 'CONFIRMED') return false;
      const chain = getCategoryChain(tx.categoryId);
      return chain.includes(budgetCategoryId);
    });

    // Get pending recurrings for this category
    const pendingRecurrings: { description: string; amount: number; day: number }[] = [];
    recurrings.forEach((r) => {
      if (r.status !== 'ACTIVE' || r.type !== 'EXPENSE' || !r.categoryId) return;
      const chain = getCategoryChain(r.categoryId);
      if (!chain.includes(budgetCategoryId)) return;

      const rule = new Map<string, string>();
      (r.recurrenceRule || '').split(';').forEach((kv) => {
        const [k, v] = kv.split('=');
        if (k && v) rule.set(k.toUpperCase(), v.toUpperCase());
      });

      const byMonthDay = rule.get('BYMONTHDAY');
      if (!byMonthDay) return;

      const dayOfMonth = parseInt(byMonthDay, 10);
      if (dayOfMonth > currentDay) {
        pendingRecurrings.push({ description: r.description, amount: r.amount, day: dayOfMonth });
      }
    });

    return { completedTx, pendingRecurrings };
  };

  const toggleCardExpanded = (cardId: string) => {
    setExpandedCards((prev) => {
      const next = new Set(prev);
      if (next.has(cardId)) {
        next.delete(cardId);
      } else {
        next.add(cardId);
      }
      return next;
    });
  };

  const resetForm = () => {
    setForm({ categoryId: '', period, amount: '', notes: '', isRecurring: true });
    setEditingId(null);
  };

  const submitBudget = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProfileId) return;
    const payload = {
      profileId: selectedProfileId,
      categoryId: form.categoryId,
      period: form.period,
      amount: Number(form.amount),
      notes: form.notes || undefined,
      isRecurring: form.isRecurring,
    };
    try {
      if (editingId) {
        await api.put(`/budgets/${editingId}`, payload);
      } else {
        await api.post('/budgets', payload);
      }
      // reload
      const [listData, sumData] = await Promise.all([
        api.get<{ data: BudgetTarget[] }>(`/budgets?profileId=${selectedProfileId}`),
        api.get<{ data: BudgetSummaryItem[] }>(`/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
      ]);
      setTargets(listData.data || []);
      setSummary(sumData.data || []);
      resetForm();
    } catch (e) {
      console.warn('Erro ao salvar orçamento', e);
      toast('Nao foi possivel salvar o orcamento', 'error');
    }
  };


  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Orçamentos por Categoria</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center justify-end gap-3">
            <div className="flex items-center gap-2">
              <label className="text-white/70 text-sm">Período:</label>
              <input
                type="month"
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
                className="bg-white/10 border border-white/20 text-white rounded-lg px-3 py-1.5 text-sm"
              />
            </div>
          </div>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-6 space-y-6">
          {!selectedProfile && (
            <p className="text-white/70">Selecione um perfil para visualizar orçamentos.</p>
          )}
          {selectedProfile && (
            <>
              {loading && <p className="text-white/70">Carregando...</p>}
              {error && (
                <div className="bg-rose-500/20 border border-rose-400/40 text-rose-100 rounded-xl p-3 mb-3">
                  {error}
                </div>
              )}

              <form onSubmit={submitBudget} className="bg-white/5 border border-white/10 rounded-xl p-4 grid gap-3 md:grid-cols-5">
                <div className="md:col-span-2">
                  <label className="block text-white/70 text-sm mb-1">Categoria</label>
                  <select
                    value={form.categoryId}
                    onChange={(e) => setForm((f) => ({ ...f, categoryId: e.target.value }))}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  >
                    <option value="">Selecione</option>
                    {categories.map((c) => (
                      <option key={c.id} value={c.id} className="bg-slate-900">{c.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-white/70 text-sm mb-1">Período</label>
                  <input
                    type="month"
                    value={form.period}
                    onChange={(e) => setForm((f) => ({ ...f, period: e.target.value }))}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  />
                </div>
                <div>
                  <label className="block text-white/70 text-sm mb-1">Valor</label>
                  <input
                    type="number"
                    step="0.01"
                    value={form.amount}
                    onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))}
                    placeholder="0,00"
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  />
                </div>
                <div className="md:col-span-5 flex gap-2 items-center">
                  <input
                    type="text"
                    value={form.notes}
                    onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))}
                    placeholder="Notas (opcional)"
                    className="flex-1 px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white"
                  />
                  <label className="flex items-center gap-2 text-white/80 text-sm whitespace-nowrap">
                    <input
                      type="checkbox"
                      checked={form.isRecurring}
                      onChange={(e) => setForm((f) => ({ ...f, isRecurring: e.target.checked }))}
                      className="w-4 h-4 rounded bg-white/10 border-white/20"
                    />
                    Recorrente
                  </label>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg font-semibold border bg-emerald-500/80 hover:bg-emerald-500 text-white border-emerald-400/40"
                  >
                    {editingId ? 'Salvar alterações' : 'Adicionar meta'}
                  </button>
                  {editingId && (
                    <button type="button" onClick={resetForm} className="px-4 py-2 rounded-lg font-semibold border bg-white/10 hover:bg-white/20 text-white border-white/20">Cancelar</button>
                  )}
                </div>
              </form>

              <div className="space-y-4">
                {summary.map((s) => {
                  const cat = categories.find((c) => c.id === s.target.categoryId);
                  const catName = cat ? cat.name : s.target.categoryId;
                  const pendingRec = pendingRecurringByCategory[s.target.categoryId] || 0;
                  const pace = calculatePaceMetrics(period, s.spent, s.target.amount, pendingRec);
                  const fmt = (v: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v);

                  const statusColor = pace.status === 'ahead' ? 'text-emerald-400' : pace.status === 'behind' ? 'text-rose-400' : 'text-amber-400';
                  const statusLabel = pace.status === 'ahead' ? 'Abaixo do ritmo' : pace.status === 'behind' ? 'Acima do ritmo' : 'No ritmo';
                  const progressColor = pace.status === 'ahead' ? 'bg-emerald-500' : pace.status === 'behind' ? 'bg-rose-500' : 'bg-amber-500';

                  return (
                    <div key={s.target.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <h4 className="text-white font-semibold">{catName}</h4>
                          <span className={`text-xs px-2 py-0.5 rounded-full ${statusColor} bg-white/10`}>
                            {statusLabel}
                          </span>
                          {s.target.isRecurring && (
                            <span className="text-xs px-2 py-0.5 rounded-full text-blue-400 bg-blue-500/20">
                              Recorrente
                            </span>
                          )}
                        </div>
                        <button
                          onClick={() => {
                            setEditingId(s.target.id);
                            setForm({
                              categoryId: s.target.categoryId,
                              period: s.target.periodStart.split('T')[0].slice(0, 7),
                              amount: String(s.target.amount),
                              notes: s.target.notes || '',
                              isRecurring: s.target.isRecurring,
                            });
                          }}
                          className="px-3 py-1.5 rounded-lg text-xs border border-white/20 text-white/80 hover:bg-white/10"
                        >
                          Editar
                        </button>
                      </div>

                      {/* Progress bar */}
                      <div className="mb-3">
                        <div className="flex justify-between text-xs text-white/60 mb-1">
                          <span>{fmt(s.spent)} de {fmt(s.target.amount)}</span>
                          <span>{pace.percentSpent}% gasto</span>
                        </div>
                        <div className="h-2 bg-white/10 rounded-full overflow-hidden flex">
                          {/* Spent portion */}
                          <div
                            className={`h-full ${progressColor} transition-all`}
                            style={{ width: `${Math.min(100, pace.percentSpent)}%` }}
                          />
                          {/* Pending recurring portion (blue) */}
                          {pendingRec > 0 && (
                            <div
                              className="h-full bg-blue-500/70 transition-all"
                              style={{ width: `${Math.min(100 - pace.percentSpent, (pendingRec / s.target.amount) * 100)}%` }}
                            />
                          )}
                        </div>
                        <div className="flex justify-between text-xs text-white/40 mt-1">
                          <span>{pace.percentMonth}% do mês</span>
                          <span>{pace.daysRemaining} dias restantes</span>
                        </div>
                      </div>

                      {/* Metrics grid */}
                      <div className="grid grid-cols-3 gap-4 text-center">
                        <div className="bg-white/5 rounded-lg p-2">
                          <p className="text-xs text-white/50">Restante</p>
                          <p className={`text-sm font-semibold ${s.remaining >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                            {fmt(s.remaining)}
                          </p>
                        </div>
                        <div className="bg-white/5 rounded-lg p-2">
                          <p className="text-xs text-white/50">Projeção</p>
                          <p className={`text-sm font-semibold ${pace.projection <= s.target.amount ? 'text-emerald-400' : 'text-rose-400'}`}>
                            {fmt(pace.projection)}
                          </p>
                        </div>
                        <div className="bg-white/5 rounded-lg p-2">
                          <p className="text-xs text-white/50">Seguro/dia</p>
                          <p className={`text-sm font-semibold ${pace.safePerDay >= 0 ? 'text-white' : 'text-rose-400'}`}>
                            {fmt(pace.safePerDay)}
                          </p>
                        </div>
                      </div>

                      {/* Expandable transactions section */}
                      {(() => {
                        const { completedTx, pendingRecurrings } = getTransactionsForCategory(s.target.categoryId);
                        const isExpanded = expandedCards.has(s.target.id);
                        const hasItems = completedTx.length > 0 || pendingRecurrings.length > 0;

                        if (!hasItems) return null;

                        return (
                          <div className="mt-3 pt-3 border-t border-white/10">
                            <button
                              onClick={() => toggleCardExpanded(s.target.id)}
                              className="flex items-center gap-2 text-xs text-white/60 hover:text-white/80 transition-colors w-full"
                            >
                              <svg
                                className={`w-3 h-3 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                              >
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                              </svg>
                              <span>Ver detalhes</span>
                              <span className="text-white/40">
                                ({completedTx.length} concluido{completedTx.length !== 1 ? 's' : ''}, {pendingRecurrings.length} pendente{pendingRecurrings.length !== 1 ? 's' : ''})
                              </span>
                            </button>

                            <div
                              className={`overflow-hidden transition-all duration-300 ease-in-out ${
                                isExpanded ? 'max-h-[500px] opacity-100 mt-3' : 'max-h-0 opacity-0'
                              }`}
                            >
                              {/* Completed transactions */}
                              {completedTx.length > 0 && (
                                <div className="mb-3">
                                  <p className="text-xs text-emerald-400 mb-1">Concluidos</p>
                                  <div className="space-y-1">
                                    {completedTx.map((tx) => (
                                      <div key={tx.id} className="flex justify-between text-xs">
                                        <span className="text-white/70 truncate flex-1">
                                          {tx.description}
                                          <span className="text-white/40 ml-1">
                                            ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })})
                                          </span>
                                        </span>
                                        <span className="text-emerald-400 ml-2">{fmt(tx.amount)}</span>
                                      </div>
                                    ))}
                                    <div className="flex justify-between text-xs pt-1 border-t border-white/10 mt-1">
                                      <span className="text-white/50 font-medium">Total concluidos</span>
                                      <span className="text-emerald-400 font-medium">{fmt(completedTx.reduce((sum, tx) => sum + tx.amount, 0))}</span>
                                    </div>
                                  </div>
                                </div>
                              )}

                              {/* Pending recurrings */}
                              {pendingRecurrings.length > 0 && (
                                <div className="mb-3">
                                  <p className="text-xs text-blue-400 mb-1">Fixas pendentes</p>
                                  <div className="space-y-1">
                                    {pendingRecurrings.map((r, idx) => (
                                      <div key={idx} className="flex justify-between text-xs">
                                        <span className="text-white/70 truncate flex-1">
                                          {r.description}
                                          <span className="text-white/40 ml-1">(dia {r.day})</span>
                                        </span>
                                        <span className="text-blue-400 ml-2">{fmt(r.amount)}</span>
                                      </div>
                                    ))}
                                    <div className="flex justify-between text-xs pt-1 border-t border-white/10 mt-1">
                                      <span className="text-white/50 font-medium">Total pendentes</span>
                                      <span className="text-blue-400 font-medium">{fmt(pendingRecurrings.reduce((sum, r) => sum + r.amount, 0))}</span>
                                    </div>
                                  </div>
                                </div>
                              )}

                              {/* Grand total */}
                              {(completedTx.length > 0 || pendingRecurrings.length > 0) && (
                                <div className="flex justify-between text-xs pt-2 border-t border-white/20 mt-2">
                                  <span className="text-white font-semibold">Total geral</span>
                                  <span className="text-white font-semibold">
                                    {fmt(completedTx.reduce((sum, tx) => sum + tx.amount, 0) + pendingRecurrings.reduce((sum, r) => sum + r.amount, 0))}
                                  </span>
                                </div>
                              )}
                            </div>
                          </div>
                        );
                      })()}
                    </div>
                  );
                })}
                {summary.length === 0 && !loading && (
                  <p className="text-white/50 text-center py-8">Nenhum orçamento cadastrado para este período.</p>
                )}
              </div>

              <div className="border-t border-white/10 pt-4">
                <h3 className="text-white font-semibold mb-2">Metas cadastradas</h3>
                <ul className="text-white/80 text-sm list-disc ml-5">
                  {targets.map((t) => {
                    const cat = categories.find((c) => c.id === t.categoryId);
                    const catName = cat ? cat.name : t.categoryId;
                    return (
                    <li key={t.id}>
                      {parseLocalDate(t.periodStart).toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })} — {catName} — {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(t.amount)}
                    </li>
                    );
                  })}
                </ul>
              </div>
            </>
          )}
        </div>
      </div>
    </AppLayout>
  );
}
