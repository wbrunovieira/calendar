'use client';

import { useEffect, useMemo, useState, useCallback } from 'react';
import Link from 'next/link';
import AppLayout, { ProfileTheme } from '@/components/layout/AppLayout';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts';
import type { Profile, RecurringTransaction, Transaction, Category, BudgetSummaryItem } from '@/types/finances';

const CHART_COLORS = [
  '#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1',
  '#14b8a6', '#a855f7', '#eab308', '#22c55e', '#0ea5e9'
];

const API_BASE = 'http://localhost:3335/api/v1';

type ViewMode = 1 | 2 | 3 | 6 | 12;

interface MonthData {
  period: string;
  transactions: Transaction[];
  budgetSummary: BudgetSummaryItem[];
}

export default function VisaoMensalPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [basePeriod, setBasePeriod] = useState<string>(() => new Date().toISOString().slice(0, 7));
  const [categories, setCategories] = useState<Category[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [monthsData, setMonthsData] = useState<MonthData[]>([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>(1);
  const [expandedBudgets, setExpandedBudgets] = useState<Set<string>>(new Set());
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set());

  // Calculate periods to display based on view mode
  const periodsToShow = useMemo(() => {
    const periods: string[] = [];
    const [baseYear, baseMonth] = basePeriod.split('-').map(Number);

    for (let i = 0; i < viewMode; i++) {
      const date = new Date(baseYear, baseMonth - 1 + i, 1);
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, '0');
      periods.push(`${year}-${month}`);
    }

    return periods;
  }, [basePeriod, viewMode]);

  // Load profiles on mount
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/profiles`);
        const data = await res.json();
        const list: Profile[] = data.data || [];
        setProfiles(list);
        const defaultProfile = list.find((p) => p.name === 'Bruno Pessoal') || list[0];
        if (defaultProfile) setSelectedProfileId(defaultProfile.id);
      } catch (e) {
        console.warn('Erro ao carregar perfis', e);
      }
    })();
  }, []);

  // Load categories and recurrings (profile-based, period-independent)
  useEffect(() => {
    if (!selectedProfileId) return;

    (async () => {
      try {
        const [catRes, recRes] = await Promise.all([
          fetch(`${API_BASE}/categories?profileId=${selectedProfileId}`),
          fetch(`${API_BASE}/recurring-transactions?profileId=${selectedProfileId}`),
        ]);

        if (catRes.ok) {
          const catData = await catRes.json();
          setCategories(catData.data || []);
        }
        if (recRes.ok) {
          const recData = await recRes.json();
          setRecurrings(recData.data || []);
        }
      } catch (e) {
        console.warn('Erro ao carregar categorias e recorrentes', e);
      }
    })();
  }, [selectedProfileId]);

  // Load data for all visible periods
  useEffect(() => {
    if (!selectedProfileId) return;

    const loadAllPeriods = async () => {
      try {
        setLoading(true);

        const allMonthsData: MonthData[] = await Promise.all(
          periodsToShow.map(async (period) => {
            const [year, month] = period.split('-').map(Number);
            const start = new Date(year, month - 1, 1);
            const end = new Date(year, month, 0, 23, 59, 59);
            const fromDate = start.toISOString().slice(0, 10);
            const toDate = end.toISOString().slice(0, 10);

            const [txRes, budgetRes] = await Promise.all([
              fetch(`${API_BASE}/transactions?profileId=${selectedProfileId}&from=${fromDate}&to=${toDate}`),
              fetch(`${API_BASE}/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
            ]);

            const txData = txRes.ok ? await txRes.json() : { data: [] };
            const budgetData = budgetRes.ok ? await budgetRes.json() : { data: [] };

            return {
              period,
              transactions: txData.data || [],
              budgetSummary: budgetData.data || [],
            };
          })
        );

        setMonthsData(allMonthsData);
      } catch (e) {
        console.warn('Erro ao carregar dados dos meses', e);
      } finally {
        setLoading(false);
      }
    };

    loadAllPeriods();
  }, [selectedProfileId, periodsToShow]);

  const fmt = (v: number) =>
    new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(v);

  const getCategoryChain = useCallback((categoryId: string): string[] => {
    const chain: string[] = [categoryId];
    let current = categories.find((c) => c.id === categoryId);
    while (current?.parentId) {
      chain.push(current.parentId);
      current = categories.find((c) => c.id === current?.parentId);
    }
    return chain;
  }, [categories]);

  const toggleBudgetExpanded = (key: string) => {
    setExpandedBudgets((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const toggleSectionExpanded = (key: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const getGridCols = () => {
    switch (viewMode) {
      case 1: return 'grid-cols-1';
      case 2: return 'grid-cols-1 lg:grid-cols-2';
      case 3: return 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3';
      case 6: return 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3';
      case 12: return 'grid-cols-1 md:grid-cols-3 lg:grid-cols-4';
      default: return 'grid-cols-1';
    }
  };

  const selectedProfile = profiles.find((p) => p.id === selectedProfileId);
  const profileTheme: ProfileTheme = selectedProfile?.type === 'BUSINESS' ? 'business' : 'personal';

  return (
    <AppLayout profileTheme={profileTheme} profileName={selectedProfile?.name}>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Visao Mensal</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        {/* Profile, Period, and View Mode Selector */}
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
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <label className="text-white/70 text-sm">Inicio:</label>
                <input
                  type="month"
                  value={basePeriod}
                  onChange={(e) => setBasePeriod(e.target.value)}
                  className="bg-white/10 border border-white/20 text-white rounded-lg px-3 py-1.5 text-sm"
                />
              </div>
            </div>
          </div>

          {/* View Mode Selector */}
          <div className="flex items-center gap-2 mt-4 pt-4 border-t border-white/10">
            <p className="text-white/70 text-sm">Visualizar:</p>
            <div className="flex flex-wrap gap-2">
              {([1, 2, 3, 6, 12] as ViewMode[]).map((mode) => {
                const label = mode === 1 ? '1 mes' : mode === 12 ? '1 ano' : `${mode} meses`;
                return (
                  <button
                    key={mode}
                    onClick={() => setViewMode(mode)}
                    className={`px-3 py-1.5 rounded-xl border transition-colors text-sm ${
                      viewMode === mode
                        ? 'bg-blue-500/30 text-blue-300 border-blue-500/50'
                        : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'
                    }`}
                  >
                    {label}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* Main Content - Grid of Months */}
        {loading ? (
          <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
            <p className="text-white/50">Carregando visao do(s) mes(es)...</p>
          </div>
        ) : (
          <div className={`grid ${getGridCols()} gap-4`}>
            {monthsData.map((monthData) => (
              <MonthCard
                key={monthData.period}
                monthData={monthData}
                categories={categories}
                recurrings={recurrings}
                expandedBudgets={expandedBudgets}
                expandedSections={expandedSections}
                onToggleBudget={toggleBudgetExpanded}
                onToggleSection={toggleSectionExpanded}
                fmt={fmt}
                getCategoryChain={getCategoryChain}
                isCompact={viewMode > 1}
              />
            ))}
          </div>
        )}
      </div>
    </AppLayout>
  );
}

interface MonthCardProps {
  monthData: MonthData;
  categories: Category[];
  recurrings: RecurringTransaction[];
  expandedBudgets: Set<string>;
  expandedSections: Set<string>;
  onToggleBudget: (key: string) => void;
  onToggleSection: (key: string) => void;
  fmt: (v: number) => string;
  getCategoryChain: (categoryId: string) => string[];
  isCompact: boolean;
}

function MonthCard({
  monthData,
  categories,
  recurrings,
  expandedBudgets,
  expandedSections,
  onToggleBudget,
  onToggleSection,
  fmt,
  getCategoryChain,
  isCompact,
}: MonthCardProps) {
  const { period, transactions, budgetSummary } = monthData;
  const [year, month] = period.split('-').map(Number);
  const monthDate = new Date(year, month - 1, 1);
  const monthName = monthDate.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' });

  // Budget category IDs for this month
  const budgetCategoryIds = useMemo(() => {
    const ids = new Set<string>();
    budgetSummary.forEach((b) => ids.add(b.target.categoryId));
    return ids;
  }, [budgetSummary]);

  const isCategoryInBudget = useCallback((categoryId?: string): boolean => {
    if (!categoryId) return false;
    if (budgetCategoryIds.has(categoryId)) return true;
    const cat = categories.find((c) => c.id === categoryId);
    if (cat?.parentId) {
      if (budgetCategoryIds.has(cat.parentId)) return true;
      const parent = categories.find((c) => c.id === cat.parentId);
      if (parent?.parentId && budgetCategoryIds.has(parent.parentId)) return true;
    }
    return false;
  }, [budgetCategoryIds, categories]);

  // Recurring forecast
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
      items.push({ id: r.id, description: r.description, amount: r.amount, type: r.type, categoryId: r.categoryId, inBudget });
    });
    return items;
  }, [recurrings, isCategoryInBudget]);

  const recurringOutsideBudget = useMemo(() => recurringForecast.filter((r) => !r.inBudget), [recurringForecast]);

  // Pending transactions outside of budgets (for separate section)
  const pendingTxOutsideBudget = useMemo(() =>
    transactions.filter((tx) => {
      if (tx.status !== 'PLANNED') return false;
      return !isCategoryInBudget(tx.categoryId);
    }),
  [transactions, isCategoryInBudget]);

  // Income transactions (all, not just outside budget)
  const incomeTransactions = useMemo(() => ({
    confirmed: transactions.filter((tx) => tx.type === 'INCOME' && tx.status === 'CONFIRMED'),
    planned: transactions.filter((tx) => tx.type === 'INCOME' && tx.status === 'PLANNED'),
  }), [transactions]);

  // Totals
  const totals = useMemo(() => {
    const budgetTotal = budgetSummary.reduce((sum, b) => sum + Math.max(b.target.amount, b.spent), 0);
    const budgetSpent = budgetSummary.reduce((sum, b) => sum + b.spent, 0);
    const budgetRemaining = budgetSummary.reduce((sum, b) => sum + b.remaining, 0);
    const recurringExpense = recurringOutsideBudget.filter((r) => r.type === 'EXPENSE').reduce((sum, r) => sum + r.amount, 0);
    const recurringIncome = recurringOutsideBudget.filter((r) => r.type === 'INCOME').reduce((sum, r) => sum + r.amount, 0);
    const pendingExpense = pendingTxOutsideBudget.filter((tx) => tx.type === 'EXPENSE').reduce((sum, tx) => sum + tx.amount, 0);
    const pendingIncome = pendingTxOutsideBudget.filter((tx) => tx.type === 'INCOME').reduce((sum, tx) => sum + tx.amount, 0);
    const confirmedIncome = incomeTransactions.confirmed.reduce((sum, tx) => sum + tx.amount, 0);
    const plannedIncome = incomeTransactions.planned.reduce((sum, tx) => sum + tx.amount, 0);
    return { budgetTotal, budgetSpent, budgetRemaining, recurringExpense, recurringIncome, pendingExpense, pendingIncome, confirmedIncome, plannedIncome };
  }, [budgetSummary, recurringOutsideBudget, pendingTxOutsideBudget, incomeTransactions]);

  const getPendingRecurringForCategory = useCallback((budgetCategoryId: string) => {
    const today = new Date();
    const currentDay = today.getMonth() + 1 === month && today.getFullYear() === year ? today.getDate() : 0;
    const pending: { description: string; amount: number; day: number }[] = [];
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
        pending.push({ description: r.description, amount: r.amount, day: dayOfMonth });
      }
    });
    return pending;
  }, [recurrings, getCategoryChain, month, year]);

  const getTransactionsForCategory = useCallback((budgetCategoryId: string, status: 'CONFIRMED' | 'PLANNED') => {
    return transactions.filter((tx) => {
      if (!tx.categoryId || tx.status !== status) return false;
      const chain = getCategoryChain(tx.categoryId);
      return chain.includes(budgetCategoryId);
    });
  }, [transactions, getCategoryChain]);

  const totalExpense = totals.budgetTotal + totals.recurringExpense + totals.pendingExpense;
  const totalIncome = totals.confirmedIncome + totals.plannedIncome;
  const balance = totalIncome - totalExpense;

  // Calculate weeks in the month and expenses per week for budget categories
  const weeklyBudgetData = useMemo(() => {
    const lastDay = new Date(year, month, 0);
    const totalDays = lastDay.getDate();

    // Calculate full weeks (7 days each), remaining days go to last week
    const fullWeeks = Math.floor(totalDays / 7);
    const remainingDays = totalDays % 7;
    // If remaining days exist, merge with last week; otherwise use 4 weeks
    const numWeeks = remainingDays > 0 ? fullWeeks : fullWeeks;

    // Define week ranges - last week gets extra days
    const weeks: { start: number; end: number; label: string }[] = [];
    for (let i = 0; i < numWeeks; i++) {
      const start = i * 7 + 1;
      const isLastWeek = i === numWeeks - 1;
      const end = isLastWeek ? totalDays : (i + 1) * 7;
      weeks.push({ start, end, label: `Sem ${i + 1} (${start}-${end})` });
    }

    // Get transactions for each budget category grouped by week
    const budgetWeeklyData: {
      categoryId: string;
      categoryName: string;
      budgetAmount: number;
      weeklyBudget: number;
      weeks: { label: string; spent: number; transactions: Transaction[] }[];
      totalSpent: number;
    }[] = [];

    budgetSummary.forEach((budget) => {
      const cat = categories.find((c) => c.id === budget.target.categoryId);
      const categoryName = cat?.name || 'Categoria';
      const weeklyBudget = budget.target.amount / numWeeks;

      const weekData = weeks.map((week) => {
        const weekTx = transactions.filter((tx) => {
          if (tx.type !== 'EXPENSE' || tx.status !== 'CONFIRMED') return false;
          if (!tx.categoryId) return false;
          const chain = getCategoryChain(tx.categoryId);
          if (!chain.includes(budget.target.categoryId)) return false;
          const txDay = new Date(tx.occurredOn).getDate();
          return txDay >= week.start && txDay <= week.end;
        });
        const spent = weekTx.reduce((sum, tx) => sum + tx.amount, 0);
        return { label: week.label, spent, transactions: weekTx };
      });

      budgetWeeklyData.push({
        categoryId: budget.target.categoryId,
        categoryName,
        budgetAmount: budget.target.amount,
        weeklyBudget,
        weeks: weekData,
        totalSpent: budget.spent,
      });
    });

    return { weeks, budgetWeeklyData, numWeeks };
  }, [budgetSummary, transactions, categories, getCategoryChain, year, month]);

  // Calculate expense totals by category for pie charts
  const categoryExpenseData = useMemo(() => {
    const expenseTransactions = transactions.filter((tx) => tx.type === 'EXPENSE');

    // Group by parent category (top-level)
    const parentTotals = new Map<string, { name: string; value: number }>();
    // Group by actual category (subcategory level)
    const subTotals = new Map<string, { name: string; value: number; parentName: string }>();

    expenseTransactions.forEach((tx) => {
      if (!tx.categoryId) return;

      const cat = categories.find((c) => c.id === tx.categoryId);
      if (!cat) return;

      // Find top-level parent
      let topParent = cat;
      const parentChain = [cat.name];
      while (topParent.parentId) {
        const parent = categories.find((c) => c.id === topParent.parentId);
        if (parent) {
          topParent = parent;
          parentChain.unshift(parent.name);
        } else {
          break;
        }
      }

      // Aggregate by parent category
      const existing = parentTotals.get(topParent.id);
      if (existing) {
        existing.value += tx.amount;
      } else {
        parentTotals.set(topParent.id, { name: topParent.name, value: tx.amount });
      }

      // Aggregate by subcategory (use full path for clarity)
      const subKey = tx.categoryId;
      const subName = parentChain.length > 1 ? parentChain.slice(-2).join(' > ') : cat.name;
      const existingSub = subTotals.get(subKey);
      if (existingSub) {
        existingSub.value += tx.amount;
      } else {
        subTotals.set(subKey, { name: subName, value: tx.amount, parentName: topParent.name });
      }
    });

    const parentData = Array.from(parentTotals.values())
      .sort((a, b) => b.value - a.value);

    const subData = Array.from(subTotals.values())
      .sort((a, b) => b.value - a.value)
      .slice(0, 10); // Top 10 subcategories

    return { parentData, subData };
  }, [transactions, categories]);

  return (
    <div className="bg-white/5 border border-white/10 rounded-2xl p-4 space-y-3">
      <h3 className="text-lg font-semibold text-white capitalize border-b border-white/10 pb-2">
        {monthName}
      </h3>

      {/* Budget Summary */}
      {budgetSummary.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-xs font-medium text-white/60">Orcamentos</h4>
          <div className={`grid gap-2 ${isCompact ? 'grid-cols-1' : 'grid-cols-1 md:grid-cols-2'}`}>
            {budgetSummary.map((budget) => {
              const cat = categories.find((c) => c.id === budget.target.categoryId);
              const catName = cat?.name || 'Categoria';
              const percentSpent = budget.target.amount > 0 ? (budget.spent / budget.target.amount) * 100 : 0;
              const isOverBudget = budget.spent > budget.target.amount;
              const pendingRecurrings = getPendingRecurringForCategory(budget.target.categoryId);
              const completedTx = getTransactionsForCategory(budget.target.categoryId, 'CONFIRMED');
              const plannedTx = getTransactionsForCategory(budget.target.categoryId, 'PLANNED');
              const pendingRecurringAmount = pendingRecurrings.reduce((sum, r) => sum + r.amount, 0);
              const plannedAmount = plannedTx.reduce((sum, tx) => sum + tx.amount, 0);
              const pendingAmount = pendingRecurringAmount + plannedAmount;
              const percentPending = budget.target.amount > 0 ? (pendingAmount / budget.target.amount) * 100 : 0;
              const budgetKey = `${period}-${budget.target.id}`;
              const isExpanded = expandedBudgets.has(budgetKey);
              const hasDetails = completedTx.length > 0 || pendingRecurrings.length > 0 || plannedTx.length > 0;

              return (
                <div key={budget.target.id} className="bg-white/5 border border-white/10 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-white text-sm font-medium truncate">{catName}</span>
                    {budget.target.isRecurring && <span className="text-blue-400 text-xs">recorrente</span>}
                  </div>

                  <div className="grid grid-cols-3 gap-1 text-center mb-2">
                    <div>
                      <p className="text-[10px] text-white/40">Orc.</p>
                      <p className="text-xs font-semibold text-white">{fmt(budget.target.amount)}</p>
                    </div>
                    <div>
                      <p className="text-[10px] text-white/40">Gasto</p>
                      <p className={`text-xs font-semibold ${isOverBudget ? 'text-rose-400' : 'text-emerald-400'}`}>
                        {fmt(budget.spent)}
                      </p>
                    </div>
                    <div>
                      <p className="text-[10px] text-white/40">Resta</p>
                      <p className={`text-xs font-semibold ${budget.remaining >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                        {fmt(budget.remaining)}
                      </p>
                    </div>
                  </div>

                  {/* Progress bar */}
                  <div className="h-1.5 bg-white/10 rounded-full overflow-hidden flex">
                    <div
                      className={`h-full transition-all ${isOverBudget ? 'bg-rose-500' : 'bg-emerald-500'}`}
                      style={{ width: `${Math.min(100, percentSpent)}%` }}
                    />
                    {pendingAmount > 0 && (
                      <div
                        className="h-full bg-blue-500/70 transition-all"
                        style={{ width: `${Math.min(100 - percentSpent, percentPending)}%` }}
                      />
                    )}
                  </div>
                  <p className="text-[10px] text-white/40 mt-0.5 text-right">{Math.round(percentSpent)}%</p>

                  {/* Expandable details */}
                  {hasDetails && (
                    <div className="mt-2 pt-2 border-t border-white/10">
                      <button
                        onClick={() => onToggleBudget(budgetKey)}
                        className="flex items-center gap-1 text-[10px] text-white/50 hover:text-white/70 transition-colors w-full"
                      >
                        <svg
                          className={`w-2.5 h-2.5 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                          fill="none" stroke="currentColor" viewBox="0 0 24 24"
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                        </svg>
                        <span>Detalhes ({completedTx.length}+{pendingRecurrings.length + plannedTx.length})</span>
                      </button>

                      <div className={`overflow-hidden transition-all duration-300 ease-in-out ${isExpanded ? 'max-h-[400px] opacity-100 mt-2' : 'max-h-0 opacity-0'}`}>
                        {completedTx.length > 0 && (
                          <div className="mb-2">
                            <p className="text-[10px] text-emerald-400 mb-1">Concluidos</p>
                            <div className="space-y-0.5">
                              {completedTx.map((tx) => (
                                <div key={tx.id} className="flex justify-between text-[10px]">
                                  <span className="text-white/60 truncate flex-1">{tx.description}</span>
                                  <span className="text-emerald-400 ml-1">{fmt(tx.amount)}</span>
                                </div>
                              ))}
                              <div className="flex justify-between text-[10px] pt-0.5 border-t border-white/10">
                                <span className="text-white/40">Total</span>
                                <span className="text-emerald-400">{fmt(completedTx.reduce((s, t) => s + t.amount, 0))}</span>
                              </div>
                            </div>
                          </div>
                        )}
                        {(pendingRecurrings.length > 0 || plannedTx.length > 0) && (
                          <div className="mb-2">
                            <p className="text-[10px] text-blue-400 mb-1">Pendentes</p>
                            <div className="space-y-0.5">
                              {pendingRecurrings.map((r, idx) => (
                                <div key={idx} className="flex justify-between text-[10px]">
                                  <span className="text-white/60 truncate flex-1">{r.description} <span className="text-white/30">(fixa d{r.day})</span></span>
                                  <span className="text-blue-400 ml-1">{fmt(r.amount)}</span>
                                </div>
                              ))}
                              {plannedTx.map((tx) => (
                                <div key={tx.id} className="flex justify-between text-[10px]">
                                  <span className="text-white/60 truncate flex-1">
                                    {tx.description}
                                    <span className="text-white/30 ml-1">
                                      ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })})
                                    </span>
                                  </span>
                                  <span className="text-blue-400 ml-1">{fmt(tx.amount)}</span>
                                </div>
                              ))}
                              <div className="flex justify-between text-[10px] pt-0.5 border-t border-white/10">
                                <span className="text-white/40">Total</span>
                                <span className="text-blue-400">{fmt(pendingAmount)}</span>
                              </div>
                            </div>
                          </div>
                        )}
                        <div className="flex justify-between text-[10px] pt-1 border-t border-white/20">
                          <span className="text-white font-medium">Total geral</span>
                          <span className="text-white font-medium">{fmt(completedTx.reduce((s, t) => s + t.amount, 0) + pendingAmount)}</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {budgetSummary.length === 0 && (
        <div className="text-center py-4">
          <p className="text-white/50 text-xs">Nenhum orcamento.</p>
        </div>
      )}

      {/* Weekly Budget Breakdown */}
      {weeklyBudgetData.budgetWeeklyData.length > 0 && (
        <div className="space-y-4">
          <h4 className="text-base font-semibold text-white/70">Gastos por Semana</h4>
          {weeklyBudgetData.budgetWeeklyData.map((budgetCat) => {
            const categoryKey = `${period}-weekly-${budgetCat.categoryId}`;
            const isCategoryExpanded = expandedSections.has(categoryKey);

            return (
              <div key={budgetCat.categoryId} className="bg-white/5 border border-white/10 rounded-xl p-4">
                <div
                  className="flex items-center justify-between cursor-pointer"
                  onClick={() => onToggleSection(categoryKey)}
                >
                  <div className="flex items-center gap-2">
                    <svg
                      className={`w-4 h-4 text-white/50 transition-transform duration-200 ${isCategoryExpanded ? 'rotate-90' : ''}`}
                      fill="none" stroke="currentColor" viewBox="0 0 24 24"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                    <span className="text-white text-base font-medium">{budgetCat.categoryName}</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-white/50 text-sm">Meta: {fmt(budgetCat.weeklyBudget)}/sem</span>
                    <span className="text-white font-semibold text-sm">{fmt(budgetCat.totalSpent)}</span>
                  </div>
                </div>

                <div className={`overflow-hidden transition-all duration-300 ease-in-out ${isCategoryExpanded ? 'max-h-[800px] opacity-100 mt-4' : 'max-h-0 opacity-0'}`}>
                  <div className="space-y-3">
                    {budgetCat.weeks.map((week, idx) => {
                      const isOverWeeklyBudget = week.spent > budgetCat.weeklyBudget;
                      const percentOfWeekly = budgetCat.weeklyBudget > 0 ? (week.spent / budgetCat.weeklyBudget) * 100 : 0;
                      const weekKey = `${period}-${budgetCat.categoryId}-week-${idx}`;
                      const isExpanded = expandedSections.has(weekKey);

                      return (
                        <div key={idx} className="bg-white/5 rounded-lg p-3">
                          <div
                            className="flex items-center justify-between cursor-pointer"
                            onClick={(e) => {
                              e.stopPropagation();
                              week.transactions.length > 0 && onToggleSection(weekKey);
                            }}
                          >
                            <div className="flex items-center gap-2">
                              <span className="text-white/80 text-sm font-medium">{week.label}</span>
                              {week.transactions.length > 0 && (
                                <span className="text-white/40 text-xs">({week.transactions.length})</span>
                              )}
                            </div>
                            <div className="flex items-center gap-3">
                              <span className={`text-sm font-semibold ${isOverWeeklyBudget ? 'text-rose-400' : 'text-emerald-400'}`}>
                                {fmt(week.spent)}
                              </span>
                              <span className={`text-xs ${isOverWeeklyBudget ? 'text-rose-400/70' : 'text-white/50'}`}>
                                ({Math.round(percentOfWeekly)}%)
                              </span>
                            </div>
                          </div>
                          {/* Progress bar */}
                          <div className="h-1.5 bg-white/10 rounded-full overflow-hidden mt-2">
                            <div
                              className={`h-full transition-all ${isOverWeeklyBudget ? 'bg-rose-500' : 'bg-emerald-500'}`}
                              style={{ width: `${Math.min(100, percentOfWeekly)}%` }}
                            />
                          </div>
                          {/* Expandable transactions */}
                          {isExpanded && week.transactions.length > 0 && (
                            <div className="mt-3 pt-3 border-t border-white/10 space-y-1">
                              {week.transactions.map((tx) => (
                                <div key={tx.id} className="flex justify-between text-xs">
                                  <span className="text-white/60 truncate flex-1">
                                    {tx.description}
                                    <span className="text-white/30 ml-1">
                                      ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit' })})
                                    </span>
                                  </span>
                                  <span className="text-emerald-400 ml-2 font-medium">{fmt(tx.amount)}</span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Fixed Recurring Expenses (outside budget) */}
      {recurringOutsideBudget.filter(r => r.type === 'EXPENSE').length > 0 && (
        <div
          className="bg-white/5 border border-white/10 rounded-lg p-2 cursor-pointer hover:bg-white/10 transition-colors"
          onClick={() => onToggleSection(`${period}-recurring`)}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1">
              <span className="text-blue-400 text-xs">📅</span>
              <span className="text-white/80 text-xs">Fixas fora orc.</span>
              <span className="text-[10px] text-white/40">({recurringOutsideBudget.filter(r => r.type === 'EXPENSE').length})</span>
            </div>
            <span className="text-rose-400 text-xs font-medium">{fmt(totals.recurringExpense)}</span>
          </div>
          {expandedSections.has(`${period}-recurring`) && (
            <div className="mt-2 space-y-0.5 border-t border-white/10 pt-2">
              {recurringOutsideBudget.filter(r => r.type === 'EXPENSE').map((item) => (
                <div key={item.id} className="flex justify-between text-[10px]">
                  <span className="text-white/60 truncate">{item.description}</span>
                  <span className="text-rose-400">{fmt(item.amount)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Income */}
      {(totals.confirmedIncome > 0 || totals.plannedIncome > 0) && (
        <div
          className="bg-white/5 border border-white/10 rounded-lg p-2 cursor-pointer hover:bg-white/10 transition-colors"
          onClick={() => onToggleSection(`${period}-income`)}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1">
              <span className="text-emerald-400 text-xs">💰</span>
              <span className="text-white/80 text-xs">Receitas</span>
              <span className="text-[10px] text-white/40">({incomeTransactions.confirmed.length + incomeTransactions.planned.length})</span>
            </div>
            <span className="text-emerald-400 text-xs font-medium">{fmt(totalIncome)}</span>
          </div>
          {expandedSections.has(`${period}-income`) && (
            <div className="mt-2 space-y-1 border-t border-white/10 pt-2">
              {incomeTransactions.confirmed.length > 0 && (
                <div className="mb-2">
                  <p className="text-[10px] text-emerald-400 mb-1">Já recebido</p>
                  <div className="space-y-0.5">
                    {incomeTransactions.confirmed.map((tx) => (
                      <div key={tx.id} className="flex justify-between text-[10px]">
                        <span className="text-white/60 truncate flex-1">
                          {tx.description}
                          <span className="text-white/30 ml-1">
                            ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })})
                          </span>
                        </span>
                        <span className="text-emerald-400 ml-1">{fmt(tx.amount)}</span>
                      </div>
                    ))}
                    <div className="flex justify-between text-[10px] pt-0.5 border-t border-white/10">
                      <span className="text-white/40">Subtotal</span>
                      <span className="text-emerald-400 font-medium">{fmt(totals.confirmedIncome)}</span>
                    </div>
                  </div>
                </div>
              )}
              {incomeTransactions.planned.length > 0 && (
                <div className="mb-2">
                  <p className="text-[10px] text-blue-400 mb-1">A receber</p>
                  <div className="space-y-0.5">
                    {incomeTransactions.planned.map((tx) => (
                      <div key={tx.id} className="flex justify-between text-[10px]">
                        <span className="text-white/60 truncate flex-1">
                          {tx.description}
                          <span className="text-white/30 ml-1">
                            ({new Date(tx.occurredOn).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })})
                          </span>
                        </span>
                        <span className="text-blue-400 ml-1">{fmt(tx.amount)}</span>
                      </div>
                    ))}
                    <div className="flex justify-between text-[10px] pt-0.5 border-t border-white/10">
                      <span className="text-white/40">Subtotal</span>
                      <span className="text-blue-400 font-medium">{fmt(totals.plannedIncome)}</span>
                    </div>
                  </div>
                </div>
              )}
              <div className="flex justify-between text-[10px] pt-1 border-t border-white/20">
                <span className="text-white font-medium">Total receitas</span>
                <span className="text-emerald-400 font-medium">{fmt(totalIncome)}</span>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Pending Transactions */}
      {pendingTxOutsideBudget.length > 0 && (
        <div
          className="bg-white/5 border border-white/10 rounded-lg p-2 cursor-pointer hover:bg-white/10 transition-colors"
          onClick={() => onToggleSection(`${period}-pending`)}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1">
              <span className="text-purple-400 text-xs">⏳</span>
              <span className="text-white/80 text-xs">Pendentes</span>
              <span className="text-[10px] text-white/40">({pendingTxOutsideBudget.length})</span>
            </div>
            <div className="text-right">
              {totals.pendingExpense > 0 && <span className="text-rose-400/70 text-xs">{fmt(totals.pendingExpense)}</span>}
            </div>
          </div>
          {expandedSections.has(`${period}-pending`) && (
            <div className="mt-2 space-y-0.5 border-t border-white/10 pt-2">
              {pendingTxOutsideBudget.map((tx) => (
                <div key={tx.id} className="flex justify-between text-[10px]">
                  <span className="text-white/60 truncate">{tx.description}</span>
                  <span className={tx.type === 'EXPENSE' ? 'text-rose-400/70' : 'text-emerald-400/70'}>{fmt(tx.amount)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Summary */}
      <div className="border-t border-white/10 pt-3 space-y-1">
        <div className="flex justify-between text-xs">
          <span className="text-white/60">Orcamentos</span>
          <span className="text-white">{fmt(totals.budgetTotal)}</span>
        </div>
        {totals.recurringExpense > 0 && (
          <div className="flex justify-between text-xs">
            <span className="text-white/60">Fixas</span>
            <span className="text-white">{fmt(totals.recurringExpense)}</span>
          </div>
        )}
        <div className="flex justify-between text-xs font-medium border-t border-white/10 pt-1">
          <span className="text-white">Total despesas</span>
          <span className="text-rose-400">{fmt(totalExpense)}</span>
        </div>
        {totals.confirmedIncome > 0 && (
          <div className="flex justify-between text-xs">
            <span className="text-white/60">Já recebido</span>
            <span className="text-emerald-400">{fmt(totals.confirmedIncome)}</span>
          </div>
        )}
        {totals.plannedIncome > 0 && (
          <div className="flex justify-between text-xs">
            <span className="text-white/60">A receber</span>
            <span className="text-blue-400">{fmt(totals.plannedIncome)}</span>
          </div>
        )}
        {totalIncome > 0 && (
          <div className="flex justify-between text-xs font-medium border-t border-white/10 pt-1">
            <span className="text-white">Total receitas</span>
            <span className="text-emerald-400">{fmt(totalIncome)}</span>
          </div>
        )}
        <div className="flex justify-between text-sm font-bold border-t border-white/10 pt-1">
          <span className="text-white">Saldo</span>
          <span className={balance >= 0 ? 'text-emerald-400' : 'text-rose-400'}>{fmt(balance)}</span>
        </div>
      </div>

      {/* Pie Charts - Category Breakdown */}
      {categoryExpenseData.parentData.length > 0 && (
        <div className="border-t border-white/10 pt-6 space-y-6">
          <h4 className="text-base font-semibold text-white">Despesas por Categoria</h4>

          {/* Parent Categories Pie Chart */}
          <div className="bg-white/5 border border-white/10 rounded-xl p-4">
            <p className="text-sm text-white/70 mb-3 text-center font-medium">Categorias principais</p>
            <div className="h-72">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={categoryExpenseData.parentData}
                    cx="50%"
                    cy="50%"
                    innerRadius={isCompact ? 50 : 60}
                    outerRadius={isCompact ? 85 : 100}
                    paddingAngle={3}
                    dataKey="value"
                    label={({ name, percent }) => isCompact ? `${((percent ?? 0) * 100).toFixed(0)}%` : `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                    labelLine={!isCompact}
                  >
                    {categoryExpenseData.parentData.map((_, index) => (
                      <Cell key={`cell-parent-${index}`} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value) => fmt(typeof value === 'number' ? value : 0)}
                    contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: '8px', fontSize: '14px' }}
                    labelStyle={{ color: '#fff', fontSize: '14px' }}
                  />
                  {isCompact && <Legend wrapperStyle={{ fontSize: '12px' }} />}
                </PieChart>
              </ResponsiveContainer>
            </div>
            {/* Legend list for non-compact */}
            {!isCompact && (
              <div className="mt-4 grid grid-cols-2 gap-2">
                {categoryExpenseData.parentData.map((item, index) => (
                  <div key={item.name} className="flex items-center gap-2 text-sm">
                    <div
                      className="w-3 h-3 rounded-full flex-shrink-0"
                      style={{ backgroundColor: CHART_COLORS[index % CHART_COLORS.length] }}
                    />
                    <span className="text-white/70 truncate">{item.name}</span>
                    <span className="text-white font-medium ml-auto">{fmt(item.value)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Subcategories Pie Chart */}
          {categoryExpenseData.subData.length > 1 && (
            <div className="bg-white/5 border border-white/10 rounded-xl p-4">
              <p className="text-sm text-white/70 mb-3 text-center font-medium">Subcategorias (top 10)</p>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={categoryExpenseData.subData}
                      cx="50%"
                      cy="50%"
                      innerRadius={isCompact ? 50 : 60}
                      outerRadius={isCompact ? 85 : 100}
                      paddingAngle={3}
                      dataKey="value"
                      label={({ percent }) => `${((percent ?? 0) * 100).toFixed(0)}%`}
                      labelLine={false}
                    >
                      {categoryExpenseData.subData.map((_, index) => (
                        <Cell key={`cell-sub-${index}`} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip
                      formatter={(value) => fmt(typeof value === 'number' ? value : 0)}
                      contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: '8px', fontSize: '14px' }}
                      labelStyle={{ color: '#fff', fontSize: '14px' }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              {/* Legend list */}
              <div className="mt-4 space-y-1.5">
                {categoryExpenseData.subData.map((item, index) => (
                  <div key={`${item.name}-${index}`} className="flex items-center gap-2 text-sm">
                    <div
                      className="w-3 h-3 rounded-full flex-shrink-0"
                      style={{ backgroundColor: CHART_COLORS[index % CHART_COLORS.length] }}
                    />
                    <span className="text-white/70 truncate flex-1">{item.name}</span>
                    <span className="text-white font-medium">{fmt(item.value)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
