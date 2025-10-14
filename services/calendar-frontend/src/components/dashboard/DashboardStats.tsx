/**
 * Dashboard Stats Component
 * Displays statistics in various visualizations
 */

import { StatsData } from '@/lib/api';
import StatsCard from './StatsCard';
import StatsTable from './StatsTable';
import StatsChart from './StatsChart';
import StatsGroupCards from './StatsGroupCards';

interface DashboardStatsProps {
  stats: StatsData[];
  groupBy: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';
  dateRange: {
    startDate: string;
    endDate: string;
  };
  categoryStats?: StatsData[];
  categoryTypeStats?: StatsData[];
  periodStats?: StatsData[];
}

export default function DashboardStats({ stats, groupBy, dateRange, categoryStats, categoryTypeStats, periodStats }: DashboardStatsProps) {
  if (stats.length === 0) {
    return (
      <div className="bg-white/5 border border-white/10 rounded-xl p-12 text-center">
        <p className="text-white/60 text-lg">
          Nenhuma estatística encontrada para os filtros selecionados.
        </p>
        <p className="text-white/40 text-sm mt-2">
          Tente ajustar os filtros ou adicionar mais eventos ao calendário.
        </p>
      </div>
    );
  }

  // Calculate overall stats
  const totalEvents = stats.reduce((sum, stat) => sum + stat.total, 0);
  const completedEvents = stats.reduce((sum, stat) => sum + stat.completed, 0);
  const pendingEvents = stats.reduce((sum, stat) => sum + stat.pending, 0);
  const overallPercentage = totalEvents > 0 ? Math.round((completedEvents / totalEvents) * 100) : 0;

  // Calculate streaks
  const maxStreak = Math.max(...stats.map(s => s.streak || 0), 0);
  const perfectDays = stats.filter(s => s.percentage === 100 && s.total > 0).length;

  // Calculate average completion time if available
  const avgCompletionTimes = stats.filter(s => s.averageCompletionTime).map(s => s.averageCompletionTime!);
  const overallAvgCompletionTime = avgCompletionTimes.length > 0
    ? Math.round(avgCompletionTimes.reduce((sum, time) => sum + time, 0) / avgCompletionTimes.length)
    : null;

  return (
    <div className="space-y-6">
      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatsCard
          title="Total de Eventos"
          value={totalEvents}
          icon="📊"
          color="blue"
        />
        <StatsCard
          title="Concluídos"
          value={completedEvents}
          subtitle={`${overallPercentage}% do total`}
          icon="✅"
          color="green"
        />
        <StatsCard
          title="Pendentes"
          value={pendingEvents}
          subtitle={`${100 - overallPercentage}% do total`}
          icon="⏳"
          color="yellow"
        />
        <StatsCard
          title="Taxa de Conclusão"
          value={`${overallPercentage}%`}
          subtitle={overallPercentage >= 70 ? '🎯 Excelente!' : overallPercentage >= 50 ? '👍 Bom!' : '💪 Continue!'}
          icon="📈"
          color="purple"
        />
      </div>

      {/* Additional Metrics */}
      {(maxStreak > 0 || perfectDays > 0 || overallAvgCompletionTime) && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {maxStreak > 0 && (
            <StatsCard
              title="Maior Sequência"
              value={maxStreak}
              subtitle="dias com 100% conclusão"
              icon="🔥"
              color="orange"
            />
          )}
          {perfectDays > 0 && (
            <StatsCard
              title="Dias Perfeitos"
              value={perfectDays}
              subtitle="100% de conclusão"
              icon="🌟"
              color="yellow"
            />
          )}
          {overallAvgCompletionTime && (
            <StatsCard
              title="Tempo Médio"
              value={`${Math.floor(overallAvgCompletionTime / 60)}h ${overallAvgCompletionTime % 60}m`}
              subtitle="até conclusão"
              icon="⏱️"
              color="cyan"
            />
          )}
        </div>
      )}

      {/* Stats by Category */}
      {categoryStats && categoryStats.length > 0 && (
        <StatsGroupCards
          stats={categoryStats}
          title="Por Categoria"
          icon="📁"
        />
      )}

      {/* Stats by Category Type */}
      {categoryTypeStats && categoryTypeStats.length > 0 && (
        <StatsGroupCards
          stats={categoryTypeStats}
          title="Por Tipo"
          icon="🏷️"
        />
      )}

      {/* Stats by Period (if viewing by day/week/month) */}
      {groupBy !== 'category' && groupBy !== 'categoryType' && periodStats && periodStats.length > 0 && (
        <StatsGroupCards
          stats={periodStats.slice(-14)}
          title="Últimos 14 Dias"
          icon="📅"
        />
      )}

      {/* Chart Visualization */}
      {groupBy !== 'total' && stats.length > 1 && (
        <StatsChart stats={stats} groupBy={groupBy} />
      )}

      {/* Detailed Table */}
      <StatsTable stats={stats} groupBy={groupBy} />
    </div>
  );
}
