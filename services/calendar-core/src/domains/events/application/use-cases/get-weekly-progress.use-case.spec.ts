import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { GetWeeklyProgressUseCase } from './get-weekly-progress.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { Event } from '../../domain/entities/event.entity';
import { EventExecution } from '../../domain/entities/event-execution.entity';

describe('GetWeeklyProgressUseCase', () => {
  let useCase: GetWeeklyProgressUseCase;
  let mockEventRepository: Partial<EventRepository>;
  let mockExecutionRepository: Partial<EventExecutionRepository>;

  // Fixed date for testing: Thursday, January 23, 2025
  // Week: Monday Jan 20 - Sunday Jan 26
  const fixedToday = new Date(2025, 0, 23);

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(fixedToday);

    mockEventRepository = {
      findAll: vi.fn(),
    };

    mockExecutionRepository = {
      findByEventId: vi.fn(),
      findByEventIds: vi.fn(),
    };

    useCase = new GetWeeklyProgressUseCase(
      mockEventRepository as EventRepository,
      mockExecutionRepository as EventExecutionRepository,
    );
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const createFlexibleHabit = (overrides: Partial<Event> = {}): Event => {
    return new Event({
      id: 'habit-1',
      calendarId: 'calendar-1',
      title: 'Jiu-jitsu',
      startTime: '19:00',
      startDate: new Date(2025, 0, 1),
      eventType: 'HABIT',
      recurrenceRule: 'FREQ=WEEKLY',
      recurrenceType: 'FLEXIBLE',
      weeklyTargetCount: 2,
      weeklyPreferredDays: ['TU', 'TH'],
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

  // Helper to get the Monday of a week containing the given date
  const getWeekStart = (date: Date): Date => {
    const d = new Date(date);
    const day = d.getDay();
    const diff = d.getDate() - day + (day === 0 ? -6 : 1); // Adjust when day is Sunday
    return new Date(d.setDate(diff));
  };

  describe('execute', () => {
    it('should return empty array when no flexible habits exist', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      const result = await useCase.execute({});

      expect(result).toEqual([]);
    });

    it('should return weekly progress for a flexible habit', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);

      const result = await useCase.execute({});

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        habitId: 'habit-1',
        habitTitle: 'Jiu-jitsu',
        weeklyTargetCount: 2,
        weeklyPreferredDays: ['TU', 'TH'],
        currentWeek: {
          weekStartDate: '2025-01-20', // Monday
          targetCount: 2,
          completedCount: 0,
          completedDates: [],
          isGoalMet: false,
          daysRemaining: 4, // Thu, Fri, Sat, Sun
        },
      });
    });

    it('should calculate current week progress correctly', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Completed on Tuesday (Jan 21)
      const executions = [
        createExecution('habit-1', new Date(2025, 0, 21), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].currentWeek).toMatchObject({
        completedCount: 1,
        completedDates: ['2025-01-21'],
        isGoalMet: false,
      });
    });

    it('should mark goal as met when target is reached', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Completed on Tuesday and Wednesday (2 times, target met)
      const executions = [
        createExecution('habit-1', new Date(2025, 0, 21), true),
        createExecution('habit-1', new Date(2025, 0, 22), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].currentWeek).toMatchObject({
        completedCount: 2,
        isGoalMet: true,
      });
    });

    it('should calculate weekly streak correctly', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // 3 consecutive weeks of meeting the goal (2x each week)
      const executions = [
        // Week Jan 6-12 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 7), true),
        createExecution('habit-1', new Date(2025, 0, 9), true),
        // Week Jan 13-19 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 14), true),
        createExecution('habit-1', new Date(2025, 0, 16), true),
        // Current week Jan 20-26 (2 completions already)
        createExecution('habit-1', new Date(2025, 0, 21), true),
        createExecution('habit-1', new Date(2025, 0, 23), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].currentStreak).toBe(3);
    });

    it('should break streak when a week does not meet the goal', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Week 1: Met goal, Week 2: Missed (only 1), Week 3 (current): Met goal
      const executions = [
        // Week Jan 6-12 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 7), true),
        createExecution('habit-1', new Date(2025, 0, 9), true),
        // Week Jan 13-19 (only 1 completion - missed!)
        createExecution('habit-1', new Date(2025, 0, 14), true),
        // Current week Jan 20-26 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 21), true),
        createExecution('habit-1', new Date(2025, 0, 23), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      // Streak should be 1 (only current week counts, previous broke it)
      expect(result[0].currentStreak).toBe(1);
    });

    it('should not count current week in streak if goal not yet met', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Previous weeks met goal, current week only 1 completion
      const executions = [
        // Week Jan 13-19 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 14), true),
        createExecution('habit-1', new Date(2025, 0, 16), true),
        // Current week Jan 20-26 (only 1 completion so far)
        createExecution('habit-1', new Date(2025, 0, 21), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      // Streak should be 1 (previous week), current week not counted yet
      expect(result[0].currentStreak).toBe(1);
      expect(result[0].currentWeek.isGoalMet).toBe(false);
    });

    it('should include current week in streak if goal is met', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Previous week met goal, current week also met goal
      const executions = [
        // Week Jan 13-19 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 14), true),
        createExecution('habit-1', new Date(2025, 0, 16), true),
        // Current week Jan 20-26 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 21), true),
        createExecution('habit-1', new Date(2025, 0, 22), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      // Streak should be 2 (previous + current)
      expect(result[0].currentStreak).toBe(2);
      expect(result[0].currentWeek.isGoalMet).toBe(true);
    });

    it('should calculate longest streak correctly', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // 4 weeks streak in December, then break, then 2 weeks in January
      const executions = [
        // Week Dec 2-8 (2 completions)
        createExecution('habit-1', new Date(2024, 11, 3), true),
        createExecution('habit-1', new Date(2024, 11, 5), true),
        // Week Dec 9-15 (2 completions)
        createExecution('habit-1', new Date(2024, 11, 10), true),
        createExecution('habit-1', new Date(2024, 11, 12), true),
        // Week Dec 16-22 (2 completions)
        createExecution('habit-1', new Date(2024, 11, 17), true),
        createExecution('habit-1', new Date(2024, 11, 19), true),
        // Week Dec 23-29 (2 completions)
        createExecution('habit-1', new Date(2024, 11, 24), true),
        createExecution('habit-1', new Date(2024, 11, 26), true),
        // Week Dec 30 - Jan 5 (missed - only 1)
        createExecution('habit-1', new Date(2024, 11, 31), true),
        // Week Jan 6-12 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 7), true),
        createExecution('habit-1', new Date(2025, 0, 9), true),
        // Week Jan 13-19 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 14), true),
        createExecution('habit-1', new Date(2025, 0, 16), true),
        // Current week (2 completions)
        createExecution('habit-1', new Date(2025, 0, 21), true),
        createExecution('habit-1', new Date(2025, 0, 23), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].longestStreak).toBe(4); // December streak
      expect(result[0].currentStreak).toBe(3); // Jan 6-12, Jan 13-19, current
    });

    it('should return week history', async () => {
      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      const executions = [
        // Week Jan 6-12 (2 completions)
        createExecution('habit-1', new Date(2025, 0, 7), true),
        createExecution('habit-1', new Date(2025, 0, 9), true),
        // Week Jan 13-19 (1 completion - missed)
        createExecution('habit-1', new Date(2025, 0, 14), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].weekHistory).toBeDefined();
      expect(result[0].weekHistory.length).toBeGreaterThan(0);

      // Check week Jan 6-12 (met goal)
      const week1 = result[0].weekHistory.find(w => w.weekStartDate === '2025-01-06');
      expect(week1).toMatchObject({
        completedCount: 2,
        targetCount: 2,
        isGoalMet: true,
      });

      // Check week Jan 13-19 (missed goal)
      const week2 = result[0].weekHistory.find(w => w.weekStartDate === '2025-01-13');
      expect(week2).toMatchObject({
        completedCount: 1,
        targetCount: 2,
        isGoalMet: false,
      });
    });

    it('should filter by calendarId', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([]);

      await useCase.execute({ calendarId: 'calendar-123' });

      expect(mockEventRepository.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          calendarId: 'calendar-123',
        }),
      );
    });

    it('should only return flexible habits', async () => {
      const flexibleHabit = createFlexibleHabit({ id: 'habit-flexible' });
      const fixedHabit = new Event({
        id: 'habit-fixed',
        calendarId: 'calendar-1',
        title: 'Daily Meditation',
        startTime: '09:00',
        startDate: new Date(2025, 0, 1),
        eventType: 'HABIT',
        recurrenceRule: 'FREQ=DAILY',
        recurrenceType: 'FIXED',
        status: 'CONFIRMED',
        isActive: true,
        createdAt: new Date(),
        updatedAt: new Date(),
      });

      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([flexibleHabit, fixedHabit]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);

      const result = await useCase.execute({});

      expect(result).toHaveLength(1);
      expect(result[0].habitId).toBe('habit-flexible');
    });

    it('should handle habit with 1x per week target', async () => {
      const habit = createFlexibleHabit({
        id: 'habit-cycling',
        title: 'Bicicleta',
        weeklyTargetCount: 1,
        weeklyPreferredDays: ['SU'],
      });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Did it on Wednesday instead of Sunday
      const executions = [
        createExecution('habit-cycling', new Date(2025, 0, 22), true), // Wednesday
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0]).toMatchObject({
        weeklyTargetCount: 1,
        currentWeek: {
          completedCount: 1,
          isGoalMet: true,
        },
      });
    });

    it('should handle exceeding the weekly target', async () => {
      const habit = createFlexibleHabit({ weeklyTargetCount: 2 });
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);

      // Did 3 times this week (exceeded target)
      const executions = [
        createExecution('habit-1', new Date(2025, 0, 20), true),
        createExecution('habit-1', new Date(2025, 0, 21), true),
        createExecution('habit-1', new Date(2025, 0, 22), true),
      ];
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(executions);

      const result = await useCase.execute({});

      expect(result[0].currentWeek).toMatchObject({
        completedCount: 3,
        targetCount: 2,
        isGoalMet: true,
      });
    });

    it('should calculate days remaining correctly', async () => {
      // Test on different days of the week
      vi.setSystemTime(new Date(2025, 0, 20)); // Monday

      const habit = createFlexibleHabit();
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([habit]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);

      let result = await useCase.execute({});
      expect(result[0].currentWeek.daysRemaining).toBe(7); // Mon-Sun

      vi.setSystemTime(new Date(2025, 0, 26)); // Sunday
      result = await useCase.execute({});
      expect(result[0].currentWeek.daysRemaining).toBe(1); // Just Sunday left
    });

    it('should handle multiple flexible habits', async () => {
      const jiujitsu = createFlexibleHabit({
        id: 'habit-jj',
        title: 'Jiu-jitsu',
        weeklyTargetCount: 2,
      });
      const cycling = createFlexibleHabit({
        id: 'habit-bike',
        title: 'Bicicleta',
        weeklyTargetCount: 1,
        weeklyPreferredDays: ['SU'],
      });

      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([jiujitsu, cycling]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockImplementation(async (eventId: string) => {
        if (eventId === 'habit-jj') {
          return [
            createExecution('habit-jj', new Date(2025, 0, 21), true),
          ];
        }
        return [
          createExecution('habit-bike', new Date(2025, 0, 19), true), // Last Sunday
        ];
      });

      const result = await useCase.execute({});

      expect(result).toHaveLength(2);

      const jjResult = result.find(r => r.habitId === 'habit-jj');
      expect(jjResult?.currentWeek.completedCount).toBe(1);
      expect(jjResult?.currentWeek.isGoalMet).toBe(false);

      const bikeResult = result.find(r => r.habitId === 'habit-bike');
      expect(bikeResult?.currentWeek.completedCount).toBe(0); // Last Sunday was previous week
      expect(bikeResult?.currentWeek.isGoalMet).toBe(false);
    });
  });
});
