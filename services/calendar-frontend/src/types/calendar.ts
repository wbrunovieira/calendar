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

export interface Label {
  id: string;
  calendarId: string;
  name: string;
  color: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface EventExecution {
  id: string;
  eventId: string;
  executionDate: string;
  completed: boolean;
  notes?: string;
}

export type EventType = 'EVENT' | 'HABIT' | 'TODO' | 'REMINDER';

export interface Event {
  id: string;
  calendarId: string;
  categoryId?: string;
  categoryTypeId?: string;
  labelId?: string;
  category?: Category; // Category object from API (includes icon, color, name, type)
  categoryType?: CategoryType; // CategoryType object from API
  label?: Label; // Label object from API
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
  // Habit/Todo fields
  eventType?: EventType;
  priority?: number; // 1=high, 2=medium, 3=low
  dueDate?: string;
  displayOrder?: number;
  // Flexible weekly habit fields
  recurrenceType?: 'FIXED' | 'FLEXIBLE';
  weeklyTargetCount?: number; // Target times per week (e.g., 2)
  weeklyPreferredDays?: string[]; // Preferred days ["MO", "TU", "WE", "TH", "FR", "SA", "SU"]
  // Reminder fields
  reminderDaysBefore?: number[]; // Days before to remind (e.g., [7, 3, 1, 0])
  // Event alert reminders (minutesBefore-based)
  reminders?: EventReminder[];
}

export interface EventReminder {
  id: string;
  eventId: string;
  minutesBefore: number;
  method: string;
}

export interface ReminderInput {
  minutesBefore: number;
  method?: string;
}

export interface HabitStats {
  habitId: string;
  habitTitle: string;
  currentStreak: number;
  longestStreak: number;
  weeklyCompletions: { date: string; completed: boolean }[];
  monthlyCompletionRate: number;
  totalCompletions: number;
}

export interface WeekProgress {
  weekStartDate: string; // YYYY-MM-DD (Monday)
  targetCount: number;
  completedCount: number;
  completedDates: string[];
  isGoalMet: boolean;
  daysRemaining?: number; // Only for current week
}

export interface FlexibleHabitProgress {
  habitId: string;
  habitTitle: string;
  weeklyTargetCount: number;
  weeklyPreferredDays: string[];
  currentWeek: WeekProgress;
  weekHistory: WeekProgress[];
  currentStreak: number;
  longestStreak: number;
}
