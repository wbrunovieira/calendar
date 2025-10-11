/**
 * Time Grid Background Component
 * Displays subtle time markers in the background of day cells
 */

import { HOUR_MARKS, HOUR_MARK_SPACING_PERCENT } from '@/constants/monthView';

export default function TimeGridBackground() {
  return (
    <div className="time-grid absolute inset-0 pointer-events-none">
      {HOUR_MARKS.map((hour, i) => (
        <div
          key={i}
          className="absolute left-0 right-0 border-t border-white/10 flex items-center"
          style={{ top: `${(i + 1) * HOUR_MARK_SPACING_PERCENT}%` }}
        >
          <span className="text-[8px] text-white/30 ml-0.5">{hour.toString().padStart(2, '0')}h</span>
        </div>
      ))}
    </div>
  );
}
