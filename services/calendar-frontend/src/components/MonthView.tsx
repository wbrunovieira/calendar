/**
 * Month View Component
 * Displays the calendar in month view with day cells and events
 */

import { calendars } from '@/data/calendars';
import { Event, Category } from '@/types/calendar';
import { DAYS_OF_WEEK_SHORT, DEFAULT_EVENT_TIME } from '@/constants/calendar';
import { getDaysInMonth, getFirstDayOfMonth } from '@/utils/calendar';

interface MonthViewProps {
  currentDate: Date;
  events: Event[];
  categories: Category[];
  selectedCalendars: string[];
  onTimeSlotClick: (date: string, time: string) => void;
  onEditClick: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
}

export default function MonthView({
  currentDate,
  events,
  categories,
  selectedCalendars,
  onTimeSlotClick,
  onEditClick,
  onDeleteClick,
}: MonthViewProps) {
  // Função para obter eventos de uma data específica
  const getEventsForDate = (date: Date): Event[] => {
    const dateString = date.toISOString().split('T')[0];

    return events.filter(event => {
      // Filtrar por calendários selecionados
      if (!selectedCalendars.includes(event.calendarId)) {
        return false;
      }

      // Comparar apenas a data (YYYY-MM-DD)
      const eventDate = event.startDate.split('T')[0];
      return eventDate === dateString;
    });
  };

  const renderMonthDays = () => {
    const daysInMonth = getDaysInMonth(currentDate);
    const firstDay = getFirstDayOfMonth(currentDate);
    const days = [];
    const today = new Date();
    const isCurrentMonth =
      currentDate.getMonth() === today.getMonth() && currentDate.getFullYear() === today.getFullYear();

    // Empty cells before first day of month
    for (let i = 0; i < firstDay; i++) {
      days.push(<div key={`empty-${i}`} className="aspect-square p-1" />);
    }

    // Days of the month
    for (let day = 1; day <= daysInMonth; day++) {
      const isToday = isCurrentMonth && day === today.getDate();
      const dayDate = new Date(currentDate.getFullYear(), currentDate.getMonth(), day);
      const dayEvents = getEventsForDate(dayDate);

      days.push(
        <div
          key={day}
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
          onClick={e => {
            // Only open modal if clicking on empty space (not on an event)
            const target = e.target as HTMLElement;
            if (
              target === e.currentTarget ||
              target.tagName === 'SPAN' ||
              target.classList.contains('flex-col') ||
              target.classList.contains('time-grid')
            ) {
              const dateString = dayDate.toISOString().split('T')[0];
              onTimeSlotClick(dateString, DEFAULT_EVENT_TIME);
            }
          }}
        >
          {/* Subtle time grid background with hour labels */}
          <div className="time-grid absolute inset-0 pointer-events-none">
            {[6, 10, 14, 18, 22].map((hour, i) => (
              <div
                key={i}
                className="absolute left-0 right-0 border-t border-white/10 flex items-center"
                style={{ top: `${(i + 1) * 16.66}%` }}
              >
                <span className="text-[8px] text-white/30 ml-0.5">{hour.toString().padStart(2, '0')}h</span>
              </div>
            ))}
          </div>

          <span className="text-xs md:text-sm mb-1 relative z-10">{day}</span>
          <div className="flex flex-col gap-0.5 w-full">
            {dayEvents.slice(0, 2).map(event => {
              const category = categories.find(c => c.id === event.categoryId);
              const calendar = calendars.find(c => c.id === event.calendarId);
              const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';
              return (
                <div
                  key={event.id}
                  className="text-[8px] md:text-[9px] rounded flex items-center overflow-hidden group relative"
                  style={{
                    backgroundColor: category?.color + '80',
                  }}
                  title={`${calendar?.name} - ${event.title} - ${event.startTime}`}
                >
                  <div
                    className="px-1 py-0.5 flex items-center justify-center text-[10px]"
                    style={{ backgroundColor: calendar?.color }}
                  >
                    {calendarIcon}
                  </div>
                  <div className="px-1 py-0.5 truncate flex-1">
                    {category?.icon} {event.title}
                  </div>
                  <div className="opacity-0 group-hover:opacity-100 transition-opacity absolute top-0 right-0 flex">
                    <button
                      onClick={e => onEditClick(event, e)}
                      className="p-0.5 hover:bg-blue-600 rounded"
                      title="Editar"
                    >
                      <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                        />
                      </svg>
                    </button>
                    <button
                      onClick={e => onDeleteClick(event, e)}
                      className="p-0.5 hover:bg-red-600 rounded"
                      title="Deletar"
                    >
                      <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  </div>
                </div>
              );
            })}
            {dayEvents.length > 2 && <div className="text-[8px] opacity-70">+{dayEvents.length - 2} mais</div>}
          </div>
        </div>
      );
    }

    return days;
  };

  return (
    <>
      {/* Days of week header */}
      <div className="grid grid-cols-7 gap-1 mb-3">
        {DAYS_OF_WEEK_SHORT.map((day, index) => (
          <div
            key={day}
            className={`
              text-center font-bold text-sm md:text-base py-1.5 rounded-lg
              transition-all duration-200
              ${
                index === 0 || index === 6
                  ? 'bg-gradient-to-r from-[#792990]/30 to-[#350545]/30 text-white/90 border border-[#792990]/40'
                  : 'bg-white/5 text-white/80 border border-white/10'
              }
              hover:bg-[#792990]/40 hover:border-[#792990]/60 hover:text-white
            `}
          >
            {day}
          </div>
        ))}
      </div>
      {/* Month grid */}
      <div className="grid grid-cols-7 gap-1">{renderMonthDays()}</div>
    </>
  );
}
