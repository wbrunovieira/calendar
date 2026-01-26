import { describe, it, expect } from 'vitest';
import { Event } from './event.entity';

describe('Event Entity', () => {
  const baseEventData = {
    calendarId: 'calendar-123',
    title: 'Test Event',
    startTime: '09:00',
    startDate: new Date('2024-01-15'),
  };

  describe('create', () => {
    it('should create an event with default values', () => {
      const event = Event.create(baseEventData);

      expect(event.calendarId).toBe('calendar-123');
      expect(event.title).toBe('Test Event');
      expect(event.startTime).toBe('09:00');
      expect(event.eventType).toBe('EVENT');
      expect(event.priority).toBeNull();
      expect(event.dueDate).toBeNull();
      expect(event.status).toBe('CONFIRMED');
      expect(event.isActive).toBe(true);
    });

    it('should create a HABIT event type', () => {
      const event = Event.create({
        ...baseEventData,
        eventType: 'HABIT',
      });

      expect(event.eventType).toBe('HABIT');
      expect(event.priority).toBeNull();
    });

    it('should create a TODO event type with priority', () => {
      const event = Event.create({
        ...baseEventData,
        eventType: 'TODO',
        priority: 1,
      });

      expect(event.eventType).toBe('TODO');
      expect(event.priority).toBe(1);
    });

    it('should create a TODO with due date', () => {
      const dueDate = new Date('2024-01-20');
      const event = Event.create({
        ...baseEventData,
        eventType: 'TODO',
        priority: 2,
        dueDate,
      });

      expect(event.eventType).toBe('TODO');
      expect(event.priority).toBe(2);
      expect(event.dueDate).toEqual(dueDate);
    });

    it('should create a recurring HABIT with recurrence rule', () => {
      const event = Event.create({
        ...baseEventData,
        eventType: 'HABIT',
        recurrenceRule: 'FREQ=DAILY;INTERVAL=1',
      });

      expect(event.eventType).toBe('HABIT');
      expect(event.recurrenceRule).toBe('FREQ=DAILY;INTERVAL=1');
    });

    it('should handle all priority levels', () => {
      const highPriority = Event.create({ ...baseEventData, eventType: 'TODO', priority: 1 });
      const mediumPriority = Event.create({ ...baseEventData, eventType: 'TODO', priority: 2 });
      const lowPriority = Event.create({ ...baseEventData, eventType: 'TODO', priority: 3 });

      expect(highPriority.priority).toBe(1);
      expect(mediumPriority.priority).toBe(2);
      expect(lowPriority.priority).toBe(3);
    });

    it('should set null for undefined optional fields', () => {
      const event = Event.create(baseEventData);

      expect(event.categoryId).toBeUndefined();
      expect(event.categoryTypeId).toBeUndefined();
      expect(event.description).toBeUndefined();
      expect(event.endTime).toBeUndefined();
      expect(event.endDate).toBeUndefined();
      expect(event.recurrenceRule).toBeNull();
      expect(event.recurrenceMasterId).toBeNull();
      expect(event.googleEventId).toBeNull();
    });

    it('should set timestamps on creation', () => {
      const beforeCreate = new Date();
      const event = Event.create(baseEventData);
      const afterCreate = new Date();

      expect(event.createdAt.getTime()).toBeGreaterThanOrEqual(beforeCreate.getTime());
      expect(event.createdAt.getTime()).toBeLessThanOrEqual(afterCreate.getTime());
      expect(event.updatedAt.getTime()).toBeGreaterThanOrEqual(beforeCreate.getTime());
      expect(event.updatedAt.getTime()).toBeLessThanOrEqual(afterCreate.getTime());
    });
  });

  describe('constructor', () => {
    it('should create event from raw data', () => {
      const rawData = {
        id: 'event-123',
        calendarId: 'calendar-123',
        title: 'Raw Event',
        startTime: '10:00',
        startDate: new Date('2024-01-15'),
        eventType: 'HABIT',
        priority: null,
        dueDate: null,
        status: 'CONFIRMED',
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      const event = new Event(rawData);

      expect(event.id).toBe('event-123');
      expect(event.title).toBe('Raw Event');
      expect(event.eventType).toBe('HABIT');
    });
  });
});
