import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { GetHabitsStatsUseCase } from './get-habits-stats.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { Event } from '../../domain/entities/event.entity';
import { EventExecution } from '../../domain/entities/event-execution.entity';

describe('GetHabitsStatsUseCase', () => {
  let useCase: GetHabitsStatsUseCase;
  let mockEventRepository: Partial<EventRepository>;
  let mockExecutionRepository: Partial<EventExecutionRepository>;

  // Fixed date for testing: January 15, 2024
  const fixedToday = new Date(2024, 0, 15);

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(fixedToday);

    mockEventRepository = {
      findAll: vi.fn(),
    };

    mockExecutionRepository = {
      findByEventId: vi.fn(),
    };

    useCase = new GetHabitsStatsUseCase(
      mockEventRepository as EventRepository,
      mockExecutionRepository as EventExecutionRepository,
    );
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const createHabit = (overrides: Partial<Event> = {}): Event => {
    return new Event({
      id: 'habit-1',
      calendarId: 'calendar-1',
      title: 'Daily Meditation',
      startTime: '09:00',
      startDate: new Date(2024, 0, 1),
      eventType: 'HABIT',
      recurrenceRule: 'FREQ=DAILY',
      status: 'CONFIRMED',
      isActive: true,
      createdAt: new Date(),
      updatedAt: new Date(),
      ...overrides,
    });
  };

  const createExecution = (
    eventId: string,
    date: Date,
    completed: boolean,
  ): EventExecution => {
    return new EventExecution({
      id: `exec-${date.toISOString()}`,
      eventId,
      executionDate: date,
      completed,
      completedAt: completed ? date : null,
      createdAt: date,
      updatedAt: date,
    });
  };

  describe('execute', () => {
    it('should return empty array when no habits exist', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      const result = await useCase.execute({});

      expect(result).toEqual([]);
      expect(mockEventRepository.findAll).toHaveBeenCalledWith({
        calendarId: undefined,
        categoryId: undefined,
        eventType: 'HABIT',
      });
    });

    it('should return stats for a habit with no completions', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);

      const result = await useCase.execute({});

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        habitId: 'habit-1',
        habitTitle: 'Daily Meditation',
        currentStreak: 0,
        longestStreak: 0,
        totalCompletions: 0,
        monthlyCompletionRate: 0,
      });
    });

    it('should calculate current streak correctly', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Completions for 3 consecutive days ending yesterday (Jan 12, 13, 14)
      const executions = [
        createExecution('habit-1', new Date(2024, 0, 12), true),
        createExecution('habit-1', new Date(2024, 0, 13), true),
        createExecution('habit-1', new Date(2024, 0, 14), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].currentStreak).toBe(3);
    });

    it('should calculate streak including today if completed', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Completions for 4 consecutive days including today
      const executions = [
        createExecution('habit-1', new Date(2024, 0, 12), true),
        createExecution('habit-1', new Date(2024, 0, 13), true),
        createExecution('habit-1', new Date(2024, 0, 14), true),
        createExecution('habit-1', new Date(2024, 0, 15), true), // Today
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].currentStreak).toBe(4);
    });

    it('should reset streak when a day is missed', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Gap on Jan 13
      const executions = [
        createExecution('habit-1', new Date(2024, 0, 11), true),
        createExecution('habit-1', new Date(2024, 0, 12), true),
        // Jan 13 missing
        createExecution('habit-1', new Date(2024, 0, 14), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      // Streak should be 1 (only Jan 14)
      expect(result[0].currentStreak).toBe(1);
    });

    it('should calculate longest streak correctly', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Two streaks: 5 days (Jan 1-5) and 2 days (Jan 14-15)
      const executions = [
        createExecution('habit-1', new Date(2024, 0, 1), true),
        createExecution('habit-1', new Date(2024, 0, 2), true),
        createExecution('habit-1', new Date(2024, 0, 3), true),
        createExecution('habit-1', new Date(2024, 0, 4), true),
        createExecution('habit-1', new Date(2024, 0, 5), true),
        // Gap
        createExecution('habit-1', new Date(2024, 0, 14), true),
        createExecution('habit-1', new Date(2024, 0, 15), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].longestStreak).toBe(5);
      expect(result[0].currentStreak).toBe(2);
    });

    it('should calculate monthly completion rate', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Complete 10 out of 15 days in January (up to Jan 15)
      const executions: EventExecution[] = [];
      for (let day = 1; day <= 10; day++) {
        executions.push(createExecution('habit-1', new Date(2024, 0, day), true));
      }
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      // 10 completed / 15 expected days = 66.67% rounded to 67%
      expect(result[0].monthlyCompletionRate).toBe(67);
    });

    it('should return weekly completions for last 7 days', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Complete some days in the last week
      const executions = [
        createExecution('habit-1', new Date(2024, 0, 10), true),
        createExecution('habit-1', new Date(2024, 0, 12), true),
        createExecution('habit-1', new Date(2024, 0, 15), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].weeklyCompletions).toHaveLength(7);

      // Check specific days
      const jan10 = result[0].weeklyCompletions.find(w => w.date === '2024-01-10');
      const jan11 = result[0].weeklyCompletions.find(w => w.date === '2024-01-11');
      const jan12 = result[0].weeklyCompletions.find(w => w.date === '2024-01-12');
      const jan15 = result[0].weeklyCompletions.find(w => w.date === '2024-01-15');

      expect(jan10?.completed).toBe(true);
      expect(jan11?.completed).toBe(false);
      expect(jan12?.completed).toBe(true);
      expect(jan15?.completed).toBe(true);
    });

    it('should count total completions correctly', async () => {
      const habit = createHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const executions = [
        createExecution('habit-1', new Date(2024, 0, 1), true),
        createExecution('habit-1', new Date(2024, 0, 2), true),
        createExecution('habit-1', new Date(2024, 0, 3), false), // Not completed
        createExecution('habit-1', new Date(2024, 0, 4), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].totalCompletions).toBe(3);
    });

    it('should filter by calendarId', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({ calendarId: 'calendar-123' });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith({
        calendarId: 'calendar-123',
        categoryId: undefined,
        eventType: 'HABIT',
      });
    });

    it('should filter by categoryId', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({ categoryId: 'category-456' });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith({
        calendarId: undefined,
        categoryId: 'category-456',
        eventType: 'HABIT',
      });
    });

    it('should handle multiple habits', async () => {
      const habit1 = createHabit({ id: 'habit-1', title: 'Meditation' });
      const habit2 = createHabit({ id: 'habit-2', title: 'Exercise' });

      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit1, habit2]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockImplementation(async (eventId: string) => {
        if (eventId === 'habit-1') {
          return [createExecution('habit-1', new Date(2024, 0, 14), true)];
        }
        return [
          createExecution('habit-2', new Date(2024, 0, 13), true),
          createExecution('habit-2', new Date(2024, 0, 14), true),
        ];
      });

      const result = await useCase.execute({});

      expect(result).toHaveLength(2);
      expect(result.find(s => s.habitId === 'habit-1')?.totalCompletions).toBe(1);
      expect(result.find(s => s.habitId === 'habit-2')?.totalCompletions).toBe(2);
    });

    it('should handle weekly habits correctly', async () => {
      // A habit that only occurs on Mondays (weekday 1)
      const habit = createHabit({
        recurrenceRule: 'FREQ=WEEKLY;BYDAY=MO',
        startDate: new Date(2024, 0, 1), // Monday
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Complete the Monday (Jan 8)
      const executions = [
        createExecution('habit-1', new Date(2024, 0, 8), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      // Should have streak of 1 for the completed Monday
      expect(result[0].totalCompletions).toBe(1);
    });
  });
});
