import { describe, it, expect } from 'vitest';
import { EventReminder } from './event-reminder.entity';

describe('EventReminder', () => {
  it('should create a reminder with all fields', () => {
    const reminder = EventReminder.create({
      eventId: 'event-123',
      minutesBefore: 30,
      method: 'notification',
    });

    expect(reminder.eventId).toBe('event-123');
    expect(reminder.minutesBefore).toBe(30);
    expect(reminder.method).toBe('notification');
    expect(reminder.id).toBe('');
    expect(reminder.createdAt).toBeInstanceOf(Date);
  });

  it('should default method to notification', () => {
    const reminder = EventReminder.create({
      eventId: 'event-123',
      minutesBefore: 15,
    });

    expect(reminder.method).toBe('notification');
  });

  it('should accept minutesBefore of 0 (at event time)', () => {
    const reminder = EventReminder.create({
      eventId: 'event-123',
      minutesBefore: 0,
    });

    expect(reminder.minutesBefore).toBe(0);
  });

  it('should accept large minutesBefore (1 day = 1440)', () => {
    const reminder = EventReminder.create({
      eventId: 'event-123',
      minutesBefore: 1440,
    });

    expect(reminder.minutesBefore).toBe(1440);
  });

  it('should accept email method', () => {
    const reminder = EventReminder.create({
      eventId: 'event-123',
      minutesBefore: 60,
      method: 'email',
    });

    expect(reminder.method).toBe('email');
  });

  it('should create via constructor with existing data', () => {
    const reminder = new EventReminder({
      id: 'reminder-456',
      eventId: 'event-123',
      minutesBefore: 10,
      method: 'notification',
      createdAt: new Date('2024-01-15'),
    });

    expect(reminder.id).toBe('reminder-456');
    expect(reminder.eventId).toBe('event-123');
    expect(reminder.minutesBefore).toBe(10);
    expect(reminder.method).toBe('notification');
    expect(reminder.createdAt).toEqual(new Date('2024-01-15'));
  });
});
