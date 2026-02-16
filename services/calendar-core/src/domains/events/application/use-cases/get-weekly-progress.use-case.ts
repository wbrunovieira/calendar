import { Injectable } from '@nestjs/common';
import { EventRepository } from '../../infrastructure/repositories/event.repository';
import { EventExecutionRepository } from '../../infrastructure/repositories/event-execution.repository';

export interface WeekProgress {
  weekStartDate: string; // YYYY-MM-DD (Monday)
  targetCount: number;
  completedCount: number;
  completedDates: string[];
  isGoalMet: boolean;
  daysRemaining?: number; // Only for current week
}

export interface FlexibleHabitProgress {
  habitId: string;
  habitTitle: string;
  weeklyTargetCount: number;
  weeklyPreferredDays: string[];
  currentWeek: WeekProgress;
  weekHistory: WeekProgress[];
  currentStreak: number;
  longestStreak: number;
}

export interface GetWeeklyProgressInput {
  calendarId?: string;
  categoryId?: string;
}

/**
 * Formats a Date object to YYYY-MM-DD string using UTC to avoid timezone issues
 */
function formatDate(date: Date): string {
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  const day = String(date.getUTCDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Formats a Date object to YYYY-MM-DD string using local time (for "today" calculations)
 */
function formatLocalDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Gets the Monday of the week containing the given date (local time)
 */
function getWeekStart(date: Date): Date {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1); // Adjust when day is Sunday
  return new Date(d.setDate(diff));
}

/**
 * Gets the Monday of the week containing the given date (UTC)
 * Used for execution dates from the database which are stored in UTC
 */
function getWeekStartUTC(date: Date): Date {
  const d = new Date(date);
  d.setUTCHours(0, 0, 0, 0);
  const day = d.getUTCDay();
  const diff = d.getUTCDate() - day + (day === 0 ? -6 : 1); // Adjust when day is Sunday
  d.setUTCDate(diff);
  return d;
}

/**
 * Gets the Sunday of the week containing the given date
 */
function getWeekEnd(date: Date): Date {
  const weekStart = getWeekStart(date);
  const weekEnd = new Date(weekStart);
  weekEnd.setDate(weekStart.getDate() + 6);
  weekEnd.setHours(23, 59, 59, 999);
  return weekEnd;
}

/**
 * Calculates days remaining from the given date to the end of the week
 */
function getDaysRemaining(date: Date): number {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  const dayOfWeek = d.getDay();
  // Sunday = 0, Monday = 1, ..., Saturday = 6
  // We need days from current day to Sunday (inclusive)
  if (dayOfWeek === 0) return 1; // Sunday
  return 7 - dayOfWeek + 1; // +1 to include current day
}

@Injectable()
export class GetWeeklyProgressUseCase {
  constructor(
    private readonly eventRepository: EventRepository,
    private readonly executionRepository: EventExecutionRepository,
  ) {}

  async execute(input: GetWeeklyProgressInput): Promise<FlexibleHabitProgress[]> {
    // Fetch habits
    const habits = await this.eventRepository.findAll({
      calendarId: input.calendarId,
      categoryId: input.categoryId,
      eventType: 'HABIT',
    });

    // Filter to only flexible habits
    const flexibleHabits = habits.filter(
      (h) => h.recurrenceType === 'FLEXIBLE' && h.weeklyTargetCount,
    );

    if (flexibleHabits.length === 0) {
      return [];
    }

    const today = new Date();
    const currentWeekStart = getWeekStart(today);

    // Fetch executions for all flexible habits
    const results: FlexibleHabitProgress[] = [];

    for (const habit of flexibleHabits) {
      const executions = await this.executionRepository.findByEventId(habit.id);
      const completedExecutions = executions.filter((e) => e.completed);

      // Group executions by week (UTC — execution dates are stored as UTC midnight)
      const weekMap = new Map<string, string[]>();

      for (const exec of completedExecutions) {
        const execDate = new Date(exec.executionDate);
        const weekStart = getWeekStartUTC(execDate);
        const weekStartStr = formatDate(weekStart);

        if (!weekMap.has(weekStartStr)) {
          weekMap.set(weekStartStr, []);
        }
        weekMap.get(weekStartStr)!.push(formatDate(execDate));
      }

      // Build current week progress (formatDate on local midnight = same day for São Paulo UTC-3)
      const currentWeekStartStr = formatDate(currentWeekStart);
      const currentWeekDates = weekMap.get(currentWeekStartStr) || [];
      const currentWeekProgress: WeekProgress = {
        weekStartDate: currentWeekStartStr,
        targetCount: habit.weeklyTargetCount!,
        completedCount: currentWeekDates.length,
        completedDates: currentWeekDates.sort(),
        isGoalMet: currentWeekDates.length >= habit.weeklyTargetCount!,
        daysRemaining: getDaysRemaining(today),
      };

      // Build week history (last 12 weeks, excluding current)
      const weekHistory: WeekProgress[] = [];
      const historyStart = new Date(currentWeekStart);
      historyStart.setDate(historyStart.getDate() - 12 * 7); // Go back 12 weeks

      for (
        let weekStart = new Date(historyStart);
        weekStart < currentWeekStart;
        weekStart.setDate(weekStart.getDate() + 7)
      ) {
        const weekStartStr = formatDate(weekStart);
        const weekDates = weekMap.get(weekStartStr) || [];
        weekHistory.push({
          weekStartDate: weekStartStr,
          targetCount: habit.weeklyTargetCount!,
          completedCount: weekDates.length,
          completedDates: weekDates.sort(),
          isGoalMet: weekDates.length >= habit.weeklyTargetCount!,
        });
      }

      // Calculate streaks
      const { currentStreak, longestStreak } = this.calculateStreaks(
        weekHistory,
        currentWeekProgress,
        habit.weeklyTargetCount!,
      );

      results.push({
        habitId: habit.id,
        habitTitle: habit.title,
        weeklyTargetCount: habit.weeklyTargetCount!,
        weeklyPreferredDays: habit.weeklyPreferredDays || [],
        currentWeek: currentWeekProgress,
        weekHistory: weekHistory.reverse(), // Most recent first
        currentStreak,
        longestStreak,
      });
    }

    return results;
  }

  private calculateStreaks(
    weekHistory: WeekProgress[],
    currentWeek: WeekProgress,
    targetCount: number,
  ): { currentStreak: number; longestStreak: number } {
    // Sort weeks from most recent to oldest for current streak
    const sortedWeeks = [...weekHistory].sort(
      (a, b) => b.weekStartDate.localeCompare(a.weekStartDate),
    );

    // Current streak: count consecutive completed weeks backwards
    // Include current week if met, then count backwards through history
    let currentStreak = 0;
    if (currentWeek.isGoalMet) {
      currentStreak = 1;
    }
    for (const week of sortedWeeks) {
      if (week.isGoalMet) {
        currentStreak++;
      } else {
        break;
      }
    }

    // If current week not met, streak is just the consecutive past weeks
    if (!currentWeek.isGoalMet) {
      currentStreak = 0;
      for (const week of sortedWeeks) {
        if (week.isGoalMet) {
          currentStreak++;
        } else {
          break;
        }
      }
    }

    // Longest streak: scan chronologically through all weeks
    const chronologicalWeeks = [...weekHistory, currentWeek].sort(
      (a, b) => a.weekStartDate.localeCompare(b.weekStartDate),
    );
    let longestStreak = 0;
    let tempStreak = 0;
    for (const week of chronologicalWeeks) {
      if (week.isGoalMet) {
        tempStreak++;
        longestStreak = Math.max(longestStreak, tempStreak);
      } else {
        tempStreak = 0;
      }
    }

    return { currentStreak, longestStreak };
  }
}
