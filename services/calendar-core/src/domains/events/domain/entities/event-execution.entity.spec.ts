import { describe, it, expect } from 'vitest';
import { EventExecution } from './event-execution.entity';

describe('EventExecution', () => {
  describe('constructor', () => {
    it('should map occurrenceDate from Prisma to executionDate', () => {
      // Arrange - This is how Prisma returns data (with occurrenceDate, not executionDate)
      const prismaData = {
        id: 'exec-123',
        eventId: 'event-456',
        occurrenceDate: new Date('2026-01-30'),
        completed: true,
        completedAt: new Date('2026-01-30T12:00:00Z'),
        notes: 'Test note',
        createdAt: new Date('2026-01-30T10:00:00Z'),
        updatedAt: new Date('2026-01-30T12:00:00Z'),
      };

      // Act
      const execution = new EventExecution(prismaData);

      // Assert
      expect(execution.executionDate).toEqual(new Date('2026-01-30'));
      expect(execution.id).toBe('exec-123');
      expect(execution.eventId).toBe('event-456');
      expect(execution.completed).toBe(true);
    });

    it('should also accept executionDate directly for backwards compatibility', () => {
      // Arrange - Direct usage with executionDate
      const directData = {
        id: 'exec-789',
        eventId: 'event-101',
        executionDate: new Date('2026-01-31'),
        completed: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // Act
      const execution = new EventExecution(directData);

      // Assert
      expect(execution.executionDate).toEqual(new Date('2026-01-31'));
    });
  });

  describe('markAsCompleted', () => {
    it('should set completed to true and set completedAt', () => {
      const execution = new EventExecution({
        id: 'exec-1',
        eventId: 'event-1',
        occurrenceDate: new Date('2026-01-30'),
        completed: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      });

      execution.markAsCompleted();

      expect(execution.completed).toBe(true);
      expect(execution.completedAt).toBeInstanceOf(Date);
    });

    it('should set notes when provided', () => {
      const execution = new EventExecution({
        id: 'exec-1',
        eventId: 'event-1',
        occurrenceDate: new Date('2026-01-30'),
        completed: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      });

      execution.markAsCompleted('Completed with notes');

      expect(execution.notes).toBe('Completed with notes');
    });
  });

  describe('markAsIncomplete', () => {
    it('should set completed to false and clear completedAt', () => {
      const execution = new EventExecution({
        id: 'exec-1',
        eventId: 'event-1',
        occurrenceDate: new Date('2026-01-30'),
        completed: true,
        completedAt: new Date(),
        createdAt: new Date(),
        updatedAt: new Date(),
      });

      execution.markAsIncomplete();

      expect(execution.completed).toBe(false);
      expect(execution.completedAt).toBeUndefined();
    });
  });
});
