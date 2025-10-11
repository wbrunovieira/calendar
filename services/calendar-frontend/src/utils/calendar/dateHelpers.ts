/**
 * Date helper functions for calendar operations
 */

/**
 * Get the number of days in a given month
 */
export function getDaysInMonth(date: Date): number {
  const year = date.getFullYear();
  const month = date.getMonth();
  return new Date(year, month + 1, 0).getDate();
}

/**
 * Get the first day of the month (0-6, Sunday-Saturday)
 */
export function getFirstDayOfMonth(date: Date): number {
  const year = date.getFullYear();
  const month = date.getMonth();
  return new Date(year, month, 1).getDay();
}

/**
 * Calculate week number in the year (ISO 8601)
 */
export function getWeekNumber(date: Date): number {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const dayNum = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
  return Math.ceil((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7;
}

/**
 * Calculate week number in the month (1-5)
 */
export function getWeekOfMonth(date: Date): number {
  const firstDay = new Date(date.getFullYear(), date.getMonth(), 1);
  const firstDayOfWeek = firstDay.getDay();
  const offsetDate = date.getDate() + firstDayOfWeek - 1;
  return Math.ceil(offsetDate / 7);
}

/**
 * Get all days in a week starting from Sunday
 */
export function getWeekDays(currentDate: Date): Date[] {
  const startOfWeek = new Date(currentDate);
  const day = startOfWeek.getDay();
  startOfWeek.setDate(startOfWeek.getDate() - day);

  const days: Date[] = [];
  for (let i = 0; i < 7; i++) {
    const date = new Date(startOfWeek);
    date.setDate(date.getDate() + i);
    days.push(date);
  }
  return days;
}

/**
 * Get next N days starting from current date
 */
export function getNextNDays(currentDate: Date, n: number): Date[] {
  const days: Date[] = [];
  for (let i = 0; i < n; i++) {
    const date = new Date(currentDate);
    date.setDate(date.getDate() + i);
    days.push(date);
  }
  return days;
}

/**
 * Check if a date is today
 */
export function isToday(date: Date): boolean {
  const today = new Date();
  return (
    date.getDate() === today.getDate() &&
    date.getMonth() === today.getMonth() &&
    date.getFullYear() === today.getFullYear()
  );
}

/**
 * Check if a date is tomorrow
 */
export function isTomorrow(date: Date): boolean {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  return (
    date.getDate() === tomorrow.getDate() &&
    date.getMonth() === tomorrow.getMonth() &&
    date.getFullYear() === tomorrow.getFullYear()
  );
}

/**
 * Format date to YYYY-MM-DD
 */
export function formatDateToString(date: Date): string {
  return date.toISOString().split('T')[0];
}

/**
 * Get days array based on view mode
 */
export function getDaysForView(
  viewMode: 'month' | 'week' | '3days' | 'day',
  currentDate: Date
): Date[] {
  switch (viewMode) {
    case 'week':
      return getWeekDays(currentDate);
    case '3days':
      return getNextNDays(currentDate, 3);
    case 'day':
      return [currentDate];
    default:
      return [];
  }
}
