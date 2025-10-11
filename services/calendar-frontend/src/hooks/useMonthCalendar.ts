/**
 * Custom hook for month calendar logic
 * Handles date calculations and event filtering for month view
 */

import { useMemo } from 'react';
import { Event } from '@/types/calendar';
import { getDaysInMonth, getFirstDayOfMonth } from '@/utils/calendar';

interface UseMonthCalendarProps {
  currentDate: Date;
  events: Event[];
  selectedCalendars: string[];
}

export function useMonthCalendar({ currentDate, events, selectedCalendars }: UseMonthCalendarProps) {
  const daysInMonth = getDaysInMonth(currentDate);
  const firstDay = getFirstDayOfMonth(currentDate);

  const today = useMemo(() => new Date(), []);

  const isCurrentMonth = useMemo(
    () => currentDate.getMonth() === today.getMonth() && currentDate.getFullYear() === today.getFullYear(),
    [currentDate, today]
  );

  const getEventsForDate = useMemo(
    () => (date: Date): Event[] => {
      const dateString = date.toISOString().split('T')[0];

      return events.filter(event => {
        // Filter by selected calendars
        if (!selectedCalendars.includes(event.calendarId)) {
          return false;
        }

        // Compare only the date (YYYY-MM-DD)
        const eventDate = event.startDate.split('T')[0];
        return eventDate === dateString;
      });
    },
    [events, selectedCalendars]
  );

  return {
    daysInMonth,
    firstDay,
    today,
    isCurrentMonth,
    getEventsForDate,
  };
}
