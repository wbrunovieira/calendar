import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { GetHabitsStatsUseCase } from './get-habits-stats.use-case';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { Event } from '../../domain/entities/event.entity';
import { EventExecution } from '../../domain/entities/event-execution.entity';
import { HabitSlipRepository } from '../../infrastructure/repositories/habit-slip.repository';

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

  describe('habits v2', () => {
    let slipRepository: Partial<HabitSlipRepository>;

    beforeEach(() => {
      slipRepository = { findByEventIds: vi.fn().mockResolvedValue([]) };
      useCase = new GetHabitsStatsUseCase(
        mockEventRepository as EventRepository,
        mockExecutionRepository as EventExecutionRepository,
        slipRepository as HabitSlipRepository,
      );
    });

    const slip = (date: string, trigger?: string) => ({
      id: `slip-${date}`,
      eventId: 'habit-1',
      slipDate: new Date(date), // DATE columns come back as UTC midnight
      trigger: trigger ?? null,
      note: null,
    });

    it('reads completions stored as UTC midnight on their own calendar day (regression)', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([createHabit()]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([
        createExecution('habit-1', new Date('2024-01-13'), true),
        createExecution('habit-1', new Date('2024-01-14'), true),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.currentStreak).toBe(2);
      expect(stats.weeklyCompletions.filter((d) => d.completed).map((d) => d.date)).toEqual([
        '2024-01-13',
        '2024-01-14',
      ]);
    });

    it('counts clean days since the last slip for a QUIT habit', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([createHabit({ habitKind: 'QUIT' })]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);
      vi.mocked(slipRepository.findByEventIds!).mockResolvedValue([
        slip('2024-01-10', 'Stress'),
        slip('2024-01-04', 'Stress'),
        slip('2024-01-02', 'Social'),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.habitKind).toBe('QUIT');
      expect(stats.currentStreak).toBe(5); // Jan 11..15
      expect(stats.longestStreak).toBe(5); // Jan 5..9 and Jan 11..15
      expect(stats.slipsLast30Days).toBe(3);
      expect(stats.topTrigger).toBe('Stress');
      expect(stats.todaySlip).toBeNull();
    });

    it('reports a slip logged today and resets the clean count', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([createHabit({ habitKind: 'QUIT' })]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);
      vi.mocked(slipRepository.findByEventIds!).mockResolvedValue([slip('2024-01-15', 'Tired')]);

      const [stats] = await useCase.execute({});

      expect(stats.currentStreak).toBe(0);
      expect(stats.todaySlip).toEqual({ trigger: 'Tired', note: null });
    });

    it('computes the completion rate over the last 30 days, excluding today', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ startDate: new Date(2023, 11, 1) }),
      ]);
      const done = Array.from({ length: 10 }, (_, i) =>
        createExecution('habit-1', new Date(2024, 0, 14 - i * 2), true),
      );
      done.push(createExecution('habit-1', new Date(2024, 0, 15), true)); // today: ignored
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue(done);

      const [stats] = await useCase.execute({});

      expect(stats.completionRate30d).toBe(33);
    });

    it('returns a 16-week heatmap ending today with null outside the schedule', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([createHabit()]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([
        createExecution('habit-1', new Date(2024, 0, 14), true),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.heatmap).toHaveLength(112);
      expect(stats.heatmap[111]).toEqual({ date: '2024-01-15', value: null }); // still pending today
      expect(stats.heatmap[110]).toEqual({ date: '2024-01-14', value: 1 });
      expect(stats.heatmap.find((d) => d.date === '2023-12-31')?.value).toBeNull();
    });

    it('sums the logged amount and passes the habit settings through', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ dailyTarget: 20, targetUnit: 'pages', timeOfDay: 'EVENING', isPaused: true }),
      ]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([
        new EventExecution({ id: 'a', eventId: 'habit-1', executionDate: new Date(2024, 0, 13), completed: true, value: 20 }),
        new EventExecution({ id: 'b', eventId: 'habit-1', executionDate: new Date(2024, 0, 14), completed: false, value: 12 }),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats).toMatchObject({
        habitKind: 'BUILD',
        dailyTarget: 20,
        targetUnit: 'pages',
        timeOfDay: 'EVENING',
        isPaused: true,
        totalAmount: 32,
      });
    });
  });

  describe('habits v2 review fixes', () => {
    let slipRepository: Partial<HabitSlipRepository>;

    beforeEach(() => {
      slipRepository = { findByEventIds: vi.fn().mockResolvedValue([]) };
      useCase = new GetHabitsStatsUseCase(
        mockEventRepository as EventRepository,
        mockExecutionRepository as EventExecutionRepository,
        slipRepository as HabitSlipRepository,
      );
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([]);
    });

    const slip = (date: string) => ({
      id: date,
      eventId: 'habit-1',
      slipDate: new Date(date),
      trigger: 'Stress',
      note: null,
    });

    it('marks done days on a FLEXIBLE heatmap and leaves the rest unscheduled', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ recurrenceRule: 'FREQ=WEEKLY', recurrenceType: 'FLEXIBLE', weeklyTargetCount: 3 }),
      ]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([
        createExecution('habit-1', new Date(2024, 0, 10), true),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.heatmap.find((c) => c.date === '2024-01-10')?.value).toBe(1);
      expect(stats.heatmap.find((c) => c.date === '2024-01-11')?.value).toBeNull();
      expect(stats.completionRate30d).toBeNull();
    });

    it('freezes a paused QUIT habit on the day it was paused', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ habitKind: 'QUIT', isPaused: true, pausedAt: new Date('2024-01-10') }),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.currentStreak).toBe(10); // Jan 1..10, not 15
    });

    it('keeps a paused BUILD streak instead of letting the paused days break it', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ isPaused: true, pausedAt: new Date('2024-01-11') }),
      ]);
      vi.mocked(mockExecutionRepository.findByEventId!).mockResolvedValue([
        createExecution('habit-1', new Date(2024, 0, 8), true),
        createExecution('habit-1', new Date(2024, 0, 9), true),
        createExecution('habit-1', new Date(2024, 0, 10), true),
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.currentStreak).toBe(3);
    });

    it('counts slips in a 30-day window that includes today and 29 days back', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ habitKind: 'QUIT', startDate: new Date(2023, 10, 1) }),
      ]);
      vi.mocked(slipRepository.findByEventIds!).mockResolvedValue([
        slip('2023-12-17'), // today - 29
        slip('2023-12-16'), // today - 30
      ]);

      const [stats] = await useCase.execute({});

      expect(stats.slipsLast30Days).toBe(1);
    });

    it('ignores slips dated before the habit started', async () => {
      vi.mocked(mockEventRepository.findAll!).mockResolvedValue([
        createHabit({ habitKind: 'QUIT', startDate: new Date('2024-01-05') }),
      ]);
      vi.mocked(slipRepository.findByEventIds!).mockResolvedValue([slip('2024-01-03')]);

      const [stats] = await useCase.execute({});

      expect(stats.slipsLast30Days).toBe(0);
      expect(stats.topTrigger).toBeNull();
      expect(stats.currentStreak).toBe(11); // Jan 5..15
    });
  });
});
