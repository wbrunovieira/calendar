'use client';

import { Event, Category } from '@/types/calendar';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { useTimeSlotEventUpdate } from '@/hooks/useTimeSlotEventUpdate';
import { useEventResize } from '@/hooks/useEventResize';
import { useEventExecution } from '@/hooks/useEventExecution';
import { HOURS_ARRAY } from '@/constants/timeSlotView';
import { calculateTimeFromOffset } from '@/utils/timeCalculations';
import RecurringEventActionModal from '../modals/RecurringEventActionModal';
import TimeColumn from '../timeslot/TimeColumn';
import TimeSlotDayColumn from '../timeslot/TimeSlotDayColumn';

interface TimeSlotViewProps {
  days: Date[];
  events: Event[];
  categories: Category[];
  selectedCalendars: string[];
  onEditClick?: (event: Event, e: React.MouseEvent) => void;
  onDeleteClick: (event: Event, e: React.MouseEvent) => void;
  onEventUpdate?: () => void;
  onTimeSlotClick?: (date: string, time: string) => void;
  daysOfWeek?: string[];
  daysOfWeekFull: string[];
  monthNames: string[];
}

export function TimeSlotView({
  days,
  events,
  categories,
  selectedCalendars,
  onEditClick,
  onDeleteClick,
  onEventUpdate,
  onTimeSlotClick,
  daysOfWeek,
  daysOfWeekFull,
  monthNames,
}: TimeSlotViewProps) {
  const hours = HOURS_ARRAY;
  const today = new Date();
  const { handleDragStart, handleDragEnd, handleDragOver, handleDrop } = useDragAndDrop();

  // Custom hooks for business logic
  const {
    handleEventUpdate,
    handleRecurringActionSelect,
    handleRecurringActionClose,
    showRecurringActionModal,
    setDragContext,
    pendingUpdate,
  } = useTimeSlotEventUpdate({ events, onEventUpdate });

  const { handleResizeStart, isResizing, justResized, resizingEvent } = useEventResize({
    events,
    onEventUpdate: handleEventUpdate,
  });

  const { handleToggleExecution } = useEventExecution({ onEventUpdate });

  const getEventsForDate = (date: Date): Event[] => {
    const dateString = date.toISOString().split('T')[0];

    return events.filter(event => {
      if (!selectedCalendars.includes(event.calendarId)) {
        return false;
      }

      // Backend já expande eventos recorrentes, então apenas comparamos a data
      // Se o evento tem occurrenceDate, significa que já foi expandido pelo backend
      const eventDate = event.startDate.split('T')[0];
      return eventDate === dateString;
    });
  };

  const isSingleDay = days.length === 1;

  return (
    <>
      <RecurringEventActionModal
        isOpen={showRecurringActionModal}
        onClose={handleRecurringActionClose}
        onSelect={handleRecurringActionSelect}
        eventTitle={pendingUpdate?.eventTitle || ''}
      />
      <div className={`p-4 w-full ${isSingleDay ? 'max-w-5xl mx-auto' : ''}`}>
        <div className={`flex ${isSingleDay ? 'gap-4' : 'gap-3'} max-h-[900px] overflow-y-auto w-full ${isSingleDay ? 'bg-gradient-to-br from-white/5 to-white/0 rounded-2xl p-4 backdrop-blur-sm shadow-2xl' : ''}`}>
          {/* Time column */}
          <TimeColumn hours={hours} days={days} daysOfWeek={daysOfWeek} />

          {/* Day columns */}
          {days.map((date, dayIndex) => {
            const isToday = date.toDateString() === today.toDateString();
            const dayEvents = getEventsForDate(date);

            return (
              <TimeSlotDayColumn
                key={dayIndex}
                date={date}
                dayIndex={dayIndex}
                isToday={isToday}
                dayEvents={dayEvents}
                categories={categories}
                hours={hours}
                daysCount={days.length}
                daysOfWeek={daysOfWeek}
                daysOfWeekFull={daysOfWeekFull}
                monthNames={monthNames}
                resizingEventId={resizingEvent?.id}
                isResizing={isResizing}
                justResized={justResized}
                onDragOver={handleDragOver}
                onDrop={e => {
                  const container = e.currentTarget;
                  const rect = container.getBoundingClientRect();
                  const offsetY = e.clientY - rect.top;
                  const time = calculateTimeFromOffset(offsetY);
                  handleDrop(date, time)(e, handleEventUpdate);
                }}
                onTimeSlotClick={onTimeSlotClick}
                onEventDragStart={(event, executionDate, e) => {
                  setDragContext({
                    eventId: event.id,
                    originalDate: executionDate,
                  });
                  handleDragStart(event)(e);
                }}
                onEventDragEnd={handleDragEnd}
                onEventResizeStart={handleResizeStart}
                onEventToggleExecution={(eventId, date, isCompleted, e) => {
                  e.stopPropagation();
                  handleToggleExecution(eventId, date, isCompleted);
                }}
                onEventEdit={onEditClick}
                onEventDelete={onDeleteClick}
              />
            );
          })}
        </div>
      </div>
    </>
  );
}
