/**
 * Time Slot Day Header Component
 * Displays the header for a day column in TimeSlotView
 */

interface TimeSlotDayHeaderProps {
  date: Date;
  isToday: boolean;
  daysOfWeek?: string[];
  daysOfWeekFull: string[];
  monthNames: string[];
  daysCount: number;
}

export default function TimeSlotDayHeader({
  date,
  isToday,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
  daysCount,
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
    </div>
  );
}
