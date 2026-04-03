'use client';

import { useEffect, useMemo, useState } from 'react';
import { api } from '@/lib/api';
import type { RecurringTransaction, Transaction, Category, BankAccount, BudgetSummaryItem } from '@/types/finances';

interface MonthlyCostOverviewProps {
  profileId: string;
  categories: Category[];
  accounts: BankAccount[];
}

export default function MonthlyCostOverview({ profileId, categories }: MonthlyCostOverviewProps) {
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [budgetSummary, setBudgetSummary] = useState<BudgetSummaryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedSection, setExpandedSection] = useState<string | null>(null);

  const currentMonth = useMemo(() => {
    const now = new Date();
    return {
      year: now.getFullYear(),
      month: now.getMonth(),
      start: new Date(now.getFullYear(), now.getMonth(), 1),
      end: new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59),
      period: `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`,
    };
  }, []);

  useEffect(() => {
    if (!profileId) return;

    const loadData = async () => {
      try {
        setLoading(true);
        const fromDate = currentMonth.start.toISOString().slice(0, 10);
        const toDate = currentMonth.end.toISOString().slice(0, 10);

        const [recData, txData, budgetData] = await Promise.all([
          api.get<{ data: RecurringTransaction[] }>(`/recurring-transactions?profileId=${profileId}`),
          api.get<{ data: Transaction[] }>(`/transactions?profileId=${profileId}&from=${fromDate}&to=${toDate}`),
          api.get<{ data: BudgetSummaryItem[] }>(`/budgets/summary?profileId=${profileId}&period=${currentMonth.period}`),
        ]);

        setRecurrings(recData.data || []);
        setTransactions(txData.data || []);
        setBudgetSummary(budgetData.data || []);
      } catch (e) {
        console.warn('Erro ao carregar dados do overview', e);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [profileId, currentMonth]);

  // Get all category IDs that are covered by budgets (including parent categories)
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

  // Calculate recurring forecast for the month (excluding those covered by budgets)
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
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recurrings, budgetCategoryIds, categories]);

  // Separate recurring into those covered by budget and those not
  const { recurringOutsideBudget } = useMemo(() => {
    return {
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
    // Budget totals: use higher of budget or spent (if exceeded)
    const budgetTotal = budgetSummary.reduce((sum, b) => sum + Math.max(b.target.amount, b.spent), 0);
    const budgetSpent = budgetSummary.reduce((sum, b) => sum + b.spent, 0);
    const budgetRemaining = budgetSummary.reduce((sum, b) => sum + b.remaining, 0);

    // Recurring forecast OUTSIDE budget (fixed expenses not covered by budget)
    const recurringExpense = recurringOutsideBudget
      .filter((r) => r.type === 'EXPENSE')
      .reduce((sum, r) => sum + r.amount, 0);
    const recurringIncome = recurringOutsideBudget
      .filter((r) => r.type === 'INCOME')
      .reduce((sum, r) => sum + r.amount, 0);

    // Confirmed transactions
    const confirmedExpense = confirmedTx
      .filter((tx) => tx.type === 'EXPENSE')
      .reduce((sum, tx) => sum + tx.amount, 0);
    const confirmedIncome = confirmedTx
      .filter((tx) => tx.type === 'INCOME')
      .reduce((sum, tx) => sum + tx.amount, 0);

    // Pending transactions
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

  const monthName = currentMonth.start.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' });

  if (loading) {
    return (
      <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
        <p className="text-white/50">Carregando visao do mes...</p>
      </div>
    );
  }

  return (
    <div className="bg-white/5 border border-white/10 rounded-2xl p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-white capitalize">Visao de {monthName}</h3>
      </div>

      {/* Budget Summary - Main Focus */}
      {budgetSummary.length > 0 && (
        <div className="space-y-3">
          <h4 className="text-sm font-medium text-white/70">Orcamentos</h4>
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
                    {budget.target.isRecurring && <span className="text-blue-400 mr-2">recorrente</span>}
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
  );
}
