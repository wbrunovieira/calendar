'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { Event } from '@/types/calendar';

interface SidebarHabitsWidgetProps {
  isCollapsed: boolean;
}

export default function SidebarHabitsWidget({ isCollapsed }: SidebarHabitsWidgetProps) {
  const [habits, setHabits] = useState<Event[]>([]);
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);

  const today = new Date().toISOString().split('T')[0];

  const fetchHabits = useCallback(async () => {
    try {
      const data = await api.events.listHabits({
        startDate: today,
        endDate: today,
      });
      setHabits(data);

      // Check which habits are completed today
      const completed = new Set<string>();
      for (const habit of data) {
        if (habit.executions?.some(e => e.executionDate === today && e.completed)) {
          completed.add(habit.id);
        }
      }
      setCompletedIds(completed);
    } catch (error) {
      console.error('Failed to fetch habits:', error);
    } finally {
      setLoading(false);
    }
  }, [today]);

  useEffect(() => {
    fetchHabits();
  }, [fetchHabits]);

  const handleToggle = async (habit: Event) => {
    const habitId = habit.originalEventId || habit.id;
    const isCompleted = completedIds.has(habit.id);

    setToggling(habit.id);
    try {
      await api.events.toggleExecution(habitId, today, !isCompleted);

      setCompletedIds(prev => {
        const next = new Set(prev);
        if (isCompleted) {
          next.delete(habit.id);
        } else {
          next.add(habit.id);
        }
        return next;
      });
    } catch (error) {
      console.error('Failed to toggle habit:', error);
    } finally {
      setToggling(null);
    }
  };

  if (isCollapsed) {
    return null;
  }

  const completedCount = completedIds.size;
  const totalCount = habits.length;

  return (
    <div className="px-3 mt-4">
      <div className="bg-white/5 rounded-lg border border-white/10 overflow-hidden">
        {/* Header */}
        <div className="px-3 py-2 border-b border-white/10 flex items-center justify-between">
          <span className="text-white/80 text-xs font-medium">Habitos de Hoje</span>
          {!loading && totalCount > 0 && (
            <span className="text-white/50 text-xs">
              {completedCount}/{totalCount}
            </span>
          )}
        </div>

        {/* Content */}
        <div className="p-2 max-h-48 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-3">
              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            </div>
          ) : habits.length === 0 ? (
            <p className="text-white/40 text-xs text-center py-2">
              Nenhum habito para hoje
            </p>
          ) : (
            <ul className="space-y-1">
              {habits.map(habit => {
                const isCompleted = completedIds.has(habit.id);
                const isToggling = toggling === habit.id;

                return (
                  <li key={habit.id}>
                    <button
                      onClick={() => handleToggle(habit)}
                      disabled={isToggling}
                      className={`w-full flex items-center gap-2 px-2 py-1.5 rounded text-left transition-all duration-200 ${
                        isCompleted
                          ? 'bg-green-500/20 hover:bg-green-500/30'
                          : 'bg-white/5 hover:bg-white/10'
                      }`}
                    >
                      {/* Checkbox */}
                      <div
                        className={`w-4 h-4 rounded border flex-shrink-0 flex items-center justify-center transition-all duration-200 ${
                          isCompleted
                            ? 'bg-green-500 border-green-500'
                            : 'border-white/30 hover:border-white/50'
                        }`}
                      >
                        {isToggling ? (
                          <div className="w-2 h-2 border border-white border-t-transparent rounded-full animate-spin" />
                        ) : isCompleted ? (
                          <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                          </svg>
                        ) : null}
                      </div>

                      {/* Title */}
                      <span
                        className={`text-xs truncate ${
                          isCompleted ? 'text-white/60 line-through' : 'text-white/90'
                        }`}
                      >
                        {habit.title}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {/* Progress bar */}
        {!loading && totalCount > 0 && (
          <div className="px-3 pb-2">
            <div className="h-1 bg-white/10 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-green-500 to-emerald-400 transition-all duration-500"
                style={{ width: `${(completedCount / totalCount) * 100}%` }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
