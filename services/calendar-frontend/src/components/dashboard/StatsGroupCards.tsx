/**
 * Stats Group Cards Component
 * Displays statistics grouped by different dimensions (category, type, period)
 */

import { StatsData } from '@/lib/api';

interface StatsGroupCardsProps {
  stats: StatsData[];
  title: string;
  icon?: string;
}

const getPercentageColor = (percentage: number) => {
  if (percentage === 100) return 'from-green-500/20 to-green-600/10 border-green-500/30';
  if (percentage >= 70) return 'from-blue-500/20 to-blue-600/10 border-blue-500/30';
  if (percentage >= 40) return 'from-yellow-500/20 to-yellow-600/10 border-yellow-500/30';
  return 'from-red-500/20 to-red-600/10 border-red-500/30';
};

const getPercentageBadgeColor = (percentage: number) => {
  if (percentage === 100) return 'bg-gradient-to-r from-green-500 to-emerald-600';
  if (percentage >= 70) return 'bg-gradient-to-r from-blue-500 to-cyan-600';
  if (percentage >= 40) return 'bg-gradient-to-r from-yellow-500 to-orange-500';
  return 'bg-gradient-to-r from-red-500 to-pink-600';
};

export default function StatsGroupCards({ stats, title, icon }: StatsGroupCardsProps) {
  if (stats.length === 0) {
    return null;
  }

  // Sort by total descending
  const sortedStats = [...stats].sort((a, b) => b.total - a.total);

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        {icon && <span className="text-2xl">{icon}</span>}
        <h2 className="text-xl font-semibold text-white">{title}</h2>
        <span className="text-sm text-white/50">({sortedStats.length})</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {sortedStats.map((stat) => (
          <div
            key={stat.key}
            className={`bg-gradient-to-br ${getPercentageColor(stat.percentage)} backdrop-blur-sm rounded-xl p-4 border shadow-lg transition-all duration-300 hover:scale-105 hover:shadow-xl`}
          >
            {/* Header */}
            <div className="flex items-start justify-between mb-3">
              <h3 className="font-semibold text-white text-sm truncate flex-1">
                {stat.label}
              </h3>
              {stat.percentage === 100 && stat.total > 0 && (
                <span className="text-lg ml-2">🌟</span>
              )}
            </div>

            {/* Main Stats */}
            <div className="space-y-2">
              {/* Total Events */}
              <div className="flex items-center justify-between">
                <span className="text-xs text-white/60">Total</span>
                <span className="text-lg font-bold text-white">{stat.total}</span>
              </div>

              {/* Progress Bar */}
              <div className="relative h-2 bg-white/10 rounded-full overflow-hidden">
                <div
                  className={`absolute left-0 top-0 h-full ${getPercentageBadgeColor(stat.percentage)} transition-all duration-500`}
                  style={{ width: `${stat.percentage}%` }}
                />
              </div>

              {/* Completed/Pending */}
              <div className="flex items-center justify-between text-xs">
                <span className="text-green-400">
                  ✓ {stat.completed}
                </span>
                {stat.pending > 0 && (
                  <span className="text-yellow-400">
                    ⏳ {stat.pending}
                  </span>
                )}
              </div>

              {/* Percentage Badge */}
              <div className="flex items-center justify-center pt-2">
                <div className={`px-3 py-1 rounded-full text-xs font-semibold text-white ${getPercentageBadgeColor(stat.percentage)} shadow-md`}>
                  {stat.percentage}%
                  {stat.percentage === 100 && ' ✓'}
                </div>
              </div>

              {/* Additional Metrics */}
              {(stat.streak || stat.averageCompletionTime) && (
                <div className="pt-2 border-t border-white/10 mt-2 flex items-center justify-between text-xs text-white/60">
                  {stat.streak && (
                    <span>🔥 {stat.streak} dias</span>
                  )}
                  {stat.averageCompletionTime && (
                    <span>⏱️ {Math.floor(stat.averageCompletionTime / 60)}h</span>
                  )}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
