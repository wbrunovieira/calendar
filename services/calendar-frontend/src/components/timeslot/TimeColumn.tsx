/**
 * Time Column Component
 * Displays the time labels column for TimeSlotView
 */

interface TimeColumnProps {
  hours: number[];
  days: Date[];
  daysOfWeek?: string[];
}

export default function TimeColumn({ hours, days, daysOfWeek }: TimeColumnProps) {
  const isSingleDay = days.length === 1;

  return (
    <div className={`${isSingleDay ? 'w-24' : 'w-20'} flex-shrink-0`}>
      {/* Spacer to align with day headers */}
      <div className={`text-center ${isSingleDay ? 'mb-3 pb-3' : 'mb-2 pb-2'} border-b border-white/10 opacity-0`}>
        {days.length === 7 && daysOfWeek && <div className="text-xs opacity-70">X</div>}
        {days.length <= 3 && <div className="text-xs opacity-70">X</div>}
        {days.length === 1 && <div className="text-sm opacity-70">X</div>}
        <div className={`font-bold ${days.length === 7 ? 'text-lg' : days.length <= 3 ? 'text-2xl' : 'text-3xl'}`}>
          00
        </div>
        {days.length <= 3 && <div className="text-xs opacity-70">X</div>}
      </div>

      {/* Hour labels */}
      {hours.map(hour => (
        <div
          key={hour}
          className={`h-24 flex items-start justify-end pr-3 pt-1 ${
            isSingleDay
              ? 'text-sm text-white/80 font-medium'
              : 'text-xs text-white/70'
          }`}
        >
          {hour.toString().padStart(2, '0')}:00
        </div>
      ))}
    </div>
  );
}
