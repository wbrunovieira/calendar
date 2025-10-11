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
  return (
    <div
      className={`text-center mb-2 pb-2 border-b border-white/10 ${
        isToday ? 'bg-gradient-to-r from-[#350545] to-[#792990] rounded-lg p-2' : ''
      }`}
    >
      {/* Day of week name */}
      {daysCount === 7 && daysOfWeek && <div className="text-xs opacity-70 text-white">{daysOfWeek[date.getDay()]}</div>}
      {daysCount > 1 && daysCount <= 3 && <div className="text-xs opacity-70 text-white">{daysOfWeekFull[date.getDay()]}</div>}
      {daysCount === 1 && <div className="text-sm opacity-70 text-white">{daysOfWeekFull[date.getDay()]}</div>}

      {/* Day number */}
      <div
        className={`font-bold text-white ${
          daysCount === 7 ? 'text-lg' : daysCount <= 3 ? 'text-2xl' : 'text-3xl'
        }`}
      >
        {date.getDate()}
      </div>

      {/* Month name (only for 3 days or less) */}
      {daysCount <= 3 && <div className="text-xs opacity-70 text-white">{monthNames[date.getMonth()]}</div>}
    </div>
  );
}
