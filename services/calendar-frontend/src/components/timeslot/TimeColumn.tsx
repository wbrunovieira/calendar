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
      {/* Spacer to align with day headers - must match TimeSlotDayHeader height */}
      <div className={`text-center ${isSingleDay ? 'mb-3 pb-3 p-4' : 'mb-2 pb-2 p-2'} border-b border-white/10 opacity-0`}>
        {/* Today badge spacer */}
        {days.length === 1 && <div className="mb-2 h-6"></div>}

        {/* Day of week name */}
        {days.length === 7 && daysOfWeek && <div className="text-xs opacity-70">X</div>}
        {days.length > 1 && days.length <= 3 && <div className="text-sm">X</div>}
        {days.length === 1 && <div className="text-lg mb-1">XXXXXXXX</div>}

        {/* Day number */}
        <div className={`font-bold ${days.length === 7 ? 'text-2xl' : days.length <= 3 ? 'text-3xl' : 'text-5xl my-2'}`}>
          00
        </div>

        {/* Month and year */}
        {days.length <= 3 && <div className={`${isSingleDay ? 'text-base' : 'text-xs'}`}>XXXXXXXXX</div>}
      </div>

      {/* Hour labels - positioned to align with grid lines */}
      {hours.map((hour) => (
        <div
          key={hour}
          className={`h-24 relative ${
            isSingleDay
              ? 'text-sm text-white/80 font-medium'
              : 'text-xs text-white/70'
          }`}
        >
          <div className="absolute -top-2 right-0 pr-3">
            {hour.toString().padStart(2, '0')}:00
          </div>
        </div>
      ))}
    </div>
  );
}
