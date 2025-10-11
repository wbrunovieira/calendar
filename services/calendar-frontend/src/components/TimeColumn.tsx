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
  return (
    <div className="w-20 flex-shrink-0">
      {/* Spacer to align with day headers */}
      <div className="text-center mb-2 pb-2 border-b border-white/10 opacity-0">
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
        <div key={hour} className="h-24 flex items-start justify-end pr-2 pt-1 text-xs text-white/70">
          {hour.toString().padStart(2, '0')}:00
        </div>
      ))}
    </div>
  );
}
