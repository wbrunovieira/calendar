/**
 * Custom hook for managing calendar navigation and view mode
 */

import { useState } from 'react';
import { ViewMode } from '@/constants/calendar';

type CalendarViewMode = ViewMode | 'month' | 'week' | '3days' | 'day';

export function useCalendarNavigation() {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [viewMode, setViewMode] = useState<CalendarViewMode>('day');

  const previousPeriod = () => {
    if (viewMode === 'month') {
      setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() - 1));
    } else if (viewMode === 'week') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() - 7);
      setCurrentDate(newDate);
    } else if (viewMode === '3days') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() - 3);
      setCurrentDate(newDate);
    } else {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() - 1);
      setCurrentDate(newDate);
    }
  };

  const nextPeriod = () => {
    if (viewMode === 'month') {
      setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() + 1));
    } else if (viewMode === 'week') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() + 7);
      setCurrentDate(newDate);
    } else if (viewMode === '3days') {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() + 3);
      setCurrentDate(newDate);
    } else {
      const newDate = new Date(currentDate);
      newDate.setDate(newDate.getDate() + 1);
      setCurrentDate(newDate);
    }
  };

  const goToToday = () => {
    setCurrentDate(new Date());
  };

  const navigateToDate = (date: Date) => {
    setCurrentDate(date);
  };

  return {
    currentDate,
    viewMode,
    setViewMode,
    previousPeriod,
    nextPeriod,
    goToToday,
    navigateToDate,
  };
}
