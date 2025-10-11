/**
 * Custom hook for managing calendar selection state
 */

import { useState } from 'react';

const DEFAULT_SELECTED_CALENDARS = ['wb-digital-calendar', 'bruno-personal-calendar'];

export function useCalendarSelection() {
  const [selectedCalendars, setSelectedCalendars] = useState<string[]>(DEFAULT_SELECTED_CALENDARS);

  const toggleCalendar = (calendarId: string) => {
    setSelectedCalendars(prev => {
      if (prev.includes(calendarId)) {
        return prev.filter(id => id !== calendarId);
      } else {
        return [...prev, calendarId];
      }
    });
  };

  return {
    selectedCalendars,
    toggleCalendar,
  };
}
