import { randomUUID } from 'crypto';

/**
 * Test Fixtures for Calendar Core
 *
 * These fixtures provide reusable test data for unit and integration tests.
 */

// Fixed date for deterministic tests
export const FIXED_DATE = new Date('2024-11-16T10:00:00-03:00');

/**
 * Returns a fixed time for deterministic tests
 */
export function fixedTime(): Date {
  return FIXED_DATE;
}

/**
 * User fixtures
 */
export const createUserFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  email: 'test@example.com',
  name: 'Test User',
  timezone: 'America/Sao_Paulo',
  createdAt: fixedTime(),
  updatedAt: fixedTime(),
  ...overrides,
});

/**
 * Calendar fixtures
 */
export const createCalendarFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  name: 'Test Calendar',
  description: 'Calendar for testing',
  userId: randomUUID(),
  type: 'PROFESSIONAL',
  color: '#3B82F6',
  isActive: true,
  createdAt: fixedTime(),
  updatedAt: fixedTime(),
  ...overrides,
});

/**
 * Category Type fixtures
 */
export const createCategoryTypeFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  name: 'WORK',
  calendarId: randomUUID(),
  createdAt: fixedTime(),
  updatedAt: fixedTime(),
  ...overrides,
});

/**
 * Category fixtures
 */
export const createCategoryFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  name: 'Meetings',
  description: 'Work meetings',
  calendarId: randomUUID(),
  color: '#10B981',
  isActive: true,
  createdAt: fixedTime(),
  updatedAt: fixedTime(),
  ...overrides,
});

/**
 * Event fixtures
 */
export const createEventFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  title: 'Test Event',
  description: 'Event for testing',
  calendarId: randomUUID(),
  categoryId: null,
  categoryTypeId: null,
  startDate: '2024-11-16',
  startTime: '10:00',
  endTime: '11:00',
  duration: 60,
  location: null,
  status: 'CONFIRMED',
  recurrenceRule: null,
  recurrenceMasterId: null,
  createdAt: fixedTime(),
  updatedAt: fixedTime(),
  ...overrides,
});

/**
 * Recurring Event fixtures
 */
export const createRecurringEventFixture = (overrides?: Partial<any>) => ({
  ...createEventFixture(),
  recurrenceRule: 'FREQ=DAILY;INTERVAL=1',
  ...overrides,
});

/**
 * Event Completion fixtures
 */
export const createEventCompletionFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  eventId: randomUUID(),
  occurrenceDate: '2024-11-16',
  isCompleted: true,
  completedAt: fixedTime(),
  notes: null,
  createdAt: fixedTime(),
  updatedAt: fixedTime(),
  ...overrides,
});

/**
 * Recurrence Exception fixtures
 */
export const createRecurrenceExceptionFixture = (overrides?: Partial<any>) => ({
  id: randomUUID(),
  eventId: randomUUID(),
  exceptionDate: '2024-11-16',
  createdAt: fixedTime(),
  ...overrides,
});

/**
 * RRule options for testing
 */
export const RRULE_DAILY = 'FREQ=DAILY;INTERVAL=1';
export const RRULE_WEEKLY = 'FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR';
export const RRULE_MONTHLY = 'FREQ=MONTHLY;INTERVAL=1;BYMONTHDAY=15';
export const RRULE_YEARLY = 'FREQ=YEARLY;INTERVAL=1';

/**
 * Date range helpers for testing
 */
export const dateRange = {
  november2024: {
    start: '2024-11-01',
    end: '2024-11-30',
  },
  week: {
    start: '2024-11-11',
    end: '2024-11-17',
  },
  singleDay: {
    start: '2024-11-16',
    end: '2024-11-16',
  },
};

/**
 * Mock DTOs for testing
 */
export const createEventDTO = {
  title: 'Test Event',
  description: 'Event description',
  calendarId: randomUUID(),
  startDate: '2024-11-16',
  startTime: '10:00',
  duration: 60,
};

export const updateEventDTO = {
  title: 'Updated Event',
  description: 'Updated description',
};

export const createRecurringEventDTO = {
  ...createEventDTO,
  recurrenceRule: RRULE_DAILY,
};
