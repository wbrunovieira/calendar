'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useEventsStats } from '@/hooks/useEventsStats';
import { useCategories } from '@/hooks/useCategories';
import { useCategoryTypes } from '@/hooks/useCategoryTypes';
import DashboardFilters from '@/components/dashboard/DashboardFilters';
import DashboardStats from '@/components/dashboard/DashboardStats';

export default function DashboardPage() {
  const router = useRouter();

  // Filter states
  const [dateRange, setDateRange] = useState<{
    startDate: string;
    endDate: string;
  }>(() => {
    // Default: last 30 days
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - 30);
    return {
      startDate: startDate.toISOString().split('T')[0],
      endDate: endDate.toISOString().split('T')[0],
    };
  });

  const [selectedCalendar, setSelectedCalendar] = useState<string>('');
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [selectedCategoryType, setSelectedCategoryType] = useState<string>('');
  const [groupBy, setGroupBy] = useState<'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total'>('day');

  // Fetch data
  const { categories } = useCategories();
  const { categoryTypes } = useCategoryTypes();

  const { stats, loading, error, refetch } = useEventsStats({
    startDate: dateRange.startDate,
    endDate: dateRange.endDate,
    calendarId: selectedCalendar || undefined,
    categoryId: selectedCategory || undefined,
    categoryTypeId: selectedCategoryType || undefined,
    groupBy,
    includeBreakdown: true,
    enabled: true,
  });

  // Fetch stats by category
  const { stats: categoryStats } = useEventsStats({
    startDate: dateRange.startDate,
    endDate: dateRange.endDate,
    calendarId: selectedCalendar || undefined,
    groupBy: 'category',
    enabled: true,
  });

  // Fetch stats by category type
  const { stats: categoryTypeStats } = useEventsStats({
    startDate: dateRange.startDate,
    endDate: dateRange.endDate,
    calendarId: selectedCalendar || undefined,
    groupBy: 'categoryType',
    enabled: true,
  });

  // Fetch stats by period (last 30 days)
  const { stats: periodStats } = useEventsStats({
    startDate: dateRange.startDate,
    endDate: dateRange.endDate,
    groupBy: 'day',
    enabled: true,
  });

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#0a0118] via-[#1a0a2e] to-[#16213e] text-white">
      {/* Header */}
      <div className="border-b border-white/10 bg-black/20 backdrop-blur-sm">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <button
                onClick={() => router.push('/')}
                className="text-white/60 hover:text-white transition-colors mb-2 flex items-center gap-2"
              >
                ← Voltar ao Calendário
              </button>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">
                Dashboard de Estatísticas
              </h1>
              <p className="text-white/60 mt-1">
                Análise detalhada de produtividade e conclusão de eventos
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="container mx-auto px-4 py-8">
        {/* Filters */}
        <DashboardFilters
          dateRange={dateRange}
          selectedCalendar={selectedCalendar}
          selectedCategory={selectedCategory}
          selectedCategoryType={selectedCategoryType}
          groupBy={groupBy}
          categories={categories}
          categoryTypes={categoryTypes}
          onDateRangeChange={setDateRange}
          onCalendarChange={setSelectedCalendar}
          onCategoryChange={setSelectedCategory}
          onCategoryTypeChange={setSelectedCategoryType}
          onGroupByChange={setGroupBy}
          onRefresh={refetch}
        />

        {/* Stats Display */}
        {loading && (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-purple-500"></div>
            <p className="mt-4 text-white/60">Carregando estatísticas...</p>
          </div>
        )}

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-6 text-center">
            <p className="text-red-400">Erro ao carregar estatísticas: {error.message}</p>
            <button
              onClick={() => refetch()}
              className="mt-4 px-4 py-2 bg-red-500/20 hover:bg-red-500/30 rounded-lg transition-colors"
            >
              Tentar Novamente
            </button>
          </div>
        )}

        {!loading && !error && (
          <DashboardStats
            stats={stats}
            groupBy={groupBy}
            dateRange={dateRange}
            categoryStats={categoryStats}
            categoryTypeStats={categoryTypeStats}
            periodStats={periodStats}
          />
        )}
      </div>
    </div>
  );
}
