import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ListEventsUseCase } from './list-events.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { Event } from '../../domain/entities/event.entity';

describe('ListEventsUseCase', () => {
  let useCase: ListEventsUseCase;
  let mockEventRepository: Partial<EventRepository>;

  beforeEach(() => {
    mockEventRepository = {
      findAll: vi.fn(),
    };

    useCase = new ListEventsUseCase(mockEventRepository as EventRepository);
  });

  const createEvent = (overrides: Record<string, any> = {}): Event => {
    return new Event({
      id: 'event-1',
      calendarId: 'calendar-1',
      title: 'Test Event',
      startTime: '09:00',
      startDate: new Date(2024, 0, 15),
      eventType: 'EVENT',
      status: 'CONFIRMED',
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
      ...overrides,
    });
  };

  describe('eventType filter', () => {
    it('should filter by eventType HABIT', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({ eventType: 'HABIT' });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          eventType: 'HABIT',
        }),
      );
    });

    it('should filter by eventType TODO', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({ eventType: 'TODO' });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          eventType: 'TODO',
        }),
      );
    });

    it('should filter by eventType EVENT', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({ eventType: 'EVENT' });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          eventType: 'EVENT',
        }),
      );
    });

    it('should not filter by eventType when not provided', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({});

      expect(mockEventRepository.findAll).toHaveBeenCalledWith({});
    });
  });

  describe('eventType in returned events', () => {
    it('should include eventType in returned events', async () => {
      const habit = createEvent({ eventType: 'HABIT', title: 'Daily Meditation' });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({});

      expect(result[0].eventType).toBe('HABIT');
    });

    it('should include priority in returned TODO events', async () => {
      const todo = createEvent({
        eventType: 'TODO',
        title: 'Organize cables',
        priority: 1,
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([todo]);

      const result = await useCase.execute({});

      expect(result[0].eventType).toBe('TODO');
      expect(result[0].priority).toBe(1);
    });

    it('should include dueDate in returned TODO events', async () => {
      const dueDate = new Date(2024, 0, 20);
      const todo = createEvent({
        eventType: 'TODO',
        title: 'Organize cables',
        dueDate,
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([todo]);

      const result = await useCase.execute({});

      expect(result[0].eventType).toBe('TODO');
      expect(result[0].dueDate).toEqual(dueDate);
    });
  });

  describe('recurring habits expansion', () => {
    it('should expand recurring HABIT within date range', async () => {
      const habit = createEvent({
        eventType: 'HABIT',
        title: 'Daily Meditation',
        recurrenceRule: 'FREQ=DAILY',
        startDate: new Date(2024, 0, 1),
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({
        startDate: '2024-01-10',
        endDate: '2024-01-12',
      });

      // Should have 3 occurrences (Jan 10, 11, 12)
      expect(result).toHaveLength(3);
      result.forEach((occurrence) => {
        expect(occurrence.eventType).toBe('HABIT');
        expect(occurrence.originalEventId).toBe('event-1');
      });
    });

    it('should preserve eventType in expanded occurrences', async () => {
      const habit = createEvent({
        eventType: 'HABIT',
        title: 'Weekly Exercise',
        recurrenceRule: 'FREQ=WEEKLY;BYDAY=MO',
        startDate: new Date(2024, 0, 1), // Monday
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({
        startDate: '2024-01-08', // Monday
        endDate: '2024-01-14', // Sunday
      });

      // Should have 1 occurrence (Monday Jan 8)
      expect(result.length).toBeGreaterThanOrEqual(1);
      expect(result[0].eventType).toBe('HABIT');
    });

    it('should preserve priority in expanded TODO occurrences', async () => {
      const todo = createEvent({
        eventType: 'TODO',
        title: 'Daily Task',
        priority: 2,
        recurrenceRule: 'FREQ=DAILY',
        startDate: new Date(2024, 0, 1),
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([todo]);

      const result = await useCase.execute({
        startDate: '2024-01-10',
        endDate: '2024-01-10',
      });

      expect(result[0].eventType).toBe('TODO');
      expect(result[0].priority).toBe(2);
    });
  });

  describe('combined filters', () => {
    it('should filter by calendarId and eventType', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({
        calendarId: 'calendar-123',
        eventType: 'HABIT',
      });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          calendarId: 'calendar-123',
          eventType: 'HABIT',
        }),
      );
    });

    it('should filter by categoryId and eventType', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({
        categoryId: 'category-456',
        eventType: 'TODO',
      });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          categoryId: 'category-456',
          eventType: 'TODO',
        }),
      );
    });

    it('should filter by date range and eventType', async () => {
      const habit = createEvent({ eventType: 'HABIT' });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      await useCase.execute({
        startDate: '2024-01-01',
        endDate: '2024-01-31',
        eventType: 'HABIT',
      });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          eventType: 'HABIT',
          startDate: '2024-01-01',
          endDate: '2024-01-31',
        }),
      );
    });
  });

  describe('daily habits on future dates', () => {
    it('should show daily habit created today when viewing tomorrow', async () => {
      // Scenario: User creates a daily habit on Jan 26
      // When viewing Jan 27, the habit should appear
      const habit = createEvent({
        eventType: 'HABIT',
        title: 'Daily Reading',
        recurrenceRule: 'FREQ=DAILY',
        startDate: new Date(2026, 0, 26), // Jan 26, 2026 (creation date)
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // View tomorrow (Jan 27)
      const result = await useCase.execute({
        startDate: '2026-01-27',
        endDate: '2026-01-27',
      });

      // Should have 1 occurrence for Jan 27
      expect(result).toHaveLength(1);
      expect(result[0].occurrenceDate).toBe('2026-01-27');
      expect(result[0].eventType).toBe('HABIT');
    });

    it('should show daily habit created today when viewing multiple future days', async () => {
      // Scenario: User creates a daily habit on Jan 26
      // When viewing Jan 27-29, all 3 days should show the habit
      const habit = createEvent({
        eventType: 'HABIT',
        title: 'Daily Meditation',
        recurrenceRule: 'FREQ=DAILY',
        startDate: new Date(2026, 0, 26), // Jan 26, 2026
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({
        startDate: '2026-01-27',
        endDate: '2026-01-29',
      });

      // Should have 3 occurrences (Jan 27, 28, 29)
      expect(result).toHaveLength(3);
      expect(result.map(r => r.occurrenceDate)).toEqual([
        '2026-01-27',
        '2026-01-28',
        '2026-01-29',
      ]);
    });

    it('should NOT show daily habit when viewing days before creation', async () => {
      // Scenario: User creates a daily habit on Jan 26
      // When viewing Jan 25, the habit should NOT appear
      const habit = createEvent({
        eventType: 'HABIT',
        title: 'Daily Reading',
        recurrenceRule: 'FREQ=DAILY',
        startDate: new Date(2026, 0, 26), // Jan 26, 2026
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({
        startDate: '2026-01-25',
        endDate: '2026-01-25',
      });

      // Should have 0 occurrences (habit didn't exist on Jan 25)
      expect(result).toHaveLength(0);
    });

    it('should show daily habit on creation day and future days', async () => {
      // Scenario: User creates a daily habit on Jan 26
      // When viewing Jan 26-28, all 3 days should show the habit
      const habit = createEvent({
        eventType: 'HABIT',
        title: 'Daily Exercise',
        recurrenceRule: 'FREQ=DAILY',
        startDate: new Date(2026, 0, 26), // Jan 26, 2026
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({
        startDate: '2026-01-26',
        endDate: '2026-01-28',
      });

      // Should have 3 occurrences (Jan 26, 27, 28)
      expect(result).toHaveLength(3);
      expect(result.map(r => r.occurrenceDate)).toEqual([
        '2026-01-26',
        '2026-01-27',
        '2026-01-28',
      ]);
    });
  });

  describe('executions handling', () => {
    it('should include executions for habits', async () => {
      const habit = createEvent({
        eventType: 'HABIT',
        executions: [
          { id: 'exec-1', eventId: 'event-1', completed: true },
        ],
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({});

      expect(result[0].executions).toBeDefined();
      expect(result[0].executions).toHaveLength(1);
    });

    it('should return empty executions when none exist', async () => {
      const habit = createEvent({ eventType: 'HABIT' });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const result = await useCase.execute({});

      expect(result[0].executions).toEqual([]);
    });
  });
});
