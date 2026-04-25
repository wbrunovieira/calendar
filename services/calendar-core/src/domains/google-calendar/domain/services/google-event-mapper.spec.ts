import { describe, it, expect } from 'vitest';
import { GoogleEventMapper } from './google-event-mapper';
import { Event } from '@domains/events/domain/entities/event.entity';

function makeEvent(overrides: Partial<Event> = {}): Event {
  return Object.assign(new Event({}), {
    id: 'evt-1',
    calendarId: 'cal-1',
    title: 'Daily Standup',
    description: 'Team sync',
    startDate: new Date('2024-01-15'),
    endDate: null,
    startTime: '09:00',
    endTime: '09:30',
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

describe('GoogleEventMapper', () => {
  describe('toGoogleEvent', () => {
    it('should map a simple timed event', () => {
      const event = makeEvent();
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.summary).toBe('Daily Standup');
      expect(result.description).toBe('Team sync');
      expect(result.start).toEqual({ dateTime: '2024-01-15T09:00:00', timeZone: 'America/Sao_Paulo' });
      expect(result.end).toEqual({ dateTime: '2024-01-15T09:30:00', timeZone: 'America/Sao_Paulo' });
    });

    it('should map an all-day event (no endTime)', () => {
      const event = makeEvent({ endTime: null });
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.start).toEqual({ date: '2024-01-15' });
      expect(result.end).toEqual({ date: '2024-01-15' });
    });

    it('should map a recurring event with RRULE', () => {
      const event = makeEvent({ recurrenceRule: 'FREQ=WEEKLY;BYDAY=MO,WE,FR' });
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.recurrence).toEqual(['RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR']);
    });

    it('should map reminders to Google overrides format', () => {
      const event = makeEvent({
        reminders: [
          { id: 'r1', minutesBefore: 30, method: 'notification' },
          { id: 'r2', minutesBefore: 1440, method: 'email' },
        ],
      });
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.reminders).toEqual({
        useDefault: false,
        overrides: [
          { method: 'popup', minutes: 30 },
          { method: 'email', minutes: 1440 },
        ],
      });
    });

    it('should use default reminders when event has none', () => {
      const event = makeEvent({ reminders: [] });
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.reminders).toEqual({ useDefault: true });
    });

    it('should map CANCELLED status', () => {
      const event = makeEvent({ status: 'CANCELLED' });
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.status).toBe('cancelled');
    });

    it('should not include recurrence when event has none', () => {
      const event = makeEvent({ recurrenceRule: null });
      const result = GoogleEventMapper.toGoogleEvent(event);

      expect(result.recurrence).toBeUndefined();
    });
  });

  describe('fromGoogleEvent', () => {
    it('should map a Google timed event to local format', () => {
      const googleEvent = {
        id: 'google-evt-123',
        summary: 'Meeting',
        description: 'Weekly sync',
        start: { dateTime: '2024-01-15T14:00:00-03:00' },
        end: { dateTime: '2024-01-15T15:00:00-03:00' },
        status: 'confirmed',
      };

      const result = GoogleEventMapper.fromGoogleEvent(googleEvent, 'cal-1');

      expect(result.googleEventId).toBe('google-evt-123');
      expect(result.calendarId).toBe('cal-1');
      expect(result.title).toBe('Meeting');
      expect(result.startTime).toBe('14:00');
      expect(result.endTime).toBe('15:00');
      expect(result.status).toBe('CONFIRMED');
      expect(result.eventType).toBe('EVENT');
    });

    it('should map a Google all-day event', () => {
      const googleEvent = {
        id: 'google-allday-1',
        summary: 'Holiday',
        start: { date: '2024-12-25' },
        end: { date: '2024-12-25' },
        status: 'confirmed',
      };

      const result = GoogleEventMapper.fromGoogleEvent(googleEvent, 'cal-1');

      expect(result.startTime).toBe('00:00');
      expect(result.endTime).toBeNull();
      expect(result.startDate).toEqual(new Date('2024-12-25'));
    });

    it('should extract RRULE from Google recurrence array', () => {
      const googleEvent = {
        id: 'google-recur-1',
        summary: 'Weekly Review',
        start: { dateTime: '2024-01-15T10:00:00-03:00' },
        end: { dateTime: '2024-01-15T11:00:00-03:00' },
        recurrence: ['RRULE:FREQ=WEEKLY;BYDAY=MO'],
        status: 'confirmed',
      };

      const result = GoogleEventMapper.fromGoogleEvent(googleEvent, 'cal-1');

      expect(result.recurrenceRule).toBe('FREQ=WEEKLY;BYDAY=MO');
    });

    it('should map cancelled Google event to CANCELLED status', () => {
      const googleEvent = {
        id: 'google-cancelled-1',
        summary: 'Deleted event',
        start: { dateTime: '2024-01-15T10:00:00-03:00' },
        end: { dateTime: '2024-01-15T11:00:00-03:00' },
        status: 'cancelled',
      };

      const result = GoogleEventMapper.fromGoogleEvent(googleEvent, 'cal-1');

      expect(result.status).toBe('CANCELLED');
    });

    it('should use fallback title for events without summary', () => {
      const googleEvent = {
        id: 'google-notitle-1',
        start: { dateTime: '2024-01-15T10:00:00-03:00' },
        end: { dateTime: '2024-01-15T11:00:00-03:00' },
        status: 'confirmed',
      };

      const result = GoogleEventMapper.fromGoogleEvent(googleEvent, 'cal-1');

      expect(result.title).toBe('(sem título)');
    });
  });
});
