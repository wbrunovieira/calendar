'use client';

import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import DashboardCharts from '@/components/finances/DashboardCharts';
import { useDashboardData } from '@/hooks/useDashboardData';

const MONTHS = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
];

export default function DashboardPage() {
  const {
    profiles,
    selectedProfileId,
    setSelectedProfileId,
    selectedYear,
    setSelectedYear,
    selectedMonth,
    setSelectedMonth,
    loading,
    filteredTransactions,
    monthlyData,
    categoryData,
    incomeCategoryData,
    totals,
    accountTotals,
    cumulativeData,
    topExpenses,
  } = useDashboardData();

  return (
    <AppLayout>
      <div className="py-6 space-y-6">
        {/* Header */}
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold text-white">Dashboard Financeiro</h1>
            <p className="text-white/60 text-sm">Analise completa de receitas, despesas e tendencias</p>
          </div>
          <Link href="/" className="text-sm text-white/70 hover:text-white underline">← Voltar</Link>
        </div>

        {/* Filters */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-4">
          <div className="flex flex-wrap items-center gap-6">
            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Perfil:</span>
              <div className="flex flex-wrap gap-2">
                {profiles.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => setSelectedProfileId(p.id)}
                    className={`px-3 py-1.5 rounded-xl border transition-colors ${
                      selectedProfileId === p.id
                        ? 'bg-emerald-500/30 border-emerald-400/50 text-emerald-200'
                        : 'bg-white/5 text-white/60 hover:bg-white/10 border-white/15'
                    }`}
                  >
                    {p.type === 'PERSONAL' ? '👤' : '🏢'} {p.name}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Ano:</span>
              <select
                value={selectedYear}
                onChange={(e) => setSelectedYear(Number(e.target.value))}
                className="bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm"
              >
                {[2024, 2025, 2026].map((year) => (
                  <option key={year} value={year} className="bg-gray-800">
                    {year}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-white/70 text-sm">Mes:</span>
              <select
                value={selectedMonth ?? ''}
                onChange={(e) => setSelectedMonth(e.target.value === '' ? null : Number(e.target.value))}
                className="bg-white/10 border border-white/20 rounded-lg px-3 py-1.5 text-white text-sm"
              >
                <option value="" className="bg-gray-800">Todos</option>
                {MONTHS.map((month, index) => (
                  <option key={index} value={index} className="bg-gray-800">
                    {month}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {loading ? (
          <div className="text-center py-12 text-white/60">Carregando dados...</div>
        ) : (
          <DashboardCharts
            totals={totals}
            accountTotals={accountTotals}
            selectedYear={selectedYear}
            selectedMonth={selectedMonth}
            monthlyData={monthlyData}
            cumulativeData={cumulativeData}
            categoryData={categoryData}
            incomeCategoryData={incomeCategoryData}
            topExpenses={topExpenses}
            filteredTransactionsCount={filteredTransactions.length}
          />
        )}
      </div>
    </AppLayout>
  );
}
