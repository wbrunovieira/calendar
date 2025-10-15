/**
 * Time Slot Summary Stats Component
 * Displays aggregated statistics for the visible days
 */

import { calendars } from '@/data/calendars';

interface CalendarStat {
  calendarId: string;
  total: number;
  completed: number;
  percentage: number;
}

interface SummaryStats {
  totalEvents: number;
  completedEvents: number;
  percentage: number;
  daysCount: number;
  perfectDays: number; // Days with 100% completion
  byCalendar?: CalendarStat[];
}

interface TimeSlotSummaryStatsProps {
  stats: SummaryStats;
}

export default function TimeSlotSummaryStats({ stats }: TimeSlotSummaryStatsProps) {
  if (stats.totalEvents === 0) {
    return null;
  }

  return (
    <div className="mb-4 px-4 relative z-10">
      <div className="bg-gradient-to-br from-white/10 to-white/5 backdrop-blur-sm rounded-xl p-4 border border-white/20 shadow-lg">
        <div className="flex items-center justify-between gap-4 flex-wrap">
          {/* Title */}
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-white/70 uppercase tracking-wide">
              Resumo do Período
            </span>
            <span className="text-xs text-white/50">
              ({stats.daysCount} {stats.daysCount === 1 ? 'dia' : 'dias'})
            </span>
          </div>

          {/* Stats */}
          <div className="flex items-center gap-6 flex-wrap">
            {/* Total Events */}
            <div className="flex flex-col items-center">
              <span className="text-2xl font-bold text-white">
                {stats.totalEvents}
              </span>
              <span className="text-xs text-white/60">
                {stats.totalEvents === 1 ? 'evento' : 'eventos'}
              </span>
            </div>

            {/* Divider */}
            <div className="h-8 w-px bg-white/20" />

            {/* Completed */}
            <div className="flex flex-col items-center">
              <span className="text-2xl font-bold text-emerald-400">
                {stats.completedEvents}
              </span>
              <span className="text-xs text-white/60">
                {stats.completedEvents === 1 ? 'concluído' : 'concluídos'}
              </span>
            </div>

            {/* Divider */}
            <div className="h-8 w-px bg-white/20" />

            {/* Percentage Badge */}
            <div className={`px-4 py-2 rounded-full font-bold text-lg shadow-lg transition-all duration-300 ${
              stats.percentage === 100
                ? 'bg-gradient-to-r from-green-500 to-emerald-600 text-white shadow-green-500/30'
                : stats.percentage >= 70
                ? 'bg-gradient-to-r from-blue-500 to-cyan-600 text-white shadow-blue-500/30'
                : stats.percentage >= 40
                ? 'bg-gradient-to-r from-yellow-500 to-orange-500 text-white shadow-yellow-500/30'
                : 'bg-gradient-to-r from-red-500 to-pink-600 text-white shadow-red-500/30'
            }`}>
              {stats.percentage}%
              {stats.percentage === 100 && ' ✓'}
            </div>

            {/* Perfect Days (if any) */}
            {stats.perfectDays > 0 && (
              <>
                <div className="h-8 w-px bg-white/20" />
                <div className="flex flex-col items-center">
                  <span className="text-2xl font-bold text-yellow-400">
                    {stats.perfectDays}
                  </span>
                  <span className="text-xs text-white/60">
                    {stats.perfectDays === 1 ? 'dia' : 'dias'} perfeito{stats.perfectDays === 1 ? '' : 's'} 🌟
                  </span>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Stats by Calendar */}
        {stats.byCalendar && stats.byCalendar.length > 0 && (
          <div className="mt-4 pt-4 border-t border-white/20">
            <div className="flex items-center gap-6 flex-wrap justify-center">
              {stats.byCalendar.map(calendarStat => {
                const calendar = calendars.find(c => c.id === calendarStat.calendarId);
                const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
                const calendarName = calendar?.name || 'Desconhecido';

                return (
                  <div key={calendarStat.calendarId} className="flex items-center gap-3 bg-white/5 rounded-lg px-4 py-2">
                    {/* Calendar Icon */}
                    <div
                      className="w-10 h-10 rounded-full flex items-center justify-center text-xl"
                      style={{ backgroundColor: calendar?.color }}
                    >
                      {calendarIcon}
                    </div>

                    {/* Calendar Stats */}
                    <div className="flex flex-col">
                      <span className="text-xs text-white/70 font-semibold">{calendarName}</span>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="text-sm text-white font-bold">{calendarStat.completed}/{calendarStat.total}</span>
                        <span className={`text-xs px-2 py-0.5 rounded-full font-semibold ${
                          calendarStat.percentage === 100
                            ? 'bg-green-500/20 text-green-300'
                            : calendarStat.percentage >= 50
                            ? 'bg-blue-500/20 text-blue-300'
                            : 'bg-orange-500/20 text-orange-300'
                        }`}>
                          {calendarStat.percentage}%
                        </span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
