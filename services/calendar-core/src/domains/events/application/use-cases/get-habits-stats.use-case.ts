import { Injectable, Optional } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { RRuleHelper } from '../../domain/utils/rrule-helper';
import {
  HabitSlip,
  HabitSlipRepository,
} from '../../infrastructure/repositories/habit-slip.repository';

const HEATMAP_DAYS = 16 * 7;

export interface HabitStats {
  habitId: string;
  habitTitle: string;
  currentStreak: number;
  longestStreak: number;
  weeklyCompletions: { date: string; completed: boolean }[];
  monthlyCompletionRate: number;
  totalCompletions: number;
  habitKind: 'BUILD' | 'QUIT';
  isPaused: boolean;
  timeOfDay: string | null;
  dailyTarget: number | null;
  targetUnit: string | null;
  /** Sum of every logged amount; null when the habit has no daily target. */
  totalAmount: number | null;
  /** % of scheduled days done in the 30 days before today; null when nothing was scheduled. */
  completionRate30d: number | null;
  /** Last 16 weeks ending today: 1 done, 0 missed, null not scheduled. */
  heatmap: { date: string; value: number | null }[];
  slipsLast30Days: number;
  topTrigger: string | null;
  todaySlip: { trigger: string | null; note: string | null } | null;
}

export interface GetHabitsStatsInput {
  calendarId?: string;
  categoryId?: string;
}

@Injectable()
export class GetHabitsStatsUseCase {
  constructor(
    private readonly eventRepository: EventRepository,
    private readonly eventExecutionRepository: EventExecutionRepository,
    @Optional() private readonly habitSlipRepository?: HabitSlipRepository,
  ) {}

  async execute(input: GetHabitsStatsInput): Promise<HabitStats[]> {
    // Get all habits
    const habits = await this.eventRepository.findAll({
      calendarId: input.calendarId,
      categoryId: input.categoryId,
      eventType: 'HABIT',
    });

    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const stats: HabitStats[] = [];

    const slipsByHabit = new Map<string, HabitSlip[]>();
    const quitIds = habits.filter((h) => h.habitKind === 'QUIT').map((h) => h.id);
    if (this.habitSlipRepository && quitIds.length) {
      for (const slip of await this.habitSlipRepository.findByEventIds(quitIds)) {
        slipsByHabit.set(slip.eventId, [...(slipsByHabit.get(slip.eventId) ?? []), slip]);
      }
    }

    for (const habit of habits) {
      const isQuit = habit.habitKind === 'QUIT';
      const slips = slipsByHabit.get(habit.id) ?? [];

      // Get all executions for this habit
      const executions = await this.eventExecutionRepository.findByEventId(
        habit.id,
      );

      // Build set of completed dates for fast lookup.
      // A QUIT habit is done on every scheduled day that has no slip.
      const completedDates = new Set<string>();
      if (isQuit) {
        const slipDays = new Set(slips.map((sl) => this.formatDbDate(sl.slipDate)));
        const start = habit.startDate ? new Date(habit.startDate) : today;
        for (const d of this.getExpectedOccurrences(habit, start, today)) {
          const key = this.formatDate(d);
          if (!slipDays.has(key)) completedDates.add(key);
        }
      } else {
        for (const exec of executions) {
          if (exec.completed) {
            completedDates.add(this.formatDbDate(exec.executionDate));
          }
        }
      }

      // Calculate weekly completions (last 7 days)
      const weeklyCompletions = this.getWeeklyCompletions(
        habit,
        today,
        completedDates,
      );

      // Calculate current streak
      const currentStreak = this.calculateCurrentStreak(
        habit,
        today,
        completedDates,
      );

      // Calculate longest streak
      const longestStreak = this.calculateLongestStreak(
        habit,
        today,
        completedDates,
      );

      // Calculate monthly completion rate
      const monthlyCompletionRate = this.calculateMonthlyCompletionRate(
        habit,
        today,
        completedDates,
      );

      // Total completions
      const totalCompletions = completedDates.size;

      const recentSlips = slips.filter(
        (sl) => this.formatDbDate(sl.slipDate) >= this.formatDate(this.addDays(today, -29)),
      );
      const todaySlip = slips.find(
        (sl) => this.formatDbDate(sl.slipDate) === this.formatDate(today),
      );
      // Build habits get until tonight to tick today off; a slip today ends a clean run now.
      const streakNow = todaySlip ? 0 : currentStreak;

      stats.push({
        habitId: habit.id,
        habitTitle: habit.title,
        currentStreak: streakNow,
        longestStreak,
        weeklyCompletions,
        monthlyCompletionRate,
        totalCompletions,
        habitKind: isQuit ? 'QUIT' : 'BUILD',
        isPaused: !!habit.isPaused,
        timeOfDay: habit.timeOfDay ?? null,
        dailyTarget: habit.dailyTarget ?? null,
        targetUnit: habit.targetUnit ?? null,
        totalAmount: habit.dailyTarget
          ? executions.reduce((sum, e) => sum + (e.value ?? 0), 0)
          : null,
        completionRate30d: this.calculateRate30d(habit, today, completedDates),
        heatmap: this.buildHeatmap(habit, today, completedDates),
        slipsLast30Days: recentSlips.length,
        topTrigger: this.topTrigger(recentSlips),
        todaySlip: todaySlip
          ? { trigger: todaySlip.trigger ?? null, note: todaySlip.note ?? null }
          : null,
      });
    }

    return stats;
  }

  private getWeeklyCompletions(
    habit: any,
    today: Date,
    completedDates: Set<string>,
  ): { date: string; completed: boolean }[] {
    const result: { date: string; completed: boolean }[] = [];

    // Get expected occurrences for the last 7 days
    const weekStart = new Date(today);
    weekStart.setDate(weekStart.getDate() - 6);

    const expectedDates = this.getExpectedOccurrences(habit, weekStart, today);

    for (let i = 0; i < 7; i++) {
      const date = new Date(weekStart);
      date.setDate(date.getDate() + i);
      const dateStr = this.formatDate(date);

      // Check if this date is an expected occurrence
      const isExpected = expectedDates.some(
        (d) => this.formatDate(d) === dateStr,
      );

      if (isExpected) {
        result.push({
          date: dateStr,
          completed: completedDates.has(dateStr),
        });
      } else {
        // Not an expected day, but include in array with null-like indicator
        result.push({
          date: dateStr,
          completed: false, // Not expected, so not completed
        });
      }
    }

    return result;
  }

  private calculateCurrentStreak(
    habit: any,
    today: Date,
    completedDates: Set<string>,
  ): number {
    // Start from today or yesterday (in case today's not done yet)
    let streak = 0;
    const checkDate = new Date(today);

    // Check if today is an expected occurrence
    const todayExpected = this.isExpectedOccurrence(habit, today);
    const todayCompleted = completedDates.has(this.formatDate(today));

    // If today is expected but not completed, start from yesterday
    if (todayExpected && !todayCompleted) {
      checkDate.setDate(checkDate.getDate() - 1);
    }

    // Count backwards
    const startDate = habit.startDate
      ? new Date(habit.startDate)
      : new Date('2020-01-01');

    while (checkDate >= startDate) {
      const dateStr = this.formatDate(checkDate);
      const isExpected = this.isExpectedOccurrence(habit, checkDate);

      if (isExpected) {
        if (completedDates.has(dateStr)) {
          streak++;
        } else {
          // Streak broken
          break;
        }
      }

      checkDate.setDate(checkDate.getDate() - 1);
    }

    return streak;
  }

  private calculateLongestStreak(
    habit: any,
    today: Date,
    completedDates: Set<string>,
  ): number {
    if (completedDates.size === 0) return 0;

    // Get all expected occurrences from habit start to today
    const startDate = habit.startDate
      ? new Date(habit.startDate)
      : new Date('2020-01-01');

    const expectedDates = this.getExpectedOccurrences(habit, startDate, today);

    let longestStreak = 0;
    let currentStreak = 0;

    for (const date of expectedDates) {
      const dateStr = this.formatDate(date);

      if (completedDates.has(dateStr)) {
        currentStreak++;
        longestStreak = Math.max(longestStreak, currentStreak);
      } else {
        currentStreak = 0;
      }
    }

    return longestStreak;
  }

  private calculateMonthlyCompletionRate(
    habit: any,
    today: Date,
    completedDates: Set<string>,
  ): number {
    // Get first day of current month
    const monthStart = new Date(today.getFullYear(), today.getMonth(), 1);

    // Get expected occurrences for this month (up to today)
    const expectedDates = this.getExpectedOccurrences(habit, monthStart, today);

    if (expectedDates.length === 0) return 0;

    let completed = 0;
    for (const date of expectedDates) {
      const dateStr = this.formatDate(date);
      if (completedDates.has(dateStr)) {
        completed++;
      }
    }

    return Math.round((completed / expectedDates.length) * 100);
  }

  private calculateRate30d(
    habit: any,
    today: Date,
    completedDates: Set<string>,
  ): number | null {
    const expected = this.getExpectedOccurrences(
      habit,
      this.addDays(today, -30),
      this.addDays(today, -1),
    );
    if (!expected.length) return null;
    const done = expected.filter((d) => completedDates.has(this.formatDate(d))).length;
    return Math.round((done / expected.length) * 100);
  }

  private buildHeatmap(
    habit: any,
    today: Date,
    completedDates: Set<string>,
  ): { date: string; value: number | null }[] {
    const from = this.addDays(today, -(HEATMAP_DAYS - 1));
    const flexible = habit.recurrenceType === 'FLEXIBLE';
    const expected = new Set(
      this.getExpectedOccurrences(habit, from, today).map((d) => this.formatDate(d)),
    );
    const start = habit.startDate ? this.formatDbDate(new Date(habit.startDate)) : '';

    return Array.from({ length: HEATMAP_DAYS }, (_, i) => {
      const date = this.formatDate(this.addDays(from, i));
      // Flexible habits can be done on any day of the week
      const scheduled = flexible ? date >= start : expected.has(date);
      return { date, value: scheduled ? (completedDates.has(date) ? 1 : 0) : null };
    });
  }

  private topTrigger(slips: HabitSlip[]): string | null {
    const counts = new Map<string, number>();
    for (const sl of slips) {
      if (sl.trigger) counts.set(sl.trigger, (counts.get(sl.trigger) ?? 0) + 1);
    }
    let best: string | null = null;
    for (const [trigger, n] of counts) {
      if (best === null || n > counts.get(best)!) best = trigger;
    }
    return best;
  }

  private addDays(date: Date, n: number): Date {
    const d = new Date(date);
    d.setDate(d.getDate() + n);
    return d;
  }

  /**
   * DATE columns come back from Prisma as UTC midnight. Reading them with local getters in
   * America/Sao_Paulo lands on the previous day, so take the UTC calendar date instead.
   */
  private formatDbDate(date: Date): string {
    const d = new Date(date);
    return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`;
  }

  private getExpectedOccurrences(
    habit: any,
    startDate: Date,
    endDate: Date,
  ): Date[] {
    if (!habit.recurrenceRule) {
      // Non-recurring habit - check if startDate falls within range
      const habitStart = habit.startDate
        ? new Date(habit.startDate)
        : new Date();
      if (habitStart >= startDate && habitStart <= endDate) {
        return [habitStart];
      }
      return [];
    }

    const habitStartDate = habit.startDate
      ? new Date(habit.startDate)
      : new Date();

    return RRuleHelper.generateOccurrences(
      habit.recurrenceRule,
      habitStartDate,
      startDate,
      endDate,
    );
  }

  private isExpectedOccurrence(habit: any, date: Date): boolean {
    const occurrences = this.getExpectedOccurrences(habit, date, date);
    return occurrences.some(
      (d) => this.formatDate(d) === this.formatDate(date),
    );
  }

  private formatDate(date: Date): string {
    const d = new Date(date);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  }
}
