/**
 * Stats Table Component
 * Displays statistics in a detailed table format
 */

import { StatsData } from '@/lib/api';

interface StatsTableProps {
  stats: StatsData[];
  groupBy: 'day' | 'week' | 'month' | 'category' | 'categoryType' | 'total';
}

export default function StatsTable({ stats, groupBy }: StatsTableProps) {
  // Sort stats by date/label
  const sortedStats = [...stats].sort((a, b) => {
    if (groupBy === 'day' || groupBy === 'week' || groupBy === 'month') {
      // Sort by key (date) descending (most recent first)
      return b.key.localeCompare(a.key);
    }
    // Sort by total descending (most events first)
    return b.total - a.total;
  });

  const getPercentageColor = (percentage: number) => {
    if (percentage === 100) return 'text-green-400';
    if (percentage >= 70) return 'text-blue-400';
    if (percentage >= 40) return 'text-yellow-400';
    return 'text-red-400';
  };

  const getPercentageBg = (percentage: number) => {
    if (percentage === 100) return 'bg-green-500/20 border-green-500/30';
    if (percentage >= 70) return 'bg-blue-500/20 border-blue-500/30';
    if (percentage >= 40) return 'bg-yellow-500/20 border-yellow-500/30';
    return 'bg-red-500/20 border-red-500/30';
  };

  return (
    <div className="bg-gradient-to-br from-white/10 to-white/5 backdrop-blur-sm rounded-xl border border-white/20 shadow-lg overflow-hidden">
      <div className="px-6 py-4 border-b border-white/10">
        <h2 className="text-lg font-semibold text-white">Detalhamento</h2>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-white/5">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-white/70 uppercase tracking-wider">
                {groupBy === 'category' ? 'Categoria' : groupBy === 'categoryType' ? 'Tipo' : 'Período'}
              </th>
              <th className="px-6 py-3 text-center text-xs font-medium text-white/70 uppercase tracking-wider">
                Total
              </th>
              <th className="px-6 py-3 text-center text-xs font-medium text-white/70 uppercase tracking-wider">
                Concluídos
              </th>
              <th className="px-6 py-3 text-center text-xs font-medium text-white/70 uppercase tracking-wider">
                Pendentes
              </th>
              <th className="px-6 py-3 text-center text-xs font-medium text-white/70 uppercase tracking-wider">
                Taxa
              </th>
              {sortedStats.some(s => s.streak) && (
                <th className="px-6 py-3 text-center text-xs font-medium text-white/70 uppercase tracking-wider">
                  Sequência
                </th>
              )}
            </tr>
          </thead>
          <tbody className="divide-y divide-white/10">
            {sortedStats.map((stat) => (
              <tr key={stat.key} className="hover:bg-white/5 transition-colors">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-white">
                  {stat.label}
                  {stat.percentage === 100 && stat.total > 0 && (
                    <span className="ml-2">🌟</span>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-center text-white/80">
                  {stat.total}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-center text-green-400">
                  {stat.completed}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-center text-yellow-400">
                  {stat.pending}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-center">
                  <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold border ${getPercentageBg(stat.percentage)}`}>
                    <span className={getPercentageColor(stat.percentage)}>
                      {stat.percentage}%
                    </span>
                    {stat.percentage === 100 && <span className="ml-1">✓</span>}
                  </span>
                </td>
                {sortedStats.some(s => s.streak) && (
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-center text-orange-400">
                    {stat.streak ? `${stat.streak} 🔥` : '-'}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
          {sortedStats.length === 0 && (
            <tbody>
              <tr>
                <td colSpan={6} className="px-6 py-12 text-center text-white/40">
                  Nenhum dado disponível
                </td>
              </tr>
            </tbody>
          )}
        </table>
      </div>
    </div>
  );
}
