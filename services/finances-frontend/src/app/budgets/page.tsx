'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout, { useProfile } from '@/components/layout/AppLayout';
import { useToast } from '@/components/ui/Toast';
import { api } from '@/lib/api';
import BudgetCard from '@/components/finances/BudgetCard';
import type { BudgetTarget, BudgetSummaryItem, Category, RecurringTransaction, Transaction } from '@/types/finances';
import { parseLocalDate } from '@/utils/format';

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

  const projection = spent + pendingRecurring;
  const remaining = budget - spent - pendingRecurring;
  const safePerDay = daysRemaining > 0 ? remaining / daysRemaining : remaining;

  let status: 'on_track' | 'ahead' | 'behind' = 'on_track';
  if (percentSpent > percentMonth + 10) {
    status = 'behind';
  } else if (percentSpent < percentMonth - 10) {
    status = 'ahead';
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

  const pendingRecurringByCategory = useMemo(() => {
    const [year, month] = period.split('-').map(Number);
    const today = new Date();
    const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year
      ? today.getDate()
      : 0;

    const getCategoryChain = (categoryId: string): string[] => {
      const chain: string[] = [categoryId];
      let current = categories.find((c) => c.id === categoryId);
      while (current?.parentId) {
        chain.push(current.parentId);
        current = categories.find((c) => c.id === current?.parentId);
      }
      return chain;
    };

    const pendingMap: Record<string, number> = {};

    summary.forEach((s) => {
      const budgetCategoryId = s.target.categoryId;
      let pendingAmount = 0;

      recurrings.forEach((r) => {
        if (r.status !== 'ACTIVE' || r.type !== 'EXPENSE') return;
        if (!r.categoryId) return;

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
          pendingAmount += r.amount;
        }
      });

      pendingMap[budgetCategoryId] = pendingAmount;
    });

    return pendingMap;
  }, [period, categories, recurrings, summary]);

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
                  const pendingRec = pendingRecurringByCategory[s.target.categoryId] || 0;
                  const pace = calculatePaceMetrics(period, s.spent, s.target.amount, pendingRec);

                  return (
                    <BudgetCard
                      key={s.target.id}
                      summary={s}
                      categories={categories}
                      recurrings={recurrings}
                      transactions={transactions}
                      period={period}
                      pendingRecurring={pendingRec}
                      pace={pace}
                      isExpanded={expandedCards.has(s.target.id)}
                      onToggleExpand={() => toggleCardExpanded(s.target.id)}
                      onEdit={() => {
                        setEditingId(s.target.id);
                        setForm({
                          categoryId: s.target.categoryId,
                          period: s.target.periodStart.split('T')[0].slice(0, 7),
                          amount: String(s.target.amount),
                          notes: s.target.notes || '',
                          isRecurring: s.target.isRecurring,
                        });
                      }}
                    />
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
