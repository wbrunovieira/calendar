'use client';

import { useEffect, useMemo, useState, useCallback } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import { useProfile } from '@/contexts/ProfileContext';
import { api } from '@/lib/api';
import type { RecurringTransaction, Transaction, Category, BudgetSummaryItem } from '@/types/finances';
import MonthCard from '@/components/finances/MonthCard';

type ViewMode = 1 | 2 | 3 | 6 | 12;

interface MonthData {
  period: string;
  transactions: Transaction[];
  budgetSummary: BudgetSummaryItem[];
}

export default function VisaoMensalPage() {
  const { selectedProfileId } = useProfile();
  const [basePeriod, setBasePeriod] = useState<string>(() => new Date().toISOString().slice(0, 7));
  const [categories, setCategories] = useState<Category[]>([]);
  const [recurrings, setRecurrings] = useState<RecurringTransaction[]>([]);
  const [monthsData, setMonthsData] = useState<MonthData[]>([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>(1);
  const [expandedBudgets, setExpandedBudgets] = useState<Set<string>>(new Set());
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set());

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

  useEffect(() => {
    if (!selectedProfileId) return;

    (async () => {
      try {
        const [catData, recData] = await Promise.all([
          api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}`),
          api.get<{ data: RecurringTransaction[] }>(`/recurring-transactions?profileId=${selectedProfileId}`),
        ]);

        setCategories(catData.data || []);
        setRecurrings(recData.data || []);
      } catch (e) {
        console.warn('Erro ao carregar categorias e recorrentes', e);
      }
    })();
  }, [selectedProfileId]);

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

            let txData: { data: Transaction[] } = { data: [] };
            let budgetData: { data: BudgetSummaryItem[] } = { data: [] };
            try {
              [txData, budgetData] = await Promise.all([
                api.get<{ data: Transaction[] }>(`/transactions?profileId=${selectedProfileId}&from=${fromDate}&to=${toDate}`),
                api.get<{ data: BudgetSummaryItem[] }>(`/budgets/summary?profileId=${selectedProfileId}&period=${period}`),
              ]);
            } catch { /* graceful fallback */ }

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

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-white">Visao Mensal</h2>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        {/* Period and View Mode Selector */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <label className="text-white/70 text-sm">Inicio:</label>
              <input
                type="month"
                value={basePeriod}
                onChange={(e) => setBasePeriod(e.target.value)}
                className="bg-white/10 border border-white/20 text-white rounded-lg px-3 py-1.5 text-sm"
              />
            </div>
            <div className="flex items-center gap-2">
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
        </div>

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
