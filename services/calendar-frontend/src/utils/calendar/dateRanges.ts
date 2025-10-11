/**
 * Date range calculation utilities
 */

import { DATE_RANGE_CONFIG } from '@/constants/calendar';

export interface DateRange {
  startDate: string;
  endDate: string;
}

/**
 * Calculate the default date range for event fetching
 * (3 months in the past to 1 year in the future)
 */
export function getDefaultDateRange(): DateRange {
  const today = new Date();

  const startDate = new Date(today);
  startDate.setMonth(startDate.getMonth() - DATE_RANGE_CONFIG.PAST_MONTHS);

  const endDate = new Date(today);
  endDate.setFullYear(endDate.getFullYear() + DATE_RANGE_CONFIG.FUTURE_YEARS);

  return {
    startDate: startDate.toISOString().split('T')[0],
    endDate: endDate.toISOString().split('T')[0],
  };
}

/**
 * Calculate date range for a specific month
 */
export function getMonthDateRange(date: Date): DateRange {
  const year = date.getFullYear();
  const month = date.getMonth();

  const startDate = new Date(year, month, 1);
  const endDate = new Date(year, month + 1, 0);

  return {
    startDate: startDate.toISOString().split('T')[0],
    endDate: endDate.toISOString().split('T')[0],
  };
}

/**
 * Calculate date range for a specific week
 */
export function getWeekDateRange(date: Date): DateRange {
  const startOfWeek = new Date(date);
  const day = startOfWeek.getDay();
  startOfWeek.setDate(startOfWeek.getDate() - day);

  const endOfWeek = new Date(startOfWeek);
  endOfWeek.setDate(endOfWeek.getDate() + 6);

  return {
    startDate: startOfWeek.toISOString().split('T')[0],
    endDate: endOfWeek.toISOString().split('T')[0],
  };
}
