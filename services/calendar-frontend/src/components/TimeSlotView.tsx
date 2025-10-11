'use client';

import { useState, useEffect, useCallback } from 'react';
import { Event, Category } from '@/types/calendar';
import { calendars } from '@/data/calendars';
import { useDragAndDrop } from '@/hooks/useDragAndDrop';
import { api } from '@/lib/api';
import RecurringEventActionModal, { RecurringEventAction } from './RecurringEventActionModal';

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
  const [showRecurringActionModal, setShowRecurringActionModal] = useState(false);
  const [dragContext, setDragContext] = useState<{ eventId: string; originalDate: string } | null>(null);
  const [pendingUpdate, setPendingUpdate] = useState<{
    eventId: string;
    eventTitle: string;
    originalDate: string;
    newDate: string;
    newTime?: string;
    newEndTime?: string;
  } | null>(null);

  const handleEventUpdate = async (eventId: string, newDate: string, newTime?: string, newEndTime?: string, recurringAction?: RecurringEventAction, originalDate?: string) => {
    try {
      const event = events.find(e => e.id === eventId);
      if (!event) return;

      // Determine the original date - use dragContext, passed originalDate, or event's startDate
      const eventOriginalDate = originalDate || dragContext?.originalDate || event.startDate.split('T')[0];

      console.log('handleEventUpdate called:', {
        eventId,
        eventTitle: event.title,
        isRecurring: event.isRecurring,
        originalDate: eventOriginalDate,
        dragContextDate: dragContext?.originalDate,
        newDate,
        newTime,
        recurringAction
      });

      // If event is recurring and no action specified, show the modal
      if (event.isRecurring && !recurringAction) {
        setPendingUpdate({
          eventId,
          eventTitle: event.title,
          originalDate: eventOriginalDate,
          newDate,
          newTime,
          newEndTime,
        });
        setShowRecurringActionModal(true);
        return;
      }

      // Extract the base event ID if it's an expanded recurring event
      let baseEventId = eventId;
      const datePattern = /-\d{4}-\d{2}-\d{2}$/;
      if (datePattern.test(eventId)) {
        baseEventId = eventId.replace(datePattern, '');
      }

      const updatePayload: any = {
        startDate: newDate,
        ...(newTime && { startTime: newTime }),
        ...(newEndTime && { endTime: newEndTime }),
      };

      // Add recurring action scope if provided
      if (recurringAction) {
        updatePayload.recurringEditScope = recurringAction;
        // For all scopes, we need to pass the original occurrence date
        updatePayload.occurrenceDate = eventOriginalDate;
      }

      await api.events.update(baseEventId, updatePayload);

      // Clear drag context after update
      setDragContext(null);

      if (onEventUpdate) {
        onEventUpdate();
      }
    } catch (error) {
      console.error('Error updating event:', error);
      // Clear drag context even on error
      setDragContext(null);
    }
  };

  const handleRecurringActionSelect = async (action: RecurringEventAction) => {
    setShowRecurringActionModal(false);

    if (pendingUpdate) {
      await handleEventUpdate(
        pendingUpdate.eventId,
        pendingUpdate.newDate,
        pendingUpdate.newTime,
        pendingUpdate.newEndTime,
        action,
        pendingUpdate.originalDate
      );
      setPendingUpdate(null);
    }
  };

  const handleRecurringActionClose = () => {
    setShowRecurringActionModal(false);
    setPendingUpdate(null);
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
      const newHeight = Math.max(48, resizingEvent.originalHeight + deltaY); // Minimum 48px (30 minutes)
      eventElement.style.height = `${newHeight}px`;
    } else {
      // Resizing from top - change start time
      const newTop = resizingEvent.originalTop + deltaY;
      const newHeight = resizingEvent.originalHeight - deltaY;

      if (newHeight >= 48) { // Minimum 48px (30 minutes)
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

    // Convert pixels to time (96px per hour)
    const startMinutesFromMidnight = (newTop / 96) * 60 + (6 * 60); // 6am offset
    const durationMinutes = (newHeight / 96) * 60;
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

      // Backend já expande eventos recorrentes, então apenas comparamos a data
      // Se o evento tem occurrenceDate, significa que já foi expandido pelo backend
      const eventDate = event.startDate.split('T')[0];
      return eventDate === dateString;
    });
  };

  // Calculate overlap positions for events
  const calculateEventPositions = (dayEvents: Event[]) => {
    // Parse event times and calculate positions
    const eventsWithTimes = dayEvents.map(event => {
      const startParts = event.startTime.split(':');
      const startHours = parseInt(startParts[0], 10);
      const startMinutes = parseInt(startParts[1] || '0', 10);
      const startTotalMinutes = startHours * 60 + startMinutes;

      let endTotalMinutes = startTotalMinutes + 60; // Default 1 hour
      if (event.endTime) {
        const endParts = event.endTime.split(':');
        const endHours = parseInt(endParts[0], 10);
        const endMinutes = parseInt(endParts[1] || '0', 10);
        endTotalMinutes = endHours * 60 + endMinutes;
      }

      return {
        event,
        start: startTotalMinutes,
        end: endTotalMinutes,
        column: 0,
        totalColumns: 1,
      };
    });

    // Sort by start time, then by duration (longer first)
    eventsWithTimes.sort((a, b) => {
      if (a.start !== b.start) return a.start - b.start;
      return (b.end - b.start) - (a.end - a.start);
    });

    // Detect overlaps and assign columns
    const columns: Array<Array<typeof eventsWithTimes[0]>> = [];

    eventsWithTimes.forEach(eventTime => {
      // Find a column where this event doesn't overlap with any existing event
      let placedInColumn = false;

      for (let i = 0; i < columns.length; i++) {
        const column = columns[i];
        const hasOverlap = column.some(existing =>
          !(eventTime.end <= existing.start || eventTime.start >= existing.end)
        );

        if (!hasOverlap) {
          column.push(eventTime);
          eventTime.column = i;
          placedInColumn = true;
          break;
        }
      }

      // If no suitable column found, create a new one
      if (!placedInColumn) {
        columns.push([eventTime]);
        eventTime.column = columns.length - 1;
      }
    });

    // Calculate total columns for each event group
    eventsWithTimes.forEach(eventTime => {
      // Find all events that overlap with this one
      const overlappingEvents = eventsWithTimes.filter(other =>
        !(eventTime.end <= other.start || eventTime.start >= other.end)
      );

      // Find the maximum column number among overlapping events
      const maxColumn = Math.max(...overlappingEvents.map(e => e.column));
      eventTime.totalColumns = maxColumn + 1;
    });

    // Create a map for quick lookup
    const positionMap = new Map();
    eventsWithTimes.forEach(eventTime => {
      positionMap.set(eventTime.event.id, {
        column: eventTime.column,
        totalColumns: eventTime.totalColumns,
      });
    });

    return positionMap;
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
        <div className="w-20 flex-shrink-0">
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
            <div key={hour} className="h-24 flex items-start justify-end pr-2 pt-1 text-xs text-white/70">
              {hour.toString().padStart(2, '0')}:00
            </div>
          ))}
        </div>

        {/* Day columns */}
        {days.map((date, dayIndex) => {
          const isToday = date.toDateString() === today.toDateString();
          const dayEvents = getEventsForDate(date);
          const eventPositions = calculateEventPositions(dayEvents);

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
                className="flex-1 relative border-l border-white/10 min-h-[1632px] transition-colors hover:bg-white/5 cursor-pointer"
                onDragOver={handleDragOver}
                onDrop={(e) => {
                  // Calculate exact time and date based on mouse position
                  const container = e.currentTarget;
                  const rect = container.getBoundingClientRect();
                  const offsetY = e.clientY - rect.top;

                  // Calculate which hour slot we're in (96px per hour)
                  const totalHours = offsetY / 96;
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
                    const totalHours = offsetY / 96;
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
                    className="h-24 border-b border-white/5 relative"
                  >
                    <div className="h-12 border-b border-dashed border-white/5" />
                    <div className="h-12" />
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

                  // Calculate position and height based on time (96px per hour)
                  let topPosition = 0;
                  let eventHeight = 96; // Default 1 hour

                  try {
                    // Parse start time
                    const startParts = event.startTime.split(':');
                    const startHours = parseInt(startParts[0], 10);
                    const startMinutes = parseInt(startParts[1] || '0', 10);
                    topPosition = ((startHours - 6) * 96) + (startMinutes / 60 * 96);

                    // Parse end time if exists
                    if (event.endTime) {
                      const endParts = event.endTime.split(':');
                      const endHours = parseInt(endParts[0], 10);
                      const endMinutes = parseInt(endParts[1] || '0', 10);

                      const startTotalMinutes = startHours * 60 + startMinutes;
                      const endTotalMinutes = endHours * 60 + endMinutes;
                      const durationMinutes = endTotalMinutes - startTotalMinutes;

                      eventHeight = (durationMinutes / 60) * 96; // 96px per hour
                    }
                  } catch (error) {
                    console.error('Error parsing time for event:', event, error);
                  }

                  const textSize = days.length === 7 ? 'text-sm' : days.length <= 3 ? 'text-base' : 'text-lg';
                  const iconSize = days.length === 7 ? 'text-lg' : days.length <= 3 ? 'text-xl' : 'text-2xl';
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
                      <div className="flex flex-1 overflow-hidden cursor-move relative">
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
                        <div className="flex gap-1 pr-1">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleToggleExecution(event.id, date, isCompleted);
                            }}
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-green-600 rounded self-start"
                            title={isCompleted ? "Marcar como não realizado" : "Marcar como realizado"}
                          >
                            {isCompleted ? (
                              <svg className={days.length === 7 ? "w-5 h-5" : "w-6 h-6"} fill="currentColor" viewBox="0 0 24 24">
                                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                              </svg>
                            ) : (
                              <svg className={days.length === 7 ? "w-5 h-5" : "w-6 h-6"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
                              <svg className={days.length === 7 ? "w-5 h-5" : "w-6 h-6"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                              </svg>
                            </button>
                          )}
                          <button
                            onClick={(e) => onDeleteClick(event, e)}
                            className="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-red-600 rounded self-start"
                            title="Deletar"
                          >
                            <svg className={days.length === 7 ? "w-5 h-5" : "w-6 h-6"} fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
                    <div className={`text-white/30 ${days.length === 7 ? 'text-sm' : 'text-base'}`}>
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
    </>
  );
}
