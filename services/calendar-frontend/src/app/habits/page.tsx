'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import type { Event, Calendar, FlexibleHabitProgress, HabitStats } from '@/types/calendar';
import ProfileSelector from '@/components/habits/ProfileSelector';
import CategoryBadge from '@/components/habits/CategoryBadge';
import CategoryTypeBadge from '@/components/habits/CategoryTypeBadge';
import LabelBadge from '@/components/labels/LabelBadge';
import CreateHabitTodoModal from '@/components/habits/CreateHabitTodoModal';
import EditTodoModal from '@/components/habits/EditTodoModal';
import EditHabitModal from '@/components/habits/EditHabitModal';
import HabitAmountStepper from '@/components/habits/HabitAmountStepper';
import HabitDetailPanel from '@/components/habits/HabitDetailPanel';
import HabitHeatmap from '@/components/habits/HabitHeatmap';
import QuitHabitCard from '@/components/habits/QuitHabitCard';
import {
  HABIT_IDEAS,
  type HabitIdea,
  aggregateHeatmap,
  groupByTimeOfDay,
  isQuitHabit,
  loggedAmount,
} from '@/lib/habits';

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

// Get week days (Monday to Sunday) with optional offset from current week
const getWeekDays = (offset: number = 0) => {
  const today = new Date();
  const todayStr = formatLocalDate(today);
  const dayOfWeek = today.getDay(); // 0 = Sunday, 1 = Monday, ...
  const diffToMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;

  const monday = new Date(today);
  monday.setDate(today.getDate() + diffToMonday + offset * 7);

  const days: { date: string; dayCode: string; dayLabel: string; isToday: boolean; isPast: boolean }[] = [];
  const dayLabels = ['Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sab', 'Dom'];
  const dayCodes = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];

  const todayStart = new Date(today);
  todayStart.setHours(0, 0, 0, 0);

  for (let i = 0; i < 7; i++) {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    const dateStr = formatLocalDate(d);
    days.push({
      date: dateStr,
      dayCode: dayCodes[i],
      dayLabel: dayLabels[i],
      isToday: dateStr === todayStr,
      isPast: d < todayStart,
    });
  }

  return days;
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

export default function HabitsPage() {
  const [activeTab, setActiveTab] = useState<TabType>('habits');
  const [habits, setHabits] = useState<Event[]>([]);
  const [todos, setTodos] = useState<Event[]>([]);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [calendarsLoading, setCalendarsLoading] = useState(true);
  const [showCompleted, setShowCompleted] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingTodo, setEditingTodo] = useState<Event | null>(null);
  const [editingHabit, setEditingHabit] = useState<Event | null>(null);
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [flexibleProgress, setFlexibleProgress] = useState<FlexibleHabitProgress[]>([]);
  const [weekOffset, setWeekOffset] = useState(0);
  const [habitStats, setHabitStats] = useState<HabitStats[]>([]);
  const [expandedHabitId, setExpandedHabitId] = useState<string | null>(null);

  const today = formatLocalDate(new Date());
  const currentWeekDays = useMemo(() => getWeekDays(0), []);
  const navigatedWeekDays = useMemo(() => getWeekDays(weekOffset), [weekOffset]);

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

      const progressParams = selectedCalendarId ? { calendarId: selectedCalendarId } : undefined;

      const [habitsData, todosData, progressData, statsData] = await Promise.all([
        api.events.listHabits(params),
        api.events.listTodos(params),
        api.events.getWeeklyProgress(progressParams),
        api.events.getHabitsStats(progressParams),
      ]);

      setHabits(habitsData || []);
      setTodos(todosData || []);
      setFlexibleProgress(progressData || []);
      setHabitStats(statsData || []);
    } catch (error) {
      console.error('Error fetching data:', error);
    } finally {
      setLoading(false);
    }
  }, [dateRange, selectedCalendarId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Background refresh without loading spinner
  const refreshData = useCallback(async () => {
    try {
      const params = {
        ...dateRange,
        ...(selectedCalendarId && { calendarId: selectedCalendarId }),
      };
      const progressParams = selectedCalendarId ? { calendarId: selectedCalendarId } : undefined;

      const [habitsData, todosData, progressData, statsData] = await Promise.all([
        api.events.listHabits(params),
        api.events.listTodos(params),
        api.events.getWeeklyProgress(progressParams),
        api.events.getHabitsStats(progressParams),
      ]);

      setHabits(habitsData || []);
      setTodos(todosData || []);
      setFlexibleProgress(progressData || []);
      setHabitStats(statsData || []);
    } catch (error) {
      console.error('Error refreshing data:', error);
    }
  }, [dateRange, selectedCalendarId]);

  const toggleHabitCompletion = async (habit: Event) => {
    const eventId = habit.originalEventId || habit.id;
    const occurrenceDate = habit.occurrenceDate || today;
    const currentlyCompleted = habit.executions?.some(
      e => e.executionDate.split('T')[0] === occurrenceDate && e.completed
    );

    // Optimistic update — toggle execution in local state immediately
    setHabits(prev => prev.map(h => {
      if (h.id !== habit.id) return h;
      const executions = h.executions || [];
      if (currentlyCompleted) {
        return { ...h, executions: executions.filter(e => !(e.executionDate.split('T')[0] === occurrenceDate && e.completed)) };
      }
      return { ...h, executions: [...executions, { id: 'temp', eventId, executionDate: occurrenceDate, completed: true }] };
    }));

    try {
      await api.events.toggleExecution(eventId, occurrenceDate, !currentlyCompleted);
      refreshData();
    } catch (error) {
      console.error('Error toggling habit:', error);
      refreshData(); // Revert by refetching
    }
  };

  // Amount habits: the check fills the day to its target, or clears it
  const setHabitAmount = async (habit: Event, value: number) => {
    const eventId = habit.originalEventId || habit.id;
    const date = habit.occurrenceDate || today;
    const target = habit.dailyTarget || 0;

    setHabits(prev => prev.map(h => {
      if (h.id !== habit.id) return h;
      const others = (h.executions || []).filter(e => e.executionDate.split('T')[0] !== date);
      return { ...h, executions: [...others, { id: 'temp', eventId, executionDate: date, completed: value >= target, value }] };
    }));

    try {
      await api.events.logHabitAmount(eventId, date, value);
    } catch (error) {
      console.error('Error logging habit amount:', error);
    }
    refreshData();
  };

  const logSlip = async (habit: Event, trigger?: string, note?: string) => {
    try {
      await api.events.logHabitSlip(habit.originalEventId || habit.id, today, trigger, note);
    } catch (error) {
      console.error('Error logging slip:', error);
    }
    await refreshData();
  };

  const undoSlip = async (habit: Event) => {
    try {
      await api.events.removeHabitSlip(habit.originalEventId || habit.id, today);
    } catch (error) {
      console.error('Error removing slip:', error);
    }
    await refreshData();
  };

  const togglePause = async (habitId: string, paused: boolean) => {
    try {
      await api.events.update(habitId, { isPaused: !paused });
    } catch (error) {
      console.error('Error pausing habit:', error);
    }
    refreshData();
  };

  const createFromIdea = async (idea: HabitIdea) => {
    const calendarId = selectedCalendarId || calendars[0]?.id;
    if (!calendarId) return;
    try {
      await api.events.create({
        calendarId,
        title: idea.title,
        eventType: 'HABIT',
        startDate: today,
        startTime: idea.timeOfDay === 'EVENING' ? '21:00' : idea.timeOfDay === 'MORNING' ? '07:00' : '09:00',
        recurrenceRule: 'FREQ=DAILY',
        recurrenceType: 'FIXED',
        habitKind: idea.habitKind,
        dailyTarget: idea.dailyTarget ?? null,
        targetUnit: idea.targetUnit ?? null,
        timeOfDay: idea.timeOfDay ?? 'ANYTIME',
      });
    } catch (error) {
      console.error('Error creating habit from idea:', error);
    }
    fetchData();
  };

  const toggleFlexibleHabitDay = async (habit: Event, dateStr: string, completedDates: string[]) => {
    const eventId = habit.originalEventId || habit.id;
    const isCompleted = completedDates.includes(dateStr);

    try {
      await api.events.toggleExecution(eventId, dateStr, !isCompleted);
      refreshData();
    } catch (error) {
      console.error('Error toggling flexible habit day:', error);
    }
  };

  const toggleTodoCompletion = async (todo: Event) => {
    const currentlyCompleted = todo.executions?.some(e => e.completed);

    // Optimistic update
    setTodos(prev => prev.map(t => {
      if (t.id !== todo.id) return t;
      const executions = t.executions || [];
      if (currentlyCompleted) {
        return { ...t, executions: executions.filter(e => !e.completed) };
      }
      return { ...t, executions: [...executions, { id: 'temp', eventId: todo.id, executionDate: today, completed: true }] };
    }));

    try {
      await api.events.toggleExecution(todo.id, today, !currentlyCompleted);
      refreshData();
    } catch (error) {
      console.error('Error toggling todo:', error);
      refreshData(); // Revert by refetching
    }
  };

  // Get calendar name for display
  const getCalendarName = (calendarId: string) => {
    const calendar = calendars.find(c => c.id === calendarId);
    return calendar?.name || '';
  };

  // Group habits by unique habit (not occurrence) - separate fixed and flexible
  const statsById = useMemo(() => new Map(habitStats.map(st => [st.habitId, st])), [habitStats]);

  const { uniqueHabits, fixedHabits, flexibleHabitsWithProgress, quitHabits, pausedHabits } = useMemo(() => {
    const habitMap = new Map<string, Event & { streak: number }>();

    habits.forEach(habit => {
      const masterId = habit.originalEventId || habit.id;
      if (!habitMap.has(masterId)) {
        habitMap.set(masterId, { ...habit, streak: statsById.get(masterId)?.currentStreak ?? 0 });
      }
    });

    const all = Array.from(habitMap.values());
    const active = all.filter(h => !h.isPaused);

    // Separate fixed, flexible and quit habits
    const fixed = active.filter(h => h.recurrenceType !== 'FLEXIBLE' && !isQuitHabit(h));

    // Match flexible habits with their progress data
    const flexible = active
      .filter(h => h.recurrenceType === 'FLEXIBLE' && !isQuitHabit(h))
      .map(h => {
        const progress = flexibleProgress.find(p => p.habitId === (h.originalEventId || h.id));
        return { ...h, progress };
      });

    return {
      uniqueHabits: all,
      fixedHabits: fixed,
      flexibleHabitsWithProgress: flexible,
      quitHabits: active.filter(isQuitHabit),
      pausedHabits: all.filter(h => h.isPaused),
    };
  }, [habits, flexibleProgress, statsById]);

  // Today's habits (occurrences for today)
  const todayHabits = useMemo(() => {
    return habits.filter(h => {
      const occDate = h.occurrenceDate || h.startDate?.split('T')[0];
      return occDate === today && !h.isPaused && !isQuitHabit(h);
    });
  }, [habits, today]);

  const todayGroups = useMemo(() => groupByTimeOfDay(todayHabits), [todayHabits]);

  const summary = useMemo(() => {
    const isDone = (h: Event) =>
      h.executions?.some(e => e.completed && e.executionDate.split('T')[0] === today) ?? false;
    const rates = habitStats
      .filter(st => st.habitKind !== 'QUIT' && !st.isPaused && st.completionRate30d !== null)
      .map(st => st.completionRate30d as number);
    const cleanRuns = habitStats.filter(st => st.habitKind === 'QUIT' && !st.isPaused).map(st => st.currentStreak);
    return {
      doneToday: todayHabits.filter(isDone).length,
      rate30: rates.length ? Math.round(rates.reduce((a, b) => a + b, 0) / rates.length) : null,
      longestClean: cleanRuns.length ? Math.max(...cleanRuns) : null,
    };
  }, [todayHabits, habitStats, today]);

  const overallHeatmap = useMemo(() => aggregateHeatmap(habitStats), [habitStats]);

  const existingTitles = useMemo(
    () => new Set(uniqueHabits.map(h => h.title.toLowerCase())),
    [uniqueHabits],
  );

  // Pending and completed todos - sorted by displayOrder
  const pendingTodos = useMemo(() => {
    return todos
      .filter(t => !t.executions?.some(e => e.completed))
      .sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0));
  }, [todos]);

  const completedTodos = useMemo(() => {
    return todos
      .filter(t => t.executions?.some(e => e.completed))
      .sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0));
  }, [todos]);

  // Drag and drop handlers
  const handleDragStart = (e: React.DragEvent, id: string) => {
    setDraggedId(id);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = async (e: React.DragEvent, targetId: string, list: (Event & { streak?: number })[]) => {
    e.preventDefault();
    if (!draggedId || draggedId === targetId) {
      setDraggedId(null);
      return;
    }

    const draggedIndex = list.findIndex(item => (item.originalEventId || item.id) === draggedId);
    const targetIndex = list.findIndex(item => (item.originalEventId || item.id) === targetId);

    if (draggedIndex === -1 || targetIndex === -1) {
      setDraggedId(null);
      return;
    }

    // Reorder the list
    const newList = [...list];
    const [draggedItem] = newList.splice(draggedIndex, 1);
    newList.splice(targetIndex, 0, draggedItem);

    // Get the master IDs in the new order
    const orderedIds = newList.map(item => item.originalEventId || item.id);

    try {
      await api.events.reorder(orderedIds);
      await fetchData();
    } catch (error) {
      console.error('Error reordering:', error);
    }

    setDraggedId(null);
  };

  const handleDragEnd = () => {
    setDraggedId(null);
  };

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

        {/* Tabs and Create Button */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex gap-2">
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

          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            <span>Criar {activeTab === 'habits' ? 'Habito' : 'Tarefa'}</span>
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
                {/* Summary */}
                <div className="grid grid-cols-3 gap-3">
                  {[
                    { label: 'Feitos hoje', value: `${summary.doneToday}/${todayHabits.length}` },
                    { label: 'Conclusão, últimos 30 dias', value: summary.rate30 === null ? '–' : `${summary.rate30}%` },
                    { label: 'Maior sequência limpa agora', value: summary.longestClean === null ? '–' : String(summary.longestClean) },
                  ].map(c => (
                    <div key={c.label} className="bg-white/10 rounded-2xl p-4 border border-white/20">
                      <div className="text-2xl font-bold text-white tabular-nums">{c.value}</div>
                      <div className="text-xs text-white/60">{c.label}</div>
                    </div>
                  ))}
                </div>

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
                    <div className="space-y-4">
                      {todayGroups.map(group => (
                      <div key={group.key}>
                      {todayGroups.length > 1 && (
                        <h3 className="text-xs font-bold uppercase tracking-wide text-white/50 mb-2">{group.label}</h3>
                      )}
                      <div className="space-y-3">
                      {group.habits.map(habit => {
                        const isCompleted = habit.executions?.some(
                          e => e.executionDate.split('T')[0] === today && e.completed
                        );
                        const habitId = habit.originalEventId || habit.id;
                        const target = habit.dailyTarget || 0;
                        const amount = target ? loggedAmount(habit, habit.occurrenceDate || today) : 0;

                        return (
                          <div
                            key={habit.id}
                            draggable
                            onDragStart={(e) => handleDragStart(e, habitId)}
                            onDragOver={handleDragOver}
                            onDrop={(e) => handleDrop(e, habitId, todayHabits)}
                            onDragEnd={handleDragEnd}
                            className={`flex items-center justify-between p-4 rounded-xl transition-colors cursor-grab active:cursor-grabbing ${
                              isCompleted ? 'bg-emerald-500/20 border border-emerald-400/30' : 'bg-white/5 hover:bg-white/10'
                            } ${draggedId === habitId ? 'opacity-50' : ''}`}
                          >
                            <div className="flex items-center gap-3">
                              <svg className="w-4 h-4 text-white/30 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                              </svg>
                              <button
                                onClick={() => (target ? setHabitAmount(habit, isCompleted ? 0 : target) : toggleHabitCompletion(habit))}
                                aria-label={`Marcar ${habit.title}`}
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
                                {(habit.categoryType || habit.category || habit.label) && (
                                  <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                                    {habit.categoryType && <CategoryTypeBadge categoryType={habit.categoryType} />}
                                    {habit.category && <CategoryBadge category={habit.category} />}
                                    {habit.label && <LabelBadge label={habit.label} />}
                                  </div>
                                )}
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              {target > 0 && (
                                <HabitAmountStepper
                                  value={amount}
                                  target={target}
                                  unit={habit.targetUnit}
                                  onChange={(v) => setHabitAmount(habit, v)}
                                />
                              )}
                              {selectedCalendarId === null && habit.calendarId && (
                                <span className="text-xs text-white/40 bg-white/5 px-2 py-1 rounded">
                                  {getCalendarName(habit.calendarId)}
                                </span>
                              )}
                              <span className="text-white/40 text-sm">{habit.startTime}</span>
                              <button
                                onClick={() => setEditingHabit(habit)}
                                className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                                title="Editar habito"
                              >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                </svg>
                              </button>
                            </div>
                          </div>
                        );
                      })}
                      </div>
                      </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Fixed Habits (Daily/Monthly) */}
                {fixedHabits.length > 0 && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    <h2 className="text-lg font-semibold text-white mb-4">Habitos Fixos</h2>
                    <div className="space-y-3">
                      {fixedHabits.map(habit => {
                        const habitId = habit.originalEventId || habit.id;
                        return (
                          <div
                            key={habit.id}
                            draggable
                            onDragStart={(e) => handleDragStart(e, habitId)}
                            onDragOver={handleDragOver}
                            onDrop={(e) => handleDrop(e, habitId, fixedHabits)}
                            onDragEnd={handleDragEnd}
                            className={`flex flex-wrap items-center justify-between p-4 bg-white/5 rounded-xl hover:bg-white/10 transition-colors cursor-grab active:cursor-grabbing ${
                              draggedId === habitId ? 'opacity-50' : ''
                            }`}
                          >
                            <div className="flex items-center gap-3">
                              <svg className="w-4 h-4 text-white/30 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                              </svg>
                              <div>
                                <button
                                  type="button"
                                  onClick={() => setExpandedHabitId(expandedHabitId === habitId ? null : habitId)}
                                  className="font-medium text-white hover:underline text-left"
                                  title="Ver estatísticas"
                                >
                                  {habit.title}
                                </button>
                                <p className="text-sm text-white/50">
                                  {habit.recurrenceFrequency === 'daily' && 'Diario'}
                                  {habit.recurrenceFrequency === 'weekly' && 'Semanal'}
                                  {habit.recurrenceFrequency === 'monthly' && 'Mensal'}
                                  {habit.startTime && ` • ${habit.startTime}`}
                                  {selectedCalendarId === null && habit.calendarId && ` • ${getCalendarName(habit.calendarId)}`}
                                </p>
                                {(habit.categoryType || habit.category || habit.label) && (
                                  <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                                    {habit.categoryType && <CategoryTypeBadge categoryType={habit.categoryType} />}
                                    {habit.category && <CategoryBadge category={habit.category} />}
                                    {habit.label && <LabelBadge label={habit.label} />}
                                  </div>
                                )}
                              </div>
                            </div>
                            <div className="flex items-center gap-3">
                              {habit.streak > 0 && (
                                <span className="flex items-center gap-1 px-2 py-1 bg-orange-500/20 text-orange-400 rounded-lg text-sm">
                                  <span>🔥</span>
                                  <span>{habit.streak} dias</span>
                                </span>
                              )}
                              <button
                                onClick={() => setEditingHabit(habit)}
                                className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                                title="Editar habito"
                              >
                                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                </svg>
                              </button>
                            </div>
                            {expandedHabitId === habitId && statsById.get(habitId) && (
                              <div className="basis-full">
                                <HabitDetailPanel
                                  stats={statsById.get(habitId)!}
                                  onTogglePause={() => togglePause(habitId, false)}
                                />
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}

                {/* Flexible Weekly Habits */}
                {flexibleHabitsWithProgress.length > 0 && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    {/* Header with week navigation */}
                    <div className="flex items-center justify-between mb-4">
                      <h2 className="text-lg font-semibold text-white">Habitos Semanais Flexiveis</h2>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => setWeekOffset(prev => Math.max(prev - 1, -12))}
                          disabled={weekOffset <= -12}
                          className="p-1.5 text-white/60 hover:text-white hover:bg-white/10 rounded-lg transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                          title="Semana anterior"
                        >
                          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                          </svg>
                        </button>
                        <button
                          onClick={() => setWeekOffset(0)}
                          className={`px-3 py-1 rounded-lg text-sm transition-colors ${
                            weekOffset === 0
                              ? 'bg-purple-600/50 text-white'
                              : 'bg-white/10 text-white/70 hover:bg-white/20 hover:text-white'
                          }`}
                        >
                          {weekOffset === 0
                            ? 'Esta semana'
                            : `${formatDisplayDate(navigatedWeekDays[0].date)} - ${formatDisplayDate(navigatedWeekDays[6].date)}`
                          }
                        </button>
                        <button
                          onClick={() => setWeekOffset(prev => Math.min(prev + 1, 4))}
                          disabled={weekOffset >= 4}
                          className="p-1.5 text-white/60 hover:text-white hover:bg-white/10 rounded-lg transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                          title="Proxima semana"
                        >
                          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </button>
                      </div>
                    </div>
                    <div className="space-y-4">
                      {flexibleHabitsWithProgress.map(habit => {
                        const habitId = habit.originalEventId || habit.id;
                        const progress = habit.progress;
                        const navigatedWeekStart = navigatedWeekDays[0].date;
                        const weekData = weekOffset === 0
                          ? progress?.currentWeek
                          : progress?.weekHistory?.find(w => w.weekStartDate === navigatedWeekStart);
                        const completed = weekData?.completedCount || 0;
                        const target = weekData?.targetCount || habit.weeklyTargetCount || 2;
                        const percentage = Math.min((completed / target) * 100, 100);
                        const isGoalMet = weekData?.isGoalMet || completed >= target;
                        const weeklyStreak = weekOffset === 0 ? (progress?.currentStreak || 0) : 0;
                        const completedDates = weekData?.completedDates || [];
                        const preferredDays = habit.weeklyPreferredDays || [];

                        return (
                          <div
                            key={habit.id}
                            draggable
                            onDragStart={(e) => handleDragStart(e, habitId)}
                            onDragOver={handleDragOver}
                            onDrop={(e) => handleDrop(e, habitId, flexibleHabitsWithProgress)}
                            onDragEnd={handleDragEnd}
                            className={`p-4 rounded-xl transition-colors cursor-grab active:cursor-grabbing ${
                              isGoalMet ? 'bg-emerald-500/20 border border-emerald-400/30' : 'bg-white/5 hover:bg-white/10'
                            } ${draggedId === habitId ? 'opacity-50' : ''}`}
                          >
                            {/* Header */}
                            <div className="flex items-center justify-between mb-3">
                              <div className="flex items-center gap-3">
                                <svg className="w-4 h-4 text-white/30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                                </svg>
                                <div>
                                  <p className={`font-medium ${isGoalMet ? 'text-emerald-300' : 'text-white'}`}>
                                    {habit.title}
                                  </p>
                                  <p className="text-sm text-white/50">
                                    {target}x por semana
                                    {habit.startTime && ` • ${habit.startTime}`}
                                    {selectedCalendarId === null && habit.calendarId && ` • ${getCalendarName(habit.calendarId)}`}
                                  </p>
                                  {(habit.categoryType || habit.category || habit.label) && (
                                    <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                                      {habit.categoryType && <CategoryTypeBadge categoryType={habit.categoryType} />}
                                      {habit.category && <CategoryBadge category={habit.category} />}
                                      {habit.label && <LabelBadge label={habit.label} />}
                                    </div>
                                  )}
                                </div>
                              </div>
                              <div className="flex items-center gap-3">
                                {isGoalMet && (
                                  <span className="flex items-center gap-1 px-2 py-1 bg-emerald-500/20 text-emerald-400 rounded-lg text-sm">
                                    <span>🏆</span>
                                  </span>
                                )}
                                {weeklyStreak > 1 && (
                                  <span className="flex items-center gap-1 px-2 py-1 bg-purple-500/20 text-purple-400 rounded-lg text-sm">
                                    <span>🔥 {weeklyStreak} semanas</span>
                                  </span>
                                )}
                                <span className={`text-lg font-bold ${isGoalMet ? 'text-emerald-400' : 'text-white'}`}>
                                  {completed}/{target}
                                </span>
                                <button
                                  onClick={() => setEditingHabit(habit)}
                                  className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                                  title="Editar habito"
                                >
                                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                  </svg>
                                </button>
                              </div>
                            </div>

                            {/* Week Days Grid */}
                            <div className="flex gap-1 mb-3">
                              {navigatedWeekDays.map((day) => {
                                const isDayCompleted = completedDates.includes(day.date);
                                const isPreferred = preferredDays.includes(day.dayCode);

                                return (
                                  <button
                                    key={day.date}
                                    onClick={() => toggleFlexibleHabitDay(habit, day.date, completedDates)}
                                    className={`flex-1 py-2 px-1 rounded-lg text-center transition-all ${
                                      isDayCompleted
                                        ? 'bg-emerald-500 text-white'
                                        : day.isToday
                                          ? 'bg-purple-500/30 text-white border border-purple-400/50'
                                          : 'bg-white/10 text-white/70 hover:bg-white/20'
                                    } ${isPreferred && !isDayCompleted ? 'ring-1 ring-purple-400/50' : ''}`}
                                    title={`${day.dayLabel} ${day.date.split('-').reverse().slice(0, 2).join('/')}${isPreferred ? ' (preferido)' : ''}`}
                                  >
                                    <div className="text-xs font-medium">{day.dayLabel}</div>
                                    <div className="text-[10px] opacity-70">{day.date.split('-')[2]}</div>
                                    {isDayCompleted && (
                                      <svg className="w-4 h-4 mx-auto mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                                      </svg>
                                    )}
                                    {isPreferred && !isDayCompleted && (
                                      <div className="w-1.5 h-1.5 rounded-full bg-purple-400 mx-auto mt-1" />
                                    )}
                                  </button>
                                );
                              })}
                            </div>

                            {/* Progress Bar */}
                            <div className="w-full bg-white/10 rounded-full h-2 overflow-hidden">
                              <div
                                className={`h-full rounded-full transition-all duration-300 ${
                                  isGoalMet ? 'bg-emerald-500' : 'bg-purple-500'
                                }`}
                                style={{ width: `${percentage}%` }}
                              />
                            </div>

                            {/* Legend and days remaining */}
                            <div className="mt-2 flex items-center justify-between text-xs text-white/40">
                              <div className="flex items-center gap-3">
                                <span className="flex items-center gap-1">
                                  <div className="w-2 h-2 rounded-full bg-emerald-500" />
                                  Concluido
                                </span>
                                {preferredDays.length > 0 && (
                                  <span className="flex items-center gap-1">
                                    <div className="w-2 h-2 rounded-full bg-purple-400" />
                                    Preferido
                                  </span>
                                )}
                              </div>
                              {!isGoalMet && weekOffset === 0 && weekData?.daysRemaining !== undefined && weekData.daysRemaining > 0 && (
                                <span>
                                  {target - completed} {target - completed === 1 ? 'vez faltando' : 'vezes faltando'} • {weekData.daysRemaining} dias
                                </span>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}

                {/* Habits to quit */}
                {quitHabits.length > 0 && (
                  <div>
                    <h2 className="text-lg font-semibold text-white mb-3">Para largar</h2>
                    <div className="grid gap-4 sm:grid-cols-2">
                      {quitHabits.map(habit => {
                        const habitId = habit.originalEventId || habit.id;
                        return (
                          <QuitHabitCard
                            key={habitId}
                            habit={habit}
                            stats={statsById.get(habitId)}
                            expanded={expandedHabitId === habitId}
                            onToggleExpanded={() => setExpandedHabitId(expandedHabitId === habitId ? null : habitId)}
                            onEdit={() => setEditingHabit(habit)}
                            onSlip={(trigger, note) => logSlip(habit, trigger, note)}
                            onUndoSlip={() => undoSlip(habit)}
                            onTogglePause={() => togglePause(habitId, false)}
                          />
                        );
                      })}
                    </div>
                  </div>
                )}

                {/* All habits heatmap */}
                {overallHeatmap.length > 0 && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    <h2 className="text-lg font-semibold text-white mb-4">Todos os hábitos, últimas 16 semanas</h2>
                    <HabitHeatmap cells={overallHeatmap} tone="purple" />
                  </div>
                )}

                {/* Ideas */}
                {HABIT_IDEAS.some(i => !existingTitles.has(i.title.toLowerCase())) && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    <h2 className="text-lg font-semibold text-white mb-3">Ideias</h2>
                    <div className="flex flex-wrap gap-2">
                      {HABIT_IDEAS.filter(i => !existingTitles.has(i.title.toLowerCase())).map(idea => (
                        <button
                          key={idea.title}
                          type="button"
                          onClick={() => createFromIdea(idea)}
                          className="px-3 py-1.5 rounded-full text-sm border border-dashed border-white/30 text-white/70 hover:border-white hover:text-white"
                        >
                          {idea.habitKind === 'QUIT' ? 'Largar: ' : '+ '}
                          {idea.title}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {/* Paused */}
                {pausedHabits.length > 0 && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    <h2 className="text-lg font-semibold text-white mb-3">Pausados</h2>
                    <div className="space-y-2">
                      {pausedHabits.map(habit => {
                        const habitId = habit.originalEventId || habit.id;
                        return (
                          <div key={habitId} className="flex items-center justify-between">
                            <span className="text-white/70">{habit.title}</span>
                            <button
                              type="button"
                              onClick={() => togglePause(habitId, true)}
                              className="px-3 py-1 rounded-lg text-sm bg-white/10 text-white/80 hover:bg-white/20"
                            >
                              Retomar
                            </button>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}

                {/* Empty state */}
                {uniqueHabits.length === 0 && (
                  <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-6 border border-white/20">
                    <p className="text-white/60 text-center py-4">Nenhum habito cadastrado</p>
                  </div>
                )}
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
                      {pendingTodos.map(todo => {
                          const priority = getPriorityLabel(todo.priority);
                          const dueStatus = getDueDateStatus(todo.dueDate);

                          return (
                            <div
                              key={todo.id}
                              draggable
                              onDragStart={(e) => handleDragStart(e, todo.id)}
                              onDragOver={handleDragOver}
                              onDrop={(e) => handleDrop(e, todo.id, pendingTodos)}
                              onDragEnd={handleDragEnd}
                              className={`flex items-center justify-between p-4 bg-white/5 rounded-xl hover:bg-white/10 transition-colors cursor-grab active:cursor-grabbing ${
                                draggedId === todo.id ? 'opacity-50' : ''
                              }`}
                            >
                              <div className="flex items-center gap-3">
                                <svg className="w-4 h-4 text-white/30 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                                </svg>
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
                                    {todo.label && (
                                      <LabelBadge label={todo.label} />
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
                                <button
                                  onClick={() => setEditingTodo(todo)}
                                  className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                                  title="Editar tarefa"
                                >
                                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                                  </svg>
                                </button>
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
                            draggable
                            onDragStart={(e) => handleDragStart(e, todo.id)}
                            onDragOver={handleDragOver}
                            onDrop={(e) => handleDrop(e, todo.id, completedTodos)}
                            onDragEnd={handleDragEnd}
                            className={`flex items-center justify-between p-4 bg-emerald-500/10 rounded-xl cursor-grab active:cursor-grabbing ${
                              draggedId === todo.id ? 'opacity-50' : ''
                            }`}
                          >
                            <div className="flex items-center gap-3">
                              <svg className="w-4 h-4 text-white/30 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                              </svg>
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
                                  {todo.label && (
                                    <LabelBadge label={todo.label} />
                                  )}
                                </div>
                              </div>
                            </div>
                            <button
                              onClick={() => setEditingTodo(todo)}
                              className="p-1.5 text-white/40 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                              title="Editar tarefa"
                            >
                              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                              </svg>
                            </button>
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

      {/* Create Modal */}
      <CreateHabitTodoModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onCreated={fetchData}
        eventType={activeTab === 'habits' ? 'HABIT' : 'TODO'}
        calendars={calendars}
        selectedCalendarId={selectedCalendarId}
      />

      {/* Edit Todo Modal */}
      <EditTodoModal
        isOpen={editingTodo !== null}
        onClose={() => setEditingTodo(null)}
        onUpdated={fetchData}
        todo={editingTodo}
        calendars={calendars}
      />

      {/* Edit Habit Modal */}
      <EditHabitModal
        isOpen={editingHabit !== null}
        onClose={() => setEditingHabit(null)}
        onUpdated={fetchData}
        habit={editingHabit}
        calendars={calendars}
      />
    </div>
  );
}
