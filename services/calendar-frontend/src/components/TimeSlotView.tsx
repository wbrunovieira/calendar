'use client';

import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { useTimeSlotEventUpdate } from '@/hooks/useTimeSlotEventUpdate';
import { useEventResize } from '@/hooks/useEventResize';
import { useEventExecution } from '@/hooks/useEventExecution';
import {
  PIXELS_PER_HOUR,
  HOURS_ARRAY,
} from '@/constants/timeSlotView';
import {
  timeToPixels,
  calculateEventHeight,
  calculateTimeFromOffset,
} from '@/utils/timeCalculations';
import RecurringEventActionModal from './RecurringEventActionModal';
import TimeColumn from './TimeColumn';
import TimeSlotDayHeader from './TimeSlotDayHeader';
import TimeSlotGrid from './TimeSlotGrid';
import EmptyDayMessage from './EmptyDayMessage';

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
                  const calendarIcon = calendar?.type === 'professional' ? '💼' : '👤';

                  // Check execution status from event executions
                  const executionDate = date.toISOString().split('T')[0];
                  const execution = event.executions?.find(exec =>
                    exec.executionDate.split('T')[0] === executionDate
                  );
                  const isCompleted = execution?.completed || false;

                  // Calculate position and height based on time
                  let topPosition = 0;
                  let eventHeight = PIXELS_PER_HOUR; // Default 1 hour

                  try {
                    topPosition = timeToPixels(event.startTime);

                    if (event.endTime) {
                      eventHeight = calculateEventHeight(event.startTime, event.endTime);
                    }
                  } catch {
                    // Error parsing time for event
                  }

                  const textSize = days.length === 7 ? 'text-sm' : days.length <= 3 ? 'text-base' : 'text-lg';
                  const padding = days.length === 7 ? 'px-2 py-2' : days.length <= 3 ? 'px-3 py-2' : 'px-4 py-3';

                  // Get position info for overlapping events
                  const positionInfo = eventPositions.get(event.id) || { column: 0, totalColumns: 1 };
                  const widthPercentage = 100 / positionInfo.totalColumns;
                  const leftPercentage = (positionInfo.column * widthPercentage);

                  return (
                    <div
                      key={event.id}
                      data-event-id={event.id}
                      draggable
                      onDragStart={(e) => {
                        // Capture the date of this specific occurrence
                        setDragContext({
                          eventId: event.id,
                          originalDate: executionDate
                        });
                        handleDragStart(event)(e);
                      }}
                      onDragEnd={handleDragEnd}
                      className={`absolute rounded-lg text-white flex flex-col overflow-visible group ${
                        isCompleted
                          ? 'ring-2 ring-green-500 shadow-lg shadow-green-500/50'
                          : ''
                      } transition-all duration-300`}
                      style={{
                        backgroundColor: isCompleted
                          ? category?.color + '60'
                          : category?.color + '90',
                        top: `${topPosition}px`,
                        height: `${eventHeight}px`,
                        minHeight: days.length === 7 ? '96px' : days.length <= 3 ? '110px' : '120px',
                        left: `${leftPercentage}%`,
                        width: `${widthPercentage}%`,
                        zIndex: 10 + positionInfo.column,
                        cursor: resizingEvent?.id === event.id ? 'ns-resize' : 'move',
                      }}
                    >
                      {/* Calendar icon badge - positioned at top left */}
                      <div
                        className="absolute -top-2 -left-2 rounded-full flex items-center justify-center shadow-lg z-30"
                        style={{
                          backgroundColor: calendar?.color,
                          width: days.length === 7 ? '24px' : days.length <= 3 ? '28px' : '32px',
                          height: days.length === 7 ? '24px' : days.length <= 3 ? '28px' : '32px',
                          fontSize: days.length === 7 ? '12px' : days.length <= 3 ? '14px' : '16px'
                        }}
                      >
                        {calendarIcon}
                      </div>

                      {/* Top resize handle */}
                      <div
                        className="absolute top-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
                        onMouseDown={(e) => handleResizeStart(event.id, 'top', e, eventHeight, topPosition)}
                        title="Arrastar para ajustar início"
                      >
                        <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-t-lg" />
                      </div>

                      {/* Event content */}
                      <div className="flex flex-1 overflow-visible cursor-move relative">
                        <div className={`flex-1 ${padding} flex flex-col ${isCompleted ? 'relative' : ''}`}>
                          {/* Category Icon - centered, prominent */}
                          <div className="text-center pb-1">
                            <span className={`${days.length === 7 ? 'text-2xl' : days.length <= 3 ? 'text-3xl' : 'text-4xl'}`}>
                              {category?.icon}
                            </span>
                          </div>

                          {/* Time - single line, bold */}
                          <div className={`font-bold ${days.length === 7 ? 'text-xs' : textSize} text-center leading-none whitespace-nowrap pb-2 ${isCompleted ? 'line-through opacity-70' : ''}`}>
                            {event.startTime}{event.endTime && ` - ${event.endTime}`}
                          </div>

                          {/* Subtle divider */}
                          <div className="w-full h-px bg-white/20 mb-2"></div>

                          {/* Event Title - more space */}
                          <div className={`${textSize} leading-relaxed ${isCompleted ? 'line-through opacity-70' : ''}`}>
                            {event.title}
                          </div>

                          {isCompleted && (
                            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                              <div className="bg-green-500 rounded-full p-2 shadow-lg animate-pulse">
                                <svg className={days.length === 7 ? "w-6 h-6" : "w-8 h-8"} fill="white" viewBox="0 0 24 24">
                                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                                </svg>
                              </div>
                            </div>
                          )}
                        </div>
                        <div className="flex flex-col gap-1 pr-1 pt-1">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleToggleExecution(event.id, date, isCompleted);
                            }}
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-green-600 rounded"
                            title={isCompleted ? "Marcar como não realizado" : "Marcar como realizado"}
                          >
                            {isCompleted ? (
                              <svg className={days.length === 7 ? "w-4 h-4" : "w-5 h-5"} fill="currentColor" viewBox="0 0 24 24">
                                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                              </svg>
                            ) : (
                              <svg className={days.length === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <rect x="3" y="3" width="18" height="18" rx="2" strokeWidth={2} />
                              </svg>
                            )}
                          </button>
                          {onEditClick && (
                            <button
                              onClick={(e) => onEditClick(event, e)}
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-blue-600 rounded"
                              title="Editar"
                            >
                              <svg className={days.length === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                              </svg>
                            </button>
                          )}
                          <button
                            onClick={(e) => onDeleteClick(event, e)}
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-600 rounded"
                            title="Deletar"
                          >
                            <svg className={days.length === 7 ? "w-4 h-4" : "w-5 h-5"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>
                        </div>
                      </div>

                      {/* Bottom resize handle */}
                      <div
                        className="absolute bottom-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
                        onMouseDown={(e) => handleResizeStart(event.id, 'bottom', e, eventHeight, topPosition)}
                        title="Arrastar para ajustar fim"
                      >
                        <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-b-lg mt-2" />
                      </div>
                    </div>
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
