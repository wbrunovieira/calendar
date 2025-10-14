/**
 * Stats Chart Component
 * Displays statistics in a visual bar chart
 */

import { StatsData } from '@/lib/api';

interface StatsChartProps {
  stats: StatsData[];
  groupBy: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';
}

export default function StatsChart({ stats, groupBy }: StatsChartProps) {
  // Sort stats by date/label
  const sortedStats = [...stats].sort((a, b) => {
    if (groupBy === 'day' || groupBy === 'week' || groupBy === 'month') {
      // Sort by key (date) ascending (oldest first for timeline)
      return a.key.localeCompare(b.key);
    }
    // Sort by total descending
    return b.total - a.total;
  });

  // Limit to last 30 entries for better visualization
  const displayStats = sortedStats.slice(-30);

  // Find max value for scaling
  const maxValue = Math.max(...displayStats.map(s => s.total), 1);

  const getBarColor = (percentage: number) => {
    if (percentage === 100) return 'bg-gradient-to-t from-green-500 to-emerald-600';
    if (percentage >= 70) return 'bg-gradient-to-t from-blue-500 to-cyan-600';
    if (percentage >= 40) return 'bg-gradient-to-t from-yellow-500 to-orange-500';
    return 'bg-gradient-to-t from-red-500 to-pink-600';
  };

  return (
    <div className="bg-gradient-to-br from-white/10 to-white/5 backdrop-blur-sm rounded-xl p-6 border border-white/20 shadow-lg">
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-white mb-2">Visualização Temporal</h2>
        <p className="text-sm text-white/60">
          {displayStats.length < sortedStats.length && `Mostrando últimos ${displayStats.length} de ${sortedStats.length}`}
        </p>
      </div>

      {/* Chart */}
      <div className="relative">
        {/* Grid lines */}
        <div className="absolute inset-0 flex flex-col justify-between">
          {[0, 25, 50, 75, 100].map((percentage) => (
            <div key={percentage} className="border-t border-white/5 relative">
              <span className="absolute -left-8 -top-2 text-xs text-white/40">
                {percentage}%
              </span>
            </div>
          ))}
        </div>

        {/* Bars */}
        <div className="flex items-end justify-between gap-1 h-64 relative z-10 pl-8">
          {displayStats.map((stat) => {
            const heightPercentage = (stat.total / maxValue) * 100;
            const completionPercentage = stat.total > 0 ? (stat.completed / stat.total) * 100 : 0;

            return (
              <div
                key={stat.key}
                className="flex-1 flex flex-col items-center group cursor-pointer"
                title={`${stat.label}: ${stat.completed}/${stat.total} (${stat.percentage}%)`}
              >
                {/* Bar container */}
                <div className="w-full relative" style={{ height: `${heightPercentage}%`, minHeight: stat.total > 0 ? '8px' : '0' }}>
                  {/* Completed portion */}
                  <div
                    className={`w-full absolute bottom-0 rounded-t-lg ${getBarColor(stat.percentage)} transition-all duration-300 group-hover:opacity-80`}
                    style={{ height: `${completionPercentage}%` }}
                  />
                  {/* Pending portion */}
                  {stat.pending > 0 && (
                    <div
                      className="w-full absolute bottom-0 bg-white/10 rounded-t-lg"
                      style={{ height: '100%' }}
                    />
                  )}
                  {/* Completed overlay */}
                  <div
                    className={`w-full absolute bottom-0 rounded-t-lg ${getBarColor(stat.percentage)} transition-all duration-300 group-hover:opacity-80 shadow-lg`}
                    style={{ height: `${completionPercentage}%` }}
                  />
                </div>

                {/* Label */}
                <div className="mt-2 text-[10px] text-white/50 text-center truncate w-full group-hover:text-white/80 transition-colors">
                  {groupBy === 'day' ? new Date(stat.key).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
                    : groupBy === 'week' ? `S${stat.label.split(' ')[1]}`
                    : groupBy === 'month' ? new Date(stat.key + '-01').toLocaleDateString('pt-BR', { month: 'short' })
                    : stat.label.slice(0, 8)}
                </div>

                {/* Tooltip on hover */}
                <div className="opacity-0 group-hover:opacity-100 absolute -top-16 left-1/2 transform -translate-x-1/2 bg-black/90 text-white text-xs rounded-lg px-3 py-2 pointer-events-none transition-opacity duration-200 whitespace-nowrap z-50 shadow-xl">
                  <div className="font-semibold mb-1">{stat.label}</div>
                  <div>Total: {stat.total}</div>
                  <div className="text-green-400">Concluídos: {stat.completed}</div>
                  <div className="text-yellow-400">Pendentes: {stat.pending}</div>
                  <div className="font-semibold mt-1">Taxa: {stat.percentage}%</div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="mt-6 flex items-center justify-center gap-6 text-xs text-white/60">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-gradient-to-t from-green-500 to-emerald-600"></div>
          <span>100%</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-gradient-to-t from-blue-500 to-cyan-600"></div>
          <span>≥70%</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-gradient-to-t from-yellow-500 to-orange-500"></div>
          <span>≥40%</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded bg-gradient-to-t from-red-500 to-pink-600"></div>
          <span>&lt;40%</span>
        </div>
      </div>
    </div>
  );
}
