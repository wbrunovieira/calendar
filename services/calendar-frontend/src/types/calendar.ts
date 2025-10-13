export type CalendarType = 'professional' | 'personal';

export interface Calendar {
  id: string;
  name: string;
  email: string;
  color: string;
  type: CalendarType;
  isActive: boolean;
}

export interface CategoryType {
  id: string;
  calendarId: string;
  name: string;
  value: string;
  icon?: string;
  color: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Category {
  id: string;
  calendarId: string;
  name: string;
  icon: string;
  color: string;
  type?: string; // Legacy field, now optional
  categoryTypes?: CategoryType[]; // New many-to-many relationship
  isActive: boolean;
}

export interface EventExecution {
  id: string;
  eventId: string;
  executionDate: string;
  completed: boolean;
  notes?: string;
}

export interface Event {
  id: string;
  calendarId: string;
  categoryId?: string;
  categoryTypeId?: string;
  category?: Category; // Category object from API (includes icon, color, name, type)
  categoryType?: CategoryType; // CategoryType object from API
  title: string;
  description?: string;
  startTime: string;
  endTime?: string;
  startDate: string;
  endDate?: string;
  isRecurring: boolean;
  recurrenceFrequency?: 'daily' | 'weekly' | 'monthly' | 'yearly';
  recurrenceInterval?: number;
  recurrenceDaysOfWeek?: number[];
  recurrenceEndDate?: string;
  isActive: boolean;
  executions?: EventExecution[];
  // For recurring event occurrences (expanded by backend)
  originalEventId?: string; // The ID of the original recurring event
  occurrenceDate?: string;  // The specific date (YYYY-MM-DD) this occurrence represents
}
