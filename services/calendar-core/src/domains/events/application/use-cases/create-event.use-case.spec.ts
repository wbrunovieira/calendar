import { describe, it, expect, beforeEach, vi } from 'vitest';
import { CreateEventUseCase } from './create-event.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { Event } from '../../domain/entities/event.entity';

describe('CreateEventUseCase', () => {
  let useCase: CreateEventUseCase;
  let mockEventRepository: Partial<EventRepository>;

  beforeEach(() => {
    mockEventRepository = {
      create: vi.fn(),
    };

    useCase = new CreateEventUseCase(mockEventRepository as EventRepository);
  });

  const baseEventDto = {
    calendarId: 'calendar-123',
    title: 'Test Event',
    startTime: '09:00',
    startDate: '2024-01-15',
  };

  describe('eventType handling', () => {
    it('should create event with default eventType EVENT', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute(baseEventDto);

      expect(result.eventType).toBe('EVENT');
      expect(mockEventRepository.create).toHaveBeenCalledWith(
        expect.objectContaining({
          eventType: 'EVENT',
        }),
        undefined,
      );
    });

    it('should create HABIT event type', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
      });

      expect(result.eventType).toBe('HABIT');
    });

    it('should create TODO event type', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'TODO',
      });

      expect(result.eventType).toBe('TODO');
    });
  });

  describe('priority handling', () => {
    it('should set priority to null by default', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute(baseEventDto);

      expect(result.priority).toBeNull();
    });

    it('should set high priority (1) for TODO', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'TODO',
        priority: 1,
      });

      expect(result.priority).toBe(1);
    });

    it('should set medium priority (2) for TODO', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'TODO',
        priority: 2,
      });

      expect(result.priority).toBe(2);
    });

    it('should set low priority (3) for TODO', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'TODO',
        priority: 3,
      });

      expect(result.priority).toBe(3);
    });
  });

  describe('dueDate handling', () => {
    it('should set dueDate to null by default', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute(baseEventDto);

      expect(result.dueDate).toBeNull();
    });

    it('should parse and set dueDate for TODO', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'TODO',
        dueDate: '2024-01-20',
      });

      expect(result.dueDate).toBeInstanceOf(Date);
      expect(result.dueDate?.getFullYear()).toBe(2024);
      expect(result.dueDate?.getMonth()).toBe(0); // January
      expect(result.dueDate?.getDate()).toBe(20);
    });
  });

  describe('recurring HABIT creation', () => {
    it('should create daily recurring HABIT', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        isRecurring: true,
        recurrenceFrequency: 'daily',
        recurrenceInterval: 1,
      });

      expect(result.eventType).toBe('HABIT');
      expect(result.recurrenceRule).toBe('FREQ=DAILY');
    });

    it('should create weekly recurring HABIT', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        isRecurring: true,
        recurrenceFrequency: 'weekly',
        recurrenceInterval: 1,
        recurrenceDaysOfWeek: [1, 3, 5], // Mon, Wed, Fri
      });

      expect(result.eventType).toBe('HABIT');
      expect(result.recurrenceRule).toContain('FREQ=WEEKLY');
      expect(result.recurrenceRule).toContain('BYDAY=MO,WE,FR');
    });

    it('should create monthly recurring HABIT', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        isRecurring: true,
        recurrenceFrequency: 'monthly',
        recurrenceInterval: 1,
      });

      expect(result.eventType).toBe('HABIT');
      expect(result.recurrenceRule).toBe('FREQ=MONTHLY');
    });
  });

  describe('direct recurrenceRule handling', () => {
    it('should use recurrenceRule directly when provided (frontend HABIT creation)', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      // This is how the frontend creates HABITs - sending recurrenceRule directly
      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        recurrenceRule: 'FREQ=DAILY',
      });

      expect(result.eventType).toBe('HABIT');
      expect(result.recurrenceRule).toBe('FREQ=DAILY');
    });

    it('should use recurrenceRule directly for weekly habit', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        recurrenceRule: 'FREQ=WEEKLY',
      });

      expect(result.eventType).toBe('HABIT');
      expect(result.recurrenceRule).toBe('FREQ=WEEKLY');
    });

    it('should use recurrenceRule directly for monthly habit', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        recurrenceRule: 'FREQ=MONTHLY',
      });

      expect(result.eventType).toBe('HABIT');
      expect(result.recurrenceRule).toBe('FREQ=MONTHLY');
    });

    it('should prefer direct recurrenceRule over legacy format', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      // If both are provided, direct recurrenceRule should win
      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'HABIT',
        recurrenceRule: 'FREQ=DAILY',
        isRecurring: true,
        recurrenceFrequency: 'weekly', // This should be ignored
      });

      expect(result.recurrenceRule).toBe('FREQ=DAILY');
    });
  });

  describe('complete TODO creation', () => {
    it('should create TODO with all fields', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'TODO',
        priority: 1,
        dueDate: '2024-01-25',
        description: 'Organize cables',
        categoryId: 'category-456',
      });

      expect(result.eventType).toBe('TODO');
      expect(result.priority).toBe(1);
      expect(result.dueDate).toBeInstanceOf(Date);
      expect(result.description).toBe('Organize cables');
      expect(result.categoryId).toBe('category-456');
    });
  });

  describe('REMINDER eventType handling', () => {
    it('should create REMINDER event type', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'REMINDER',
      });

      expect(result.eventType).toBe('REMINDER');
    });

    it('should set reminderDaysBefore to null by default', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute(baseEventDto);

      expect(result.reminderDaysBefore).toBeNull();
    });

    it('should set reminderDaysBefore for REMINDER', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'REMINDER',
        reminderDaysBefore: [7, 3, 1, 0],
      });

      expect(result.eventType).toBe('REMINDER');
      expect(result.reminderDaysBefore).toEqual([7, 3, 1, 0]);
    });

    it('should create REMINDER with priority', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'REMINDER',
        priority: 1,
        reminderDaysBefore: [3, 1],
      });

      expect(result.eventType).toBe('REMINDER');
      expect(result.priority).toBe(1);
      expect(result.reminderDaysBefore).toEqual([3, 1]);
    });

    it('should create complete REMINDER with all fields', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      const result = await useCase.execute({
        ...baseEventDto,
        eventType: 'REMINDER',
        priority: 2,
        reminderDaysBefore: [7, 3, 1, 0],
        description: 'Cobrar depoimento do cliente',
        categoryId: 'category-456',
      });

      expect(result.eventType).toBe('REMINDER');
      expect(result.priority).toBe(2);
      expect(result.reminderDaysBefore).toEqual([7, 3, 1, 0]);
      expect(result.description).toBe('Cobrar depoimento do cliente');
      expect(result.categoryId).toBe('category-456');
    });
  });

  describe('event reminders (alerts)', () => {
    it('should pass reminders to repository when provided', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      await useCase.execute({
        ...baseEventDto,
        reminders: [
          { minutesBefore: 30, method: 'notification' },
          { minutesBefore: 10 },
        ],
      });

      expect(mockEventRepository.create).toHaveBeenCalledWith(
        expect.any(Object),
        [
          { minutesBefore: 30, method: 'notification' },
          { minutesBefore: 10 },
        ],
      );
    });

    it('should pass undefined reminders when not provided', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      await useCase.execute(baseEventDto);

      expect(mockEventRepository.create).toHaveBeenCalledWith(
        expect.any(Object),
        undefined,
      );
    });

    it('should pass empty array when reminders is empty', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      await useCase.execute({
        ...baseEventDto,
        reminders: [],
      });

      expect(mockEventRepository.create).toHaveBeenCalledWith(
        expect.any(Object),
        [],
      );
    });

    it('should pass single reminder (0 minutes = at event time)', async () => {
      vi.mocked(mockEventRepository.create!).mockImplementation(async (event) => event);

      await useCase.execute({
        ...baseEventDto,
        reminders: [{ minutesBefore: 0, method: 'notification' }],
      });

      expect(mockEventRepository.create).toHaveBeenCalledWith(
        expect.any(Object),
        [{ minutesBefore: 0, method: 'notification' }],
      );
    });
  });
});
