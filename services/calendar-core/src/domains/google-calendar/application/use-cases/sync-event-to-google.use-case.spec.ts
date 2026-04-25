import { describe, it, expect, beforeEach, vi } from 'vitest';
import { SyncEventToGoogleUseCase } from './sync-event-to-google.use-case';
import { Event } from '@domains/events/domain/entities/event.entity';
import { Calendar } from '@domains/calendars/domain/entities/calendar.entity';

const mockApiClient = {
  createEvent: vi.fn(),
  updateEvent: vi.fn(),
  deleteEvent: vi.fn(),
};

function makeCalendar(overrides: Partial<Calendar> = {}): Calendar {
  return Calendar.create({
    id: 'cal-1',
    userId: 'user-1',
    name: 'WB Digital Solutions',
    color: '#0077B5',
    type: 'professional',
    email: 'bruno@wbdigitalsolutions.com',
    googleCalendarId: 'primary',
    ...overrides,
  });
}

function makeEvent(overrides: Partial<Event> = {}): Event {
  return Object.assign(new Event({}), {
    id: 'evt-1',
    calendarId: 'cal-1',
    title: 'Team Meeting',
    startDate: new Date('2024-01-15'),
    startTime: '09:00',
    endTime: '10:00',
    endDate: null,
    recurrenceRule: null,
    status: 'CONFIRMED',
    eventType: 'EVENT',
    googleEventId: null,
    reminders: [],
    isActive: true,
    createdAt: new Date(),
    updatedAt: new Date(),
    ...overrides,
  });
}

describe('SyncEventToGoogleUseCase', () => {
  let useCase: SyncEventToGoogleUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new SyncEventToGoogleUseCase(mockApiClient as any);
  });

  describe('onCreate', () => {
    it('should create event in Google and return googleEventId', async () => {
      const event = makeEvent();
      const calendar = makeCalendar();
      mockApiClient.createEvent.mockResolvedValue('google-evt-abc');

      const result = await useCase.onCreate(event, calendar);

      expect(result).toBe('google-evt-abc');
      expect(mockApiClient.createEvent).toHaveBeenCalledWith(
        'primary',
        'bruno@wbdigitalsolutions.com',
        expect.objectContaining({ summary: 'Team Meeting' }),
      );
    });

    it('should return null when calendar has no googleCalendarId', async () => {
      const event = makeEvent();
      const calendar = makeCalendar({ googleCalendarId: null });

      const result = await useCase.onCreate(event, calendar);

      expect(result).toBeNull();
      expect(mockApiClient.createEvent).not.toHaveBeenCalled();
    });

    it('should return null when calendar has no email', async () => {
      const event = makeEvent();
      const calendar = makeCalendar({ email: null });

      const result = await useCase.onCreate(event, calendar);

      expect(result).toBeNull();
      expect(mockApiClient.createEvent).not.toHaveBeenCalled();
    });

    it('should return null and log error when API call fails', async () => {
      const event = makeEvent();
      const calendar = makeCalendar();
      mockApiClient.createEvent.mockRejectedValue(new Error('API quota exceeded'));

      const result = await useCase.onCreate(event, calendar);

      expect(result).toBeNull();
    });
  });

  describe('onUpdate', () => {
    it('should update event in Google when googleEventId is set', async () => {
      const event = makeEvent({ googleEventId: 'google-evt-123' });
      const calendar = makeCalendar();
      mockApiClient.updateEvent.mockResolvedValue(undefined);

      await useCase.onUpdate(event, calendar);

      expect(mockApiClient.updateEvent).toHaveBeenCalledWith(
        'primary',
        'google-evt-123',
        'bruno@wbdigitalsolutions.com',
        expect.objectContaining({ summary: 'Team Meeting' }),
      );
    });

    it('should skip update when event has no googleEventId', async () => {
      const event = makeEvent({ googleEventId: null });
      const calendar = makeCalendar();

      await useCase.onUpdate(event, calendar);

      expect(mockApiClient.updateEvent).not.toHaveBeenCalled();
    });

    it('should not throw when API call fails', async () => {
      const event = makeEvent({ googleEventId: 'google-evt-123' });
      const calendar = makeCalendar();
      mockApiClient.updateEvent.mockRejectedValue(new Error('Network error'));

      await expect(useCase.onUpdate(event, calendar)).resolves.not.toThrow();
    });
  });

  describe('onDelete', () => {
    it('should delete event in Google when googleEventId and calendar are set', async () => {
      const calendar = makeCalendar();
      mockApiClient.deleteEvent.mockResolvedValue(undefined);

      await useCase.onDelete('google-evt-123', calendar);

      expect(mockApiClient.deleteEvent).toHaveBeenCalledWith(
        'primary',
        'google-evt-123',
        'bruno@wbdigitalsolutions.com',
      );
    });

    it('should skip delete when calendar has no googleCalendarId', async () => {
      const calendar = makeCalendar({ googleCalendarId: null });

      await useCase.onDelete('google-evt-123', calendar);

      expect(mockApiClient.deleteEvent).not.toHaveBeenCalled();
    });

    it('should not throw when API call fails', async () => {
      const calendar = makeCalendar();
      mockApiClient.deleteEvent.mockRejectedValue(new Error('Event not found'));

      await expect(useCase.onDelete('google-evt-123', calendar)).resolves.not.toThrow();
    });
  });
});
