/**
 * Custom hook for handling event resizing in TimeSlotView
 * Manages resize state, mouse events, and time calculations
 */

import { useState, useEffect, useCallback } from 'react';
import { Event } from '@/types/calendar';
import { PIXELS_PER_HOUR, START_HOUR, MIN_EVENT_HEIGHT } from '@/constants/timeSlotView';
import { pixelsToTime } from '@/utils/timeCalculations';

interface ResizingEvent {
  id: string;
  edge: 'top' | 'bottom';
  startY: number;
  originalHeight: number;
  originalTop: number;
}

interface UseEventResizeProps {
  events: Event[];
  onEventUpdate: (
    eventId: string,
    newDate: string,
    newTime?: string,
    newEndTime?: string
  ) => Promise<void>;
}

export function useEventResize({ events, onEventUpdate }: UseEventResizeProps) {
  const [resizingEvent, setResizingEvent] = useState<ResizingEvent | null>(null);
  const [justResized, setJustResized] = useState(false);
  const [isResizing, setIsResizing] = useState(false);

  const handleResizeStart = (
    eventId: string,
    edge: 'top' | 'bottom',
    e: React.MouseEvent,
    currentHeight: number,
    currentTop: number
  ) => {
    e.stopPropagation();
    e.preventDefault();

    // Disable dragging on the event element
    const eventElement = document.querySelector(`[data-event-id="${eventId}"]`) as HTMLElement;
    if (eventElement) {
      eventElement.setAttribute('draggable', 'false');
    }

    setIsResizing(true);
    setResizingEvent({
      id: eventId,
      edge,
      startY: e.clientY,
      originalHeight: currentHeight,
      originalTop: currentTop,
    });
  };

  const handleResizeMove = useCallback(
    (e: MouseEvent) => {
      if (!resizingEvent) return;

      const deltaY = e.clientY - resizingEvent.startY;

      // Find the event element
      const eventElement = document.querySelector(`[data-event-id="${resizingEvent.id}"]`) as HTMLElement;
      if (!eventElement) return;

      if (resizingEvent.edge === 'bottom') {
        // Resizing from bottom - change end time
        const newHeight = Math.max(MIN_EVENT_HEIGHT, resizingEvent.originalHeight + deltaY);
        eventElement.style.height = `${newHeight}px`;
      } else {
        // Resizing from top - change start time
        const newTop = resizingEvent.originalTop + deltaY;
        const newHeight = resizingEvent.originalHeight - deltaY;

        if (newHeight >= MIN_EVENT_HEIGHT) {
          eventElement.style.top = `${newTop}px`;
          eventElement.style.height = `${newHeight}px`;
        }
      }
    },
    [resizingEvent]
  );

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

    // Convert pixels to time using utils
    const newStartTime = pixelsToTime(newTop);
    const startMinutesFromMidnight = (newTop / PIXELS_PER_HOUR) * 60 + START_HOUR * 60;
    const durationMinutes = (newHeight / PIXELS_PER_HOUR) * 60;
    const endMinutesFromMidnight = startMinutesFromMidnight + durationMinutes;
    const endHours = Math.floor(endMinutesFromMidnight / 60);
    const endMins = Math.round(endMinutesFromMidnight % 60);
    const newEndTime = `${endHours.toString().padStart(2, '0')}:${endMins.toString().padStart(2, '0')}`;

    const dateString = event.startDate.split('T')[0];

    // Set flags to prevent modal opening BEFORE any async operations
    setJustResized(true);
    setIsResizing(false);
    setResizingEvent(null);

    await onEventUpdate(event.id, dateString, newStartTime, newEndTime);

    // Keep the flag for longer to ensure click event doesn't trigger
    setTimeout(() => setJustResized(false), 300);
  }, [resizingEvent, events, onEventUpdate]);

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

  return {
    handleResizeStart,
    isResizing,
    justResized,
    resizingEvent,
  };
}
