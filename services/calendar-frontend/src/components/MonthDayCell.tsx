/**
 * Month Day Cell Component
 * Displays a single day cell in month view with events
 */

import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { DEFAULT_EVENT_TIME } from '@/constants/calendar';
import { MAX_VISIBLE_EVENTS } from '@/constants/monthView';
import TimeGridBackground from './TimeGridBackground';
import MonthEventCard from './MonthEventCard';

interface MonthDayCellProps {
  day: number;
  date: Date;
  isToday: boolean;
  events: Event[];
  categories: Category[];
  onTimeSlotClick: (date: string, time: string) => void;
  onEditClick: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
}

export default function MonthDayCell({
  day,
  date,
  isToday,
  events,
  categories,
  onTimeSlotClick,
  onEditClick,
  onDeleteClick,
}: MonthDayCellProps) {
  const visibleEvents = events.slice(0, MAX_VISIBLE_EVENTS);
  const remainingCount = events.length - MAX_VISIBLE_EVENTS;

  const handleCellClick = (e: React.MouseEvent) => {
    // Only open modal if clicking on empty space (not on an event)
    const target = e.target as HTMLElement;
    if (
      target === e.currentTarget ||
      target.tagName === 'SPAN' ||
      target.classList.contains('flex-col') ||
      target.classList.contains('time-grid')
    ) {
      const dateString = date.toISOString().split('T')[0];
      onTimeSlotClick(dateString, DEFAULT_EVENT_TIME);
    }
  };

  return (
    <div
      className={`
        aspect-square p-2 flex flex-col items-start justify-start
        cursor-pointer transition-all duration-200
        border border-white/10
        hover:border-[#792990]/50 hover:shadow-lg hover:shadow-[#792990]/20
        relative overflow-hidden
        ${isToday ? 'text-white font-bold border-[#792990] shadow-xl shadow-[#792990]/30' : 'text-white'}
      `}
      style={
        isToday
          ? { background: 'linear-gradient(135deg, #350545 0%, #792990 100%)' }
          : { backgroundColor: 'rgba(255, 255, 255, 0.02)' }
      }
      onClick={handleCellClick}
    >
      <TimeGridBackground />

      <span className="text-xs md:text-sm mb-1 relative z-10">{day}</span>

      <div className="flex flex-col gap-0.5 w-full">
        {visibleEvents.map(event => {
          const category = categories.find(c => c.id === event.categoryId);
          const calendar = calendars.find(c => c.id === event.calendarId);
          const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

          return (
            <MonthEventCard
              key={event.id}
              event={event}
              category={category}
              calendarColor={calendar?.color || '#999'}
              calendarName={calendar?.name || 'Unknown'}
              calendarIcon={calendarIcon}
              onEditClick={onEditClick}
              onDeleteClick={onDeleteClick}
            />
          );
        })}

        {remainingCount > 0 && <div className="text-[8px] opacity-70">+{remainingCount} mais</div>}
      </div>
    </div>
  );
}
