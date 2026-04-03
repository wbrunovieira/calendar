'use client';

import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  Legend,
  AreaChart,
  Area,
} from 'recharts';
import { formatCurrency, formatCurrencyCompact, parseLocalDate } from '@/utils/format';

const COLORS = [
  '#10b981', '#3b82f6', '#8b5cf6', '#f59e0b', '#ef4444',
  '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#6366f1',
];

const MONTHS = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
];

interface MonthlyDataItem {
  month: string;
  monthIndex: number;
  income: number;
  expense: number;
  balance: number;
}

interface CumulativeDataItem extends MonthlyDataItem {
  cumulative: number;
}

interface CategoryDataItem {
  [key: string]: unknown;
  id: string;
  name: string;
  value: number;
  color: string;
  subcategories: { name: string; value: number }[];
}

interface IncomeCategoryItem {
  [key: string]: unknown;
  name: string;
  value: number;
  color: string;
}

interface TopExpenseItem {
  id: string;
  description: string;
  amount: number;
  occurredOn: string;
  categoryName: string;
}

interface DashboardChartsProps {
  totals: { income: number; expense: number; balance: number };
  accountTotals: { available: number; investments: number; total: number };
  selectedYear: number;
  selectedMonth: number | null;
  monthlyData: MonthlyDataItem[];
  cumulativeData: CumulativeDataItem[];
  categoryData: CategoryDataItem[];
  incomeCategoryData: IncomeCategoryItem[];
  topExpenses: TopExpenseItem[];
  filteredTransactionsCount: number;
}

const CustomTooltip = ({ active, payload, label }: { active?: boolean; payload?: Array<{ name: string; value: number; color: string }>; label?: string }) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-gray-900/95 border border-white/20 rounded-lg p-3 shadow-xl">
        <p className="text-white/80 text-sm font-medium mb-1">{label}</p>
        {payload.map((entry, index) => (
          <p key={index} className="text-sm" style={{ color: entry.color }}>
            {entry.name}: {formatCurrency(entry.value)}
          </p>
        ))}
      </div>
    );
  }
  return null;
};

export default function DashboardCharts({
  totals,
  accountTotals,
  selectedYear,
  selectedMonth,
  monthlyData,
  cumulativeData,
  categoryData,
  incomeCategoryData,
  topExpenses,
  filteredTransactionsCount,
}: DashboardChartsProps) {
  return (
    <>
      {/* KPI Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <div className="bg-gradient-to-br from-emerald-600/20 to-emerald-900/20 border border-emerald-500/30 rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-2xl">📈</span>
            <span className="text-emerald-300/80 text-sm">Receitas</span>
          </div>
          <div className="text-3xl font-bold text-emerald-400">
            {formatCurrencyCompact(totals.income)}
          </div>
          <p className="text-emerald-300/60 text-xs mt-1">
            {selectedMonth !== null ? MONTHS[selectedMonth] : 'Ano'} de {selectedYear}
          </p>
        </div>

        <div className="bg-gradient-to-br from-rose-600/20 to-rose-900/20 border border-rose-500/30 rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-2xl">📉</span>
            <span className="text-rose-300/80 text-sm">Despesas</span>
          </div>
          <div className="text-3xl font-bold text-rose-400">
            {formatCurrencyCompact(totals.expense)}
          </div>
          <p className="text-rose-300/60 text-xs mt-1">
            {selectedMonth !== null ? MONTHS[selectedMonth] : 'Ano'} de {selectedYear}
          </p>
        </div>

        <div className={`bg-gradient-to-br ${totals.balance >= 0 ? 'from-blue-600/20 to-blue-900/20 border-blue-500/30' : 'from-amber-600/20 to-amber-900/20 border-amber-500/30'} border rounded-2xl p-5`}>
          <div className="flex items-center gap-2 mb-2">
            <span className="text-2xl">💰</span>
            <span className={`${totals.balance >= 0 ? 'text-blue-300/80' : 'text-amber-300/80'} text-sm`}>Saldo</span>
          </div>
          <div className={`text-3xl font-bold ${totals.balance >= 0 ? 'text-blue-400' : 'text-amber-400'}`}>
            {formatCurrencyCompact(totals.balance)}
          </div>
          <p className={`${totals.balance >= 0 ? 'text-blue-300/60' : 'text-amber-300/60'} text-xs mt-1`}>
            Receitas - Despesas
          </p>
        </div>

        <div className="bg-gradient-to-br from-purple-600/20 to-purple-900/20 border border-purple-500/30 rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-2xl">🏦</span>
            <span className="text-purple-300/80 text-sm">Patrimonio</span>
          </div>
          <div className="text-3xl font-bold text-purple-400">
            {formatCurrencyCompact(accountTotals.total)}
          </div>
          <p className="text-purple-300/60 text-xs mt-1">
            {formatCurrencyCompact(accountTotals.available)} disponivel + {formatCurrencyCompact(accountTotals.investments)} investido
          </p>
        </div>
      </div>

      {/* Charts Row 1 */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Monthly Income vs Expense Bar Chart */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Receitas vs Despesas por Mes</h3>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={monthlyData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                <XAxis dataKey="month" tick={{ fill: '#94a3b8', fontSize: 12 }} />
                <YAxis tick={{ fill: '#94a3b8', fontSize: 12 }} tickFormatter={(v) => formatCurrencyCompact(v)} />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                <Bar dataKey="income" name="Receitas" fill="#10b981" radius={[4, 4, 0, 0]} />
                <Bar dataKey="expense" name="Despesas" fill="#f43f5e" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Cumulative Balance Line Chart */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Evolucao do Saldo Acumulado</h3>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={cumulativeData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <defs>
                  <linearGradient id="colorBalance" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
                <XAxis dataKey="month" tick={{ fill: '#94a3b8', fontSize: 12 }} />
                <YAxis tick={{ fill: '#94a3b8', fontSize: 12 }} tickFormatter={(v) => formatCurrencyCompact(v)} />
                <Tooltip content={<CustomTooltip />} />
                <Area
                  type="monotone"
                  dataKey="cumulative"
                  name="Saldo Acumulado"
                  stroke="#3b82f6"
                  fill="url(#colorBalance)"
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Charts Row 2 - Pie Charts */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Expense by Category Pie */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">
            Despesas por Categoria
            {selectedMonth !== null && <span className="text-white/50 text-sm font-normal ml-2">({MONTHS[selectedMonth]})</span>}
          </h3>
          {categoryData.length === 0 ? (
            <div className="h-64 flex items-center justify-center text-white/50">
              Nenhuma despesa no periodo
            </div>
          ) : (
            <div className="flex flex-col lg:flex-row items-center gap-4">
              <div className="h-64 w-64">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={categoryData}
                      cx="50%"
                      cy="50%"
                      innerRadius={50}
                      outerRadius={90}
                      paddingAngle={2}
                      dataKey="value"
                    >
                      {categoryData.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip formatter={(value) => formatCurrency(Number(value))} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="flex-1 space-y-1 max-h-64 overflow-y-auto">
                {categoryData.map((cat, index) => (
                  <div key={cat.id}>
                    <div className="flex items-center justify-between gap-2 text-sm py-1">
                      <div className="flex items-center gap-2">
                        <div
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: COLORS[index % COLORS.length] }}
                        />
                        <span className="text-white/80 font-medium">{cat.name}</span>
                      </div>
                      <span className="text-white/60 font-medium">{formatCurrency(cat.value)}</span>
                    </div>
                    {cat.subcategories && cat.subcategories.length > 0 && (
                      <div className="ml-5 space-y-0.5 border-l border-white/10 pl-3 mb-1">
                        {cat.subcategories.map((sub) => (
                          <div key={sub.name} className="flex items-center justify-between gap-2 text-xs">
                            <span className="text-white/50">↳ {sub.name}</span>
                            <span className="text-white/40">{formatCurrency(sub.value)}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Income by Category Pie */}
        <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">
            Receitas por Categoria
            {selectedMonth !== null && <span className="text-white/50 text-sm font-normal ml-2">({MONTHS[selectedMonth]})</span>}
          </h3>
          {incomeCategoryData.length === 0 ? (
            <div className="h-64 flex items-center justify-center text-white/50">
              Nenhuma receita no periodo
            </div>
          ) : (
            <div className="flex flex-col lg:flex-row items-center gap-4">
              <div className="h-64 w-64">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={incomeCategoryData}
                      cx="50%"
                      cy="50%"
                      innerRadius={50}
                      outerRadius={90}
                      paddingAngle={2}
                      dataKey="value"
                    >
                      {incomeCategoryData.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip formatter={(value) => formatCurrency(Number(value))} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="flex-1 space-y-2 max-h-64 overflow-y-auto">
                {incomeCategoryData.map((cat, index) => (
                  <div key={cat.name} className="flex items-center justify-between gap-2 text-sm">
                    <div className="flex items-center gap-2">
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: COLORS[index % COLORS.length] }}
                      />
                      <span className="text-white/80">{cat.name}</span>
                    </div>
                    <span className="text-white/60 font-medium">{formatCurrency(cat.value)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Monthly Balance Trend */}
      <div className="bg-white/5 border border-white/10 rounded-2xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Tendencia Mensal</h3>
        <div className="h-72">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={monthlyData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
              <XAxis dataKey="month" tick={{ fill: '#94a3b8', fontSize: 12 }} />
              <YAxis tick={{ fill: '#94a3b8', fontSize: 12 }} tickFormatter={(v) => formatCurrencyCompact(v)} />
              <Tooltip content={<CustomTooltip />} />
              <Legend />
              <Line type="monotone" dataKey="income" name="Receitas" stroke="#10b981" strokeWidth={2} dot={{ fill: '#10b981', strokeWidth: 2 }} />
              <Line type="monotone" dataKey="expense" name="Despesas" stroke="#f43f5e" strokeWidth={2} dot={{ fill: '#f43f5e', strokeWidth: 2 }} />
              <Line type="monotone" dataKey="balance" name="Saldo" stroke="#3b82f6" strokeWidth={2} strokeDasharray="5 5" dot={{ fill: '#3b82f6', strokeWidth: 2 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Top Expenses & Summary */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Top 5 Expenses */}
        <div className="lg:col-span-2 bg-white/5 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">
            Maiores Despesas
            {selectedMonth !== null && <span className="text-white/50 text-sm font-normal ml-2">({MONTHS[selectedMonth]})</span>}
          </h3>
          {topExpenses.length === 0 ? (
            <p className="text-white/50">Nenhuma despesa no periodo</p>
          ) : (
            <div className="space-y-3">
              {topExpenses.map((tx, index) => (
                <div
                  key={tx.id}
                  className="flex items-center justify-between bg-white/5 rounded-xl px-4 py-3 border border-white/10"
                >
                  <div className="flex items-center gap-4">
                    <div className="w-8 h-8 rounded-full bg-rose-500/20 flex items-center justify-center text-rose-400 font-bold text-sm">
                      {index + 1}
                    </div>
                    <div>
                      <p className="text-white font-medium">{tx.description}</p>
                      <p className="text-white/50 text-xs">
                        {tx.categoryName} • {parseLocalDate(tx.occurredOn).toLocaleDateString('pt-BR')}
                      </p>
                    </div>
                  </div>
                  <div className="text-rose-400 font-semibold">
                    -{formatCurrency(tx.amount)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Quick Stats */}
        <div className="bg-gradient-to-br from-slate-800/50 to-slate-900/50 border border-white/10 rounded-2xl p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Resumo Rapido</h3>
          <div className="space-y-4">
            <div className="flex justify-between items-center py-2 border-b border-white/10">
              <span className="text-white/60">Total Transacoes</span>
              <span className="text-white font-semibold">{filteredTransactionsCount}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-white/10">
              <span className="text-white/60">Media Despesa</span>
              <span className="text-white font-semibold">
                {formatCurrency(
                  categoryData.length > 0
                    ? categoryData.reduce((sum, c) => sum + c.value, 0) / categoryData.length
                    : 0
                )}
              </span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-white/10">
              <span className="text-white/60">Categorias Usadas</span>
              <span className="text-white font-semibold">{categoryData.length}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-white/10">
              <span className="text-white/60">Maior Categoria</span>
              <span className="text-white font-semibold">
                {categoryData[0]?.name || '-'}
              </span>
            </div>
            <div className="flex justify-between items-center py-2">
              <span className="text-white/60">% Economia</span>
              <span className={`font-semibold ${totals.income > 0 && totals.balance > 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                {totals.income > 0
                  ? `${((totals.balance / totals.income) * 100).toFixed(1)}%`
                  : '0%'}
              </span>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
