import { describe, it, expect, beforeEach, vi } from 'vitest';
import { UpdateEventUseCase } from './update-event.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';

describe('UpdateEventUseCase', () => {
  let useCase: UpdateEventUseCase;
  let mockEventRepository: Partial<EventRepository>;

  const existingEvent = {
    id: 'event-123',
    calendarId: 'calendar-123',
    title: 'Original Title',
    eventType: 'TODO',
    priority: null,
    dueDate: null,
    startDate: new Date('2024-01-15'),
    recurrenceRule: null,
  };

  beforeEach(() => {
    mockEventRepository = {
      findById: vi.fn().mockResolvedValue(existingEvent),
      update: vi.fn().mockImplementation(async (id, data) => ({ ...existingEvent, ...data })),
    };

    useCase = new UpdateEventUseCase(mockEventRepository as EventRepository);
  });

  describe('dueDate handling', () => {
    it('should update dueDate when provided', async () => {
      const result = await useCase.execute('event-123', {
        dueDate: '2024-01-28',
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.objectContaining({
          dueDate: expect.any(Date),
        }),
        undefined,
      );
      expect(result.dueDate).toBeInstanceOf(Date);
      expect(result.dueDate?.getFullYear()).toBe(2024);
      expect(result.dueDate?.getMonth()).toBe(0); // January
      expect(result.dueDate?.getDate()).toBe(28);
    });

    it('should clear dueDate when null is provided', async () => {
      const eventWithDueDate = {
        ...existingEvent,
        dueDate: new Date('2024-01-20'),
      };
      vi.mocked(mockEventRepository.findById!).mockResolvedValue(eventWithDueDate);

      await useCase.execute('event-123', {
        dueDate: null as unknown as string,
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.objectContaining({
          dueDate: null,
        }),
        undefined,
      );
    });

    it('should not update dueDate when not provided', async () => {
      await useCase.execute('event-123', {
        title: 'New Title',
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.not.objectContaining({
          dueDate: expect.anything(),
        }),
        undefined,
      );
    });
  });

  describe('priority handling', () => {
    it('should update priority when provided', async () => {
      const result = await useCase.execute('event-123', {
        priority: 1,
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.objectContaining({
          priority: 1,
        }),
        undefined,
      );
      expect(result.priority).toBe(1);
    });

    it('should clear priority when null is provided', async () => {
      const eventWithPriority = {
        ...existingEvent,
        priority: 2,
      };
      vi.mocked(mockEventRepository.findById!).mockResolvedValue(eventWithPriority);

      await useCase.execute('event-123', {
        priority: null as unknown as number,
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.objectContaining({
          priority: null,
        }),
        undefined,
      );
    });

    it('should update to high priority (1)', async () => {
      const result = await useCase.execute('event-123', {
        priority: 1,
      });

      expect(result.priority).toBe(1);
    });

    it('should update to medium priority (2)', async () => {
      const result = await useCase.execute('event-123', {
        priority: 2,
      });

      expect(result.priority).toBe(2);
    });

    it('should update to low priority (3)', async () => {
      const result = await useCase.execute('event-123', {
        priority: 3,
      });

      expect(result.priority).toBe(3);
    });
  });

  describe('combined TODO update', () => {
    it('should update both dueDate and priority together', async () => {
      const result = await useCase.execute('event-123', {
        dueDate: '2024-02-15',
        priority: 2,
        title: 'Updated Task',
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.objectContaining({
          dueDate: expect.any(Date),
          priority: 2,
          title: 'Updated Task',
        }),
        undefined,
      );
      expect(result.dueDate?.getDate()).toBe(15);
      expect(result.dueDate?.getMonth()).toBe(1); // February
      expect(result.priority).toBe(2);
    });
  });

  describe('event reminders (alerts)', () => {
    it('should pass reminders to repository when provided', async () => {
      await useCase.execute('event-123', {
        title: 'Updated',
        reminders: [{ minutesBefore: 30 }, { minutesBefore: 10, method: 'notification' }],
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.objectContaining({ title: 'Updated' }),
        [{ minutesBefore: 30 }, { minutesBefore: 10, method: 'notification' }],
      );
    });

    it('should pass undefined reminders when not in dto', async () => {
      await useCase.execute('event-123', {
        title: 'Updated',
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.any(Object),
        undefined,
      );
    });

    it('should pass empty array to clear all reminders', async () => {
      await useCase.execute('event-123', {
        reminders: [],
      });

      expect(mockEventRepository.update).toHaveBeenCalledWith(
        'event-123',
        expect.any(Object),
        [],
      );
    });
  });
});
