/**
 * Days of Week Selector Component
 * Allows selection of multiple days of the week
 */

import { DAYS_OF_WEEK_OPTIONS } from '@/constants/calendar';

interface DaysOfWeekSelectorProps {
  selectedDays: number[];
  onToggleDay: (day: number) => void;
}

export default function DaysOfWeekSelector({ selectedDays, onToggleDay }: DaysOfWeekSelectorProps) {
  return (
    <div>
      <label className="block text-sm font-semibold text-white mb-2">Dias da Semana</label>
      <div className="flex gap-2 flex-wrap">
        {DAYS_OF_WEEK_OPTIONS.map(day => (
          <button
            key={day.value}
            type="button"
            onClick={() => onToggleDay(day.value)}
            className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              selectedDays.includes(day.value)
                ? 'bg-[#792990] text-white'
                : 'bg-white/10 text-white border border-white/20'
            }`}
          >
            {day.label}
          </button>
        ))}
      </div>
    </div>
  );
}
