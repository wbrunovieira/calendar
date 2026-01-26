'use client';

/**
 * Day View Habits Section
 * Shows today's habits with completion toggle below the calendar day view
 */

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import { formatDateToString } from '@/utils/calendar/dateHelpers';
import type { Event } from '@/types/calendar';

interface DayViewHabitsSectionProps {
  date: Date;
  onHabitToggled?: () => void;
}

export default function DayViewHabitsSection({ date, onHabitToggled }: DayViewHabitsSectionProps) {
  const [habits, setHabits] = useState<Event[]>([]);
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);

  const dateString = formatDateToString(date);

  const fetchHabits = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api.events.listHabits({
        startDate: dateString,
        endDate: dateString,
      });

      // Sort by displayOrder
      const sortedHabits = [...data].sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0));
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
  }, [dateString]);

  useEffect(() => {
    fetchHabits();
  }, [fetchHabits]);

  const handleToggle = async (habit: Event) => {
    const habitId = habit.originalEventId || habit.id;
    const isCompleted = completedIds.has(habit.id);

    setToggling(habit.id);
    try {
      await api.events.toggleExecution(habitId, dateString, !isCompleted);

      setCompletedIds(prev => {
        const next = new Set(prev);
        if (isCompleted) {
          next.delete(habit.id);
        } else {
          next.add(habit.id);
        }
        return next;
      });

      if (onHabitToggled) {
        onHabitToggled();
      }
    } catch (error) {
      console.error('Failed to toggle habit:', error);
    } finally {
      setToggling(null);
    }
  };

  const completedCount = completedIds.size;
  const totalCount = habits.length;

  // Don't render if no habits
  if (!loading && habits.length === 0) {
    return null;
  }

  return (
    <div className="mt-4 bg-gradient-to-br from-white/10 to-white/5 rounded-xl border border-white/20 overflow-hidden shadow-lg">
      {/* Header */}
      <div className="px-4 py-3 border-b border-white/10 flex items-center justify-between bg-white/5">
        <div className="flex items-center gap-2">
          <span className="text-lg">🎯</span>
          <span className="text-white font-medium">Habitos de Hoje</span>
        </div>
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
      </div>

      {/* Content */}
      <div className="p-3">
        {loading ? (
          <div className="flex items-center justify-center py-6">
            <div className="w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          </div>
        ) : (
          <ul className="space-y-2">
            {habits.map(habit => {
              const isCompleted = completedIds.has(habit.id);
              const isToggling = toggling === habit.id;

              return (
                <li key={habit.id}>
                  <button
                    onClick={() => handleToggle(habit)}
                    disabled={isToggling}
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
                      {isToggling ? (
                        <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
                      ) : isCompleted ? (
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
            })}
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
  );
}
