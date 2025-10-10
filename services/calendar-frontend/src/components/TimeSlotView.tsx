'use client';

import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { api } from '@/lib/api';

interface TimeSlotViewProps {
  days: Date[];
  events: Event[];
  categories: Category[];
  selectedCalendars: string[];
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
  onEventUpdate?: () => void;
  daysOfWeek?: string[];
  daysOfWeekFull: string[];
  monthNames: string[];
}

export function TimeSlotView({
  days,
  events,
  categories,
  selectedCalendars,
  onDeleteClick,
  onEventUpdate,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
}: TimeSlotViewProps) {
  const hours = Array.from({ length: 17 }, (_, i) => i + 6); // 6am to 10pm
  const today = new Date();
  const { handleDragStart, handleDragEnd, handleDragOver, handleDrop } = useDragAndDrop();

  const handleEventUpdate = async (eventId: string, newDate: string, newTime?: string) => {
    try {
      const event = events.find(e => e.id === eventId);
      if (!event) return;

      await api.events.update(eventId, {
        startDate: newDate,
        ...(newTime && { startTime: newTime }),
      });

      if (onEventUpdate) {
        onEventUpdate();
      }
    } catch (error) {
      console.error('Error updating event:', error);
    }
  };

  const getEventsForDate = (date: Date): Event[] => {
    const dateString = date.toISOString().split('T')[0];

    return events.filter(event => {
      if (!selectedCalendars.includes(event.calendarId)) {
        return false;
      }

      if (!event.isRecurring) {
        const eventDate = event.startDate.split('T')[0];
        return eventDate === dateString;
      }

      if (event.recurrenceFrequency === 'daily') {
        return true;
      }

      if (event.recurrenceFrequency === 'weekly' && event.recurrenceDaysOfWeek) {
        const dayOfWeek = date.getDay();
        return event.recurrenceDaysOfWeek.includes(dayOfWeek);
      }

      return false;
    });
  };

  return (
    <div className="p-4 w-full">
      <div className="flex gap-2 max-h-[600px] overflow-y-auto w-full">
        {/* Time column */}
        <div className="w-16 flex-shrink-0">
          {/* Spacer to align with day headers */}
          <div className="text-center mb-2 pb-2 border-b border-white/10 opacity-0">
            {days.length === 7 && daysOfWeek && (
              <div className="text-xs opacity-70">X</div>
            )}
            {days.length <= 3 && (
              <div className="text-xs opacity-70">X</div>
            )}
            {days.length === 1 && (
              <div className="text-sm opacity-70">X</div>
            )}
            <div className={`font-bold ${days.length === 7 ? 'text-lg' : days.length <= 3 ? 'text-2xl' : 'text-3xl'}`}>
              00
            </div>
            {days.length <= 3 && (
              <div className="text-xs opacity-70">X</div>
            )}
          </div>
          {hours.map((hour) => (
            <div key={hour} className="h-16 flex items-start justify-end pr-2 pt-1 text-xs text-white/70">
              {hour.toString().padStart(2, '0')}:00
            </div>
          ))}
        </div>

        {/* Day columns */}
        {days.map((date, dayIndex) => {
          const isToday = date.toDateString() === today.toDateString();
          const dayEvents = getEventsForDate(date);

          return (
            <div key={dayIndex} className="flex-1 min-w-0 flex flex-col">
              {/* Day header */}
              <div className={`text-center mb-2 pb-2 border-b border-white/10 ${isToday ? 'bg-gradient-to-r from-[#350545] to-[#792990] rounded-lg p-2' : ''}`}>
                {days.length === 7 && daysOfWeek && (
                  <div className="text-xs opacity-70 text-white">{daysOfWeek[date.getDay()]}</div>
                )}
                {days.length <= 3 && (
                  <div className="text-xs opacity-70 text-white">{daysOfWeekFull[date.getDay()]}</div>
                )}
                {days.length === 1 && (
                  <div className="text-sm opacity-70 text-white">{daysOfWeekFull[date.getDay()]}</div>
                )}
                <div className={`font-bold text-white ${days.length === 7 ? 'text-lg' : days.length <= 3 ? 'text-2xl' : 'text-3xl'}`}>
                  {date.getDate()}
                </div>
                {days.length <= 3 && (
                  <div className="text-xs opacity-70 text-white">{monthNames[date.getMonth()]}</div>
                )}
              </div>

              {/* Time grid for this day */}
              <div className="flex-1 relative border-l border-white/10 min-h-[1088px]">
                {/* Time slot grid */}
                {hours.map((hour) => (
                  <div
                    key={hour}
                    className="h-16 border-b border-white/5 relative"
                    onDragOver={handleDragOver}
                    onDrop={(e) => {
                      // Calculate exact time based on mouse position within the slot
                      const rect = e.currentTarget.getBoundingClientRect();
                      const offsetY = e.clientY - rect.top;
                      const minutes = Math.floor((offsetY / 64) * 60);
                      const time = `${hour.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
                      handleDrop(date, time)(e, handleEventUpdate);
                    }}
                  >
                    <div className="h-8 border-b border-dashed border-white/5" />
                    <div className="h-8" />
                  </div>
                ))}

                {/* Events positioned absolutely */}
                {dayEvents.map((event) => {
                  const category = categories.find(c => c.id === event.categoryId);
                  const calendar = calendars.find(c => c.id === event.calendarId);
                  const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

                  // Calculate position based on time (offset by 6 hours since we start at 6am)
                  let topPosition = 0;
                  try {
                    const timeParts = event.startTime.split(':');
                    const eventHours = parseInt(timeParts[0], 10);
                    const minutes = parseInt(timeParts[1] || '0', 10);
                    topPosition = ((eventHours - 6) * 64) + (minutes / 60 * 64); // 64px per hour (h-16), offset by 6
                  } catch (error) {
                    console.error('Error parsing time for event:', event, error);
                  }

                  const textSize = days.length === 7 ? 'text-[8px]' : days.length <= 3 ? 'text-[10px]' : 'text-xs';
                  const iconSize = days.length === 7 ? 'text-[8px]' : days.length <= 3 ? 'text-xs' : 'text-sm';
                  const padding = days.length === 7 ? 'px-0.5 py-0.5' : days.length <= 3 ? 'px-2 py-1' : 'px-2 py-1';

                  return (
                    <div
                      key={event.id}
                      draggable
                      onDragStart={handleDragStart(event)}
                      onDragEnd={handleDragEnd}
                      className={`absolute rounded-lg text-white flex overflow-hidden group cursor-move ${days.length === 7 ? 'left-0.5 right-0.5' : 'left-2 right-2'}`}
                      style={{
                        backgroundColor: category?.color + '90',
                        top: `${topPosition}px`,
                        minHeight: '48px',
                        zIndex: 10,
                      }}
                    >
                      <div
                        className={`${padding} flex items-center justify-center ${iconSize}`}
                        style={{ backgroundColor: calendar?.color }}
                      >
                        {calendarIcon}
                      </div>
                      <div className={`flex-1 ${padding}`}>
                        <div className={`font-semibold ${textSize} flex items-center gap-1`}>
                          <span>{category?.icon}</span>
                          <span className="truncate">{event.startTime}</span>
                        </div>
                        <div className={`${textSize} truncate`}>{event.title}</div>
                      </div>
                      <button
                        onClick={(e) => onDeleteClick(event, e)}
                        className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-600 rounded"
                        title="Deletar"
                      >
                        <svg className={days.length === 7 ? "w-2 h-2" : "w-3 h-3"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  );
                })}

                {/* Show message if no events for this day */}
                {dayEvents.length === 0 && (
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className={`text-white/30 ${days.length === 7 ? 'text-[8px]' : 'text-xs'}`}>
                      {days.length === 7 ? 'Vazio' : 'Sem eventos'}
                    </div>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
