/**
 * Time Slot Day Header Component
 * Displays the header for a day column in TimeSlotView
 */

interface DayStats {
  total: number;
  completed: number;
  percentage: number;
}

interface TimeSlotDayHeaderProps {
  date: Date;
  isToday: boolean;
  daysOfWeek?: string[];
  daysOfWeekFull: string[];
  monthNames: string[];
  daysCount: number;
  stats?: DayStats;
}

export default function TimeSlotDayHeader({
  date,
  isToday,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
  daysCount,
  stats,
}: TimeSlotDayHeaderProps) {
  // Para visualização de 1 dia, vamos criar um header mais rico
  const isSingleDay = daysCount === 1;

  return (
    <div
      className={`text-center mb-3 pb-3 border-b transition-all duration-300 ${
        isToday
          ? 'bg-gradient-to-br from-[#350545] via-[#4a0860] to-[#792990] rounded-xl p-4 shadow-lg shadow-purple-900/50 border-purple-600/30'
          : 'border-white/10 p-2'
      }`}
    >
      {/* Today badge for single day view */}
      {isSingleDay && isToday && (
        <div className="mb-2">
          <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-white/20 backdrop-blur-sm rounded-full text-xs font-semibold text-white shadow-md">
            <span className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></span>
            Hoje
          </span>
        </div>
      )}

      {/* Day of week name */}
      {daysCount === 7 && daysOfWeek && (
        <div className="text-xs opacity-70 text-white uppercase tracking-wider">
          {daysOfWeek[date.getDay()]}
        </div>
      )}
      {daysCount > 1 && daysCount <= 3 && (
        <div className="text-sm opacity-80 text-white font-medium uppercase tracking-wide">
          {daysOfWeekFull[date.getDay()]}
        </div>
      )}
      {daysCount === 1 && (
        <div className="text-lg opacity-90 text-white font-semibold uppercase tracking-wider mb-1">
          {daysOfWeekFull[date.getDay()]}
        </div>
      )}

      {/* Day number */}
      <div
        className={`font-extrabold text-white ${
          daysCount === 7
            ? 'text-2xl'
            : daysCount <= 3
            ? 'text-3xl'
            : 'text-5xl my-2'
        } ${isSingleDay ? 'drop-shadow-lg' : ''}`}
      >
        {date.getDate()}
      </div>

      {/* Month and year */}
      {daysCount <= 3 && (
        <div className={`opacity-80 text-white ${isSingleDay ? 'text-base font-medium' : 'text-xs'}`}>
          {monthNames[date.getMonth()]}
          {isSingleDay && ` de ${date.getFullYear()}`}
        </div>
      )}

      {/* Stats Badge */}
      {stats && stats.total > 0 && (
        <div className="mt-2">
          <div
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-white font-semibold shadow-md transition-all duration-300 ${
              stats.percentage === 100
                ? 'bg-gradient-to-r from-green-500 to-emerald-600 shadow-green-500/30'
                : stats.percentage >= 70
                ? 'bg-gradient-to-r from-blue-500 to-cyan-600 shadow-blue-500/30'
                : stats.percentage >= 40
                ? 'bg-gradient-to-r from-yellow-500 to-orange-500 shadow-yellow-500/30'
                : 'bg-gradient-to-r from-red-500 to-pink-600 shadow-red-500/30'
            } ${
              daysCount === 7 ? 'text-[10px]' : daysCount === 3 ? 'text-xs' : 'text-sm'
            }`}
          >
            <span className="font-bold">{stats.completed}/{stats.total}</span>
            <span className="opacity-75">·</span>
            <span>{stats.percentage}%</span>
            {stats.percentage === 100 && <span>✓</span>}
          </div>
        </div>
      )}
    </div>
  );
}
