/**
 * Calendar Footer Component
 * Displays calendar selector and legend information
 */

import { calendars } from '@/data/calendars';

interface CalendarFooterProps {
  selectedCalendars: string[];
  onToggleCalendar: (calendarId: string) => void;
}

export default function CalendarFooter({ selectedCalendars, onToggleCalendar }: CalendarFooterProps) {
  return (
    <>
      {/* Calendar Selector */}
      <div className="px-3 py-2 border-t" style={{ backgroundColor: '#350545', borderColor: '#792990' }}>
        <div className="flex items-center justify-center gap-3 flex-wrap">
          {calendars.map(calendar => (
            <button
              key={calendar.id}
              onClick={() => onToggleCalendar(calendar.id)}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                selectedCalendars.includes(calendar.id)
                  ? 'bg-white/20 text-white'
                  : 'bg-white/5 text-white/50 hover:bg-white/10'
              }`}
            >
              <div className="w-3 h-3 rounded-full" style={{ backgroundColor: calendar.color }} />
              <span>{calendar.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Footer Info */}
      <div className="px-3 py-2 border-t" style={{ backgroundColor: '#350545', borderColor: '#792990' }}>
        <div className="flex items-center justify-center gap-4 text-xs text-white flex-wrap">
          <div className="flex items-center gap-1">
            <div
              className="w-3 h-3 rounded"
              style={{ background: 'linear-gradient(135deg, #792990 0%, #ffffff 100%)' }}
            ></div>
            <span>Hoje</span>
          </div>
          <div className="h-3 w-px bg-white/20"></div>
          <div className="flex items-center gap-1.5">
            <span>💼</span>
            <div className="w-3 h-3 rounded" style={{ backgroundColor: '#350545' }}></div>
            <span>WB Digital</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span>👤</span>
            <div className="w-3 h-3 rounded" style={{ backgroundColor: '#792990' }}></div>
            <span>Pessoal</span>
          </div>
        </div>
      </div>
    </>
  );
}
