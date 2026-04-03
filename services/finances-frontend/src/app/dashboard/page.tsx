'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import AppLayout from '@/components/layout/AppLayout';
import type { Profile, Category, Transaction, BankAccount } from '@/types/finances';
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
import { api } from '@/lib/api';
import { formatCurrency, formatCurrencyCompact, parseLocalDate } from '@/utils/format';

const COLORS = [
  '#10b981', '#3b82f6', '#8b5cf6', '#f59e0b', '#ef4444',
  '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#6366f1',
];

const MONTHS = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro'
];

const MONTHS_SHORT = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

export default function DashboardPage() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState<number | null>(null); // null = all months
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const data = await api.get<{ data: Profile[] }>('/profiles');
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
    setLoading(true);
    try {
      // Fetch full year of transactions
      const startDate = `${selectedYear}-01-01`;
      const endDate = `${selectedYear}-12-31`;

      const [catData, txData, accData] = await Promise.all([
        api.get<{ data: Category[] }>(`/categories?profileId=${selectedProfileId}`),
        api.get<{ data: Transaction[] }>(`/transactions?profileId=${selectedProfileId}&from=${startDate}&to=${endDate}`),
        api.get<{ data: BankAccount[] }>(`/bank-accounts?profileId=${selectedProfileId}`),
      ]);

      setCategories(catData.data || []);
      setTransactions(txData.data || []);
      setAccounts(accData.data || []);
    } catch (e) {
      console.warn('Erro ao carregar dados', e);
    } finally {
      setLoading(false);
    }
  }, [selectedProfileId, selectedYear]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Filter transactions by selected month (if any)
  const filteredTransactions = useMemo(() => {
    if (selectedMonth === null) return transactions;
    return transactions.filter((tx) => {
      const txMonth = parseLocalDate(tx.occurredOn).getMonth();
      return txMonth === selectedMonth;
    });
  }, [transactions, selectedMonth]);

  // Monthly data for bar/line charts
  const monthlyData = useMemo(() => {
    const data = MONTHS_SHORT.map((month, index) => {
      const monthTx = transactions.filter((tx) => {
        const txMonth = parseLocalDate(tx.occurredOn).getMonth();
        return txMonth === index && tx.status === 'CONFIRMED';
      });

      const income = monthTx
        .filter((tx) => tx.type === 'INCOME')
        .reduce((sum, tx) => sum + tx.amount, 0);

      const expense = monthTx
        .filter((tx) => tx.type === 'EXPENSE')
        .reduce((sum, tx) => sum + tx.amount, 0);

      return {
        month,
        monthIndex: index,
        income,
        expense,
        balance: income - expense,
      };
    });
    return data;
  }, [transactions]);

  // Category breakdown for pie chart - grouped by parent
  const categoryData = useMemo(() => {
    const expenseTx = filteredTransactions.filter(
      (tx) => tx.type === 'EXPENSE' && tx.status === 'CONFIRMED'
    );

    // Group by parent category
    const byParent: Record<string, { total: number; subcategories: Record<string, number> }> = {};

    expenseTx.forEach((tx) => {
      const category = categories.find((c) => c.id === tx.categoryId);
      const parentId = category?.parentId || category?.id || 'sem-categoria';
      if (!byParent[parentId]) {
        byParent[parentId] = { total: 0, subcategories: {} };
      }
      byParent[parentId].total += tx.amount;

      // Track subcategory breakdown
      if (category?.parentId) {
        const subName = category.name;
        byParent[parentId].subcategories[subName] = (byParent[parentId].subcategories[subName] || 0) + tx.amount;
      }
    });

    return Object.entries(byParent)
      .map(([categoryId, data]) => {
        const category = categories.find((c) => c.id === categoryId);
        return {
          id: categoryId,
          name: category?.name || 'Sem categoria',
          value: data.total,
          color: category?.color || '#64748b',
          subcategories: Object.entries(data.subcategories)
            .map(([name, value]) => ({ name, value }))
            .sort((a, b) => b.value - a.value),
        };
      })
      .sort((a, b) => b.value - a.value);
  }, [filteredTransactions, categories]);

  // Income category breakdown
  const incomeCategoryData = useMemo(() => {
    const incomeTx = filteredTransactions.filter(
      (tx) => tx.type === 'INCOME' && tx.status === 'CONFIRMED'
    );

    const byCategory: Record<string, number> = {};
    incomeTx.forEach((tx) => {
      const catId = tx.categoryId || 'sem-categoria';
      byCategory[catId] = (byCategory[catId] || 0) + tx.amount;
    });

    return Object.entries(byCategory)
      .map(([categoryId, amount]) => {
        const category = categories.find((c) => c.id === categoryId);
        return {
          name: category?.name || 'Sem categoria',
          value: amount,
          color: category?.color || '#64748b',
        };
      })
      .sort((a, b) => b.value - a.value);
  }, [filteredTransactions, categories]);

  // Totals
  const totals = useMemo(() => {
    const confirmed = filteredTransactions.filter((tx) => tx.status === 'CONFIRMED');
    const income = confirmed
      .filter((tx) => tx.type === 'INCOME')
      .reduce((sum, tx) => sum + tx.amount, 0);
    const expense = confirmed
      .filter((tx) => tx.type === 'EXPENSE')
      .reduce((sum, tx) => sum + tx.amount, 0);
    return { income, expense, balance: income - expense };
  }, [filteredTransactions]);

  // Account totals
  const accountTotals = useMemo(() => {
    const profileAccounts = accounts.filter((acc) => acc.profileId === selectedProfileId);
    const available = profileAccounts
      .filter((acc) => acc.type !== 'CREDIT_CARD' && acc.type !== 'INVESTMENT')
      .reduce((sum, acc) => sum + acc.currentBalance, 0);
    const investments = profileAccounts
      .filter((acc) => acc.type === 'INVESTMENT')
      .reduce((sum, acc) => sum + acc.currentBalance, 0);
    return { available, investments, total: available + investments };
  }, [accounts, selectedProfileId]);

  // Cumulative balance over months
  const cumulativeData = useMemo(() => {
    let cumulative = 0;
    return monthlyData.map((m) => {
      cumulative += m.balance;
      return { ...m, cumulative };
    });
  }, [monthlyData]);

  // Top expenses
  const topExpenses = useMemo(() => {
    return filteredTransactions
      .filter((tx) => tx.type === 'EXPENSE' && tx.status === 'CONFIRMED')
      .sort((a, b) => b.amount - a.amount)
      .slice(0, 5)
      .map((tx) => ({
        ...tx,
        categoryName: categories.find((c) => c.id === tx.categoryId)?.name || 'Sem categoria',
      }));
  }, [filteredTransactions, categories]);


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
            {/* Profile selector */}
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

            {/* Year selector */}
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

            {/* Month selector */}
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
                            {categoryData.map((entry, index) => (
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
                            {incomeCategoryData.map((entry, index) => (
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
                    <Line
                      type="monotone"
                      dataKey="income"
                      name="Receitas"
                      stroke="#10b981"
                      strokeWidth={2}
                      dot={{ fill: '#10b981', strokeWidth: 2 }}
                    />
                    <Line
                      type="monotone"
                      dataKey="expense"
                      name="Despesas"
                      stroke="#f43f5e"
                      strokeWidth={2}
                      dot={{ fill: '#f43f5e', strokeWidth: 2 }}
                    />
                    <Line
                      type="monotone"
                      dataKey="balance"
                      name="Saldo"
                      stroke="#3b82f6"
                      strokeWidth={2}
                      strokeDasharray="5 5"
                      dot={{ fill: '#3b82f6', strokeWidth: 2 }}
                    />
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
                    <span className="text-white font-semibold">{filteredTransactions.length}</span>
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
        )}
      </div>
    </AppLayout>
  );
}
