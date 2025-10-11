/**
 * Time helper functions for calendar operations
 */

/**
 * Calculate end time by adding hours to start time
 */
export function calculateEndTime(startTime: string, hoursToAdd: number = 1): string {
  if (!startTime) return '';

  const [hours, minutes] = startTime.split(':').map(Number);
  const endHour = (hours + hoursToAdd) % 24;
  return `${endHour.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
}

/**
 * Increment time by specified hours
 */
export function incrementTime(time: string, hoursToAdd: number = 1): string {
  return calculateEndTime(time, hoursToAdd);
}

/**
 * Validate if time string is in valid format (HH:MM)
 */
export function isValidTimeFormat(time: string): boolean {
  const timeRegex = /^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/;
  return timeRegex.test(time);
}

/**
 * Parse time string to hours and minutes
 */
export function parseTime(time: string): { hours: number; minutes: number } | null {
  if (!isValidTimeFormat(time)) return null;

  const [hours, minutes] = time.split(':').map(Number);
  return { hours, minutes };
}
