/**
 * Time calculation utilities for TimeSlotView
 */

import { PIXELS_PER_HOUR, START_HOUR } from '@/constants/timeSlotView';

/**
 * Parse time string (HH:MM) to hours and minutes
 */
export function parseTimeString(time: string): { hours: number; minutes: number } {
  const parts = time.split(':');
  const hours = parseInt(parts[0], 10);
  const minutes = parseInt(parts[1] || '0', 10);
  return { hours, minutes };
}

/**
 * Convert time to total minutes from midnight
 */
export function timeToMinutes(time: string): number {
  const { hours, minutes } = parseTimeString(time);
  return hours * 60 + minutes;
}

/**
 * Convert time string to pixel position (relative to START_HOUR)
 */
export function timeToPixels(time: string): number {
  const { hours, minutes } = parseTimeString(time);
  return ((hours - START_HOUR) * PIXELS_PER_HOUR) + (minutes / 60 * PIXELS_PER_HOUR);
}

/**
 * Convert pixel position to time string
 */
export function pixelsToTime(pixels: number): string {
  const totalHours = pixels / PIXELS_PER_HOUR;
  const hour = Math.floor(totalHours) + START_HOUR;
  const fractionalHour = totalHours - Math.floor(totalHours);
  const minutes = Math.floor(fractionalHour * 60);

  return `${hour.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
}

/**
 * Calculate event height in pixels based on start and end time
 */
export function calculateEventHeight(startTime: string, endTime: string): number {
  const startMinutes = timeToMinutes(startTime);
  const endMinutes = timeToMinutes(endTime);
  const durationMinutes = endMinutes - startMinutes;

  return (durationMinutes / 60) * PIXELS_PER_HOUR;
}

/**
 * Calculate time from pixel offset and clamp to bounds
 */
export function calculateTimeFromOffset(offsetY: number, startHour: number = START_HOUR, endHour: number = 22): string {
  const totalHours = offsetY / PIXELS_PER_HOUR;
  const hour = Math.floor(totalHours) + startHour;
  const fractionalHour = totalHours - Math.floor(totalHours);
  const minutes = Math.floor(fractionalHour * 60);

  const clampedHour = Math.max(startHour, Math.min(endHour, hour));
  const clampedMinutes = Math.max(0, Math.min(59, minutes));

  return `${clampedHour.toString().padStart(2, '0')}:${clampedMinutes.toString().padStart(2, '0')}`;
}

/**
 * Extract base event ID (remove date suffix from expanded recurring events)
 */
export function extractBaseEventId(eventId: string): string {
  const datePattern = /-\d{4}-\d{2}-\d{2}$/;
  if (datePattern.test(eventId)) {
    return eventId.replace(datePattern, '');
  }
  return eventId;
}
