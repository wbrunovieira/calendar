import { Injectable } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';
import { RRuleHelper } from '../../domain/utils/rrule-helper';

export interface HabitStats {
  habitId: string;
  habitTitle: string;
  currentStreak: number;
  longestStreak: number;
  weeklyCompletions: { date: string; completed: boolean }[];
  monthlyCompletionRate: number;
  totalCompletions: number;
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

    for (const habit of habits) {
      // Get all executions for this habit
      const executions = await this.eventExecutionRepository.findByEventId(
        habit.id,
      );

      // Build set of completed dates for fast lookup
      const completedDates = new Set<string>();
      for (const exec of executions) {
        if (exec.completed) {
          const dateStr = this.formatDate(exec.executionDate);
          completedDates.add(dateStr);
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

      stats.push({
        habitId: habit.id,
        habitTitle: habit.title,
        currentStreak,
        longestStreak,
        weeklyCompletions,
        monthlyCompletionRate,
        totalCompletions,
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
