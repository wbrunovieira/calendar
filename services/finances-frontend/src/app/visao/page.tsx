'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import type { Profile, RecurringTransaction, Transaction, Category, BankAccount, BudgetSummaryItem } from '@/types/finances';

const API_BASE = 'http://localhost:3335/api/v1';

export default function VisaoMensalPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [period, setPeriod] = useState<string>(() => new Date().toISOString().slice(0, 7));
  const [categories, setCategories] = useState<Category[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [budgetSummary, setBudgetSummary] = useState<BudgetSummaryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedSection, setExpandedSection] = useState<string | null>(null);

  const selectedProfile = useMemo(
    () => profiles.find((p) => p.id === selectedProfileId) || null,
    [profiles, selectedProfileId],
  );

  const selectedMonth = useMemo(() => {
    const [year, month] = period.split('-').map(Number);
    return {
      year,
      month: month - 1,
      start: new Date(year, month - 1, 1),
      end: new Date(year, month, 0, 23, 59, 59),
      period,
    };
  }, [period]);

  // Load profiles on mount
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/profiles`);
        const data = await res.json();
        const list: Profile[] = data.data || [];
        setProfiles(list);
        // Default to "Bruno Pessoal" or first profile
        const defaultProfile = list.find((p) => p.name === 'Bruno Pessoal') || list[0];
        if (defaultProfile) setSelectedProfileId(defaultProfile.id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  // Load data when profile or period changes
  useEffect(() => {
    if (!selectedProfileId) return;

    const loadData = async () => {
      try {
        setLoading(true);
        const fromDate = selectedMonth.start.toISOString().slice(0, 10);
        const toDate = selectedMonth.end.toISOString().slice(0, 10);

        const [catRes, accRes, recRes, txRes, budgetRes] = await Promise.all([
          fetch(`${API_BASE}/categories?profileId=${selectedProfileId}`),
          fetch(`${API_BASE}/bank-accounts?profileId=${selectedProfileId}`),
          fetch(`${API_BASE}/recurring-transactions?profileId=${selectedProfileId}`),
          fetch(`${API_BASE}/transactions?profileId=${selectedProfileId}&from=${fromDate}&to=${toDate}`),
          fetch(`${API_BASE}/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
        ]);

        if (catRes.ok) {
          const catData = await catRes.json();
          setCategories(catData.data || []);
        }
        if (accRes.ok) {
          const accData = await accRes.json();
          setAccounts(accData.data || []);
        }
        if (recRes.ok) {
          const recData = await recRes.json();
          setRecurrings(recData.data || []);
        }
        if (txRes.ok) {
          const txData = await txRes.json();
          setTransactions(txData.data || []);
        }
        if (budgetRes.ok) {
          const budgetData = await budgetRes.json();
          setBudgetSummary(budgetData.data || []);
        }
      } catch (e) {
        console.warn('Erro ao carregar dados do overview', e);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [selectedProfileId, period, selectedMonth]);

  // Get all category IDs that are covered by budgets
  const budgetCategoryIds = useMemo(() => {
    const ids = new Set<string>();
    budgetSummary.forEach((b) => {
      ids.add(b.target.categoryId);
    });
    return ids;
  }, [budgetSummary]);

  // Check if a category is covered by a budget (either directly or via parent)
  const isCategoryInBudget = (categoryId?: string): boolean => {
    if (!categoryId) return false;

    // Direct match
    if (budgetCategoryIds.has(categoryId)) return true;

    // Check if parent category has budget
    const cat = categories.find((c) => c.id === categoryId);
    if (cat?.parentId) {
      if (budgetCategoryIds.has(cat.parentId)) return true;
      // Check grandparent
      const parent = categories.find((c) => c.id === cat.parentId);
      if (parent?.parentId && budgetCategoryIds.has(parent.parentId)) return true;
    }

    return false;
  };

  // Calculate recurring forecast for the month
  const recurringForecast = useMemo(() => {
    const items: { id: string; description: string; amount: number; type: string; categoryId?: string; inBudget: boolean }[] = [];

    recurrings.forEach((r) => {
      if (r.status !== 'ACTIVE') return;

      const rule = new Map<string, string>();
      (r.recurrenceRule || '').split(';').forEach((kv) => {
        const [k, v] = kv.split('=');
        if (k && v) rule.set(k.toUpperCase(), v.toUpperCase());
      });

      const byMonthDay = rule.get('BYMONTHDAY');
      if (!byMonthDay) return;

      const inBudget = isCategoryInBudget(r.categoryId);

      items.push({
        id: r.id,
        description: r.description,
        amount: r.amount,
        type: r.type,
        categoryId: r.categoryId,
        inBudget,
      });
    });

    return items;
  }, [recurrings, budgetCategoryIds, categories]);

  // Separate recurring into those covered by budget and those not
  const { recurringOutsideBudget } = useMemo(() => {
    return {
      recurringInBudget: recurringForecast.filter((r) => r.inBudget),
      recurringOutsideBudget: recurringForecast.filter((r) => !r.inBudget),
    };
  }, [recurringForecast]);

  // Separate transactions
  const { confirmedTx, pendingTx } = useMemo(() => {
    return {
      confirmedTx: transactions.filter((tx) => tx.status === 'CONFIRMED'),
      pendingTx: transactions.filter((tx) => tx.status === 'PLANNED'),
    };
  }, [transactions]);

  // Calculate totals
  const totals = useMemo(() => {
    // For total calculation: use the higher of budget or spent (if exceeded)
    const budgetTotal = budgetSummary.reduce((sum, b) => sum + Math.max(b.target.amount, b.spent), 0);
    const budgetSpent = budgetSummary.reduce((sum, b) => sum + b.spent, 0);
    const budgetRemaining = budgetSummary.reduce((sum, b) => sum + b.remaining, 0);

    const recurringExpense = recurringOutsideBudget
      .filter((r) => r.type === 'EXPENSE')
      .reduce((sum, r) => sum + r.amount, 0);
    const recurringIncome = recurringOutsideBudget
      .filter((r) => r.type === 'INCOME')
      .reduce((sum, r) => sum + r.amount, 0);

    const confirmedExpense = confirmedTx
      .filter((tx) => tx.type === 'EXPENSE')
      .reduce((sum, tx) => sum + tx.amount, 0);
    const confirmedIncome = confirmedTx
      .filter((tx) => tx.type === 'INCOME')
      .reduce((sum, tx) => sum + tx.amount, 0);

    const pendingExpense = pendingTx
      .filter((tx) => tx.type === 'EXPENSE')
      .reduce((sum, tx) => sum + tx.amount, 0);
    const pendingIncome = pendingTx
      .filter((tx) => tx.type === 'INCOME')
      .reduce((sum, tx) => sum + tx.amount, 0);

    return {
      budgetTotal,
      budgetSpent,
      budgetRemaining,
      recurringExpense,
      recurringIncome,
      confirmedExpense,
      confirmedIncome,
      pendingExpense,
      pendingIncome,
      totalPlannedExpense: budgetTotal + recurringExpense,
      totalPlannedIncome: recurringIncome,
    };
  }, [budgetSummary, recurringOutsideBudget, confirmedTx, pendingTx]);

  const fmt = (v: number) =>
    new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v);

  const getCategoryName = (categoryId?: string) => {
    if (!categoryId) return 'Sem categoria';
    const cat = categories.find((c) => c.id === categoryId);
    return cat?.name || categoryId;
  };

  const monthName = selectedMonth.start.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' });

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white capitalize">Visao de {monthName}</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        {/* Profile and Period Selector */}
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
              <label className="text-white/70 text-sm">Mes:</label>
              <input
                type="month"
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
                className="bg-white/10 border border-white/20 text-white rounded-lg px-3 py-1.5 text-sm"
              />
            </div>
          </div>
        </div>

        {/* Main Content */}
        {loading ? (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
            <p className="text-white/50">Carregando visao do mes...</p>
          </div>
        ) : (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-6 space-y-4">
            {/* Budget Summary - Main Focus */}
            {budgetSummary.length > 0 && (
              <div className="space-y-3">
                <h4 className="text-sm font-medium text-white/70">Orcamentos</h4>
                <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                  {budgetSummary.map((budget) => {
                    const cat = categories.find((c) => c.id === budget.target.categoryId);
                    const catName = cat?.name || 'Categoria';
                    const percentSpent = budget.target.amount > 0 ? (budget.spent / budget.target.amount) * 100 : 0;
                    const isOverBudget = budget.spent > budget.target.amount;

                    return (
                      <div key={budget.target.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-white font-medium">{catName}</span>
                          <span className="text-white/50 text-sm">
                            {budget.target.isRecurring && <span className="text-blue-400">recorrente</span>}
                          </span>
                        </div>

                        <div className="grid grid-cols-3 gap-2 text-center mb-3">
                          <div>
                            <p className="text-xs text-white/50">Orcamento</p>
                            <p className="text-sm font-semibold text-white">{fmt(budget.target.amount)}</p>
                          </div>
                          <div>
                            <p className="text-xs text-white/50">Gasto</p>
                            <p className={`text-sm font-semibold ${isOverBudget ? 'text-rose-400' : 'text-emerald-400'}`}>
                              {fmt(budget.spent)}
                            </p>
                          </div>
                          <div>
                            <p className="text-xs text-white/50">Restante</p>
                            <p className={`text-sm font-semibold ${budget.remaining >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                              {fmt(budget.remaining)}
                            </p>
                          </div>
                        </div>

                        {/* Progress bar */}
                        <div className="h-2 bg-white/10 rounded-full overflow-hidden">
                          <div
                            className={`h-full transition-all ${isOverBudget ? 'bg-rose-500' : 'bg-emerald-500'}`}
                            style={{ width: `${Math.min(100, percentSpent)}%` }}
                          />
                        </div>
                        <p className="text-xs text-white/40 mt-1 text-right">{Math.round(percentSpent)}% utilizado</p>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {budgetSummary.length === 0 && (
              <div className="text-center py-8">
                <p className="text-white/50">Nenhum orcamento cadastrado para este mes.</p>
                <Link href="/budgets" className="text-blue-400 hover:text-blue-300 text-sm underline mt-2 inline-block">
                  Criar orcamentos
                </Link>
              </div>
            )}

            {/* Fixed Recurring Expenses (outside budget) */}
            {recurringOutsideBudget.filter(r => r.type === 'EXPENSE').length > 0 && (
              <div
                className="bg-white/5 border border-white/10 rounded-xl p-3 cursor-pointer hover:bg-white/10 transition-colors"
                onClick={() => setExpandedSection(expandedSection === 'recurring' ? null : 'recurring')}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-blue-400">📅</span>
                    <span className="text-white/90">Despesas fixas (fora do orcamento)</span>
                    <span className="text-xs text-white/50">({recurringOutsideBudget.filter(r => r.type === 'EXPENSE').length})</span>
                  </div>
                  <span className="text-rose-400 text-sm font-medium">{fmt(totals.recurringExpense)}</span>
                </div>
                {expandedSection === 'recurring' && (
                  <div className="mt-3 space-y-1 border-t border-white/10 pt-3">
                    {recurringOutsideBudget.filter(r => r.type === 'EXPENSE').map((item) => (
                      <div key={item.id} className="flex justify-between text-sm">
                        <span className="text-white/70">
                          {item.description}
                          <span className="text-white/40 ml-2 text-xs">({getCategoryName(item.categoryId)})</span>
                        </span>
                        <span className="text-rose-400">{fmt(item.amount)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Receitas Recorrentes (outside budget) */}
            {totals.recurringIncome > 0 && (
              <div
                className="bg-white/5 border border-white/10 rounded-xl p-3 cursor-pointer hover:bg-white/10 transition-colors"
                onClick={() => setExpandedSection(expandedSection === 'income' ? null : 'income')}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-emerald-400">💰</span>
                    <span className="text-white/90">Receitas previstas</span>
                    <span className="text-xs text-white/50">({recurringOutsideBudget.filter(r => r.type === 'INCOME').length})</span>
                  </div>
                  <span className="text-emerald-400 text-sm font-medium">{fmt(totals.recurringIncome)}</span>
                </div>
                {expandedSection === 'income' && (
                  <div className="mt-3 space-y-1 border-t border-white/10 pt-3">
                    {recurringOutsideBudget.filter(r => r.type === 'INCOME').map((item) => (
                      <div key={item.id} className="flex justify-between text-sm">
                        <span className="text-white/70">{item.description}</span>
                        <span className="text-emerald-400">{fmt(item.amount)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Pending Transactions */}
            {pendingTx.length > 0 && (
              <div
                className="bg-white/5 border border-white/10 rounded-xl p-3 cursor-pointer hover:bg-white/10 transition-colors"
                onClick={() => setExpandedSection(expandedSection === 'pending' ? null : 'pending')}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-purple-400">⏳</span>
                    <span className="text-white/90">Pendentes de confirmacao</span>
                    <span className="text-xs text-white/50">({pendingTx.length})</span>
                  </div>
                  <div className="text-right">
                    {totals.pendingExpense > 0 && <span className="text-rose-400/70 text-sm">{fmt(totals.pendingExpense)}</span>}
                    {totals.pendingIncome > 0 && <span className="text-emerald-400/70 text-sm ml-2">+{fmt(totals.pendingIncome)}</span>}
                  </div>
                </div>
                {expandedSection === 'pending' && (
                  <div className="mt-3 space-y-1 border-t border-white/10 pt-3">
                    {pendingTx.map((tx) => (
                      <div key={tx.id} className="flex justify-between text-sm">
                        <span className="text-white/70">
                          {tx.description}
                          <span className="text-white/40 ml-2 text-xs">
                            ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })})
                          </span>
                        </span>
                        <span className={tx.type === 'EXPENSE' ? 'text-rose-400/70' : 'text-emerald-400/70'}>
                          {fmt(tx.amount)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Summary */}
            <div className="border-t border-white/10 pt-4 space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-white/70">Orcamentos (gastos variaveis)</span>
                <span className="text-white">{fmt(totals.budgetTotal)}</span>
              </div>
              {totals.recurringExpense > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-white/70">Despesas fixas (fora orcamento)</span>
                  <span className="text-white">{fmt(totals.recurringExpense)}</span>
                </div>
              )}
              {totals.pendingExpense > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-white/70">Pendentes de confirmacao</span>
                  <span className="text-white/70">{fmt(totals.pendingExpense)}</span>
                </div>
              )}
              <div className="flex justify-between text-sm font-medium border-t border-white/10 pt-2">
                <span className="text-white">Custo total previsto</span>
                <span className="text-rose-400">{fmt(totals.budgetTotal + totals.recurringExpense + totals.pendingExpense)}</span>
              </div>
              {(totals.recurringIncome > 0 || totals.pendingIncome > 0) && (
                <div className="flex justify-between text-sm font-medium">
                  <span className="text-white">Receitas previstas</span>
                  <span className="text-emerald-400">{fmt(totals.recurringIncome + totals.pendingIncome)}</span>
                </div>
              )}
              <div className="flex justify-between text-lg font-bold border-t border-white/10 pt-2">
                <span className="text-white">Saldo previsto</span>
                <span className={(totals.recurringIncome + totals.pendingIncome) - (totals.budgetTotal + totals.recurringExpense + totals.pendingExpense) >= 0 ? 'text-emerald-400' : 'text-rose-400'}>
                  {fmt((totals.recurringIncome + totals.pendingIncome) - (totals.budgetTotal + totals.recurringExpense + totals.pendingExpense))}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
