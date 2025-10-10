'use client';

import { useState, useEffect, useCallback } from 'react';
import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { api } from '@/lib/api';

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
  const hours = Array.from({ length: 17 }, (_, i) => i + 6); // 6am to 10pm
  const today = new Date();
  const { handleDragStart, handleDragEnd, handleDragOver, handleDrop } = useDragAndDrop();
  const [resizingEvent, setResizingEvent] = useState<{ id: string; edge: 'top' | 'bottom'; startY: number; originalHeight: number; originalTop: number } | null>(null);
  const [justResized, setJustResized] = useState(false);
  const [isResizing, setIsResizing] = useState(false);

  const handleEventUpdate = async (eventId: string, newDate: string, newTime?: string, newEndTime?: string) => {
    try {
      const event = events.find(e => e.id === eventId);
      if (!event) return;

      await api.events.update(eventId, {
        startDate: newDate,
        ...(newTime && { startTime: newTime }),
        ...(newEndTime && { endTime: newEndTime }),
      });

      if (onEventUpdate) {
        onEventUpdate();
      }
    } catch (error) {
      console.error('Error updating event:', error);
    }
  };

  const handleResizeStart = (eventId: string, edge: 'top' | 'bottom', e: React.MouseEvent, currentHeight: number, currentTop: number) => {
    e.stopPropagation();
    e.preventDefault();

    // Disable dragging on the event element
    const eventElement = document.querySelector(`[data-event-id="${eventId}"]`) as HTMLElement;
    if (eventElement) {
      eventElement.setAttribute('draggable', 'false');
    }

    setIsResizing(true);
    setResizingEvent({ id: eventId, edge, startY: e.clientY, originalHeight: currentHeight, originalTop: currentTop });
  };

  const handleResizeMove = useCallback((e: MouseEvent) => {
    if (!resizingEvent) return;

    const deltaY = e.clientY - resizingEvent.startY;

    // Find the event element
    const eventElement = document.querySelector(`[data-event-id="${resizingEvent.id}"]`) as HTMLElement;
    if (!eventElement) return;

    if (resizingEvent.edge === 'bottom') {
      // Resizing from bottom - change end time
      const newHeight = Math.max(32, resizingEvent.originalHeight + deltaY); // Minimum 32px (30 minutes)
      eventElement.style.height = `${newHeight}px`;
    } else {
      // Resizing from top - change start time
      const newTop = resizingEvent.originalTop + deltaY;
      const newHeight = resizingEvent.originalHeight - deltaY;

      if (newHeight >= 32) { // Minimum 32px (30 minutes)
        eventElement.style.top = `${newTop}px`;
        eventElement.style.height = `${newHeight}px`;
      }
    }
  }, [resizingEvent]);

  const handleResizeEnd = useCallback(async () => {
    if (!resizingEvent) return;

    const event = events.find(ev => ev.id === resizingEvent.id);
    if (!event) {
      setResizingEvent(null);
      setIsResizing(false);
      return;
    }

    const eventElement = document.querySelector(`[data-event-id="${resizingEvent.id}"]`) as HTMLElement;
    if (!eventElement) {
      setResizingEvent(null);
      setIsResizing(false);
      return;
    }

    // Re-enable dragging
    eventElement.setAttribute('draggable', 'true');

    const newTop = parseFloat(eventElement.style.top);
    const newHeight = parseFloat(eventElement.style.height);

    // Convert pixels to time
    const startMinutesFromMidnight = (newTop / 64) * 60 + (6 * 60); // 6am offset
    const durationMinutes = (newHeight / 64) * 60;
    const endMinutesFromMidnight = startMinutesFromMidnight + durationMinutes;

    const startHours = Math.floor(startMinutesFromMidnight / 60);
    const startMins = Math.round(startMinutesFromMidnight % 60);
    const endHours = Math.floor(endMinutesFromMidnight / 60);
    const endMins = Math.round(endMinutesFromMidnight % 60);

    const newStartTime = `${startHours.toString().padStart(2, '0')}:${startMins.toString().padStart(2, '0')}`;
    const newEndTime = `${endHours.toString().padStart(2, '0')}:${endMins.toString().padStart(2, '0')}`;

    const dateString = event.startDate.split('T')[0];

    // Set flags to prevent modal opening BEFORE any async operations
    setJustResized(true);
    setIsResizing(false);
    setResizingEvent(null);

    await handleEventUpdate(event.id, dateString, newStartTime, newEndTime);

    // Keep the flag for longer to ensure click event doesn't trigger
    setTimeout(() => setJustResized(false), 300);
  }, [resizingEvent, events, handleEventUpdate]);

  // Handle checkbox toggle
  const handleToggleExecution = async (eventId: string, date: Date, currentlyCompleted: boolean) => {
    const executionDate = date.toISOString().split('T')[0];

    try {
      await api.events.toggleExecution(
        eventId,
        executionDate,
        !currentlyCompleted
      );

      // Reload events to get updated execution status
      if (onEventUpdate) {
        onEventUpdate();
      }
    } catch (error) {
      console.error('Error toggling execution:', error);
    }
  };

  // Add mouse event listeners for resize
  useEffect(() => {
    if (resizingEvent) {
      window.addEventListener('mousemove', handleResizeMove);
      window.addEventListener('mouseup', handleResizeEnd);

      return () => {
        window.removeEventListener('mousemove', handleResizeMove);
        window.removeEventListener('mouseup', handleResizeEnd);
      };
    }
  }, [resizingEvent, handleResizeMove, handleResizeEnd]);

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
              <div
                className="flex-1 relative border-l border-white/10 min-h-[1088px] transition-colors hover:bg-white/5 cursor-pointer"
                onDragOver={handleDragOver}
                onDrop={(e) => {
                  // Calculate exact time and date based on mouse position
                  const container = e.currentTarget;
                  const rect = container.getBoundingClientRect();
                  const offsetY = e.clientY - rect.top;

                  // Calculate which hour slot we're in (64px per hour)
                  const totalHours = offsetY / 64;
                  const hour = Math.floor(totalHours) + 6; // Add 6 because we start at 6am
                  const fractionalHour = totalHours - Math.floor(totalHours);
                  const minutes = Math.floor(fractionalHour * 60);

                  // Ensure hour is within bounds (6am-10pm)
                  const clampedHour = Math.max(6, Math.min(22, hour));
                  const clampedMinutes = Math.max(0, Math.min(59, minutes));

                  const time = `${clampedHour.toString().padStart(2, '0')}:${clampedMinutes.toString().padStart(2, '0')}`;
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

                    // Calculate time based on click position
                    const totalHours = offsetY / 64;
                    const hour = Math.floor(totalHours) + 6;
                    const fractionalHour = totalHours - Math.floor(totalHours);
                    const minutes = Math.floor(fractionalHour * 60);

                    const clampedHour = Math.max(6, Math.min(22, hour));
                    const clampedMinutes = Math.max(0, Math.min(59, minutes));

                    const time = `${clampedHour.toString().padStart(2, '0')}:${clampedMinutes.toString().padStart(2, '0')}`;
                    const dateString = date.toISOString().split('T')[0];

                    if (onTimeSlotClick) {
                      onTimeSlotClick(dateString, time);
                    }
                  }
                }}
              >
                {/* Time slot grid */}
                {hours.map((hour) => (
                  <div
                    key={hour}
                    className="h-16 border-b border-white/5 relative"
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

                  // Check execution status from event executions
                  const executionDate = date.toISOString().split('T')[0];
                  const execution = event.executions?.find(exec =>
                    exec.executionDate.split('T')[0] === executionDate
                  );
                  const isCompleted = execution?.completed || false;

                  // Calculate position and height based on time
                  let topPosition = 0;
                  let eventHeight = 64; // Default 1 hour

                  try {
                    // Parse start time
                    const startParts = event.startTime.split(':');
                    const startHours = parseInt(startParts[0], 10);
                    const startMinutes = parseInt(startParts[1] || '0', 10);
                    topPosition = ((startHours - 6) * 64) + (startMinutes / 60 * 64);

                    // Parse end time if exists
                    if (event.endTime) {
                      const endParts = event.endTime.split(':');
                      const endHours = parseInt(endParts[0], 10);
                      const endMinutes = parseInt(endParts[1] || '0', 10);

                      const startTotalMinutes = startHours * 60 + startMinutes;
                      const endTotalMinutes = endHours * 60 + endMinutes;
                      const durationMinutes = endTotalMinutes - startTotalMinutes;

                      eventHeight = (durationMinutes / 60) * 64; // 64px per hour
                    }
                  } catch (error) {
                    console.error('Error parsing time for event:', event, error);
                  }

                  const textSize = days.length === 7 ? 'text-[8px]' : days.length <= 3 ? 'text-[10px]' : 'text-xs';
                  const iconSize = days.length === 7 ? 'text-[8px]' : days.length <= 3 ? 'text-xs' : 'text-sm';
                  const padding = days.length === 7 ? 'px-0.5 py-0.5' : days.length <= 3 ? 'px-2 py-1' : 'px-2 py-1';

                  return (
                    <div
                      key={event.id}
                      data-event-id={event.id}
                      draggable
                      onDragStart={handleDragStart(event)}
                      onDragEnd={handleDragEnd}
                      className={`absolute rounded-lg text-white flex flex-col overflow-visible group ${days.length === 7 ? 'left-0.5 right-0.5' : 'left-2 right-2'} ${
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
                        minHeight: '48px',
                        zIndex: 10,
                        cursor: resizingEvent?.id === event.id ? 'ns-resize' : 'move',
                      }}
                    >
                      {/* Top resize handle */}
                      <div
                        className="absolute top-0 left-0 right-0 h-3 cursor-ns-resize group/handle z-20"
                        onMouseDown={(e) => handleResizeStart(event.id, 'top', e, eventHeight, topPosition)}
                        title="Arrastar para ajustar início"
                      >
                        <div className="h-1 bg-white/0 group-hover/handle:bg-white/40 transition-all rounded-t-lg" />
                      </div>

                      {/* Event content */}
                      <div className="flex flex-1 overflow-hidden cursor-move relative">
                        <div
                          className={`${padding} flex items-center justify-center ${iconSize}`}
                          style={{ backgroundColor: calendar?.color }}
                        >
                          {calendarIcon}
                        </div>
                        <div className={`flex-1 ${padding} flex flex-col justify-center ${isCompleted ? 'relative' : ''}`}>
                          <div className={`font-semibold ${textSize} flex items-center gap-1 ${isCompleted ? 'line-through opacity-70' : ''}`}>
                            <span>{category?.icon}</span>
                            <span className="truncate">{event.startTime}</span>
                            {event.endTime && <span className="truncate">- {event.endTime}</span>}
                          </div>
                          <div className={`${textSize} truncate ${isCompleted ? 'line-through opacity-70' : ''}`}>{event.title}</div>
                          {isCompleted && (
                            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                              <div className="bg-green-500 rounded-full p-1 shadow-lg animate-pulse">
                                <svg className={days.length === 7 ? "w-4 h-4" : "w-6 h-6"} fill="white" viewBox="0 0 24 24">
                                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                                </svg>
                              </div>
                            </div>
                          )}
                        </div>
                        <div className="flex gap-1">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleToggleExecution(event.id, date, isCompleted);
                            }}
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-green-600 rounded self-start"
                            title={isCompleted ? "Marcar como não realizado" : "Marcar como realizado"}
                          >
                            {isCompleted ? (
                              <svg className={days.length === 7 ? "w-3 h-3" : "w-4 h-4"} fill="currentColor" viewBox="0 0 24 24">
                                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                              </svg>
                            ) : (
                              <svg className={days.length === 7 ? "w-3 h-3" : "w-4 h-4"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <rect x="3" y="3" width="18" height="18" rx="2" strokeWidth={2} />
                              </svg>
                            )}
                          </button>
                          {onEditClick && (
                            <button
                              onClick={(e) => onEditClick(event, e)}
                              className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-blue-600 rounded self-start"
                              title="Editar"
                            >
                              <svg className={days.length === 7 ? "w-3 h-3" : "w-4 h-4"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                              </svg>
                            </button>
                          )}
                          <button
                            onClick={(e) => onDeleteClick(event, e)}
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-600 rounded self-start"
                            title="Deletar"
                          >
                            <svg className={days.length === 7 ? "w-3 h-3" : "w-4 h-4"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
