'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import type { Profile, BudgetTarget, BudgetSummaryItem, Category } from '@/types/finances';

const API_BASE = 'http://localhost:3335/api/v1';

// Parse date without timezone conversion
const parseLocalDate = (value: string) => {
  const datePart = value.split('T')[0];
  const [year, month, day] = datePart.split('-').map(Number);
  return new Date(year, month - 1, day);
};

// Calculate pace tracking metrics
const calculatePaceMetrics = (period: string, spent: number, budget: number) => {
  const [year, month] = period.split('-').map(Number);
  const firstDay = new Date(year, month - 1, 1);
  const lastDay = new Date(year, month, 0);
  const today = new Date();

  const totalDays = lastDay.getDate();
  const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year
    ? today.getDate()
    : (today > lastDay ? totalDays : 0);

  const daysRemaining = Math.max(0, totalDays - currentDay);
  const percentMonth = totalDays > 0 ? (currentDay / totalDays) * 100 : 0;
  const percentSpent = budget > 0 ? (spent / budget) * 100 : 0;

  // Projection: if we continue at current pace
  const dailyAverage = currentDay > 0 ? spent / currentDay : 0;
  const projection = dailyAverage * totalDays;

  // Safe to spend per day
  const remaining = budget - spent;
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
  };
};

export default function BudgetsPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [period, setPeriod] = useState<string>(() => new Date().toISOString().slice(0, 7));
  const [targets, setTargets] = useState<BudgetTarget[]>([]);
  const [summary, setSummary] = useState<BudgetSummaryItem[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<{ categoryId: string; period: string; amount: string; notes: string; isRecurring: boolean }>({ categoryId: '', period, amount: '', notes: '', isRecurring: true });
  const [editingId, setEditingId] = useState<string | null>(null);

  const selectedProfile = useMemo(
    () => profiles.find((p) => p.id === selectedProfileId) || null,
    [profiles, selectedProfileId],
  );

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

  useEffect(() => {
    if (!selectedProfileId) return;
    (async () => {
      try {
        setLoading(true);
        setError(null);
        const [listRes, sumRes, catRes] = await Promise.all([
          fetch(`${API_BASE}/budgets?profileId=${selectedProfileId}`),
          fetch(`${API_BASE}/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
          fetch(`${API_BASE}/categories?profileId=${selectedProfileId}&type=EXPENSE`),
        ]);
        if (!listRes.ok) throw new Error(`budgets ${listRes.status}`);
        if (!sumRes.ok) throw new Error(`summary ${sumRes.status}`);
        if (!catRes.ok) throw new Error(`categories ${catRes.status}`);
        const listData = await listRes.json();
        const sumData = await sumRes.json();
        const catData = await catRes.json();
        setTargets(listData.data || []);
        setSummary(sumData.data || []);
        setCategories(catData.data || []);
      } catch (e) {
        console.warn('Erro ao carregar orçamentos', e);
        setTargets([]);
        setSummary([]);
        setCategories([]);
        setError('Não foi possível carregar os orçamentos.');
      } finally {
        setLoading(false);
      }
    })();
  }, [selectedProfileId, period]);

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
      const url = editingId ? `${API_BASE}/budgets/${editingId}` : `${API_BASE}/budgets`;
      const method = editingId ? 'PUT' : 'POST';
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Falha ao salvar orçamento');
      }
      // reload
      const [listRes, sumRes] = await Promise.all([
        fetch(`${API_BASE}/budgets?profileId=${selectedProfileId}`),
        fetch(`${API_BASE}/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
      ]);
      const listData = await listRes.json();
      const sumData = await sumRes.json();
      setTargets(listData.data || []);
      setSummary(sumData.data || []);
      resetForm();
    } catch (e) {
      console.warn('Erro ao salvar orçamento', e);
      alert('Não foi possível salvar o orçamento');
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
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <p className="text-white/70 text-sm">Perfil:</p>
              <div className="flex flex-wrap gap-2">
                {profiles.map((profile) => {
                  const isSelected = selectedProfileId === profile.id;
                  return (
                    <button
                      key={profile.id}
                      onClick={() => setSelectedProfileId(profile.id)}
                      className={`px-3 py-1.5 rounded-xl border transition-colors ${
                        isSelected ? 'bg-white/20 text-white border-white/40' : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'
                      }`}
                    >
                      {profile.name}
                    </button>
                  );
                })}
              </div>
            </div>
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
                  const pace = calculatePaceMetrics(period, s.spent, s.target.amount);
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
                        <div className="h-2 bg-white/10 rounded-full overflow-hidden">
                          <div
                            className={`h-full ${progressColor} transition-all`}
                            style={{ width: `${Math.min(100, pace.percentSpent)}%` }}
                          />
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
