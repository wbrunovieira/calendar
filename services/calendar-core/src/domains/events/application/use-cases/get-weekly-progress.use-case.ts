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
 * Formats a Date object to YYYY-MM-DD string
 */
function formatDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Gets the Monday of the week containing the given date
 */
function getWeekStart(date: Date): Date {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1); // Adjust when day is Sunday
  return new Date(d.setDate(diff));
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

      // Group executions by week
      const weekMap = new Map<string, string[]>();

      for (const exec of completedExecutions) {
        const execDate = new Date(exec.executionDate);
        const weekStart = getWeekStart(execDate);
        const weekStartStr = formatDate(weekStart);

        if (!weekMap.has(weekStartStr)) {
          weekMap.set(weekStartStr, []);
        }
        weekMap.get(weekStartStr)!.push(formatDate(execDate));
      }

      // Build current week progress
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
    // Sort weeks from most recent to oldest
    const sortedWeeks = [...weekHistory].sort(
      (a, b) => b.weekStartDate.localeCompare(a.weekStartDate),
    );

    // Calculate current streak (counting backwards from most recent completed weeks)
    let currentStreak = 0;

    // If current week goal is met, include it
    if (currentWeek.isGoalMet) {
      currentStreak = 1;
    }

    // Count consecutive weeks before current week
    for (const week of sortedWeeks) {
      if (week.isGoalMet) {
        if (currentStreak > 0 || currentWeek.isGoalMet) {
          // Only continue if we've started counting
          currentStreak++;
        } else if (currentStreak === 0 && !currentWeek.isGoalMet) {
          // Current week not met, but check if last week was
          currentStreak = 1;
        }
      } else {
        // Streak broken
        if (currentStreak > 0) break;
      }
    }

    // Recalculate to handle the logic properly
    currentStreak = 0;
    if (currentWeek.isGoalMet) {
      currentStreak = 1;
    }

    // Check consecutive completed weeks before current
    for (const week of sortedWeeks) {
      if (week.isGoalMet) {
        currentStreak++;
      } else {
        break;
      }
    }

    // If current week is not met but we have previous streak, don't add current week
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

    // Calculate longest streak (across all weeks including current)
    const allWeeks = currentWeek.isGoalMet
      ? [currentWeek, ...sortedWeeks]
      : sortedWeeks;

    // Sort chronologically for longest streak calculation
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
