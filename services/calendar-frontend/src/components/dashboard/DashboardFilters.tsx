/**
 * Dashboard Filters Component
 * Provides advanced filtering options for statistics
 */

import { Category, CategoryType } from '@/types/calendar';
import { calendars } from '@/data/calendars';

interface DashboardFiltersProps {
  dateRange: {
    startDate: string;
    endDate: string;
  };
  selectedCalendar: string;
  selectedCategory: string;
  selectedCategoryType: string;
  groupBy: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';
  categories: Category[];
  categoryTypes: CategoryType[];
  onDateRangeChange: (range: { startDate: string; endDate: string }) => void;
  onCalendarChange: (calendarId: string) => void;
  onCategoryChange: (categoryId: string) => void;
  onCategoryTypeChange: (categoryTypeId: string) => void;
  onGroupByChange: (groupBy: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total') => void;
  onRefresh: () => void;
}

export default function DashboardFilters({
  dateRange,
  selectedCalendar,
  selectedCategory,
  selectedCategoryType,
  groupBy,
  categories,
  categoryTypes,
  onDateRangeChange,
  onCalendarChange,
  onCategoryChange,
  onCategoryTypeChange,
  onGroupByChange,
  onRefresh,
}: DashboardFiltersProps) {
  // Quick date range presets
  const setPresetRange = (days: number) => {
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - days);
    onDateRangeChange({
      startDate: startDate.toISOString().split('T')[0],
      endDate: endDate.toISOString().split('T')[0],
    });
  };

  return (
    <div className="bg-gradient-to-br from-white/10 to-white/5 backdrop-blur-sm rounded-xl p-6 border border-white/20 shadow-lg mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Filtros</h2>
        <button
          onClick={onRefresh}
          className="px-3 py-1.5 bg-purple-500/20 hover:bg-purple-500/30 border border-purple-500/30 rounded-lg transition-colors text-sm"
        >
          🔄 Atualizar
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* Date Range */}
        <div className="col-span-full">
          <label className="block text-sm font-medium text-white/70 mb-2">
            Período
          </label>
          <div className="flex gap-2 flex-wrap">
            <button
              onClick={() => setPresetRange(7)}
              className="px-3 py-1.5 bg-white/10 hover:bg-white/20 rounded-lg transition-colors text-sm"
            >
              7 dias
            </button>
            <button
              onClick={() => setPresetRange(30)}
              className="px-3 py-1.5 bg-white/10 hover:bg-white/20 rounded-lg transition-colors text-sm"
            >
              30 dias
            </button>
            <button
              onClick={() => setPresetRange(90)}
              className="px-3 py-1.5 bg-white/10 hover:bg-white/20 rounded-lg transition-colors text-sm"
            >
              90 dias
            </button>
            <button
              onClick={() => setPresetRange(365)}
              className="px-3 py-1.5 bg-white/10 hover:bg-white/20 rounded-lg transition-colors text-sm"
            >
              1 ano
            </button>
          </div>
          <div className="flex gap-2 mt-3">
            <div className="flex-1">
              <label className="block text-xs text-white/50 mb-1">Data Inicial</label>
              <input
                type="date"
                value={dateRange.startDate}
                onChange={(e) => onDateRangeChange({ ...dateRange, startDate: e.target.value })}
                className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>
            <div className="flex-1">
              <label className="block text-xs text-white/50 mb-1">Data Final</label>
              <input
                type="date"
                value={dateRange.endDate}
                onChange={(e) => onDateRangeChange({ ...dateRange, endDate: e.target.value })}
                className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>
          </div>
        </div>

        {/* Calendar Filter */}
        <div>
          <label className="block text-sm font-medium text-white/70 mb-2">
            Calendário
          </label>
          <select
            value={selectedCalendar}
            onChange={(e) => onCalendarChange(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
          >
            <option value="">Todos</option>
            {calendars.map((calendar) => (
              <option key={calendar.id} value={calendar.id}>
                {calendar.type === 'professional' ? '💼' : '👤'} {calendar.name}
              </option>
            ))}
          </select>
        </div>

        {/* Category Filter */}
        <div>
          <label className="block text-sm font-medium text-white/70 mb-2">
            Categoria
          </label>
          <select
            value={selectedCategory}
            onChange={(e) => onCategoryChange(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
          >
            <option value="">Todas</option>
            {categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </select>
        </div>

        {/* Category Type Filter */}
        <div>
          <label className="block text-sm font-medium text-white/70 mb-2">
            Tipo de Categoria
          </label>
          <select
            value={selectedCategoryType}
            onChange={(e) => onCategoryTypeChange(e.target.value)}
            className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
          >
            <option value="">Todos</option>
            {categoryTypes.map((type) => (
              <option key={type.id} value={type.id}>
                {type.icon} {type.name}
              </option>
            ))}
          </select>
        </div>

        {/* Group By */}
        <div className="col-span-full">
          <label className="block text-sm font-medium text-white/70 mb-2">
            Agrupar Por
          </label>
          <div className="flex gap-2 flex-wrap">
            <button
              onClick={() => onGroupByChange('day')}
              className={`px-4 py-2 rounded-lg transition-colors ${
                groupBy === 'day'
                  ? 'bg-purple-500 text-white'
                  : 'bg-white/10 hover:bg-white/20 text-white/70'
              }`}
            >
              Dia
            </button>
            <button
              onClick={() => onGroupByChange('week')}
              className={`px-4 py-2 rounded-lg transition-colors ${
                groupBy === 'week'
                  ? 'bg-purple-500 text-white'
                  : 'bg-white/10 hover:bg-white/20 text-white/70'
              }`}
            >
              Semana
            </button>
            <button
              onClick={() => onGroupByChange('month')}
              className={`px-4 py-2 rounded-lg transition-colors ${
                groupBy === 'month'
                  ? 'bg-purple-500 text-white'
                  : 'bg-white/10 hover:bg-white/20 text-white/70'
              }`}
            >
              Mês
            </button>
            <button
              onClick={() => onGroupByChange('category')}
              className={`px-4 py-2 rounded-lg transition-colors ${
                groupBy === 'category'
                  ? 'bg-purple-500 text-white'
                  : 'bg-white/10 hover:bg-white/20 text-white/70'
              }`}
            >
              Categoria
            </button>
            <button
              onClick={() => onGroupByChange('categoryType')}
              className={`px-4 py-2 rounded-lg transition-colors ${
                groupBy === 'categoryType'
                  ? 'bg-purple-500 text-white'
                  : 'bg-white/10 hover:bg-white/20 text-white/70'
              }`}
            >
              Tipo
            </button>
            <button
              onClick={() => onGroupByChange('total')}
              className={`px-4 py-2 rounded-lg transition-colors ${
                groupBy === 'total'
                  ? 'bg-purple-500 text-white'
                  : 'bg-white/10 hover:bg-white/20 text-white/70'
              }`}
            >
              Total
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
