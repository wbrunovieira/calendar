'use client';

/**
 * Day View Habits Section
 * Shows today's habits with completion toggle below the calendar day view
 * Collapsible with smooth animation
 * Groups habits by profile/calendar
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import { api } from '@/lib/api';
import { formatDateToString } from '@/utils/calendar/dateHelpers';
import { calendars } from '@/data/calendars';
import type { Event } from '@/types/calendar';

interface DayViewHabitsSectionProps {
  date: Date;
}

export default function DayViewHabitsSection({ date }: DayViewHabitsSectionProps) {
  const [habits, setHabits] = useState<Event[]>([]);
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [isExpanded, setIsExpanded] = useState(true);

  const dateString = formatDateToString(date);

  const fetchHabits = useCallback(async () => {
    try {
      setLoading(true);
      // Use a wider date range to fetch recurring habits
      // The backend expands recurring events within the date range
      const startRange = new Date(date);
      startRange.setDate(startRange.getDate() - 30);
      const endRange = new Date(date);
      endRange.setDate(endRange.getDate() + 30);

      const data = await api.events.listHabits({
        startDate: formatDateToString(startRange),
        endDate: formatDateToString(endRange),
      });

      // Filter habits that occur on this specific date
      const todayHabits = data.filter(h => {
        const occDate = h.occurrenceDate || h.startDate?.split('T')[0];
        return occDate === dateString;
      });

      // Sort by displayOrder
      const sortedHabits = [...todayHabits].sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0));
      setHabits(sortedHabits);

      // Check which habits are completed for this date
      const completed = new Set<string>();
      for (const habit of sortedHabits) {
        const execution = habit.executions?.find(e => {
          const execDate = e.executionDate
            ? (typeof e.executionDate === 'string' ? e.executionDate.split('T')[0] : formatDateToString(new Date(e.executionDate)))
            : '';
          return execDate === dateString && e.completed;
        });
        if (execution) {
          completed.add(habit.id);
        }
      }
      setCompletedIds(completed);
    } catch (error) {
      console.error('Failed to fetch habits:', error);
    } finally {
      setLoading(false);
    }
  }, [date, dateString]);

  useEffect(() => {
    fetchHabits();
  }, [fetchHabits]);

  // Group habits by calendar/profile
  const habitsByCalendar = useMemo(() => {
    const grouped = new Map<string, Event[]>();

    for (const habit of habits) {
      const calendarId = habit.calendarId;
      if (!grouped.has(calendarId)) {
        grouped.set(calendarId, []);
      }
      grouped.get(calendarId)!.push(habit);
    }

    return grouped;
  }, [habits]);

  // Check if we have multiple profiles
  const hasMultipleProfiles = habitsByCalendar.size > 1;

  const handleToggle = async (habit: Event) => {
    const habitId = habit.originalEventId || habit.id;
    const isCompleted = completedIds.has(habit.id);

    // Optimistic update — update UI immediately
    setCompletedIds(prev => {
      const next = new Set(prev);
      if (isCompleted) {
        next.delete(habit.id);
      } else {
        next.add(habit.id);
      }
      return next;
    });

    try {
      await api.events.toggleExecution(habitId, dateString, !isCompleted);
    } catch (error) {
      console.error('Failed to toggle habit:', error);
      // Revert on error
      setCompletedIds(prev => {
        const next = new Set(prev);
        if (isCompleted) {
          next.add(habit.id);
        } else {
          next.delete(habit.id);
        }
        return next;
      });
    }
  };

  const completedCount = completedIds.size;
  const totalCount = habits.length;

  // Render a single habit item
  const renderHabitItem = (habit: Event) => {
    const isCompleted = completedIds.has(habit.id);

    return (
      <li key={habit.id}>
        <button
          onClick={() => handleToggle(habit)}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-all duration-200 ${
            isCompleted
              ? 'bg-green-500/20 hover:bg-green-500/30 border border-green-500/30'
              : 'bg-white/5 hover:bg-white/10 border border-white/10 hover:border-white/20'
          }`}
        >
          {/* Checkbox */}
          <div
            className={`w-5 h-5 rounded-md border-2 flex-shrink-0 flex items-center justify-center transition-all duration-200 ${
              isCompleted
                ? 'bg-green-500 border-green-500'
                : 'border-white/40 hover:border-white/60'
            }`}
          >
            {isCompleted ? (
              <svg className="w-3.5 h-3.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
              </svg>
            ) : null}
          </div>

          {/* Title and time */}
          <div className="flex-1 min-w-0">
            <span
              className={`block text-sm font-medium truncate ${
                isCompleted ? 'text-white/60 line-through' : 'text-white'
              }`}
            >
              {habit.title}
            </span>
            {habit.startTime && (
              <span className="text-xs text-white/40">
                {habit.startTime}
              </span>
            )}
          </div>

          {/* Completed indicator */}
          {isCompleted && (
            <span className="text-green-400 text-xs font-medium">Concluido</span>
          )}
        </button>
      </li>
    );
  };

  return (
    <div className="mt-4 bg-gradient-to-br from-white/10 to-white/5 rounded-xl border border-white/20 overflow-hidden shadow-lg">
      {/* Header - Clickable to expand/collapse */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 py-3 border-b border-white/10 flex items-center justify-between bg-white/5 hover:bg-white/10 transition-colors duration-200"
      >
        <div className="flex items-center gap-2">
          <span className="text-lg">🎯</span>
          <span className="text-white font-medium">Habitos de Hoje</span>
        </div>
        <div className="flex items-center gap-3">
          {!loading && totalCount > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-white/90 font-semibold text-lg">
                {completedCount}/{totalCount}
              </span>
              {completedCount === totalCount && totalCount > 0 && (
                <span className="text-green-400">✓</span>
              )}
            </div>
          )}
          {/* Chevron icon with rotation animation */}
          <svg
            className={`w-5 h-5 text-white/60 transition-transform duration-300 ease-in-out ${
              isExpanded ? 'rotate-180' : 'rotate-0'
            }`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </button>

      {/* Collapsible Content with smooth animation */}
      <div
        className={`transition-all duration-300 ease-in-out overflow-hidden ${
          isExpanded ? 'max-h-[500px] opacity-100' : 'max-h-0 opacity-0'
        }`}
      >
        {/* Content */}
        <div className="p-3 max-h-64 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-6">
              <div className="w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            </div>
          ) : habits.length === 0 ? (
            <p className="text-white/40 text-sm text-center py-4">
              Nenhum habito para hoje
            </p>
          ) : hasMultipleProfiles ? (
            // Grouped by profile
            <div className="space-y-4">
              {Array.from(habitsByCalendar.entries()).map(([calendarId, calendarHabits]) => {
                const calendar = calendars.find(c => c.id === calendarId);
                const calendarCompletedCount = calendarHabits.filter(h => completedIds.has(h.id)).length;

                return (
                  <div key={calendarId}>
                    {/* Profile header */}
                    <div className="flex items-center gap-2 mb-2 px-1">
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: calendar?.color || '#666' }}
                      />
                      <span className="text-white/70 text-xs font-medium">
                        {calendar?.name || 'Desconhecido'}
                      </span>
                      <span className="text-white/40 text-xs">
                        ({calendarCompletedCount}/{calendarHabits.length})
                      </span>
                    </div>
                    {/* Habits list */}
                    <ul className="space-y-2">
                      {calendarHabits.map(renderHabitItem)}
                    </ul>
                  </div>
                );
              })}
            </div>
          ) : (
            // Single profile - no grouping needed
            <ul className="space-y-2">
              {habits.map(renderHabitItem)}
            </ul>
          )}
        </div>

        {/* Progress bar */}
        {!loading && totalCount > 0 && (
          <div className="px-4 pb-3">
            <div className="h-2 bg-white/10 rounded-full overflow-hidden">
              <div
                className={`h-full transition-all duration-500 ${
                  completedCount === totalCount
                    ? 'bg-gradient-to-r from-green-500 to-emerald-400'
                    : 'bg-gradient-to-r from-blue-500 to-cyan-400'
                }`}
                style={{ width: `${totalCount > 0 ? (completedCount / totalCount) * 100 : 0}%` }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
