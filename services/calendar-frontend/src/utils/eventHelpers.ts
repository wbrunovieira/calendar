/**
 * Event helper functions
 * Business logic for event operations
 */

import { ReminderInput } from '@/types/calendar';

interface EventFormData {
  calendarId: string;
  categoryId: string;
  categoryTypeId: string;
  title: string;
  description: string;
  startTime: string;
  endTime: string;
  startDate: string;
  endDate: string;
  isRecurring: boolean;
  recurrenceFrequency: 'daily' | 'weekly' | 'monthly' | 'yearly';
  recurrenceInterval: number;
  recurrenceDaysOfWeek: number[];
  recurrenceEndDate: string;
  reminders?: ReminderInput[];
}

/**
 * Build event payload for API submission
 * Only includes non-empty optional fields
 */
export function buildEventPayload(formData: EventFormData): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    calendarId: formData.calendarId,
    title: formData.title,
    startTime: formData.startTime,
    startDate: formData.startDate,
    isRecurring: formData.isRecurring,
  };

  // Only add optional fields if they have values
  if (formData.categoryId) payload.categoryId = formData.categoryId;
  if (formData.categoryTypeId) payload.categoryTypeId = formData.categoryTypeId;
  if (formData.description) payload.description = formData.description;
  if (formData.endTime) payload.endTime = formData.endTime;
  if (formData.endDate) payload.endDate = formData.endDate;

  // Add recurrence fields if event is recurring
  if (formData.isRecurring) {
    payload.recurrenceFrequency = formData.recurrenceFrequency;
    payload.recurrenceInterval = formData.recurrenceInterval;

    if (formData.recurrenceFrequency === 'weekly') {
      // If no days selected, use the day of week from start date
      if (formData.recurrenceDaysOfWeek.length > 0) {
        payload.recurrenceDaysOfWeek = formData.recurrenceDaysOfWeek;
      } else if (formData.startDate) {
        // Get day of week from start date (0 = Sunday, 6 = Saturday)
        const startDate = new Date(formData.startDate);
        const dayOfWeek = startDate.getDay();
        payload.recurrenceDaysOfWeek = [dayOfWeek];
      }
    }

    if (formData.recurrenceEndDate) {
      payload.recurrenceEndDate = formData.recurrenceEndDate;
    }
  }

  // Add reminders if present
  if (formData.reminders && formData.reminders.length > 0) {
    payload.reminders = formData.reminders;
  }

  return payload;
}

/**
 * Validate required event fields
 */
export function validateEventForm(formData: EventFormData): string | null {
  if (!formData.calendarId) return 'Selecione um calendário';
  if (!formData.title) return 'Digite um título';
  if (!formData.startTime) return 'Selecione a hora de início';
  if (!formData.startDate) return 'Selecione a data de início';
  return null;
}
