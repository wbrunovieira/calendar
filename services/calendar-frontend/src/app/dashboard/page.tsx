'use client';

import { useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import type { Event, Calendar, HabitStats, FlexibleHabitProgress } from '@/types/calendar';

const formatLocalDate = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const getWeekDays = () => {
  const today = new Date();
  const days = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(d.getDate() - i);
    days.push(formatLocalDate(d));
  }
  return days;
};

const getMonthDays = () => {
  const today = new Date();
  const days = [];
  for (let i = 29; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(d.getDate() - i);
    days.push(formatLocalDate(d));
  }
  return days;
};

const getDayLabel = (dateStr: string) => {
  const [year, month, day] = dateStr.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  return date.toLocaleDateString('pt-BR', { weekday: 'short' }).slice(0, 3);
};

const getShortDate = (dateStr: string) => {
  const [, month, day] = dateStr.split('-').map(Number);
  return `${day}/${month}`;
};

export default function HabitsDashboardPage() {
  const [habits, setHabits] = useState<Event[]>([]);
  const [habitStats, setHabitStats] = useState<HabitStats[]>([]);
  const [flexibleProgress, setFlexibleProgress] = useState<FlexibleHabitProgress[]>([]);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');
  const [loading, setLoading] = useState(true);

  const today = formatLocalDate(new Date());
  const weekDays = useMemo(() => getWeekDays(), []);
  const monthDays = useMemo(() => getMonthDays(), []);

  // Fetch data
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const [calendarsData, habitsData, statsData, flexibleData] = await Promise.all([
          api.calendars.list(),
          api.events.listHabits({
            startDate: monthDays[0],
            endDate: today,
            ...(selectedCalendarId && { calendarId: selectedCalendarId }),
          }),
          api.events.getHabitsStats(
            selectedCalendarId ? { calendarId: selectedCalendarId } : undefined
          ),
          api.events.getWeeklyProgress(
            selectedCalendarId ? { calendarId: selectedCalendarId } : undefined
          ),
        ]);

        setCalendars(calendarsData || []);
        setHabits(habitsData || []);
        setHabitStats(statsData || []);
        setFlexibleProgress(flexibleData || []);
      } catch (error) {
        console.error('Error fetching data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [selectedCalendarId, today, monthDays]);

  // Calculate daily completions for the week
  const weeklyData = useMemo(() => {
    const data: { date: string; total: number; completed: number }[] = [];

    for (const day of weekDays) {
      const dayHabits = habits.filter(h => {
        const occDate = h.occurrenceDate || h.startDate?.split('T')[0];
        return occDate === day;
      });

      const completed = dayHabits.filter(h =>
        h.executions?.some(e => {
          const execDate = typeof e.executionDate === 'string'
            ? e.executionDate.split('T')[0]
            : new Date(e.executionDate).toISOString().split('T')[0];
          return execDate === day && e.completed;
        })
      ).length;

      data.push({
        date: day,
        total: dayHabits.length,
        completed,
      });
    }

    return data;
  }, [habits, weekDays]);

  // Calculate monthly completions
  const monthlyData = useMemo(() => {
    const data: { date: string; total: number; completed: number }[] = [];

    for (const day of monthDays) {
      const dayHabits = habits.filter(h => {
        const occDate = h.occurrenceDate || h.startDate?.split('T')[0];
        return occDate === day;
      });

      const completed = dayHabits.filter(h =>
        h.executions?.some(e => {
          const execDate = typeof e.executionDate === 'string'
            ? e.executionDate.split('T')[0]
            : new Date(e.executionDate).toISOString().split('T')[0];
          return execDate === day && e.completed;
        })
      ).length;

      data.push({
        date: day,
        total: dayHabits.length,
        completed,
      });
    }

    return data;
  }, [habits, monthDays]);

  // Summary stats
  const summaryStats = useMemo(() => {
    const weekTotal = weeklyData.reduce((sum, d) => sum + d.total, 0);
    const weekCompleted = weeklyData.reduce((sum, d) => sum + d.completed, 0);
    const monthTotal = monthlyData.reduce((sum, d) => sum + d.total, 0);
    const monthCompleted = monthlyData.reduce((sum, d) => sum + d.completed, 0);

    // Today's stats
    const todayData = weeklyData.find(d => d.date === today);
    const todayTotal = todayData?.total || 0;
    const todayCompleted = todayData?.completed || 0;

    // Best streak from habitStats
    const bestStreak = Math.max(...habitStats.map(s => s.longestStreak), 0);
    const currentStreak = Math.max(...habitStats.map(s => s.currentStreak), 0);

    return {
      todayTotal,
      todayCompleted,
      todayPercentage: todayTotal > 0 ? Math.round((todayCompleted / todayTotal) * 100) : 0,
      weekTotal,
      weekCompleted,
      weekPercentage: weekTotal > 0 ? Math.round((weekCompleted / weekTotal) * 100) : 0,
      monthTotal,
      monthCompleted,
      monthPercentage: monthTotal > 0 ? Math.round((monthCompleted / monthTotal) * 100) : 0,
      bestStreak,
      currentStreak,
    };
  }, [weeklyData, monthlyData, habitStats, today]);

  // Get color based on percentage
  const getBarColor = (completed: number, total: number) => {
    if (total === 0) return 'bg-white/10';
    const pct = (completed / total) * 100;
    if (pct === 100) return 'bg-emerald-500';
    if (pct >= 70) return 'bg-blue-500';
    if (pct >= 40) return 'bg-yellow-500';
    if (pct > 0) return 'bg-orange-500';
    return 'bg-red-500/50';
  };

  const getHeatmapColor = (completed: number, total: number) => {
    if (total === 0) return 'bg-white/5';
    const pct = (completed / total) * 100;
    if (pct === 100) return 'bg-emerald-500';
    if (pct >= 70) return 'bg-emerald-500/70';
    if (pct >= 40) return 'bg-emerald-500/40';
    if (pct > 0) return 'bg-emerald-500/20';
    return 'bg-red-500/30';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#0a0118] via-[#1a0a2e] to-[#16213e] text-white p-6">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <Link
            href="/"
            className="inline-flex items-center gap-2 text-white/60 hover:text-white transition-colors mb-4"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            <span>Voltar ao Calendario</span>
          </Link>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent">
            Dashboard de Habitos
          </h1>
          <p className="text-white/60 mt-1">Acompanhe seu progresso e mantenha a consistencia</p>
        </div>

        {/* Calendar Filter */}
        <div className="mb-6">
          <select
            value={selectedCalendarId}
            onChange={(e) => setSelectedCalendarId(e.target.value)}
            className="bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
          >
            <option value="">Todos os Perfis</option>
            {calendars.map((cal) => (
              <option key={cal.id} value={cal.id}>{cal.name}</option>
            ))}
          </select>
        </div>

        {loading ? (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500"></div>
            <p className="mt-4 text-white/60">Carregando estatisticas...</p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Summary Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="bg-white/10 backdrop-blur-sm rounded-xl p-4 border border-white/20">
                <div className="text-3xl mb-2">📅</div>
                <div className="text-2xl font-bold text-white">
                  {summaryStats.todayCompleted}/{summaryStats.todayTotal}
                </div>
                <div className="text-white/60 text-sm">Hoje</div>
                <div className="mt-2 h-2 bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-emerald-500 transition-all"
                    style={{ width: `${summaryStats.todayPercentage}%` }}
                  />
                </div>
                <div className="text-right text-xs text-white/50 mt-1">{summaryStats.todayPercentage}%</div>
              </div>

              <div className="bg-white/10 backdrop-blur-sm rounded-xl p-4 border border-white/20">
                <div className="text-3xl mb-2">📊</div>
                <div className="text-2xl font-bold text-white">
                  {summaryStats.weekCompleted}/{summaryStats.weekTotal}
                </div>
                <div className="text-white/60 text-sm">Semana</div>
                <div className="mt-2 h-2 bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all"
                    style={{ width: `${summaryStats.weekPercentage}%` }}
                  />
                </div>
                <div className="text-right text-xs text-white/50 mt-1">{summaryStats.weekPercentage}%</div>
              </div>

              <div className="bg-white/10 backdrop-blur-sm rounded-xl p-4 border border-white/20">
                <div className="text-3xl mb-2">📈</div>
                <div className="text-2xl font-bold text-white">
                  {summaryStats.monthCompleted}/{summaryStats.monthTotal}
                </div>
                <div className="text-white/60 text-sm">Mes (30 dias)</div>
                <div className="mt-2 h-2 bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-purple-500 transition-all"
                    style={{ width: `${summaryStats.monthPercentage}%` }}
                  />
                </div>
                <div className="text-right text-xs text-white/50 mt-1">{summaryStats.monthPercentage}%</div>
              </div>

              <div className="bg-white/10 backdrop-blur-sm rounded-xl p-4 border border-white/20">
                <div className="text-3xl mb-2">🔥</div>
                <div className="text-2xl font-bold text-orange-400">{summaryStats.currentStreak}</div>
                <div className="text-white/60 text-sm">Sequencia Atual</div>
                <div className="mt-2 text-xs text-white/50">
                  Recorde: {summaryStats.bestStreak} dias
                </div>
              </div>
            </div>

            {/* Flexible Habits Weekly Progress */}
            {flexibleProgress.length > 0 && (
              <div className="bg-gradient-to-r from-purple-500/20 via-pink-500/20 to-orange-500/20 backdrop-blur-sm rounded-xl p-6 border border-purple-400/30">
                <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
                  <span>🎯</span>
                  <span>Habitos Flexiveis - Progresso Semanal</span>
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {flexibleProgress.map((habit) => {
                    const percentage = habit.weekProgress.targetCount > 0
                      ? Math.round((habit.weekProgress.completedCount / habit.weekProgress.targetCount) * 100)
                      : 0;
                    const isGoalMet = habit.weekProgress.isGoalMet;

                    return (
                      <div
                        key={habit.habitId}
                        className={`p-4 rounded-xl ${isGoalMet ? 'bg-emerald-500/20 border border-emerald-400/30' : 'bg-white/10 border border-white/20'}`}
                      >
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-medium text-white">{habit.habitTitle}</span>
                          {isGoalMet && <span className="text-emerald-400">✓</span>}
                        </div>
                        <div className="flex items-center gap-3">
                          <div className="flex-1 h-3 bg-white/10 rounded-full overflow-hidden">
                            <div
                              className={`h-full transition-all ${isGoalMet ? 'bg-emerald-500' : 'bg-purple-500'}`}
                              style={{ width: `${Math.min(percentage, 100)}%` }}
                            />
                          </div>
                          <span className={`text-lg font-bold ${isGoalMet ? 'text-emerald-400' : 'text-white'}`}>
                            {habit.weekProgress.completedCount}/{habit.weekProgress.targetCount}
                          </span>
                        </div>
                        {habit.weekProgress.completedDates.length > 0 && (
                          <div className="mt-2 text-xs text-white/50">
                            Concluido: {habit.weekProgress.completedDates.map(d => {
                              const [, m, day] = d.split('-');
                              return `${day}/${m}`;
                            }).join(', ')}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Weekly Chart */}
            <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/20">
              <h2 className="text-lg font-semibold text-white mb-4">Ultimos 7 Dias (Habitos Fixos)</h2>
              <div className="flex items-end justify-between gap-2 h-48">
                {weeklyData.map((day) => {
                  const percentage = day.total > 0 ? (day.completed / day.total) * 100 : 0;
                  const isToday = day.date === today;

                  return (
                    <div key={day.date} className="flex-1 flex flex-col items-center">
                      <div className="text-xs text-white/50 mb-1">
                        {day.completed}/{day.total}
                      </div>
                      <div className="w-full h-32 bg-white/5 rounded-lg relative overflow-hidden">
                        <div
                          className={`absolute bottom-0 w-full rounded-lg transition-all ${getBarColor(day.completed, day.total)}`}
                          style={{ height: `${percentage}%` }}
                        />
                      </div>
                      <div className={`text-xs mt-2 ${isToday ? 'text-emerald-400 font-bold' : 'text-white/50'}`}>
                        {getDayLabel(day.date)}
                      </div>
                      <div className={`text-xs ${isToday ? 'text-emerald-400' : 'text-white/30'}`}>
                        {getShortDate(day.date)}
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Legend */}
              <div className="mt-4 flex items-center justify-center gap-4 text-xs text-white/50">
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-emerald-500"></div>
                  <span>100%</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-blue-500"></div>
                  <span>70%+</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-yellow-500"></div>
                  <span>40%+</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-orange-500"></div>
                  <span>&lt;40%</span>
                </div>
              </div>
            </div>

            {/* Monthly Heatmap */}
            <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/20">
              <h2 className="text-lg font-semibold text-white mb-4">Ultimos 30 Dias</h2>
              <div className="grid grid-cols-10 gap-1">
                {monthlyData.map((day) => {
                  const isToday = day.date === today;
                  return (
                    <div
                      key={day.date}
                      className={`aspect-square rounded ${getHeatmapColor(day.completed, day.total)} ${isToday ? 'ring-2 ring-emerald-400' : ''}`}
                      title={`${getShortDate(day.date)}: ${day.completed}/${day.total}`}
                    />
                  );
                })}
              </div>
              <div className="mt-4 flex items-center justify-between text-xs text-white/50">
                <span>{getShortDate(monthDays[0])}</span>
                <div className="flex items-center gap-1">
                  <span>Menos</span>
                  <div className="w-3 h-3 rounded bg-white/5"></div>
                  <div className="w-3 h-3 rounded bg-emerald-500/20"></div>
                  <div className="w-3 h-3 rounded bg-emerald-500/40"></div>
                  <div className="w-3 h-3 rounded bg-emerald-500/70"></div>
                  <div className="w-3 h-3 rounded bg-emerald-500"></div>
                  <span>Mais</span>
                </div>
                <span>{getShortDate(monthDays[monthDays.length - 1])}</span>
              </div>
            </div>

            {/* Individual Habit Stats */}
            {habitStats.length > 0 && (
              <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 border border-white/20">
                <h2 className="text-lg font-semibold text-white mb-4">Por Habito</h2>
                <div className="space-y-3">
                  {habitStats.map((stat) => (
                    <div key={stat.habitId} className="flex items-center justify-between p-3 bg-white/5 rounded-lg">
                      <div>
                        <div className="font-medium text-white">{stat.habitTitle}</div>
                        <div className="text-xs text-white/50">
                          {stat.totalCompletions} conclusoes totais
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <div className="text-right">
                          <div className="text-sm text-white/70">{stat.monthlyCompletionRate}%</div>
                          <div className="text-xs text-white/40">no mes</div>
                        </div>
                        {stat.currentStreak > 0 && (
                          <div className="flex items-center gap-1 px-2 py-1 bg-orange-500/20 rounded-lg">
                            <span>🔥</span>
                            <span className="text-orange-400 font-medium">{stat.currentStreak}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Link to Habits Page */}
            <div className="text-center">
              <Link
                href="/habits"
                className="inline-flex items-center gap-2 px-6 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition-colors"
              >
                <span>Gerenciar Habitos</span>
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </Link>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
