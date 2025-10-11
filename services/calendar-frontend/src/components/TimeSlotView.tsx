'use client';

import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { useTimeSlotEventUpdate } from '@/hooks/useTimeSlotEventUpdate';
import { useEventResize } from '@/hooks/useEventResize';
import { useEventExecution } from '@/hooks/useEventExecution';
import { HOURS_ARRAY } from '@/constants/timeSlotView';
import { calculateTimeFromOffset } from '@/utils/timeCalculations';
import RecurringEventActionModal from './RecurringEventActionModal';
import TimeColumn from './TimeColumn';
import TimeSlotDayHeader from './TimeSlotDayHeader';
import TimeSlotGrid from './TimeSlotGrid';
import EmptyDayMessage from './EmptyDayMessage';
import TimeSlotEventCard from './TimeSlotEventCard';

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


  return (
    <>
      <RecurringEventActionModal
        isOpen={showRecurringActionModal}
        onClose={handleRecurringActionClose}
        onSelect={handleRecurringActionSelect}
        eventTitle={pendingUpdate?.eventTitle || ''}
      />
      <div className="p-4 w-full">
        <div className="flex gap-3 max-h-[900px] overflow-y-auto w-full">
        {/* Time column */}
        <TimeColumn hours={hours} days={days} daysOfWeek={daysOfWeek} />

        {/* Day columns */}
        {days.map((date, dayIndex) => {
          const isToday = date.toDateString() === today.toDateString();
          const dayEvents = getEventsForDate(date);

          // Calculate event positions for this day
          const eventsWithTimes = dayEvents.map(event => {
            const startTotalMinutes = event.startTime.split(':').map(Number).reduce((h, m) => h * 60 + m);
            const endTotalMinutes = event.endTime ? event.endTime.split(':').map(Number).reduce((h, m) => h * 60 + m) : startTotalMinutes + 60;
            return {
              event,
              start: startTotalMinutes,
              end: endTotalMinutes,
              column: 0,
              totalColumns: 1,
            };
          });

          eventsWithTimes.sort((a, b) => a.start !== b.start ? a.start - b.start : (b.end - b.start) - (a.end - a.start));

          const columns: Array<Array<typeof eventsWithTimes[0]>> = [];
          eventsWithTimes.forEach(eventTime => {
            let placedInColumn = false;
            for (let i = 0; i < columns.length; i++) {
              const column = columns[i];
              const hasOverlap = column.some(existing => !(eventTime.end <= existing.start || eventTime.start >= existing.end));
              if (!hasOverlap) {
                column.push(eventTime);
                eventTime.column = i;
                placedInColumn = true;
                break;
              }
            }
            if (!placedInColumn) {
              columns.push([eventTime]);
              eventTime.column = columns.length - 1;
            }
          });

          eventsWithTimes.forEach(eventTime => {
            const overlappingEvents = eventsWithTimes.filter(other => !(eventTime.end <= other.start || eventTime.start >= other.end));
            const maxColumn = Math.max(...overlappingEvents.map(e => e.column));
            eventTime.totalColumns = maxColumn + 1;
          });

          const eventPositions = new Map();
          eventsWithTimes.forEach(eventTime => {
            eventPositions.set(eventTime.event.id, {
              column: eventTime.column,
              totalColumns: eventTime.totalColumns,
            });
          });

          return (
            <div key={dayIndex} className="flex-1 min-w-0 flex flex-col">
              {/* Day header */}
              <TimeSlotDayHeader
                date={date}
                isToday={isToday}
                daysOfWeek={daysOfWeek}
                daysOfWeekFull={daysOfWeekFull}
                monthNames={monthNames}
                daysCount={days.length}
              />

              {/* Time grid for this day */}
              <div
                className="flex-1 relative border-l border-white/10 min-h-[1632px] transition-colors hover:bg-white/5 cursor-pointer"
                onDragOver={handleDragOver}
                onDrop={(e) => {
                  // Calculate exact time and date based on mouse position
                  const container = e.currentTarget;
                  const rect = container.getBoundingClientRect();
                  const offsetY = e.clientY - rect.top;

                  const time = calculateTimeFromOffset(offsetY);
                  handleDrop(date, time)(e, handleEventUpdate);
                }}
                onClick={(e) => {
                  // Don't open modal if currently resizing or just finished resizing
                  if (isResizing || justResized) {
                    return;
                  }

                  // Only open modal if clicking on empty space (not on an event or button)
                  const target = e.target as HTMLElement;
                  const isEvent = target.closest('.cursor-move'); // Event cards have cursor-move
                  const isButton = target.closest('button'); // Delete buttons
                  const isResizeHandle = target.closest('.group\\/handle'); // Resize handles

                  if (!isEvent && !isButton && !isResizeHandle) {
                    const container = e.currentTarget;
                    const rect = container.getBoundingClientRect();
                    const offsetY = e.clientY - rect.top;

                    const time = calculateTimeFromOffset(offsetY);
                    const dateString = date.toISOString().split('T')[0];

                    if (onTimeSlotClick) {
                      onTimeSlotClick(dateString, time);
                    }
                  }
                }}
              >
                {/* Time slot grid */}
                <TimeSlotGrid hours={hours} />

                {/* Events positioned absolutely */}
                {dayEvents.map((event) => {
                  const category = categories.find(c => c.id === event.categoryId);
                  const calendar = calendars.find(c => c.id === event.calendarId);
                  const executionDate = date.toISOString().split('T')[0];
                  const positionInfo = eventPositions.get(event.id) || { column: 0, totalColumns: 1 };

                  return (
                    <TimeSlotEventCard
                      key={event.id}
                      event={event}
                      date={date}
                      category={category}
                      calendar={calendar}
                      daysCount={days.length}
                      positionInfo={positionInfo}
                      resizingEventId={resizingEvent?.id}
                      onDragStart={(e) => {
                        setDragContext({
                          eventId: event.id,
                          originalDate: executionDate
                        });
                        handleDragStart(event)(e);
                      }}
                      onDragEnd={handleDragEnd}
                      onResizeStart={(edge, e, height, top) => handleResizeStart(event.id, edge, e, height, top)}
                      onToggleExecution={(e) => {
                        e.stopPropagation();
                        handleToggleExecution(event.id, date, event.executions?.find(exec => exec.executionDate.split('T')[0] === executionDate)?.completed || false);
                      }}
                      onEditClick={onEditClick ? (e) => onEditClick(event, e) : undefined}
                      onDeleteClick={(e) => onDeleteClick(event, e)}
                    />
                  );
                })}

                {/* Show message if no events for this day */}
                {dayEvents.length === 0 && (
                  <EmptyDayMessage daysCount={days.length} />
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
    </>
  );
}
