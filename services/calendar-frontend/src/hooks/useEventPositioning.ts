/**
 * Hook for calculating event positions to handle overlapping events
 */

import { useMemo } from 'react';
import { Event } from '@/types/calendar';
import { timeToMinutes } from '@/utils/timeCalculations';

interface EventPosition {
  column: number;
  totalColumns: number;
}

export function useEventPositioning(dayEvents: Event[]): Map<string, EventPosition> {
  return useMemo(() => {
    // Parse event times and calculate positions
    const eventsWithTimes = dayEvents.map(event => {
      const startTotalMinutes = timeToMinutes(event.startTime);

      let endTotalMinutes = startTotalMinutes + 60; // Default 1 hour
      if (event.endTime) {
        endTotalMinutes = timeToMinutes(event.endTime);
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
        const hasOverlap = column.some(existing => !(eventTime.end <= existing.start || eventTime.start >= existing.end));

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
      const overlappingEvents = eventsWithTimes.filter(
        other => !(eventTime.end <= other.start || eventTime.start >= other.end)
      );

      // Find the maximum column number among overlapping events
      const maxColumn = Math.max(...overlappingEvents.map(e => e.column));
      eventTime.totalColumns = maxColumn + 1;
    });

    // Create a map for quick lookup
    const positionMap = new Map<string, EventPosition>();
    eventsWithTimes.forEach(eventTime => {
      positionMap.set(eventTime.event.id, {
        column: eventTime.column,
        totalColumns: eventTime.totalColumns,
      });
    });

    return positionMap;
  }, [dayEvents]);
}
