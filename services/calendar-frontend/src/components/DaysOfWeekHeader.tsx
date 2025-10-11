/**
 * Days of Week Header Component
 * Displays the days of the week header for calendar views
 */

import { DAYS_OF_WEEK_SHORT } from '@/constants/calendar';

export default function DaysOfWeekHeader() {
  return (
    <div className="grid grid-cols-7 gap-1 mb-3">
      {DAYS_OF_WEEK_SHORT.map((day, index) => {
        const isWeekend = index === 0 || index === 6;
        return (
          <div
            key={day}
            className={`
              text-center font-bold text-sm md:text-base py-1.5 rounded-lg
              transition-all duration-200
              ${
                isWeekend
                  ? 'bg-gradient-to-r from-[#792990]/30 to-[#350545]/30 text-white/90 border border-[#792990]/40'
                  : 'bg-white/5 text-white/80 border border-white/10'
              }
              hover:bg-[#792990]/40 hover:border-[#792990]/60 hover:text-white
            `}
          >
            {day}
          </div>
        );
      })}
    </div>
  );
}
