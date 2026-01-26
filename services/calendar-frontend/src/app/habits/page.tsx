'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import type { Event, Calendar } from '@/types/calendar';
import ProfileSelector from '@/components/habits/ProfileSelector';
import CategoryBadge from '@/components/habits/CategoryBadge';

type TabType = 'habits' | 'todos';

const formatLocalDate = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const formatDisplayDate = (dateStr: string) => {
  if (!dateStr) return '';
  const cleanDate = dateStr.includes('T') ? dateStr.split('T')[0] : dateStr;
  const [year, month, day] = cleanDate.split('-').map(Number);
  if (!year || !month || !day) return '';
  return `${String(day).padStart(2, '0')}/${String(month).padStart(2, '0')}`;
};

const getPriorityLabel = (priority?: number) => {
  switch (priority) {
    case 1: return { label: 'Alta', color: 'text-red-400', bg: 'bg-red-500/20' };
    case 2: return { label: 'Media', color: 'text-yellow-400', bg: 'bg-yellow-500/20' };
    case 3: return { label: 'Baixa', color: 'text-blue-400', bg: 'bg-blue-500/20' };
    default: return null;
  }
};

const getDueDateStatus = (dueDate?: string) => {
  if (!dueDate) return null;
  const today = formatLocalDate(new Date());
  const cleanDate = dueDate.includes('T') ? dueDate.split('T')[0] : dueDate;

  if (cleanDate < today) return { label: 'Atrasado', color: 'text-red-400' };
  if (cleanDate === today) return { label: 'Hoje', color: 'text-amber-400' };
  return { label: formatDisplayDate(dueDate), color: 'text-white/60' };
};

// Calculate streak for a habit based on its executions
const calculateStreak = (habit: Event): number => {
  if (!habit.executions || habit.executions.length === 0) return 0;

  const today = new Date();
  today.setHours(0, 0, 0, 0);

  // Get completed dates sorted descending
  const completedDates = habit.executions
    .filter(e => e.completed)
    .map(e => {
      const d = new Date(e.executionDate);
      d.setHours(0, 0, 0, 0);
      return d.getTime();
    })
    .sort((a, b) => b - a);

  if (completedDates.length === 0) return 0;

  let streak = 0;
  let currentDate = today.getTime();

  // Check if today or yesterday was completed
  const oneDayMs = 24 * 60 * 60 * 1000;
  const yesterday = currentDate - oneDayMs;

  if (!completedDates.includes(currentDate) && !completedDates.includes(yesterday)) {
    return 0;
  }

  // Start counting from the most recent completed date
  currentDate = completedDates[0];

  for (const completedDate of completedDates) {
    if (completedDate === currentDate) {
      streak++;
      currentDate -= oneDayMs;
    } else if (completedDate < currentDate - oneDayMs) {
      break;
    }
  }

  return streak;
};

export default function HabitsPage() {
  const [activeTab, setActiveTab] = useState<TabType>('habits');
  const [habits, setHabits] = useState<Event[]>([]);
  const [todos, setTodos] = useState<Event[]>([]);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [calendarsLoading, setCalendarsLoading] = useState(true);
  const [showCompleted, setShowCompleted] = useState(false);

  const today = formatLocalDate(new Date());

  // Calculate date range for fetching (last 30 days to next 30 days)
  const dateRange = useMemo(() => {
    const start = new Date();
    start.setDate(start.getDate() - 30);
    const end = new Date();
    end.setDate(end.getDate() + 30);
    return {
      startDate: formatLocalDate(start),
      endDate: formatLocalDate(end),
    };
  }, []);

  // Fetch calendars on mount
  useEffect(() => {
    const fetchCalendars = async () => {
      try {
        setCalendarsLoading(true);
        const calendarsData = await api.calendars.list();
        setCalendars(calendarsData || []);
      } catch (error) {
        console.error('Error fetching calendars:', error);
      } finally {
        setCalendarsLoading(false);
      }
    };
    fetchCalendars();
  }, []);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      const params = {
        ...dateRange,
        ...(selectedCalendarId && { calendarId: selectedCalendarId }),
      };

      const [habitsData, todosData] = await Promise.all([
        api.events.listHabits(params),
        api.events.listTodos(params),
      ]);

      setHabits(habitsData || []);
      setTodos(todosData || []);
    } catch (error) {
      console.error('Error fetching data:', error);
    } finally {
      setLoading(false);
    }
  }, [dateRange, selectedCalendarId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const toggleHabitCompletion = async (habit: Event) => {
    const eventId = habit.originalEventId || habit.id;
    const occurrenceDate = habit.occurrenceDate || today;
    const currentlyCompleted = habit.executions?.some(
      e => e.executionDate.split('T')[0] === occurrenceDate && e.completed
    );

    try {
      await api.events.toggleExecution(eventId, occurrenceDate, !currentlyCompleted);
      await fetchData();
    } catch (error) {
      console.error('Error toggling habit:', error);
    }
  };

  const toggleTodoCompletion = async (todo: Event) => {
    const currentlyCompleted = todo.executions?.some(e => e.completed);

    try {
      await api.events.toggleExecution(todo.id, today, !currentlyCompleted);
      await fetchData();
    } catch (error) {
      console.error('Error toggling todo:', error);
    }
  };

  // Get calendar name for display
  const getCalendarName = (calendarId: string) => {
    const calendar = calendars.find(c => c.id === calendarId);
    return calendar?.name || '';
  };

  // Group habits by unique habit (not occurrence)
  const uniqueHabits = useMemo(() => {
    const habitMap = new Map<string, Event & { streak: number }>();

    habits.forEach(habit => {
      const masterId = habit.originalEventId || habit.id;
      if (!habitMap.has(masterId)) {
        habitMap.set(masterId, { ...habit, streak: calculateStreak(habit) });
      }
    });

    return Array.from(habitMap.values());
  }, [habits]);

  // Today's habits (occurrences for today)
  const todayHabits = useMemo(() => {
    return habits.filter(h => {
      const occDate = h.occurrenceDate || h.startDate?.split('T')[0];
      return occDate === today;
    });
  }, [habits, today]);

  // Pending and completed todos
  const pendingTodos = useMemo(() => {
    return todos.filter(t => !t.executions?.some(e => e.completed));
  }, [todos]);

  const completedTodos = useMemo(() => {
    return todos.filter(t => t.executions?.some(e => e.completed));
  }, [todos]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#350545] via-[#4a0860] to-[#792990] p-6">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <Link
            href="/"
            className="inline-flex items-center gap-2 text-white/60 hover:text-white transition-colors mb-4"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            <span>Voltar ao Calendario</span>
          </Link>
          <h1 className="text-2xl font-bold text-white mb-2">Habitos & Tarefas</h1>
          <p className="text-white/60">Gerencie seus habitos diarios e tarefas pendentes</p>
        </div>

        {/* Profile Selector */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-white/70 mb-2">Perfil</h3>
          <ProfileSelector
            calendars={calendars}
            selectedId={selectedCalendarId}
            onSelect={setSelectedCalendarId}
            loading={calendarsLoading}
          />
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setActiveTab('habits')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              activeTab === 'habits'
                ? 'bg-purple-600 text-white'
                : 'bg-white/10 text-white/70 hover:bg-white/20'
            }`}
          >
            Habitos ({uniqueHabits.length})
          </button>
          <button
            onClick={() => setActiveTab('todos')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              activeTab === 'todos'
                ? 'bg-purple-600 text-white'
                : 'bg-white/10 text-white/70 hover:bg-white/20'
            }`}
          >
            Tarefas ({pendingTodos.length})
          </button>
        </div>

        {/* Loading State */}
        {loading ? (
          <div className="animate-pulse space-y-4">
            <div className="h-64 bg-white/10 rounded-2xl"></div>
          </div>
        ) : (
          <>
            {/* Habits Tab */}
            {activeTab === 'habits' && (
              <div className="space-y-6">
                {/* Today's Habits */}
                <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                  <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
                    <span>Habitos de Hoje</span>
                    <span className="text-sm font-normal text-white/60">
                      {todayHabits.filter(h => h.executions?.some(e => e.completed && e.executionDate.split('T')[0] === today)).length}/{todayHabits.length}
                    </span>
                  </h2>

                  {todayHabits.length === 0 ? (
                    <p className="text-white/60 text-center py-4">Nenhum habito para hoje</p>
                  ) : (
                    <div className="space-y-3">
                      {todayHabits.map(habit => {
                        const isCompleted = habit.executions?.some(
                          e => e.executionDate.split('T')[0] === today && e.completed
                        );

                        return (
                          <div
                            key={habit.id}
                            className={`flex items-center justify-between p-4 rounded-xl transition-colors ${
                              isCompleted ? 'bg-emerald-500/20 border border-emerald-400/30' : 'bg-white/5 hover:bg-white/10'
                            }`}
                          >
                            <div className="flex items-center gap-3">
                              <button
                                onClick={() => toggleHabitCompletion(habit)}
                                className={`w-6 h-6 rounded-full border-2 flex items-center justify-center transition-colors ${
                                  isCompleted
                                    ? 'bg-emerald-500 border-emerald-500 text-white'
                                    : 'border-white/40 hover:border-white/60'
                                }`}
                              >
                                {isCompleted && (
                                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                  </svg>
                                )}
                              </button>
                              <div>
                                <p className={`font-medium ${isCompleted ? 'text-emerald-300 line-through' : 'text-white'}`}>
                                  {habit.title}
                                </p>
                                {habit.description && (
                                  <p className="text-sm text-white/50">{habit.description}</p>
                                )}
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              {selectedCalendarId === null && habit.calendarId && (
                                <span className="text-xs text-white/40 bg-white/5 px-2 py-1 rounded">
                                  {getCalendarName(habit.calendarId)}
                                </span>
                              )}
                              <span className="text-white/40 text-sm">{habit.startTime}</span>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>

                {/* All Habits */}
                <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                  <h2 className="text-lg font-semibold text-white mb-4">Todos os Habitos</h2>

                  {uniqueHabits.length === 0 ? (
                    <p className="text-white/60 text-center py-4">Nenhum habito cadastrado</p>
                  ) : (
                    <div className="space-y-3">
                      {uniqueHabits.map(habit => (
                        <div
                          key={habit.id}
                          className="flex items-center justify-between p-4 bg-white/5 rounded-xl hover:bg-white/10 transition-colors"
                        >
                          <div className="flex items-center gap-3">
                            <div>
                              <p className="font-medium text-white">{habit.title}</p>
                              <p className="text-sm text-white/50">
                                {habit.recurrenceFrequency === 'daily' && 'Diario'}
                                {habit.recurrenceFrequency === 'weekly' && 'Semanal'}
                                {habit.recurrenceFrequency === 'monthly' && 'Mensal'}
                                {habit.startTime && ` • ${habit.startTime}`}
                                {selectedCalendarId === null && habit.calendarId && ` • ${getCalendarName(habit.calendarId)}`}
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center gap-3">
                            {habit.streak > 0 && (
                              <span className="flex items-center gap-1 px-2 py-1 bg-orange-500/20 text-orange-400 rounded-lg text-sm">
                                <span>🔥</span>
                                <span>{habit.streak}</span>
                              </span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Todos Tab */}
            {activeTab === 'todos' && (
              <div className="space-y-6">
                {/* Pending Todos */}
                <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                  <h2 className="text-lg font-semibold text-white mb-4">
                    Tarefas Pendentes ({pendingTodos.length})
                  </h2>

                  {pendingTodos.length === 0 ? (
                    <p className="text-white/60 text-center py-4">Nenhuma tarefa pendente</p>
                  ) : (
                    <div className="space-y-3">
                      {pendingTodos
                        .sort((a, b) => {
                          // Sort by priority first (1 = high priority comes first)
                          if (a.priority && b.priority) return a.priority - b.priority;
                          if (a.priority) return -1;
                          if (b.priority) return 1;
                          // Then by due date
                          if (a.dueDate && b.dueDate) return a.dueDate.localeCompare(b.dueDate);
                          if (a.dueDate) return -1;
                          if (b.dueDate) return 1;
                          return 0;
                        })
                        .map(todo => {
                          const priority = getPriorityLabel(todo.priority);
                          const dueStatus = getDueDateStatus(todo.dueDate);

                          return (
                            <div
                              key={todo.id}
                              className="flex items-center justify-between p-4 bg-white/5 rounded-xl hover:bg-white/10 transition-colors"
                            >
                              <div className="flex items-center gap-3">
                                <button
                                  onClick={() => toggleTodoCompletion(todo)}
                                  className="w-6 h-6 rounded border-2 border-white/40 hover:border-white/60 transition-colors"
                                />
                                <div>
                                  <p className="font-medium text-white">{todo.title}</p>
                                  {todo.description && (
                                    <p className="text-sm text-white/50">{todo.description}</p>
                                  )}
                                  <div className="flex items-center gap-2 mt-1">
                                    {selectedCalendarId === null && todo.calendarId && (
                                      <span className="text-xs text-white/40">{getCalendarName(todo.calendarId)}</span>
                                    )}
                                    {todo.category && (
                                      <CategoryBadge category={todo.category} />
                                    )}
                                  </div>
                                </div>
                              </div>
                              <div className="flex items-center gap-2">
                                {priority && (
                                  <span className={`px-2 py-1 rounded text-xs ${priority.bg} ${priority.color}`}>
                                    {priority.label}
                                  </span>
                                )}
                                {dueStatus && (
                                  <span className={`text-sm ${dueStatus.color}`}>
                                    {dueStatus.label}
                                  </span>
                                )}
                              </div>
                            </div>
                          );
                        })}
                    </div>
                  )}
                </div>

                {/* Completed Todos */}
                {completedTodos.length > 0 && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    <button
                      onClick={() => setShowCompleted(!showCompleted)}
                      className="flex items-center justify-between w-full text-left"
                    >
                      <h2 className="text-lg font-semibold text-white">
                        Concluidas ({completedTodos.length})
                      </h2>
                      <svg
                        className={`w-5 h-5 text-white/60 transition-transform ${showCompleted ? 'rotate-180' : ''}`}
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>

                    {showCompleted && (
                      <div className="mt-4 space-y-3">
                        {completedTodos.map(todo => (
                          <div
                            key={todo.id}
                            className="flex items-center justify-between p-4 bg-emerald-500/10 rounded-xl"
                          >
                            <div className="flex items-center gap-3">
                              <button
                                onClick={() => toggleTodoCompletion(todo)}
                                className="w-6 h-6 rounded border-2 bg-emerald-500 border-emerald-500 flex items-center justify-center"
                              >
                                <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                </svg>
                              </button>
                              <div>
                                <p className="font-medium text-white/60 line-through">{todo.title}</p>
                                <div className="flex items-center gap-2 mt-1">
                                  {selectedCalendarId === null && todo.calendarId && (
                                    <span className="text-xs text-white/40">{getCalendarName(todo.calendarId)}</span>
                                  )}
                                  {todo.category && (
                                    <CategoryBadge category={todo.category} />
                                  )}
                                </div>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
